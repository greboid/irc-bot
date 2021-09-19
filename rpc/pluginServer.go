package rpc

import (
	"context"
	"strings"

	"github.com/ergochat/irc-go/ircevent"
	"github.com/ergochat/irc-go/ircmsg"
)

type IRCFunctions interface {
	GetChannels() []string
	CurrentNick() string
	RemoveCallback(id ircevent.CallbackID)
	AddCallback(string, func(ircmsg.Message)) ircevent.CallbackID
}

type IRCSender interface {
	Join(string) error
	Part(string) error
	SendRawf(string, ...interface{}) error
	SendRelayMessage(channel string, nickname string, message string) error
}

type pluginServer struct {
	sender    IRCSender
	functions IRCFunctions
}

func (ps *pluginServer) SendRelayMessage(_ context.Context, message *RelayMessage) (*Error, error) {
	err := ps.sender.SendRelayMessage(message.Channel, message.Nick, message.Message)
	if err != nil {
		return &Error{
			Message: err.Error(),
		}, err
	}
	return &Error{
		Message: "",
	}, nil
}

func (ps *pluginServer) JoinChannel(_ context.Context, channel *Channel) (*Error, error) {
	err := ps.sender.Join(channel.Name)
	if err != nil {
		return &Error{
			Message: channel.Name,
		}, err
	}
	return &Error{
		Message: "",
	}, nil
}

func (ps *pluginServer) LeaveChannel(_ context.Context, channel *Channel) (*Error, error) {
	err := ps.sender.Part(channel.Name)
	if err != nil {
		return &Error{
			Message: channel.Name,
		}, err
	}
	return &Error{
		Message: "",
	}, nil
}

func (ps *pluginServer) ListChannel(_ context.Context, _ *Empty) (*ChannelList, error) {
	return &ChannelList{
		Name: ps.functions.GetChannels(),
	}, nil
}

func (ps *pluginServer) mustEmbedUnimplementedIRCPluginServer() {
}

func (ps *pluginServer) SendChannelMessage(_ context.Context, req *ChannelMessage) (*Error, error) {
	err := ps.sender.SendRawf("PRIVMSG %s :%s", req.Channel, req.Message)
	if err != nil {
		return &Error{
			Message: err.Error(),
		}, err
	}
	return &Error{
		Message: "",
	}, nil
}
func (ps *pluginServer) SendRawMessage(_ context.Context, req *RawMessage) (*Error, error) {
	err := ps.sender.SendRawf("%s", req.Message)
	if err != nil {
		return &Error{
			Message: err.Error(),
		}, err
	}
	return &Error{
		Message: "",
	}, nil
}

func (ps *pluginServer) GetMessages(channel *Channel, stream IRCPlugin_GetMessagesServer) error {
	exitLoop := make(chan bool, 1)
	chanMessage := make(chan *ircmsg.Message, 1)
	channelName := channel.Name
	defer ps.functions.RemoveCallback(ps.functions.AddCallback("PART", func(message ircmsg.Message) {
		if message.Params[1] == channelName {
			exitLoop <- true
		}
	}))
	defer ps.functions.RemoveCallback(ps.functions.AddCallback("KICK", func(message ircmsg.Message) {
		if message.Params[1] == ps.functions.CurrentNick() && message.Params[1] == channelName {
			exitLoop <- true
		}
	}))
	defer ps.functions.RemoveCallback(ps.functions.AddCallback("PRIVMSG", func(message ircmsg.Message) {
		if channelName == "*" || strings.ToLower(message.Params[0]) == strings.ToLower(channelName) {
			chanMessage <- &message
		}
	}))
	for {
		select {
		case <-exitLoop:
			return nil
		case msg := <-chanMessage:
			if err := stream.Send(&ChannelMessage{
				Channel: strings.ToLower(msg.Params[0]),
				Message: strings.Join(msg.Params[1:], " "),
				Tags:    msg.AllTags(),
				Source:  msg.Prefix,
			}); err != nil {
				return err
			}
		}
	}
}

func (ps *pluginServer) Ping(context.Context, *Empty) (*Empty, error) {
	return &Empty{}, nil
}
