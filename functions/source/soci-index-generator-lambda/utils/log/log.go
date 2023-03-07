// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package logging provides log functions with common contextual information such as Aws Request Id, Repository Name, Image Digest, etc.
package log

import (
	"context"

	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func Error(ctx context.Context, msg string, err error) {
	logEvent := log.Error().Err(err)
	addContext(ctx, logEvent)
	logEvent.Msg(msg)
}

func Warn(ctx context.Context, msg string) {
	logEvent := log.Warn()
	addContext(ctx, logEvent)
	logEvent.Msg(msg)
}

func Info(ctx context.Context, msg string) {
	logEvent := log.Info()
	addContext(ctx, logEvent)
	logEvent.Msg(msg)
}

// Add more context to the log event
func addContext(ctx context.Context, logEvent *zerolog.Event) {
	contextKeys := []string{
		"RegistryURL",
		"RepositoryName",
		"ImageDigest",
		"ImageTag",
		"SOCIIndexDigest"}

	for _, contextKey := range contextKeys {
		if value := ctx.Value(contextKey); value != nil {
			logEvent.Str(contextKey, value.(string))
		}
	}

	lambdaCtx, _ := lambdacontext.FromContext(ctx)
	logEvent.Str("RequestId", lambdaCtx.AwsRequestID)
}
