// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package soci

import (
	"context"
	"errors"
	"fmt"
	"oras.land/oras-go/v2/content/oci"
	"path"

	"github.com/awslabs/soci-snapshotter/soci"
	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/content/local"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/platforms"

	"github.com/aws-ia/cfn-aws-soci-index-builder/soci-index-generator-lambda/utils/log"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

type EcrSoci struct {
	containerdStore content.Store     // containerd's content store
	OciStore        oci.Store         // oci's content store
	artifactsDb     *soci.ArtifactsDb // SOCI artifacts db
}

const artifactsStoreName = "store"
const artifactsDbName = "artifacts.db"

// Authenticate with  ECR and initialize the ECR and SOCI wrapper
func Init(ctx context.Context, registryUrl string, dataDir string) (*EcrSoci, error) {
	containerdStore, err := initContainerdStore(dataDir)
	if err != nil {
		return nil, err
	}

	ociStore, err := initOciStore(ctx, dataDir)
	if err != nil {
		return nil, err
	}

	artifactsDb, err := initSociArtifactsDb(dataDir)
	if err != nil {
		return nil, err
	}

	return &EcrSoci{containerdStore: *containerdStore, OciStore: *ociStore, artifactsDb: artifactsDb}, nil
}

// Init containerd store
func initContainerdStore(dataDir string) (*content.Store, error) {
	containerdStore, err := local.NewStore(path.Join(dataDir, artifactsStoreName))
	return &containerdStore, err
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
func (ecrSoci *EcrSoci) BuildIndex(ctx context.Context, image images.Image) (*ocispec.Descriptor, error) {
	log.Info(ctx, "Building SOCI index")
	platform := platforms.DefaultSpec() // TODO: make this a user option

	builder, err := soci.NewIndexBuilder(ecrSoci.containerdStore, &ecrSoci.OciStore, ecrSoci.artifactsDb, soci.WithMinLayerSize(0), soci.WithPlatform(platform), soci.WithLegacyRegistrySupport)
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
	err = soci.WriteSociIndex(ctx, index, &ecrSoci.OciStore, ecrSoci.artifactsDb)
	if err != nil {
		return nil, err
	}

	// Get SOCI indices for the image from the oras store
	// The most recent one is stored last
	// TODO: consider making soci's WriteSociIndex to return the descriptor directly
	indexDescriptorInfos, err := soci.GetIndexDescriptorCollection(ctx, ecrSoci.containerdStore, ecrSoci.artifactsDb, image, []ocispec.Platform{platform})
	if err != nil {
		return nil, err
	}
	if len(indexDescriptorInfos) == 0 {
		return nil, errors.New("No SOCI indices found in oras store")
	}
	return &indexDescriptorInfos[len(indexDescriptorInfos)-1].Descriptor, nil
}
