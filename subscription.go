package tbcomctl

import (
	"context"
	"fmt"

	tb "gopkg.in/tucnak/telebot.v3"
)

// SubChecker is controller to check the chat subscription.
type SubChecker struct {
	commonCtl
	chats     []int64
	showList  bool
	chatCache map[int64]*tb.Chat
	pl        *Picklist
}

type SCOption func(sc *SubChecker)

func SCOptShowList(b bool) SCOption {
	return func(sc *SubChecker) {
		sc.showList = true
	}
}

func SCOptFallbackLang(lang string) SCOption {
	return func(sc *SubChecker) {
		optFallbackLang(lang)(&sc.commonCtl)
	}
}

// NewSubChecker creates new subscription checker that checks the subscription
// on the desired channels.  Boter must be added to channels for this to work.
func NewSubChecker(name string, t Texter, chats []int64, opts ...SCOption) *SubChecker {
	sc := &SubChecker{
		commonCtl: newCommonCtl(name),
		chats:     chats,
	}
	for _, o := range opts {
		o(sc)
	}
	// SubChecker uses picklist for its filthy job.
	pl := NewPicklist(
		"$subcheck"+randString(8), // assigning a fake name
		&TVC{TextFn: t.Text, ValuesFn: sc.valuesFn, CBfn: sc.callback},
		PickOptRemoveButtons(true),
	)
	sc.pl = pl
	return sc
}

func (sc *SubChecker) valuesFn(_ context.Context, c tb.Context) ([]string, error) {
	p := PrinterContext(c, sc.pl.fallbackLang)
	return []string{p.Sprintf(MsgSubCheck)}, nil
}

func (sc *SubChecker) callback(_ context.Context, c tb.Context) error {
	b := c.Bot()

	// check if the user is subCount
	var subCount int // count of subscribed channels.
	for _, chID := range sc.chats {
		ch, err := sc.cachedChat(c, chID)
		if err != nil {
			return fmt.Errorf("internal error: can't resolve chat ID: %d: %w", chID, err)
		}
		cm, err := b.ChatMemberOf(ch, c.Sender())
		if err != nil {
			lg.Printf("error: %s", err)
			continue
		}
		dlg.Printf("user %s has role %s", Userinfo(c.Sender()), cm.Role)
		if !(cm.Role == "left" || cm.Role == "kicked" || cm.Role == "") {
			subCount++
		}
	}
	if len(sc.chats) != subCount {
		// show alert if not
		pr := Printer(c.Sender().LanguageCode)
		return &Error{Type: TErrRetry, Msg: pr.Sprintf(MsgSubNoSub), Alert: true}
	}
	return nil
}

func (sc *SubChecker) Handler(c tb.Context) error {
	return sc.pl.Handler(c)
}

// cachedChat tries to get the chat information from cache, if it fails, gets
// the chat information via API.
func (sc *SubChecker) cachedChat(c tb.Context, id int64) (*tb.Chat, error) {
	if sc.chatCache == nil {
		sc.chatCache = make(map[int64]*tb.Chat)
	}
	ch, ok := sc.chatCache[id]
	if !ok {
		var err error
		ch, err = c.Bot().ChatByID(id)
		if err != nil {
			return ch, err
		}
		sc.chatCache[id] = ch
	} else {
		lg.Printf("using cached value for: %s", id)
	}
	return ch, nil
}
