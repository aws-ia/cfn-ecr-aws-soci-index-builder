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

	"github.com/aws-ia/cfn-aws-soci-index-builder/soci-index-generator-lambda/ecrsoci"
	"github.com/aws-ia/cfn-aws-soci-index-builder/soci-index-generator-lambda/events"
	"github.com/aws-ia/cfn-aws-soci-index-builder/soci-index-generator-lambda/utils/fs"
	"github.com/aws-ia/cfn-aws-soci-index-builder/soci-index-generator-lambda/utils/log"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/containerd/containerd/images"
)

func HandleRequest(ctx context.Context, event events.ECRImageActionEvent) (string, error) {
	ctx, err := validateEvent(ctx, event)
	if err != nil {
		return lambdaError(ctx, "ECRImageActionEvent validation error", err)
	}

	repo := event.Detail.RepositoryName
	digest := event.Detail.ImageDigest
	registryUrl := buildEcrRegistryUrl(event)
	ctx = context.WithValue(ctx, "RegistryURL", registryUrl)

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

	ecrSoci, err := ecrsoci.Init(ctx, registryUrl, dataDir)
	if err != nil {
		return lambdaError(ctx, "Registry client or local storage initialization error", err)
	}

	desc, err := ecrSoci.Pull(ctx, repo, digest)
	if err != nil {
		return lambdaError(ctx, "Image pull error", err)
	}

	image := images.Image{
		Name:   repo + "@" + digest,
		Target: *desc,
	}

	indexDescriptor, err := ecrSoci.BuildIndex(ctx, image)
	if err != nil {
		return lambdaError(ctx, "SOCI index build error", err)
	}
	ctx = context.WithValue(ctx, "SOCIIndexDigest", indexDescriptor.Digest.String())

	err = ecrSoci.PushIndex(ctx, *indexDescriptor, repo)
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

// Log and return the lambda handler error
func lambdaError(ctx context.Context, msg string, err error) (string, error) {
	log.Error(ctx, msg, err)
	return msg, err
}

func main() {
	lambda.Start(HandleRequest)
}
