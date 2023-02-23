// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package ecrsoci can pull down images from a private ECR repository, build a SOCI index from an image, and push a SOCI index to a private ECR repository
package ecrsoci

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"

	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/oci"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"

	"github.com/awslabs/soci-snapshotter/soci"
	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/content/local"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/platforms"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

type EcrSoci struct {
	registry        *remote.Registry // remote ECR registry
	containerdStore content.Store    // containerd's content store
	ociStore        oci.Store        // oci's content store
}

const artifactsStoreName = "/tmp/store"
const artifactsDbDir = "/tmp/soci"

var artifactsDbPath = path.Join(artifactsDbDir, "artifacts.db")

var RegistryNotSupportingOciArtifacts = errors.New("Registry does not support OCI artifacts")

// Authenticate with  ECR and initialize the ECR and SOCI wrapper
func Init(ctx context.Context, registryUrl string) (*EcrSoci, error) {
	registry, err := initRegistry(registryUrl)
	if err != nil {
		return nil, err
	}

	containerdStore, err := initContainerdStore()
	if err != nil {
		return nil, err
	}

	ociStore, err := initOciStore(ctx)
	if err != nil {
		return nil, err
	}

	return &EcrSoci{registry: registry, containerdStore: *containerdStore, ociStore: *ociStore}, nil
}

// Init containerd store
func initContainerdStore() (*content.Store, error) {
	if _, err := os.Stat(artifactsDbDir); os.IsNotExist(err) {
		if err = os.Mkdir(artifactsDbDir, 0711); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	containerdStore, err := local.NewStore(artifactsStoreName)
	return &containerdStore, err
}

// Init OCI artifact store
func initOciStore(ctx context.Context) (*oci.Store, error) {
	return oci.NewWithContext(ctx, artifactsStoreName)
}

// Initialize a remote registry
func initRegistry(registryUrl string) (*remote.Registry, error) {
	registry, err := remote.NewRegistry(registryUrl)
	if err != nil {
		return nil, err
	}
	if isEcrRegistry(registryUrl) {
		err := authorizeEcr(registry)
		if err != nil {
			return nil, err
		}
	}
	return registry, nil
}

// Check if a registry is an ECR registry
func isEcrRegistry(registryUrl string) bool {
	ecrRegistryUrlRegex := "\\d{12}\\.dkr\\.ecr\\.\\S+\\.amazonaws\\.com"
	match, err := regexp.MatchString(ecrRegistryUrlRegex, registryUrl)
	if err != nil {
		panic(err)
	}
	return match
}

// Authorize ECR registry
func authorizeEcr(ecrRegistry *remote.Registry) error {
	// getting ecr auth token
	input := &ecr.GetAuthorizationTokenInput{}
	var ecrClient *ecr.ECR
	ecrEndpoint := os.Getenv("ECR_ENDPOINT") // set this env var for custom, i.e. non default, aws ecr endpoint
	if ecrEndpoint != "" {
		ecrClient = ecr.New(session.New(&aws.Config{Endpoint: aws.String(ecrEndpoint)}))
	} else {
		ecrClient = ecr.New(session.New())
	}
	getAuthorizationTokenResponse, err := ecrClient.GetAuthorizationToken(input)
	if err != nil {
		return err
	}

	if len(getAuthorizationTokenResponse.AuthorizationData) == 0 {
		return errors.New("Couldn't authorize with ECR: empty authorization data returned")
	}

	ecrAuthorizationToken := getAuthorizationTokenResponse.AuthorizationData[0].AuthorizationToken
	if len(*ecrAuthorizationToken) == 0 {
		return errors.New("Couldn't authorize with ECR: empty authorization token returned")
	}

	ecrRegistry.RepositoryOptions.Client = &auth.Client{
		Header: http.Header{
			"Authorization": {"Basic " + *ecrAuthorizationToken},
			"User-Agent":    {"SOCI Index Builder (oras-go)"},
		},
	}
	return nil
}

// Pull an image from the remote registry
// imageReference can be either a digest or a tag
func (ecrSoci *EcrSoci) Pull(ctx context.Context, repositoryName string, imageReference string) (*ocispec.Descriptor, error) {
	repo, err := ecrSoci.registry.Repository(ctx, repositoryName)
	if err != nil {
		return nil, err
	}

	imageDescriptor, err := oras.Copy(ctx, repo, imageReference, &ecrSoci.ociStore, imageReference, oras.DefaultCopyOptions)
	if err != nil {
		return nil, err
	}

	return &imageDescriptor, nil
}

// Build soci index for an image and returns its ocispec.Descriptor
func (ecrSoci *EcrSoci) BuildIndex(ctx context.Context, image images.Image) (*ocispec.Descriptor, error) {
	platform := platforms.DefaultSpec() // TODO: make this a user option

	// Create new instance of an ArtifactDB
	artifactDb, err := soci.NewDB(artifactsDbPath)
	if err != nil {
		return nil, err
	}

	builder, err := soci.NewIndexBuilder(ecrSoci.containerdStore, &ecrSoci.ociStore, artifactDb, soci.WithMinLayerSize(0), soci.WithPlatform(platform), soci.WithLegacyRegistrySupport)
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
	err = soci.WriteSociIndex(ctx, index, &ecrSoci.ociStore, artifactDb)
	if err != nil {
		return nil, err
	}

	// Get SOCI indices for the image from the oras store
	// The most recent one is stored last
	// TODO: consider making soci's WriteSociIndex to return the descriptor directly
	indexDescriptorInfos, err := soci.GetIndexDescriptorCollection(ctx, ecrSoci.containerdStore, artifactDb, image, []ocispec.Platform{platform})
	if err != nil {
		return nil, err
	}
	if len(indexDescriptorInfos) == 0 {
		return nil, errors.New("No SOCI indices found in oras store")
	}
	return &indexDescriptorInfos[len(indexDescriptorInfos)-1].Descriptor, nil
}

// Push soci index to the remote registry
func (ecrSoci *EcrSoci) PushIndex(ctx context.Context, indexDesc ocispec.Descriptor, repositoryName string) error {

	repo, err := ecrSoci.registry.Repository(ctx, repositoryName)
	if err != nil {
		return err
	}

	err = oras.CopyGraph(ctx, &ecrSoci.ociStore, repo, indexDesc, oras.DefaultCopyGraphOptions)
	if err != nil {
		// TODO: There might be a better way to check if a registry supporting OCI or not
		if strings.Contains(err.Error(), "Response status code 405: unsupported: Invalid parameter at 'ImageManifest' failed to satisfy constraint: 'Invalid JSON syntax'") {
			fmt.Printf("[WARN] Error when pushing [err: %v]", err)
			return RegistryNotSupportingOciArtifacts
		}
		return err
	}
	return nil
}
