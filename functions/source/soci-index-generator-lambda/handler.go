// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/awslabs/soci-index-generator-lambda/ecrsoci"
	"github.com/awslabs/soci-index-generator-lambda/events"
	"github.com/containerd/containerd/images"
	"github.com/rs/zerolog/log"
)

func HandleRequest(ctx context.Context, event events.ECRImageActionEvent) (string, error) {
	lambdaContext, _ := lambdacontext.FromContext(ctx)
	actionType := event.Detail.ActionType
	if actionType != "PUSH" {
		err := fmt.Errorf("The event's 'detail.action-type' must be 'PUSH'.")
		log.Error().Err(err).Str("RequestId", lambdaContext.AwsRequestID).Msg("ECR event validation error")
		return "ECR event validation error", err
	}
	registryUrl := buildEcrRegistryUrl(event)
	repo := event.Detail.RepositoryName
	digest := event.Detail.ImageDigest

	ecrSoci, err := ecrsoci.Init(ctx, registryUrl)
	if err != nil {
		log.Error().Err(err).Str("RequestId", lambdaContext.AwsRequestID).Msg("EcrSoci initialization error")
		return "EcrSoci initialization error", err
	}

	log.Info().Str("RequestId", lambdaContext.AwsRequestID).Str("Repository", repo).Str("ImageDigest", digest).Msg("Pulling image")
	desc, err := ecrSoci.Pull(ctx, repo, digest)
	if err != nil {
		log.Error().Err(err).Str("RequestId", lambdaContext.AwsRequestID).Msg("Pulling image error")
		return "Pulling image error", err
	}

	image := images.Image{
		Name:   repo + "@" + digest,
		Target: *desc,
	}

	log.Info().Str("RequestId", lambdaContext.AwsRequestID).Str("Repository", repo).Str("ImageDigest", digest).Msg("Building SOCI index ")
	indexDescriptor, err := ecrSoci.BuildIndex(ctx, image)
	if err != nil {
		log.Error().Err(err).Str("RequestId", lambdaContext.AwsRequestID).Msg("Building SOCI index error")
		return "Building SOCI index error", err
	}

	log.Info().Str("RequestId", lambdaContext.AwsRequestID).Str("Repository", repo).Msg("Pushing SOCI index ")
	err = ecrSoci.PushIndex(ctx, *indexDescriptor, repo)
	if err != nil {
		log.Error().Err(err).Str("RequestId", lambdaContext.AwsRequestID).Msg("Pushing SOCI index error")
		return "Pushing SOCI index error", err
	}

	log.Info().Str("RequestId", lambdaContext.AwsRequestID).Str("Repository", repo).Str("ImageDigest", digest).Msg("Successfully built and pushed SOCI index")
	return "Successfully built and pushed SOCI index", nil
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
