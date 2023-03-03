// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package ecrsoci

import (
	"context"
	"os"
	"testing"

	"github.com/containerd/containerd/images"
)

// Test all functions of of ecrsoci package
// TODO: turn this into an integration test
func TestAll(t *testing.T) {
	ctx := context.Background()

	registryHost := os.Getenv("REGISTRY_HOST")

	ecrSoci, err := Init(ctx, registryHost, "/tmp")

	desc, err := ecrSoci.Pull(ctx, "redis", "latest")
	if err != nil {
		panic(err)
	}

	image := images.Image{
		Name:   "redis:latest",
		Target: *desc,
	}

	indexDescriptor, err := ecrSoci.BuildIndex(ctx, image)

	ecrSoci.PushIndex(ctx, *indexDescriptor, "redis")
}
