package irc

import (
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ergochat/irc-go/ircevent"
	"github.com/ergochat/irc-go/ircmsg"
)

//Logger interface for loosely typed printf-style formatted error logging messages
type Logger interface {
	// Debugf uses fmt.Sprintf to log a templated message with an debug priority
	Debugf(template string, args ...interface{})
	// Infof uses fmt.Sprintf to log a templated message with an info priority
	Infof(template string, args ...interface{})
	// Warnf uses fmt.Sprintf to log a templated message with an warning priority
	Warnf(template string, args ...interface{})
	// Errorf uses fmt.Sprintf to log a templated message with an error priority
	Errorf(template string, args ...interface{})
	// Panicf uses fmt.Sprintf to log a templated message with an error priority
	Panicf(template string, args ...interface{})
	// Fatalf uses fmt.Sprintf to log a templated message with a fatal priority and then exit the application
	Fatalf(template string, args ...interface{})
}

type Connection struct {
	connection   *ircevent.Connection
	FloodProfile string
	logger           Logger
	connected bool
	limiter *RateLimiter
}

func NewIRC(server, password, nickname, realname string, useTLS, useSasl bool, saslUser, saslPass string,
	logger Logger, floodProfile string) *Connection {
	connection := &Connection{
		connection: &ircevent.Connection{
			Server:          server,
			Nick:            nickname,
			User:            nickname,
			RealName:        realname,
			Password:        password,
			SASLLogin:       saslUser,
			SASLPassword:    saslPass,
			SASLMech:        "PLAIN",
			Timeout:         1 * time.Minute,
			KeepAlive:       4 * time.Minute,
			UseTLS:          useTLS,
			UseSASL:         useSasl,
			EnableCTCP:      true,
			Debug: true,
			TLSConfig: &tls.Config{
				InsecureSkipVerify:          true,
			},
			QuitMessage: " ",
			Log: log.Default(),
		},
		FloodProfile: floodProfile,
		logger:       logger,
	}
	connection.limiter = connection.NewRateLimiter(floodProfile)
	logger.Infof("Creating new IRC")
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

func (irc *Connection) Connect() error {
	irc.logger.Infof("Connecting to IRC: %s", irc.connection.Server)
	err := irc.connection.Connect()
	if err != nil {
		return err
	}
	return nil
}

func (irc *Connection) Wait() {
	irc.logger.Debugf("Waiting for IRC to finish")
	irc.connection.Loop()
	irc.logger.Debugf("IRC Finished")
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
			irc.logger.Errorf("Error connecting: %s", err.Error())
			irc.logger.Infof("Retrying connect in %d", retryDelay)
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
