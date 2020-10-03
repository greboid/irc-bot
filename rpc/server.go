package rpc

import (
	"context"
	"crypto/tls"
	"fmt"

	"github.com/greboid/irc/v4/irc"
	grpcmiddleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpcauth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func NewGrpcServer(conn *irc.Connection, eventManager *irc.EventManager, rpcPort int, plugins []Plugin, webPort int, logger *zap.SugaredLogger) GrpcServer {
	return GrpcServer{
		conn:         conn,
		eventManager: eventManager,
		rpcPort:      rpcPort,
		plugins:      plugins,
		webPort:      webPort,
		logger:       logger,
	}
}

type GrpcServer struct {
	conn         *irc.Connection
	eventManager *irc.EventManager
	rpcPort      int
	plugins      []Plugin
	webPort      int
	logger       *zap.SugaredLogger
}

func (s *GrpcServer) StartGRPC() {
	certificate, err := generateSelfSignedCert()
	if err != nil {
		s.logger.Fatalf("failed to generate certificate: %s", err.Error())
		return
	}
	s.logger.Infof("Starting RPC server: %d", s.rpcPort)
	lis, err := tls.Listen("tcp", fmt.Sprintf(":%d", s.rpcPort), &tls.Config{Certificates: []tls.Certificate{*certificate}})
	if err != nil {
		s.logger.Fatalf("failed to listen: %v", err)
		return
	}
	grpcServer := grpc.NewServer(
		grpc.StreamInterceptor(grpcmiddleware.ChainStreamServer(grpcauth.StreamServerInterceptor(s.authPlugin))),
		grpc.UnaryInterceptor(grpcmiddleware.ChainUnaryServer(grpcauth.UnaryServerInterceptor(s.authPlugin))),
	)
	httpsServer := NewHttpServer(s.webPort, s.plugins, s.logger)
	RegisterIRCPluginServer(grpcServer, &pluginServer{s.conn, s.eventManager})
	RegisterHTTPPluginServer(grpcServer, httpsServer)
	s.logger.Infof("Starting HTTP Server: %d", s.webPort)
	httpsServer.Start()
	err = grpcServer.Serve(lis)
	if err != nil {
		s.logger.Errorf("Error listening: %s", err.Error())
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
