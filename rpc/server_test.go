package rpc

import (
	"context"
	"crypto/tls"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func TestGRPCServerTLS(t *testing.T) {
	certificate, err := generateSelfSignedCert()
	if err != nil {
		t.Fatal(err)
	}
	server, err := NewGrpcServer(0, "test=secret", 0)
	if err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { listener.Close() })
	grpcServer := server.newGRPCServer(*certificate)
	RegisterIRCPluginServer(grpcServer, &pluginServer{})
	serveDone := make(chan error, 1)
	go func() { serveDone <- grpcServer.Serve(listener) }()
	t.Cleanup(func() {
		grpcServer.Stop()
		if err := <-serveDone; err != nil {
			t.Errorf("Serve: %v", err)
		}
	})

	// Match the plugins' TLS setup: the bot uses a self-signed certificate.
	conn, err := grpc.NewClient(listener.Addr().String(), grpc.WithTransportCredentials(
		credentials.NewTLS(&tls.Config{InsecureSkipVerify: true}),
	))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	client := NewIRCPluginClient(conn)

	for _, tt := range []struct {
		name  string
		token string
		code  codes.Code
	}{
		{name: "valid token", token: "secret", code: codes.OK},
		{name: "invalid token", token: "wrong", code: codes.Unauthenticated},
		{name: "missing token", code: codes.Unauthenticated},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if tt.token != "" {
				ctx = CtxWithToken(ctx, "bearer", tt.token)
			}
			var remote peer.Peer
			_, err := client.Ping(ctx, &Empty{}, grpc.Peer(&remote))
			if status.Code(err) != tt.code {
				t.Fatalf("Ping: got %v, want %v", err, tt.code)
			}
			info, ok := remote.AuthInfo.(credentials.TLSInfo)
			if !ok {
				t.Fatalf("expected TLS peer information, got %T", remote.AuthInfo)
			}
			if got := info.State.NegotiatedProtocol; got != "h2" {
				t.Errorf("negotiated ALPN protocol = %q, want h2", got)
			}
		})
	}
}
