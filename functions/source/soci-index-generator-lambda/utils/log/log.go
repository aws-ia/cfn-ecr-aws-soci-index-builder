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

// add more context, such as Aws Request Id, image repository, image Digest to log event
func addContext(ctx context.Context, logEvent *zerolog.Event) {
	if v := ctx.Value("RepositoryName"); v != nil {
		logEvent.Str("RepositoryName", v.(string))
	}
	if v := ctx.Value("ImageDigest"); v != nil {
		logEvent.Str("ImageDigest", v.(string))
	}
	if v := ctx.Value("RegistryURL"); v != nil {
		logEvent.Str("RegistryURL", v.(string))
	}
	lambdaCtx, _ := lambdacontext.FromContext(ctx)
	logEvent.Str("RequestId", lambdaCtx.AwsRequestID)
}
