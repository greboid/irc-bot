package rpc

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"os"

	"github.com/greboid/irc-bot/v6/bot"
	grpcmiddleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpcauth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func NewGrpcServer(rpcPort int, pluginString string, webPort int) (*GrpcServer, error) {
	plugins, err := ParsePluginString(pluginString)
	if err != nil {
		return nil, err
	}
	return &GrpcServer{
		rpcPort: rpcPort,
		plugins: plugins,
		webPort: webPort,
	}, nil
}

type GrpcServer struct {
	rpcPort int
	plugins []Plugin
	webPort int
}

func (s *GrpcServer) StartGRPC(bot *bot.Bot) {
	certificate, err := generateSelfSignedCert()
	if err != nil {
		slog.Error("failed to generate certificate", "error", err)
		os.Exit(1)
		return
	}
	slog.Info("Starting RPC server", "port", s.rpcPort)
	lis, err := tls.Listen("tcp", fmt.Sprintf(":%d", s.rpcPort), &tls.Config{Certificates: []tls.Certificate{*certificate}})
	if err != nil {
		slog.Error("failed to listen", "error", err)
		os.Exit(1)
		return
	}
	grpcServer := grpc.NewServer(
		grpc.StreamInterceptor(grpcmiddleware.ChainStreamServer(grpcauth.StreamServerInterceptor(s.authPlugin))),
		grpc.UnaryInterceptor(grpcmiddleware.ChainUnaryServer(grpcauth.UnaryServerInterceptor(s.authPlugin))),
	)
	httpsServer := NewHttpServer(s.webPort, s.plugins)
	RegisterIRCPluginServer(grpcServer, &pluginServer{bot.Connection, bot})
	RegisterHTTPPluginServer(grpcServer, httpsServer)
	slog.Info("Starting HTTP Server", "port", s.webPort)
	httpsServer.Start()
	err = grpcServer.Serve(lis)
	if err != nil {
		slog.Error("Error listening", "error", err)
		return
	}
}

func (s *GrpcServer) authPlugin(ctx context.Context) (context.Context, error) {
	token, err := grpcauth.AuthFromMD(ctx, "bearer")
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid auth token: %s", err.Error())
	}
	if !s.checkPlugin(token) {
		return nil, status.Errorf(codes.Unauthenticated, "access denied")
	}
	return ctx, nil
}

func (s *GrpcServer) checkPlugin(token string) bool {
	for _, plugin := range s.plugins {
		if plugin.Token == token {
			return true
		}
	}
	return false
}
