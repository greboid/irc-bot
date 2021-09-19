package rpc

import (
	"context"
	"fmt"
	"reflect"
	"testing"
)

type fakeIRCSender struct {
	sendMessages []string
}

func (s *fakeIRCSender) SendRelayMessage(channel string, nickname string, message string) error {
	panic("implement me")
}

func (s *fakeIRCSender) Join(s2 string) error {
	panic("implement me")
}

func (s *fakeIRCSender) Part(s2 string) error {
	panic("implement me")
}

func (s *fakeIRCSender) SendRawf(string string, i ...interface{}) error {
	fmt.Printf("----\n")
	fmt.Printf(string, i...)
	fmt.Printf("\n")
	fmt.Printf("----\n")
	s.sendMessages = append(s.sendMessages, fmt.Sprintf(string, i...))
	return nil
}

func Test_pluginServer_SendChannelMessage(t *testing.T) {
	tests := []struct {
		name         string
		sender       *fakeIRCSender
		req          *ChannelMessage
		wantErr      bool
		wantMessages []string
	}{
		{
			name:   "Send channel message",
			sender: &fakeIRCSender{},
			req: &ChannelMessage{
				Channel: "#test",
				Message: "This is a test",
				Source:  "",
				Tags:    nil,
			},
			wantErr:      false,
			wantMessages: []string{"PRIVMSG #test :This is a test"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := &pluginServer{
				sender:    tt.sender,
				functions: nil,
			}
			_, err := ps.SendChannelMessage(context.Background(), tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("SendChannelMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(tt.sender.sendMessages, tt.wantMessages) {
				t.Errorf("SendChannelMessage() got = %#+v, want %#+v", tt.sender.sendMessages, tt.wantMessages)
			}
		})
	}
}
