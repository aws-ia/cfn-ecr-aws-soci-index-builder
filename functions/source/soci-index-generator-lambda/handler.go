// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/awslabs/soci-index-generator-lambda/ecrsoci"
	"github.com/awslabs/soci-index-generator-lambda/events"
	"github.com/awslabs/soci-index-generator-lambda/utils/fs"
	"github.com/awslabs/soci-index-generator-lambda/utils/log"
	"github.com/containerd/containerd/images"
)

func HandleRequest(ctx context.Context, event events.ECRImageActionEvent) (string, error) {
	// Getting information about the registry, repository, and image
	// Those information are then stored in the application context
	registryUrl := buildEcrRegistryUrl(event)
	ctx = context.WithValue(ctx, "RegistryURL", registryUrl)
	repo := event.Detail.RepositoryName
	ctx = context.WithValue(ctx, "RepositoryName", repo)
	digest := event.Detail.ImageDigest
	ctx = context.WithValue(ctx, "ImageDigest", digest)

	// Directory in lambda storage to store images and SOCI artifacts db
	dataDir, err := createTempDir(ctx)
	if err != nil {
		return lambdaError(ctx, "Data directory create error", err)
	}
	defer cleanUp(ctx, dataDir)

	// the channel to signal the deadline monitor goroutine to exit early
	quitChannel := make(chan int)
	defer func() {
		quitChannel <- 1
	}()

	setDeadline(ctx, quitChannel, dataDir)

	actionType := event.Detail.ActionType
	if actionType != "PUSH" {
		err := fmt.Errorf("The event's 'detail.action-type' must be 'PUSH'.")
		return lambdaError(ctx, "ECR event validation error", err)
	}

	ecrSoci, err := ecrsoci.Init(ctx, registryUrl, dataDir)
	if err != nil {
		return lambdaError(ctx, "EcrSoci initialization error", err)
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

	err = ecrSoci.PushIndex(ctx, *indexDescriptor, repo)
	if err != nil {
		return lambdaError(ctx, "SOCI index push error", err)
	}

	log.Info(ctx, "Successfully built and pushed SOCI index")
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

	log.Info(ctx, "Creating a directory to store images and SOCI artifactsdb")
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
			default:
				time.Sleep(1 * time.Second)
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
