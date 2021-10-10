package auth

import (
	"context"

	"google.golang.org/grpc"
)

type AuthFunc func(ctx context.Context) (context.Context, error)

func UnaryServerInterceptor(authFunc AuthFunc) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		ctx, err := authFunc(ctx)
		if err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}

func StreamServerInterceptor(authFunc AuthFunc) grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		_, err := authFunc(stream.Context())
		if err != nil {
			return err
		}
		return handler(srv, stream)
	}
}
