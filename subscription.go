package tbcomctl

import (
	"context"

	tb "gopkg.in/tucnak/telebot.v2"
)

// SubChecker is controller to check the chat subscription.
type SubChecker struct {
	commonCtl
	chats    []string
	showList bool
}

type SCOption func(sc *SubChecker)

func SCOptShowList(b bool) SCOption {
	return func(sc *SubChecker) {
		sc.showList = true
	}
}

func NewSubChecker(b Boter, name string, textFn TextFunc, chats []string, opts ...SCOption) *SubChecker {
	sc := &SubChecker{
		commonCtl: newCommonCtl(b, name, textFn),
		chats:     chats,
	}
	for _, o := range opts {
		o(sc)
	}
	return sc
}

func (sc *SubChecker) Handler(m *tb.Message) {
	ctx := WithController(context.Background(), sc)
	text, err := sc.textFn(ctx, m.Sender)
	if err != nil {
		lg.Printf("%s textfn error: %s", caller(0), err)
		return
	}
	// TODO: add list of channels depending on showList
	outbound, err := sc.b.Send(m.Sender, text)
	if err != nil {
		lg.Printf("%s error: %s", caller(0), err)
		return
	}
	sc.register(outbound.ID)
	sc.logOutgoingMsg(outbound)
}
