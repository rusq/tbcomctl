package tbcomctl

import (
	"context"
	"fmt"

	tb "gopkg.in/tucnak/telebot.v2"
)

// SubChecker is controller to check the chat subscription.
type SubChecker struct {
	commonCtl
	chats     []string
	showList  bool
	chatCache map[string]*tb.Chat
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
func NewSubChecker(b BotChecker, name string, textFn TextFunc, chats []string, opts ...SCOption) *SubChecker {
	sc := &SubChecker{
		commonCtl: newCommonCtl(b, name, textFn),
		chats:     chats,
	}
	for _, o := range opts {
		o(sc)
	}
	pl := NewPicklist(b, "", textFn, sc.valuesFn, sc.callback, PickOptRemoveButtons(true))
	sc.pl = pl
	return sc
}

func (sc *SubChecker) valuesFn(ctx context.Context, u *tb.User) ([]string, error) {
	p := Printer(u.LanguageCode, sc.pl.lang)
	return []string{p.Sprintf(MsgSubCheck)}, nil
}

func (sc *SubChecker) callback(ctx context.Context, cb *tb.Callback) error {
	b := sc.b.(BotChecker)
	// check if the user is subscribed
	var subscribed int
	// show alert if not
	for _, chID := range sc.chats {
		ch, err := sc.cachedChat(chID)
		if err != nil {
			return fmt.Errorf("internal error: can't resolve chat ID: %s: %s", chID, err)
		}
		cm, err := b.ChatMemberOf(ch, cb.Sender)
		if err != nil {
			lg.Printf("error: %s", err)
			continue
		}
		dlg.Printf("user %s has role %s", Userinfo(cb.Sender), cm.Role)
		if !(cm.Role == "left" || cm.Role == "kicked" || cm.Role == "") {
			subscribed++
		}
	}
	if len(sc.chats) != subscribed {
		pr := Printer(cb.Sender.LanguageCode)
		return &Error{Type: TErrRetry, Msg: pr.Sprintf(MsgSubNoSub), Alert: true}
	}
	return nil
}

func (sc *SubChecker) Handler(m *tb.Message) {
	sc.pl.Handler(m)
}

func (sc *SubChecker) cachedChat(id string) (*tb.Chat, error) {
	if sc.chatCache == nil {
		sc.chatCache = make(map[string]*tb.Chat)
	}
	b := sc.b.(BotChecker)
	ch, ok := sc.chatCache[id]
	if !ok {
		var err error
		ch, err = b.ChatByID(id)
		if err != nil {
			return ch, err
		}
		sc.chatCache[id] = ch
	} else {
		lg.Printf("using cached value for: %s", id)
	}
	return ch, nil
}
