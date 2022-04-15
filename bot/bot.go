package bot

import (
	"fmt"
	"os"
	"strings"

	"github.com/ergochat/irc-go/ircevent"
	"github.com/ergochat/irc-go/ircmsg"
	"github.com/greboid/irc-bot/v5/irc"
)

type Bot struct {
	Connection     *irc.Connection
	channels       []string
	initialChannel string
	log            irc.Logger
}

func NewBot(server, password, nickname, realname string, useTLS, useSasl bool, saslUser, saslPass string,
	logger irc.Logger, floodProfile string, initialChannel string) *Bot {
	connection := irc.NewIRC(server, password, nickname, realname, useTLS, useSasl, saslUser, saslPass, logger, floodProfile)
	bot := &Bot{
		Connection:     connection,
		channels:       []string{},
		initialChannel: initialChannel,
		log:            logger,
	}
	bot.addBotCallbacks()
	return bot
}

func (b *Bot) CurrentNick() string {
	return b.Connection.CurrentNick()
}

func (b *Bot) RemoveCallback(id ircevent.CallbackID) {
	b.Connection.RemoveCallback(id)
}

func (b *Bot) AddCallback(s string, f func(ircmsg.Message)) ircevent.CallbackID {
	return b.Connection.AddCallback(s, f)
}

func (b *Bot) Start(signals chan os.Signal) error {
	go func() {
		<-signals
		b.Connection.Quit()
	}()
	return b.Connection.ConnectAndWaitWithRetry(5)
}

func (b *Bot) GetChannels() []string {
	return b.channels
}

func (b *Bot) addBotCallbacks() {
	b.Connection.AddConnectCallback(func(message ircmsg.Message) {
		b.onConnect(b.Connection)
	})
	b.Connection.AddCallback("JOIN", func(message ircmsg.Message) {
		nuh, err := message.NUH()
		if err == nil {
			if nuh.Name == b.Connection.CurrentNick() {
				b.addToChannels(message.Params[0])
			}
		}
	})
	b.Connection.AddCallback("KICK", func(message ircmsg.Message) {
		if message.Params[1] == b.Connection.CurrentNick() {
			b.removeFromChannels(message.Params[0])
		}
	})
	b.Connection.AddCallback("PART", func(message ircmsg.Message) {
		nuh, err := message.NUH()
		if err == nil {
			if nuh.Name == b.Connection.CurrentNick() {
				b.removeFromChannels(message.Params[0])
			}
		}
	})
}

func (b *Bot) onConnect(c *irc.Connection) {
	botMode := c.ISupport()["BOT"]
	if len(botMode) > 0 {
		err := c.SetMode("+" + botMode)
		if err != nil {
			b.log.Errorf("Unable to set mode: %s", err)
		}
	}
	b.joinChannels(c)
}

func (b *Bot) joinChannels(c *irc.Connection) {
	if len(b.initialChannel) == 0 {
		return
	}
	for _, join := range b.getJoinCommands(b.initialChannel) {
		_ = c.Join(join)
	}
}

func (b *Bot) getJoinCommands(channelString string) (joinCommands []string) {
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
		joinCommands = append(joinCommands, fmt.Sprintf("%s %s", strings.Join(keyedChannels, ","), strings.Join(keys, ",")))
	}
	if len(keylessChannels) > 0 {
		joinCommands = append(joinCommands, fmt.Sprintf("%s", strings.Join(keylessChannels, ",")))
	}
	return
}

func (b *Bot) addToChannels(channel string) {
	existing := false
	for i := range b.channels {
		if b.channels[i] == channel {
			existing = true
			break
		}
	}
	if !existing {
		b.channels = append(b.channels, channel)
	}
}

func (b *Bot) removeFromChannels(channel string) {
	for i, v := range b.channels {
		if v == channel {
			b.channels = append(b.channels[:i], b.channels[i+1:]...)
			break
		}
	}
}
