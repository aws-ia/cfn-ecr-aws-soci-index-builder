// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/soci-index-generator-lambda/ecrsoci"
	"github.com/awslabs/soci-index-generator-lambda/events"
	"github.com/containerd/containerd/images"
)

func HandleRequest(ctx context.Context, event events.ECRImageActionEvent) (string, error) {
	actionType := event.Detail.ActionType
	if actionType != "PUSH" {
		return "", fmt.Errorf("The event's 'detail.action-type' must be 'PUSH'.")
	}
	registryUrl := buildEcrRegistryUrl(event)
	repo := event.Detail.RepositoryName
	digest := event.Detail.ImageDigest

	ecrSoci, err := ecrsoci.Init(ctx, registryUrl)
	if err != nil {
		return "", err
	}

	fmt.Printf("Pulling image [repo=%s, digest=%s]\n", repo, digest)
	desc, err := ecrSoci.Pull(ctx, repo, digest)
	if err != nil {
		return "", err
	}

	image := images.Image{
		Name:   repo + "@" + digest,
		Target: *desc,
	}

	fmt.Printf("Building SOCI index for the image [repo=%s, digest=%s]\n", repo, digest)
	indexDescriptor, err := ecrSoci.BuildIndex(ctx, image)
	if err != nil {
		return "", err
	}

	fmt.Printf("Pushing the SOCI index to the repo [repo=%s]\n", repo)
	err = ecrSoci.PushIndex(ctx, *indexDescriptor, repo)
	if err != nil {
		return "", err
	}

	return "Successfully pushed SOCI index for " + registryUrl + "/" + repo + "@" + digest, nil
}

// Returns ecr registry url from an image action event
func buildEcrRegistryUrl(event events.ECRImageActionEvent) string {
	var awsDomain = ".amazonaws.com"
	if strings.HasPrefix(event.Region, "cn") {
		awsDomain = ".amazonaws.com.cn"
	}
	return event.Account + ".dkr.ecr." + event.Region + awsDomain
}

func main() {
	lambda.Start(HandleRequest)
}
