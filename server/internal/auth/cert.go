package auth

import (
	"context"
	"fmt"

	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
)

func CertAuthFunc(ctx context.Context) (context.Context, error) {
	cn, err := authFromCert(ctx)
	if err != nil {
		return nil, err
	}

	return newContext(ctx, &User{Name: cn}), nil
}

func authFromCert(ctx context.Context) (string, error) {
	peer, ok := peer.FromContext(ctx)
	if !ok {
		return "", fmt.Errorf("error while getting auth data")
	}

	tls := peer.AuthInfo.(credentials.TLSInfo)

	if len(tls.State.PeerCertificates) == 0 {
		return "", fmt.Errorf("error while getting certificate data")
	}
	return tls.State.PeerCertificates[0].Subject.CommonName, nil
}

func UsernameFromCtx(ctx context.Context) (*User, bool) {
	return fromContext(ctx)
}
