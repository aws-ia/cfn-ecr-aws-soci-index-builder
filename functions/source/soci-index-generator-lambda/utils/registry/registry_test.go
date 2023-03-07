// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"context"
	"fmt"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"testing"
)

func TestGetMediaType(t *testing.T) {
	do := func(registryUrl string, repository string, digestOrTag string, expectedMediaType string) {
		// making the test context
		lc := lambdacontext.LambdaContext{}
		lc.AwsRequestID = "abcd-1234"
		ctx := lambdacontext.NewContext(context.Background(), &lc)
		registry, err := Init(ctx, registryUrl)
		if err != nil {
			panic(err)
		}

		mediaType, err := registry.GetMediaType(context.Background(), repository, digestOrTag)
		if err != nil {
			panic(err)
		}
		fmt.Println(mediaType)
		if mediaType != expectedMediaType {
			t.Fatalf("Incorrect media type. Expected %s but got %s", expectedMediaType, mediaType)
		}
	}

	do("public.ecr.aws", "docker/library/redis", "7", MediaTypeDockerManifestList)
	do("public.ecr.aws", "lambda/python", "3.10", MediaTypeDockerManifestList)
	do("public.ecr.aws", "lambda/python", "3.10-x86_64", MediaTypeDockerManifest)
	do("docker.io", "library/redis", "sha256:afd1957d6b59bfff9615d7ec07001afb4eeea39eb341fc777c0caac3fcf52187", MediaTypeDockerManifest)
}
