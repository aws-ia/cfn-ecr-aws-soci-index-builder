// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"errors"
	"github.com/aws-ia/cfn-aws-soci-index-builder/soci-index-generator-lambda/events"
	"github.com/aws-ia/cfn-aws-soci-index-builder/soci-index-generator-lambda/utils/fs"
	"github.com/aws-ia/cfn-aws-soci-index-builder/soci-index-generator-lambda/utils/log"
	registryutils "github.com/aws-ia/cfn-aws-soci-index-builder/soci-index-generator-lambda/utils/registry"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/containerd/containerd/images"
	"oras.land/oras-go/v2/content/oci"
	"path"

	"github.com/awslabs/soci-snapshotter/soci"
	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/content/local"
	"github.com/containerd/containerd/platforms"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

const artifactsStoreName = "store"
const artifactsDbName = "artifacts.db"

func HandleRequest(ctx context.Context, event events.ECRImageActionEvent) (string, error) {
	ctx, err := validateEvent(ctx, event)
	if err != nil {
		return lambdaError(ctx, "ECRImageActionEvent validation error", err)
	}

	repo := event.Detail.RepositoryName
	digest := event.Detail.ImageDigest
	registryUrl := buildEcrRegistryUrl(event)
	ctx = context.WithValue(ctx, "RegistryURL", registryUrl)

	registry, err := registryutils.Init(registryUrl)
	if err != nil {
		return lambdaError(ctx, "Remote registry initialization error", err)
	}

	err = validateImageDigest(registry, repo, digest)
	if err != nil {
		return lambdaError(ctx, "Remote image digest validation error", err)
	}

	// Directory in lambda storage to store images and SOCI artifacts
	dataDir, err := createTempDir(ctx)
	if err != nil {
		return lambdaError(ctx, "Directory create error", err)
	}
	defer cleanUp(ctx, dataDir)

	// The channel to signal the deadline monitor goroutine to exit early
	quitChannel := make(chan int)
	defer func() {
		quitChannel <- 1
	}()

	setDeadline(ctx, quitChannel, dataDir)

	ociStore, err := initOciStore(ctx, dataDir)
	if err != nil {
		return lambdaError(ctx, "OCI storage initialization error", err)
	}

	desc, err := registry.Pull(ctx, repo, ociStore, digest)
	if err != nil {
		return lambdaError(ctx, "Image pull error", err)
	}

	image := images.Image{
		Name:   repo + "@" + digest,
		Target: *desc,
	}

	indexDescriptor, err := buildIndex(ctx, dataDir, ociStore, image)
	if err != nil {
		return lambdaError(ctx, "SOCI index build error", err)
	}
	ctx = context.WithValue(ctx, "SOCIIndexDigest", indexDescriptor.Digest.String())

	err = registry.Push(ctx, ociStore, *indexDescriptor, repo)
	if err != nil {
		return lambdaError(ctx, "SOCI index push error", err)
	}

	log.Info(ctx, "Successfully built and pushed SOCI index")
	return "Successfully built and pushed SOCI index", nil
}

// Validate the given event, populating the context with relevant valid event properties
func validateEvent(ctx context.Context, event events.ECRImageActionEvent) (context.Context, error) {
	var errors []error

	if event.Source != "aws.ecr" {
		errors = append(errors, fmt.Errorf("The event's 'source' must be 'aws.ecr'"))
	}
	if event.Account == "" {
		errors = append(errors, fmt.Errorf("The event's 'account' must not be empty"))
	}
	if event.DetailType != "ECR Image Action" {
		errors = append(errors, fmt.Errorf("The event's 'detail-type' must be 'ECR Image Action'"))
	}
	if event.Detail.ActionType != "PUSH" {
		errors = append(errors, fmt.Errorf("The event's 'detail.action-type' must be 'PUSH'"))
	}
	if event.Detail.Result != "SUCCESS" {
		errors = append(errors, fmt.Errorf("The event's 'detail.result' must be 'SUCCESS'"))
	}
	if event.Detail.RepositoryName == "" {
		errors = append(errors, fmt.Errorf("The event's 'detail.repository-name' must not be empty"))
	}
	if event.Detail.ImageDigest == "" {
		errors = append(errors, fmt.Errorf("The event's 'detail.image-digest' must not be empty"))
	}

	validAccountId, err := regexp.MatchString(`[0-9]{12}`, event.Account)
	if err != nil {
		errors = append(errors, err)
	}
	if !validAccountId {
		errors = append(errors, fmt.Errorf("The event's 'account' must be a valid AWS account ID"))
	}

	validRepositoryName, err := regexp.MatchString(`(?:[a-z0-9]+(?:[._-][a-z0-9]+)*/)*[a-z0-9]+(?:[._-][a-z0-9]+)*`, event.Detail.RepositoryName)
	if err != nil {
		errors = append(errors, err)
	}
	if validRepositoryName {
		ctx = context.WithValue(ctx, "RepositoryName", event.Detail.RepositoryName)
	} else {
		errors = append(errors, fmt.Errorf("The event's 'detail.repository-name' must be a valid repository name"))
	}

	validImageDigest, err := regexp.MatchString(`[[A-Za-z][A-Za-z0-9]*(?:[-_+.][A-Za-z][A-Za-z0-9]*)*[:][A-Fa-f0-9]{32,}`, event.Detail.ImageDigest)
	if err != nil {
		errors = append(errors, err)
	}
	if validImageDigest {
		ctx = context.WithValue(ctx, "ImageDigest", event.Detail.ImageDigest)
	} else {
		errors = append(errors, fmt.Errorf("The event's 'detail.image-digest' must be a valid image digest"))
	}

	// missing/empty tag is OK
	if event.Detail.ImageTag != "" {
		validImageTag, err := regexp.MatchString(`[A-Za-z0-9_][A-Za-z0-9_.-]{0,127}`, event.Detail.ImageTag)
		if err != nil {
			errors = append(errors, err)
		}
		if validImageTag {
			ctx = context.WithValue(ctx, "ImageTag", event.Detail.ImageTag)
		} else {
			errors = append(errors, fmt.Errorf("The event's 'detail.image-tag' must be empty or a valid image tag"))
		}
	}

	if len(errors) == 0 {
		return ctx, nil
	} else {
		return ctx, errors[0]
	}
}

// Validate the given remote image digest
func validateImageDigest(registry *registryutils.Registry, repository string, digest string) error {
	mediaType, err := registry.GetMediaType(context.Background(), repository, digest)

	if err != nil {
		return err
	}

	if mediaType != registryutils.MediaTypeDockerManifest {
		return fmt.Errorf("Unexpected media type %s, expected %s", mediaType, registryutils.MediaTypeDockerManifest)
	}

	return nil
}

// Returns ecr registry url from an image action event
func buildEcrRegistryUrl(event events.ECRImageActionEvent) string {
	var awsDomain = ".amazonaws.com"
	if strings.HasPrefix(event.Region, "cn") {
		awsDomain = ".amazonaws.com.cn"
	}
	return event.Account + ".dkr.ecr." + event.Region + awsDomain
}

// Create a temp directory in /tmp
// The directory is prefixed by the Lambda's request id
func createTempDir(ctx context.Context) (string, error) {
	// free space in bytes
	freeSpace := fs.CalculateFreeSpace("/tmp")
	log.Info(ctx, fmt.Sprintf("There are %d bytes of free space in /tmp directory", freeSpace))
	if freeSpace < 6_000_000_000 {
		// this is problematic because we support images as big as 6GB
		log.Warn(ctx, fmt.Sprintf("Free space in /tmp is only %d bytes, which is less than 6GB", freeSpace))
	}

	log.Info(ctx, "Creating a directory to store images and SOCI artifacts")
	lambdaContext, _ := lambdacontext.FromContext(ctx)
	tempDir, err := os.MkdirTemp("/tmp", lambdaContext.AwsRequestID) // The temp dir name is prefixed by the request id
	return tempDir, err
}

// Clean up the data written by the Lambda
func cleanUp(ctx context.Context, dataDir string) {
	log.Info(ctx, fmt.Sprintf("Removing all files in %s", dataDir))
	if err := os.RemoveAll(dataDir); err != nil {
		log.Error(ctx, "Clean up error", err)
	}
}

// Set up deadline for the lambda to proactively clean up its data before the invocation timeout. We don't
// want to keep data in storage when the Lambda reaches its invocation timeout.
// This function creates a goroutine that will do cleanup when the invocation timeout is near.
// quitChannel is used for signaling that goroutine when the invocation ends naturally.
func setDeadline(ctx context.Context, quitChannel chan int, dataDir string) {
	// setting deadline as 10 seconds before lambda timeout.
	// reference: https://docs.aws.amazon.com/lambda/latest/dg/golang-context.html
	deadline, _ := ctx.Deadline()
	deadline = deadline.Add(-10 * time.Second)
	timeoutChannel := time.After(time.Until(deadline))
	go func() {
		for {
			select {
			case <-timeoutChannel:
				cleanUp(ctx, dataDir)
				log.Error(ctx, "Invocation timeout error", fmt.Errorf("Invocation timeout after 14 minutes and 50 seconds"))
				return
			case <-quitChannel:
				return
			}
		}
	}()
}

// Init containerd store
func initContainerdStore(dataDir string) (content.Store, error) {
	containerdStore, err := local.NewStore(path.Join(dataDir, artifactsStoreName))
	return containerdStore, err
}

// Init OCI artifact store
func initOciStore(ctx context.Context, dataDir string) (*oci.Store, error) {
	return oci.NewWithContext(ctx, path.Join(dataDir, artifactsStoreName))
}

// Init a new instance of SOCI artifacts DB
func initSociArtifactsDb(dataDir string) (*soci.ArtifactsDb, error) {
	artifactsDbPath := path.Join(dataDir, artifactsDbName)
	artifactsDb, err := soci.NewDB(artifactsDbPath)
	if err != nil {
		return nil, err
	}
	return artifactsDb, nil
}

// Build soci index for an image and returns its ocispec.Descriptor
func buildIndex(ctx context.Context, dataDir string, ociStore *oci.Store, image images.Image) (*ocispec.Descriptor, error) {
	log.Info(ctx, "Building SOCI index")
	platform := platforms.DefaultSpec() // TODO: make this a user option

	artifactsDb, err := initSociArtifactsDb(dataDir)
	if err != nil {
		return nil, err
	}

	containerdStore, err := initContainerdStore(dataDir)
	if err != nil {
		return nil, err
	}

	builder, err := soci.NewIndexBuilder(containerdStore, ociStore, artifactsDb, soci.WithMinLayerSize(0), soci.WithPlatform(platform), soci.WithLegacyRegistrySupport)
	if err != nil {
		return nil, err
	}

	// Build the SOCI index
	index, err := builder.Build(ctx, image)
	if err != nil {
		return nil, err
	}
	fmt.Println(index.ImageDigest)

	// Write the SOCI index to the oras store
	err = soci.WriteSociIndex(ctx, index, ociStore, artifactsDb)
	if err != nil {
		return nil, err
	}

	// Get SOCI indices for the image from the oras store
	// The most recent one is stored last
	// TODO: consider making soci's WriteSociIndex to return the descriptor directly
	indexDescriptorInfos, err := soci.GetIndexDescriptorCollection(ctx, containerdStore, artifactsDb, image, []ocispec.Platform{platform})
	if err != nil {
		return nil, err
	}
	if len(indexDescriptorInfos) == 0 {
		return nil, errors.New("No SOCI indices found in oras store")
	}
	return &indexDescriptorInfos[len(indexDescriptorInfos)-1].Descriptor, nil
}

// Log and return the lambda handler error
func lambdaError(ctx context.Context, msg string, err error) (string, error) {
	log.Error(ctx, msg, err)
	return msg, err
}

func main() {
	lambda.Start(HandleRequest)
}
