package grpcprotocol

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const serverKeyMetadata = "server-key"

// ServerKeyInterceptor creates a server-side interceptor that validates the server key
func ServerKeyInterceptor(validServerKey string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.Unauthenticated, "missing metadata")
		}

		serverKeys := md.Get(serverKeyMetadata)
		if len(serverKeys) == 0 || serverKeys[0] != validServerKey {
			return nil, status.Errorf(codes.Unauthenticated, "invalid server key")
		}

		return handler(ctx, req)
	}
}

// ClientServerKeyInterceptor creates a client-side interceptor that adds the server key to outgoing requests
func ClientServerKeyInterceptor(serverKey string) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		ctx = metadata.AppendToOutgoingContext(ctx, serverKeyMetadata, serverKey)
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
