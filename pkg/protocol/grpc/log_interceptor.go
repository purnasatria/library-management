package grpcprotocol

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

func LogInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	start := time.Now()

	// Proceed with the handler
	resp, err := handler(ctx, req)

	// Log details
	logEvent := log.Debug().
		Dur("duration", time.Since(start)).
		Str("method", info.FullMethod)

	if err != nil {
		logEvent = logEvent.Err(err).Str("status", status.Code(err).String())
	} else {
		logEvent = logEvent.Str("status", status.Code(nil).String())
	}

	logEvent.Send()

	return resp, err
}
