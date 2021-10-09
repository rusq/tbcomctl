package tbcomctl

import (
	"context"
	"fmt"

	tb "gopkg.in/tucnak/telebot.v3"
)

// Message is the controller that sends a message.
type Message struct {
	commonCtl
	opts []interface{}
}

var _ Controller = &Message{}

// NewMessage creates new Message Controller.  One must pass Bot instance, name
// of the controller, text function that returns the desired message and
// optionally any sendOpts that will be supplied to telebot.Bot.Send.
func NewMessage(b Boter, name string, textfn TextFunc, sendOpts ...interface{}) *Message {
	return &Message{
		commonCtl: newCommonCtl(b, name, textfn),
		opts:      sendOpts,
	}
}

// NewMessageText is a convenience wrapper for NewMessage with a fixed text.
func NewMessageText(b Boter, name, text string, sendOpts ...interface{}) *Message {
	return NewMessage(b, name, TextFn(text), sendOpts...)
}

// Handler is the Message controller's message handler.
func (m *Message) Handler(c tb.Context) error {
	ctx := WithController(context.Background(), m)
	txt, err := m.textFn(ctx, c.Sender())
	if err != nil {
		return fmt.Errorf("tbcomctl: message: text function error: %s: %w", Userinfo(c.Sender()), err)
	}

	outbound, err := m.sendOrEdit(c.Message(), txt, m.opts...)
	if err != nil {
		return fmt.Errorf("tbcomctl: message: send error: %s: %w", Userinfo(c.Sender()), err)
	}
	m.register(c.Sender(), outbound.ID)
	m.unregister(c.Sender(), outbound.ID)
	return nil
}
