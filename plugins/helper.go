package plugins

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"

	"github.com/greboid/irc-bot/v5/rpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type PluginHelper struct {
	RPCTarget     string
	RPCToken      string
	rpcConnection *grpc.ClientConn
	httpClient    rpc.HTTPPluginClient
	ircClient     rpc.IRCPluginClient
}

//NewHelper returns a PluginHelper that simplifies writing plugins by managing grpc connections and exposing a simple
//interface.
//It returns a PluginHelper or any errors encountered whilst creating
func NewHelper(target string, rpctoken string) (*PluginHelper, error) {
	if len(target) == 0 {
		return nil, fmt.Errorf("gRPC target name needs to be set")
	}
	if len(rpctoken) == 0 {
		return nil, fmt.Errorf("plugin RPC token must be set")
	}
	return &PluginHelper{
		RPCTarget: target,
		RPCToken:  rpctoken,
	}, nil
}

func (h *PluginHelper) rpcClient(ctx context.Context) (*grpc.ClientConn, error) {
	if h.rpcConnection == nil {
		creds := credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})
		conn, err := grpc.DialContext(ctx, h.RPCTarget, grpc.WithTransportCredentials(creds))
		if err != nil {
			return nil, err
		}
		h.rpcConnection = conn
	}
	return h.rpcConnection, nil
}

func (h *PluginHelper) HTTPClient() (rpc.HTTPPluginClient, error) {
	return h.HTTPClientWithContext(context.Background())
}

func (h *PluginHelper) HTTPClientWithContext(ctx context.Context) (rpc.HTTPPluginClient, error) {
	if h.httpClient == nil {
		rpcClient, err := h.rpcClient(ctx)
		if err != nil {
			return nil, err
		}
		client := rpc.NewHTTPPluginClient(rpcClient)
		h.httpClient = client
	}
	return h.httpClient, nil
}

func (h *PluginHelper) IRCClient() (rpc.IRCPluginClient, error) {
	return h.IRCClientWithContext(context.Background())
}

func (h *PluginHelper) IRCClientWithContext(ctx context.Context) (rpc.IRCPluginClient, error) {
	if h.ircClient == nil {
		rpcClient, err := h.rpcClient(ctx)
		if err != nil {
			return nil, err
		}
		client := rpc.NewIRCPluginClient(rpcClient)
		if _, err := client.Ping(rpc.CtxWithToken(ctx, "bearer", h.RPCToken), &rpc.Empty{}); err != nil {
			return nil, err
		}
		h.ircClient = client
	}
	return h.ircClient, nil
}

func (h *PluginHelper) RegisterWebhook(path string, handler func(request *rpc.HttpRequest) *rpc.HttpResponse) error {
	return h.RegisterWebhookWithContext(context.Background(), path, handler)
}

func (h *PluginHelper) RegisterWebhookWithContext(ctx context.Context, path string, handler func(request *rpc.HttpRequest) *rpc.HttpResponse) error {
	httpClient, err := h.HTTPClientWithContext(ctx)
	if err != nil {
		return err
	}
	stream, err := httpClient.GetRequest(rpc.CtxWithTokenAndPath(ctx, "bearer", h.RPCToken, path))
	if err != nil {
		return err
	}
	for {
		request, err := stream.Recv()
		if err == io.EOF {
			return err
		}
		if err != nil {
			return err
		}
		response := handler(request)
		if err = stream.Send(response); err != nil {
			return err
		}
	}
}

func (h *PluginHelper) Ping() error {
	return h.PingWithContext(context.Background())
}

func (h *PluginHelper) PingWithContext(ctx context.Context) error {
	ircClient, err := h.IRCClientWithContext(ctx)
	if err != nil {
		return err
	}
	_, err = ircClient.Ping(rpc.CtxWithToken(ctx, "bearer", h.RPCToken), &rpc.Empty{})
	return err
}

func (h *PluginHelper) SendRelayMessage(channel string, nickname string, messages ...string) error {
	return h.SendChannelMessageWithContext(context.Background(), channel, messages...)
}

func (h *PluginHelper) SendRelayMessageWithContext(ctx context.Context, channel string, nickname string, messages ...string) error {
	ircClient, err := h.IRCClientWithContext(ctx)
	if err != nil {
		return err
	}
	for index := range messages {
		_, err := ircClient.SendRelayMessage(rpc.CtxWithToken(ctx, "bearer", h.RPCToken), &rpc.RelayMessage{
			Channel: channel,
			Nick:    nickname,
			Message: messages[index],
			Tags:    nil,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (h *PluginHelper) SendChannelMessage(channel string, messages ...string) error {
	return h.SendChannelMessageWithContext(context.Background(), channel, messages...)
}

func (h *PluginHelper) SendChannelMessageWithContext(ctx context.Context, channel string, messages ...string) error {
	ircClient, err := h.IRCClientWithContext(ctx)
	if err != nil {
		return err
	}
	for index := range messages {
		_, err := ircClient.SendChannelMessage(rpc.CtxWithToken(ctx, "bearer", h.RPCToken), &rpc.ChannelMessage{
			Channel: channel,
			Message: messages[index],
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (h *PluginHelper) SendRawMessage(messages ...string) error {
	return h.SendRawMessageWithContext(context.Background(), messages...)
}

func (h *PluginHelper) SendRawMessageWithContext(ctx context.Context, messages ...string) error {
	ircClient, err := h.IRCClientWithContext(ctx)
	if err != nil {
		return err
	}
	for index := range messages {
		_, err := ircClient.SendRawMessage(rpc.CtxWithToken(ctx, "bearer", h.RPCToken), &rpc.RawMessage{
			Message: messages[index],
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (h *PluginHelper) RegisterChannelMessageHandler(channel string, handler func(message *rpc.ChannelMessage)) error {
	return h.RegisterChannelMessageHandlerWithContext(context.Background(), channel, handler)
}

func (h *PluginHelper) RegisterChannelMessageHandlerWithContext(ctx context.Context, channel string, handler func(message *rpc.ChannelMessage)) error {
	ircClient, err := h.IRCClientWithContext(ctx)
	if err != nil {
		return err
	}
	stream, err := ircClient.GetMessages(
		rpc.CtxWithToken(ctx, "bearer", h.RPCToken),
		&rpc.Channel{Name: channel},
	)
	if err != nil {
		return err
	}
	for {
		message, err := stream.Recv()
		if err == io.EOF {
			return err
		}
		if err != nil {
			return err
		}
		handler(message)
	}
}
