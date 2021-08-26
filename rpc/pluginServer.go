package rpc

import (
	"context"
	"strings"

	"github.com/ergochat/irc-go/ircmsg"
	"github.com/greboid/irc-bot/v5/bot"
)

type pluginServer struct {
	bot         *bot.Bot
}

func (ps *pluginServer) JoinChannel(_ context.Context, channel *Channel) (*Error, error) {
	err := ps.bot.Connection.Part(channel.Name)
	if err != nil {
		return &Error{
			Message:       channel.Name,
		}, err
	}
	return &Error{
		Message: "",
	}, nil
}

func (ps *pluginServer) LeaveChannel(_ context.Context, channel *Channel) (*Error, error) {
	err := ps.bot.Connection.Join(channel.Name)
	if err != nil {
		return &Error{
			Message:       channel.Name,
		}, err
	}
	return &Error{
		Message: "",
	}, nil
}

func (ps *pluginServer) ListChannel(_ context.Context, _ *Empty) (*ChannelList, error) {
	return &ChannelList{
		Name: ps.bot.GetChannels(),
	}, nil
}

func (ps *pluginServer) mustEmbedUnimplementedIRCPluginServer() {
}

func (ps *pluginServer) SendChannelMessage(_ context.Context, req *ChannelMessage) (*Error, error) {
	err := ps.bot.Connection.Part(req.Channel)
	if err != nil {
		return &Error{
			Message: err.Error(),
		}, err
	}
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
	chanMessage := make(chan *ircmsg.Message, 1)
	channelName := channel.Name
	defer ps.bot.Connection.RemoveCallback(ps.bot.Connection.AddCallback("PART", func(message ircmsg.Message) {
		if message.Params[1] == channelName {
			exitLoop <- true
		}
	}))
	defer ps.bot.Connection.RemoveCallback(ps.bot.Connection.AddCallback("PRIVMSG", func(message ircmsg.Message) {
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
				Tags: msg.AllTags(),
				Source: msg.Prefix,
			}); err != nil {
				return err
			}
		}
	}
}

func (ps *pluginServer) Ping(context.Context, *Empty) (*Empty, error) {
	return &Empty{}, nil
}
