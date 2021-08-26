package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/greboid/irc-bot/v5/bot"
	"github.com/greboid/irc-bot/v5/rpc"
	"github.com/kouhin/envflag"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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
	Debug         = flag.Bool("debug", false, "Enable IRC debug output")
	SASLAuth      = flag.Bool("sasl-auth", false, "Authenticate via SASL?")
	SASLUser      = flag.String("sasl-user", "", "SASL username")
	SASLPass      = flag.String("sasl-pass", "", "SASL password")
	RPCPort       = flag.Int("rpc-port", 8001, "gRPC server port")
	PluginsString = flag.String("plugins", "", "Comma separated list of plugins, name=token")
	FloodProfile  = flag.String("flood-profile", "restrictive", "Flood profile: restrictive, unlimited")
	WebPort       = flag.Int("web-port", 8000, "Web port for http server")
)

func main() {
	if err := envflag.Parse(); err != nil {
		fmt.Printf("Unable to load config: %s", err.Error())
		return
	}
	err, log := CreateLogger(*Debug)
	if err != nil {
		fmt.Printf("Unable to create logger: %s", err.Error())
		return
	}
	defer func() {
		err = log.Sync()
	}()
	log.Info("Starting bot")
	if len(*Server) == 0 {
		log.Fatal("Server is mandatory")
	}
	rpcServer, err := rpc.NewGrpcServer(*RPCPort, *PluginsString, *WebPort, log)
	if err != nil {
		log.Fatalf("Unable to create GRPC server: %s", err)
	}
	ircBot := bot.NewBot(*Server, *Password, *Nickname, *Realname, *TLS, *SASLAuth, *SASLUser, *SASLPass, log,
		*FloodProfile, *Channel)
	go func() {
		rpcServer.StartGRPC(ircBot)
	}()
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	err = ircBot.Start(signals)
	if err != nil {
		log.Fatal(err)
	}
	log.Info("Exiting")
}

func CreateLogger(debug bool) (error, *zap.SugaredLogger) {
	zapConfig := zap.NewDevelopmentConfig()
	zapConfig.DisableCaller = !debug
	zapConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	zapConfig.DisableStacktrace = !debug
	zapConfig.OutputPaths = []string{"stdout"}
	if debug {
		zapConfig.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	} else {
		zapConfig.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	log, err := zapConfig.Build()
	if err != nil {
		return err, nil
	}
	_, err = zap.RedirectStdLogAt(log, zap.DebugLevel)
	if err != nil {
		log.Fatal("Unable to modify standard logger")
	}
	return nil, log.Sugar()
}
