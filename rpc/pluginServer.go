package rpc

import (
	"context"
	"strings"

	"github.com/greboid/irc/v4/irc"
)

type pluginServer struct {
	conn         irc.Sender
	EventManager *irc.EventManager
}

func (ps *pluginServer) SendChannelMessage(_ context.Context, req *ChannelMessage) (*Error, error) {
	ps.conn.SendRawf("PRIVMSG %s :%s", req.Channel, req.Message)
	return &Error{
		Message: "",
	}, nil
}
func (*pluginServer) SendRawMessage(_ context.Context, _ *RawMessage) (*Error, error) {
	return &Error{
		Message: "",
	}, nil
}

func (ps *pluginServer) GetMessages(channel *Channel, stream IRCPlugin_GetMessagesServer) error {
	exitLoop := make(chan bool, 1)
	chanMessage := make(chan *irc.Message, 1)
	channelName := channel.Name
	partHandler := func(channelPart irc.Channel) {
		if channelPart.Name == channelName {
			exitLoop <- true
		}
	}
	messageHandler := func(message irc.Message) {
		if channelName == "*" || strings.ToLower(message.Params[0]) == strings.ToLower(channelName) {
			chanMessage <- &message
		}
	}
	ps.EventManager.SubscribeChannelPart(partHandler)
	defer ps.EventManager.UnsubscribeChannelPart(partHandler)
	ps.EventManager.SubscribeChannelMessage(messageHandler)
	defer ps.EventManager.UnsubscribeChannelMessage(messageHandler)
	for {
		select {
		case <-exitLoop:
			return nil
		case msg := <-chanMessage:
			if err := stream.Send(&ChannelMessage{Channel: strings.ToLower(msg.Params[0]), Message: strings.Join(msg.Params[1:], " "), Source: msg.Source}); err != nil {
				return err
			}
		}
	}
}

func (ps *pluginServer) Ping(context.Context, *Empty) (*Empty, error) {
	return &Empty{}, nil
}
