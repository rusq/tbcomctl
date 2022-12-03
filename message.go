package tbcomctl

import (
	"context"
	"fmt"

	tb "gopkg.in/telebot.v3"
)

// Message is the controller that sends a message.  It can be used to send a
// confirmation message at the end of the Form.
type Message struct {
	commonCtl
	txt  Texter
	opts []interface{}
}

var _ Controller = &Message{} // assertion

// NewMessage creates new Message Controller.  One must pass Bot instance, name
// of the controller, text function that returns the desired message and
// optionally any sendOpts that will be supplied to telebot.Bot.Send.
func NewMessage(name string, tx Texter, sendOpts ...interface{}) *Message {
	return &Message{
		commonCtl: newCommonCtl(name),
		opts:      sendOpts,
		txt:       tx,
	}
}

// NewMessageText is a convenience wrapper for NewMessage with a fixed text.
func NewMessageText(name, text string, sendOpts ...interface{}) *Message {
	return NewMessage(name, NewTexter(text), sendOpts...)
}

// Handler is the Message controller's message handler.
func (m *Message) Handler(c tb.Context) error {
	ctx := WithController(context.Background(), m)
	txt, err := m.txt.Text(ctx, c)
	if err != nil {
		return fmt.Errorf("tbcomctl: message: text function error: %s: %w", Userinfo(c.Sender()), err)
	}

	outbound, err := m.sendOrEdit(c, txt, m.opts...)
	if err != nil {
		return fmt.Errorf("tbcomctl: message: send error: %s: %w", Userinfo(c.Sender()), err)
	}
	m.reg.Register(c.Sender(), outbound.ID)
	m.reg.Unregister(c.Sender(), outbound.ID)
	return nil
}
