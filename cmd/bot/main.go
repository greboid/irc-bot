package main

import (
	"flag"
	"fmt"
	"github.com/greboid/irc-bot/v4/rpc"
	"github.com/greboid/irc/v5/irc"
	"github.com/kouhin/envflag"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

//go:generate protoc -I ../../rpc plugin.proto --go_out=plugins=grpc:../../rpc

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
	err, log := CreateLogger(*Debug)
	if err != nil {
		fmt.Printf("Unable to create logger: %s", err.Error())
		return
	}
	defer func() {
		err = log.Sync()
		if err != nil {
			panic("Unable to sync logs")
		}
	}()
	log.Info("Starting bot")
	if err = envflag.Parse(); err != nil {
		log.Fatal("Unable to load config.", zap.String("error", err.Error()))
	}
	Plugins, err := rpc.ParsePluginString(*PluginsString)
	if err != nil {
		log.Fatal("Unable to load config.", zap.String("error", err.Error()))
	}
	if len(*Server) == 0 && len(*Channel) == 0 {
		log.Fatal("Server and channel are mandatory")
	}
	eventManager := irc.NewEventManager()
	err, connection := irc.NewIRC(*Server, *Password, *Nickname, *Realname, *TLS, *SASLAuth, *SASLUser, *SASLPass, log,
		*FloodProfile, eventManager)
	if err != nil {
		log.Fatalf("Unable to launch new connection: %s", err.Error())
		return
	}
	rpcServer := rpc.NewGrpcServer(connection, eventManager, *RPCPort, Plugins, *WebPort, log)
	log.Info("Adding callbacks")
	addBotCallbacks(connection)
	go rpcServer.StartGRPC()
	err = connection.ConnectAndWaitWithRetry(5)
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
	return nil, log.Sugar()
}