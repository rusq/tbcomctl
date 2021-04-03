package tbcomctl

import (
	"context"

	tb "gopkg.in/tucnak/telebot.v2"
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
func (m *Message) Handler(msg *tb.Message) {
	ctx := WithController(context.Background(), m)
	txt, err := m.textFn(ctx, msg.Sender)
	if err != nil {
		lg.Printf("tbcomctl: message: text function error: %s: %s", Userinfo(msg.Sender), err)
		return
	}

	var outbound *tb.Message
	if m.overwrite && m.prev != nil {
		msgID, ok := m.prev.OutgoingID(msg.Sender.Recipient())
		if !ok {
			lg.Println("can't find previous message ID for %s", Userinfo(msg.Sender))
			return
		}
		prevMsg := tb.Message{ID: msgID, Chat: msg.Chat}
		outbound, err = m.b.Edit(&prevMsg,
			txt,
			m.opts...,
		)
	} else {
		outbound, err = m.b.Send(msg.Chat, txt, m.opts...)
	}
	if err != nil {
		lg.Printf("tbcomctl: message: send error: %s: %s", Userinfo(msg.Sender), err)
		return
	}
	m.register(msg.Sender, outbound.ID)
	m.unregister(msg.Sender, outbound.ID)
}
