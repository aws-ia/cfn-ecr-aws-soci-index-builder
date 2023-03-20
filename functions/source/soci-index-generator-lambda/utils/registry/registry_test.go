// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"context"
	"testing"

	"github.com/aws/aws-lambda-go/lambdacontext"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

type ExpectedResponse struct {
	MediaType string
	Config    ocispec.Descriptor
}

func TestHeadManifest(t *testing.T) {
	doTest := func(registryUrl string, repository string, digestOrTag string, expected ExpectedResponse) {
		// making the test context
		lc := lambdacontext.LambdaContext{}
		lc.AwsRequestID = "abcd-1234-test-head-manifest"
		ctx := lambdacontext.NewContext(context.Background(), &lc)
		registry, err := Init(ctx, registryUrl)
		if err != nil {
			panic(err)
		}

		descriptor, err := registry.HeadManifest(context.Background(), repository, digestOrTag)
		if err != nil {
			panic(err)
		}
		if descriptor.MediaType != expected.MediaType {
			t.Fatalf("Incorrect manifest media type. Expected %s but got %s", expected.MediaType, descriptor.MediaType)
		}
	}

	expected := ExpectedResponse{
		MediaType: MediaTypeDockerManifestList,
	}
	doTest("public.ecr.aws", "docker/library/redis", "7", expected)

	expected = ExpectedResponse{
		MediaType: MediaTypeDockerManifestList,
	}
	doTest("public.ecr.aws", "lambda/python", "3.10", expected)

	expected = ExpectedResponse{
		MediaType: MediaTypeDockerManifest,
	}
	doTest("public.ecr.aws", "lambda/python", "3.10-x86_64", expected)

	expected = ExpectedResponse{
		MediaType: MediaTypeDockerManifest,
	}
	doTest("docker.io", "library/redis", "sha256:afd1957d6b59bfff9615d7ec07001afb4eeea39eb341fc777c0caac3fcf52187", expected)
}

func TestGetManifest(t *testing.T) {
	doTest := func(registryUrl string, repository string, digestOrTag string, expected ExpectedResponse) {
		// making the test context
		lc := lambdacontext.LambdaContext{}
		lc.AwsRequestID = "abcd-1234-test-get-manifest"
		ctx := lambdacontext.NewContext(context.Background(), &lc)
		registry, err := Init(ctx, registryUrl)
		if err != nil {
			panic(err)
		}

		manifest, err := registry.GetManifest(context.Background(), repository, digestOrTag)
		if err != nil {
			panic(err)
		}
		if manifest.MediaType != expected.MediaType {
			t.Fatalf("Incorrect manifest media type. Expected %s but got %s", expected.MediaType, manifest.MediaType)
		}

		if manifest.Config.MediaType != expected.Config.MediaType {
			t.Fatalf("Incorrect config's media type. Expected %s but got %s", expected.Config.MediaType, manifest.Config.MediaType)
		}
	}

	expected := ExpectedResponse{
		MediaType: MediaTypeDockerManifestList,
		Config: ocispec.Descriptor{
			MediaType: "",
		},
	}
	doTest("public.ecr.aws", "docker/library/redis", "7", expected)

	expected = ExpectedResponse{
		MediaType: MediaTypeDockerManifestList,
		Config: ocispec.Descriptor{
			MediaType: "",
		},
	}
	doTest("public.ecr.aws", "lambda/python", "3.10", expected)

	expected = ExpectedResponse{
		MediaType: MediaTypeDockerManifest,
		Config: ocispec.Descriptor{
			MediaType: MediaTypeDockerImageConfig,
		},
	}
	doTest("public.ecr.aws", "lambda/python", "3.10-x86_64", expected)

	expected = ExpectedResponse{
		MediaType: MediaTypeDockerManifest,
		Config: ocispec.Descriptor{
			MediaType: MediaTypeDockerImageConfig,
		},
	}
	doTest("docker.io", "library/redis", "sha256:afd1957d6b59bfff9615d7ec07001afb4eeea39eb341fc777c0caac3fcf52187", expected)
}

func TestIsImage(t *testing.T) {
	// making the test context
	lc := lambdacontext.LambdaContext{}
	lc.AwsRequestID = "abcd-1234-test-get-manifest"
	ctx := lambdacontext.NewContext(context.Background(), &lc)

	manifest := ocispec.Manifest{
		MediaType: MediaTypeDockerImageConfig,
		Config: ocispec.Descriptor{
			MediaType: "soci index",
		},
	}
	if isImage(ctx, manifest.MediaType, manifest) {
		t.Fatalf("soci index is not a valid config media type for an image")
	}

	manifest = ocispec.Manifest{
		MediaType: MediaTypeOCIManifest,
		Config: ocispec.Descriptor{
			MediaType: MediaTypeOCIImageConfig,
		},
	}
	if !isImage(ctx, manifest.MediaType, manifest) {
		t.Fatalf("The manifest above is a valid OCI image manifest")
	}
}
