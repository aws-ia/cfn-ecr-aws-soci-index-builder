// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"github.com/aws-ia/cfn-aws-soci-index-builder/soci-index-generator-lambda/events"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"os"
	"testing"
	"time"
)

// This test ensures that the handler can pull an image, build, and push the SOCI index back to the repository.
// To run this test locally, you need to push an image to a private ECR repository, and set following environment variables:
// AWS_ACCOUNT_ID: your aws account id.
// AWS_REGION: the region of your private ECR repository.
// REPOSITORY_NAME: name of your private ECR repository.
// IMAGE_DIGEST: the digest of your image.
func TestHandlerHappyPath(t *testing.T) {
	event := events.ECRImageActionEvent{
		Version:    "1",
		Id:         "id",
		DetailType: "ECR Image Action",
		Source:     "aws.ecr",
		Account:    os.Getenv("AWS_ACCOUNT_ID"),
		Time:       "time",
		Region:     os.Getenv("AWS_REGION"),
		Detail: events.ECRImageActionEventDetail{
			ActionType:     "PUSH",
			Result:         "SUCCESS",
			RepositoryName: os.Getenv("REPOSITORY_NAME"),
			ImageDigest:    os.Getenv("IMAGE_DIGEST"),
			ImageTag:       "",
		},
	}

	// making the test context
	lc := lambdacontext.LambdaContext{}
	lc.AwsRequestID = "abcd-1234"
	ctx := lambdacontext.NewContext(context.Background(), &lc)
	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(time.Minute))
	defer cancel()

	resp, err := HandleRequest(ctx, event)
	if err != nil {
		t.Fatalf("HandleRequest failed %v", err)
	}

	expected_resp := "Successfully built and pushed SOCI index"
	if resp != expected_resp {
		t.Fatalf("Unexpected response. Expected %s but got %s", expected_resp, resp)
	}
}

// This test ensures that the handler can validate the input digest media type
// To run this test locally, you need to push an image to a private ECR repository, and set following environment variables:
// AWS_ACCOUNT_ID: your aws account id.
// AWS_REGION: the region of your private ECR repository.
// REPOSITORY_NAME: name of your private ECR repository.
// INVAL(D_IMAGE_DIGEST: the digest of anything that isn't an image.
func TestHandlerInvalidDigestMediaType(t *testing.T) {
	event := events.ECRImageActionEvent{
		Version:    "1",
		Id:         "id",
		DetailType: "ECR Image Action",
		Source:     "aws.ecr",
		Account:    os.Getenv("AWS_ACCOUNT_ID"),
		Time:       "time",
		Region:     os.Getenv("AWS_REGION"),
		Detail: events.ECRImageActionEventDetail{
			ActionType:     "PUSH",
			Result:         "SUCCESS",
			RepositoryName: os.Getenv("REPOSITORY_NAME"),
			ImageDigest:    os.Getenv("INVALID_IMAGE_DIGEST"),
			ImageTag:       "",
		},
	}

	// making the test context
	lc := lambdacontext.LambdaContext{}
	lc.AwsRequestID = "abcd-1234"
	ctx := lambdacontext.NewContext(context.Background(), &lc)
	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(time.Minute))
	defer cancel()

	resp, err := HandleRequest(ctx, event)
	if err != nil {
		t.Fatalf("Invalid image digest is not expected to fail")
	}

	expected_resp := "Invalid image digest"
	if resp != expected_resp {
		t.Fatalf("Unexpected response. Expected %s but got %s", expected_resp, resp)
	}
}
