package irc

import (
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/csmith/slogflags"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ergochat/irc-go/ircevent"
	"github.com/ergochat/irc-go/ircmsg"
)

type Connection struct {
	connection   *ircevent.Connection
	FloodProfile string
	connected    bool
	limiter      *RateLimiter
}

func NewIRC(server, password, nickname, realname string, useTLS, useSasl bool, saslUser, saslPass string, floodProfile string) *Connection {
	connection := &Connection{
		connection: &ircevent.Connection{
			Server:       server,
			Nick:         nickname,
			User:         nickname,
			RealName:     realname,
			Password:     password,
			SASLLogin:    saslUser,
			SASLPassword: saslPass,
			SASLMech:     "PLAIN",
			Timeout:      1 * time.Minute,
			KeepAlive:    4 * time.Minute,
			UseTLS:       useTLS,
			UseSASL:      useSasl,
			EnableCTCP:   true,
			Debug:        true,
			TLSConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
			QuitMessage: " ",
			Log:         slog.NewLogLogger(slogflags.Logger().Handler().WithGroup("rawirc"), slog.LevelDebug),
		},
		FloodProfile: floodProfile,
	}
	connection.connection.RequestCaps = append(connection.connection.RequestCaps, "draft/relaymsg")
	connection.limiter = connection.NewRateLimiter(floodProfile)
	slog.Info("Creating new IRC")
	return connection
}

func (irc *Connection) Quit() {
	irc.connection.Quit()
}

func (irc *Connection) AddConnectCallback(handler func(ircmsg.Message)) ircevent.CallbackID {
	return irc.connection.AddConnectCallback(handler)
}

func (irc *Connection) AddCallback(command string, handler func(ircmsg.Message)) ircevent.CallbackID {
	return irc.connection.AddCallback(command, handler)
}

func (irc *Connection) RemoveCallback(id ircevent.CallbackID) {
	irc.connection.RemoveCallback(id)
}

func (irc *Connection) Join(channel string) error {
	return irc.connection.Join(channel)
}

func (irc *Connection) Part(channel string) error {
	return irc.connection.Part(channel)
}

func (irc *Connection) CurrentNick() string {
	return irc.connection.CurrentNick()
}

func (irc *Connection) ISupport() map[string]string {
	return irc.connection.ISupport()
}

func (irc *Connection) AcknowledgedCaps() map[string]string {
	return irc.connection.AcknowledgedCaps()
}

func (irc *Connection) SetMode(mode string) error {
	return irc.SendRawf("MODE %s %s", irc.CurrentNick(), mode)
}

func (irc *Connection) SendRaw(line string) error {
	err := irc.limiter.Wait()
	if err != nil {
		return err
	}
	return irc.connection.SendRaw(line)
}

func (irc *Connection) SendRawf(formatLine string, args ...interface{}) error {
	return irc.SendRaw(fmt.Sprintf(formatLine, args...))
}

func (irc *Connection) SendRelayMessage(channel string, nickname string, message string) error {
	if irc.AcknowledgedCaps()["draft/relaymsg"] == "" {
		return irc.SendRawf("PRIVMSG %s :<%s> %s", channel, nickname, message)
	} else {
		return irc.SendRawf("RELAYMSG %s %s :%s", channel, nickname, message)
	}
}

func (irc *Connection) Connect() error {
	slog.Info("Connecting to IRC", "server", irc.connection.Server)
	err := irc.connection.Connect()
	if err != nil {
		return err
	}
	return nil
}

func (irc *Connection) Wait() {
	slog.Debug("Waiting for IRC to finish")
	irc.connection.Loop()
	slog.Debug("IRC Finished")
}

func (irc *Connection) ConnectAndWait() error {
	err := irc.Connect()
	if err != nil {
		return err
	}
	irc.Wait()
	return nil
}

func (irc *Connection) ConnectAndWaitWithRetry(maxRetries int) error {
	sigWait := make(chan os.Signal, 1)
	signal.Notify(sigWait, os.Interrupt)
	signal.Notify(sigWait, syscall.SIGTERM)
	retryDelay := 0
	retryCount := -1
	for {
		retryCount++
		err := irc.ConnectAndWait()
		if retryCount > maxRetries {
			return errors.New("maximum retries reached")
		}
		retryDelay = retryCount*5 + retryDelay
		if retryDelay > 300 {
			retryDelay = 300
		}
		irc.connection.ReconnectFreq = time.Duration(retryDelay) * time.Second
		if err != nil {
			slog.Error("Error connecting", "error", err)
			slog.Error("Retrying connect", "retryDelay", retryDelay)
		} else {
			return nil
		}
		sleep := time.NewTimer(time.Duration(retryDelay) * time.Second)
		select {
		case <-sleep.C:
		//NOOP
		case <-sigWait:
			return fmt.Errorf("terminate Signal received")
		}
	}
}
