package main

import (
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/csmith/envflag/v2"
	"github.com/csmith/slogflags"
	"github.com/greboid/irc-bot/v6/bot"
	"github.com/greboid/irc-bot/v6/rpc"
)

//go:generate protoc --go_out=../../rpc -I ../../rpc plugin.proto
//go:generate protoc --go-grpc_out=../../rpc -I ../../rpc plugin.proto

var (
	Server        = flag.String("server", "", "Which IRC server to connect to")
	Password      = flag.String("password", "", "The server password, if required")
	TLS           = flag.Bool("tls", true, "Connect with TLS?")
	Nickname      = flag.String("nick", "", "Nickname to use")
	Realname      = flag.String("realname", "", "'Real name' to use")
	Channel       = flag.String("channel", "", "Channels to join on connect, comma separated list (with optional space separated key with each channel)")
	SASLAuth      = flag.Bool("sasl-auth", false, "Authenticate via SASL?")
	SASLUser      = flag.String("sasl-user", "", "SASL username")
	SASLPass      = flag.String("sasl-pass", "", "SASL password")
	RPCPort       = flag.Int("rpc-port", 8001, "gRPC server port")
	PluginsString = flag.String("plugins", "", "Comma separated list of plugins, name=token")
	FloodProfile  = flag.String("flood-profile", "restrictive", "Flood profile: restrictive, unlimited")
	WebPort       = flag.Int("web-port", 8000, "Web port for http server")
)

func main() {
	envflag.Parse()
	slogflags.Logger(slogflags.WithSetDefault(true))
	slog.Info("Starting bot")
	if len(*Server) == 0 {
		slog.Error("Server is mandatory")
		os.Exit(1)
	}
	rpcServer, err := rpc.NewGrpcServer(*RPCPort, *PluginsString, *WebPort)
	if err != nil {
		slog.Error("Unable to create GRPC server", "error", err)
		os.Exit(1)
	}
	ircBot := bot.NewBot(*Server, *Password, *Nickname, *Realname, *TLS, *SASLAuth, *SASLUser, *SASLPass, *FloodProfile, *Channel)
	go func() {
		rpcServer.StartGRPC(ircBot)
	}()
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	err = ircBot.Start(signals)
	if err != nil {
		slog.Error("Unable to start bot", "error", err)
		os.Exit(1)
	}
	slog.Info("Exiting")
}
