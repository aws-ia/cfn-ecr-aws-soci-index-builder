// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package ecrsoci

import (
	"context"
	"github.com/containerd/containerd/images"
	"os"
	"testing"
)

// Test all functions of of ecrsoci package
// TODO: turn this into an integration test
func TestAll(t *testing.T) {
	ctx := context.Background()

	registryHost := os.Getenv("REGISTRY_HOST")

	ecrSoci, err := Init(ctx, registryHost)

	desc, err := ecrSoci.Pull(ctx, "redis", "latest")
	if err != nil {
		panic(err)
	}

	image := images.Image{
		Name:   "redis:latest",
		Target: *desc,
	}

	ecrSoci.CreateIndex(ctx, image)

	ecrSoci.PushIndex(ctx, image, "redis")
}
