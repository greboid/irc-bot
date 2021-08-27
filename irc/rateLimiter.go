package irc

import (
	"context"
	"math"
	"time"

	"github.com/ergochat/irc-go/ircmsg"
	"golang.org/x/time/rate"
)

func (irc *Connection) NewRateLimiter(floodProfile string) *RateLimiter {
	rl := RateLimiter{}
	rl.Init(floodProfile)
	irc.connection.AddConnectCallback(func(ircmsg.Message) {
		rl.received001 = true
	})
	return &rl
}

type RateLimiter struct {
	limiter     *rate.Limiter
	received001 bool
}

func (r *RateLimiter) Init(profile string) {
	switch profile {
	case "unlimited":
		r.limiter = rate.NewLimiter(rate.Inf, math.MaxInt)
	case "restrictive":
		r.limiter = rate.NewLimiter(rate.Every(1500 * time.Millisecond), 3)
	}
}

func (r *RateLimiter) Wait() error {
	if r.received001 {
		if err := r.limiter.WaitN(context.Background(), 1); err != nil {
			return err
		}
	}
	return nil
}
