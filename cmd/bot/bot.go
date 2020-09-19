package main

import (
	"fmt"
	"github.com/greboid/irc/v2/irc"
	"strings"
)

func addBotCallbacks(c *irc.Connection) {
	c.AddInboundHandler("001", joinChannels)
	c.AddInboundHandler("PRIVMSG", publishMessages)
}

func joinChannels(_ *irc.EventManager, c *irc.Connection, _ *irc.Message) {
	for _, join := range getJoinCommands(*Channel) {
		c.SendRawf(join)
	}
}

func getJoinCommands(channelString string) (joinCommands []string) {
	keyedChannels := make([]string, 0)
	keys := make([]string, 0)
	keylessChannels := make([]string, 0)
	channels := strings.Split(channelString, ",")
	for index := range channels {
		parts := strings.Split(channels[index], " ")
		if len(parts) == 1 {
			keylessChannels = append(keylessChannels, channels[index])
		} else if len(parts) == 2 {
			keyedChannels = append(keyedChannels, parts[0])
			keys = append(keys, parts[1])
		}
	}
	if len(keyedChannels) > 0 {
		joinCommands = append(joinCommands, fmt.Sprintf("JOIN :%s %s", strings.Join(keyedChannels, ","), strings.Join(keys, ",")))
	}
	if len(keylessChannels) > 0 {
		joinCommands = append(joinCommands, fmt.Sprintf("JOIN :%s", strings.Join(keylessChannels, ",")))
	}
	return
}

func publishMessages(em *irc.EventManager, _ *irc.Connection, m *irc.Message) {
	em.PublishChannelMessage(*m)
}
