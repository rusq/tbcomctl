package tbcomctl

import (
	"context"
	"fmt"
	"strings"

	tb "gopkg.in/tucnak/telebot.v2"
)

const (
	defNumButtons = 4 // in a row.
	maxButtons    = 8 // in a row.
)

type Picklist struct {
	commonCtl
	*buttons
	removeButtons bool

	vFn  ValuesFunc
	cbFn BtnCallbackFunc
}

var _ Controller = &Picklist{}

type PicklistOption func(p *Picklist)

func PickOptRemoveButtons(b bool) PicklistOption {
	return func(p *Picklist) {
		p.removeButtons = b
	}
}

func PickOptPrivateOnly(b bool) PicklistOption {
	return func(p *Picklist) {
		optPrivateOnly(b)(&p.commonCtl)
	}
}

func PickOptErrFunc(fn ErrFunc) PicklistOption {
	return func(p *Picklist) {
		optErrFunc(fn)(&p.commonCtl)
	}
}

func PickOptFallbackLang(lang string) PicklistOption {
	return func(p *Picklist) {
		optFallbackLang(lang)(&p.commonCtl)
	}
}

func PickOptMaxInlineButtons(n int) PicklistOption {
	return func(p *Picklist) {
		p.buttons.SetMaxButtons(n)
	}
}

// NewPicklist creates a new picklist.
func NewPicklist(b Boter, name string, textFn TextFunc, valuesFn ValuesFunc, callbackFn BtnCallbackFunc, opts ...PicklistOption) *Picklist {
	if b == nil {
		panic("bot is required")
	}
	if textFn == nil || valuesFn == nil || callbackFn == nil {
		panic("one or more of the functions not set")
	}
	p := &Picklist{
		commonCtl: newCommonCtl(b, name, textFn),
		vFn:       valuesFn,
		cbFn:      callbackFn,
		buttons:   &buttons{maxButtons: defNumButtons},
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// NewPicklistText is a convenience function to return picklist with fixed text and values.
func NewPicklistText(b Boter, name string, text string, values []string, callbackFn BtnCallbackFunc, opts ...PicklistOption) *Picklist {
	return NewPicklist(b,
		name,
		func(context.Context, *tb.User) (string, error) { return text, nil },
		func(ctx context.Context, u *tb.User) ([]string, error) { return values, nil },
		callbackFn,
		opts...,
	)
}

func (p *Picklist) Handler(m *tb.Message) {
	if p.privateOnly && !m.Private() {
		return
	}
	ctrlCtx := WithController(context.Background(), p)
	values, err := p.vFn(ctrlCtx, m.Sender)
	if err != nil {
		p.processErr(m, err)
		return
	}

	// generate markup
	markup := p.inlineMarkup(values)
	// send message with markup
	pr := Printer(m.Sender.LanguageCode, p.lang)
	text, err := p.textFn(WithController(context.Background(), p), m.Sender)
	if err != nil {
		lg.Printf("error while generating text for controller: %s: %s", p.name, err)
		p.b.Send(m.Sender, pr.Sprintf(MsgUnexpected))
		return
	}
	outbound, err := p.b.Send(m.Sender,
		fmt.Sprintf("%s\n\n%s", text, pr.Sprintf(MsgChooseVal)),
		&tb.SendOptions{ReplyMarkup: markup, ParseMode: tb.ModeHTML},
	)
	if err != nil {
		lg.Println(err)
		return
	}
	_ = p.register(outbound.ID)
	p.logOutgoingMsg(outbound, fmt.Sprintf("picklist: %q", strings.Join(values, "*")))
}

func (p *Picklist) Callback(cb *tb.Callback) {
	p.logCallback(cb)

	var resp tb.CallbackResponse
	err := p.cbFn(WithController(context.Background(), p), cb)
	switch err {
	case nil:
		resp = tb.CallbackResponse{Text: MsgOK}
	case ErrNoChange:
		resp = tb.CallbackResponse{}
	case ErrRetry:
		p.b.Respond(cb, &tb.CallbackResponse{Text: MsgRetry, ShowAlert: true})
		return
	default: //err !=nil
		p.editMsg(cb)
		p.b.Respond(cb, &tb.CallbackResponse{Text: err.Error(), ShowAlert: true})
		p.unregister(cb.Message.ID)
		return
	}

	p.SetValue(cb.Sender.Recipient(), cb.Data)
	// edit message
	p.editMsg(cb)
	p.b.Respond(cb, &resp)
	p.nextHandler(cb)
	p.unregister(cb.Message.ID)
}

func (p *Picklist) editMsg(cb *tb.Callback) bool {
	text, err := p.textFn(WithController(context.Background(), p), cb.Sender)
	if err != nil {
		lg.Println(err)
		return false
	}

	if p.removeButtons {
		if _, err := p.b.Edit(
			cb.Message,
			text,
			&tb.SendOptions{ParseMode: tb.ModeHTML},
		); err != nil {
			lg.Println(err)
			return false
		}
		return true
	}

	pr := Printer(cb.Sender.LanguageCode, p.lang)
	values, err := p.vFn(WithController(context.Background(), p), cb.Sender)
	if err != nil {
		p.processErr(convertToMsg(cb), err)
		return false
	}

	markup := p.inlineMarkup(values)
	if _, err := p.b.Edit(cb.Message,
		fmt.Sprintf("%s\n\n%s", text, pr.Sprintf(MsgChooseVal)),
		&tb.SendOptions{ParseMode: tb.ModeHTML, ReplyMarkup: markup},
	); err != nil {
		lg.Println(err)
		return false
	}

	return true
}

func (p *Picklist) inlineMarkup(values []string) *tb.ReplyMarkup {
	return ButtonMarkup(p.b, values, p.maxButtons, p.Callback)
}

func (p *Picklist) processErr(m *tb.Message, err error) {
	pr := Printer(m.Sender.LanguageCode, p.lang)
	lg.Println(err)
	if p.errFn == nil {
		p.b.Send(m.Sender, pr.Sprintf(MsgUnexpected))
	} else {
		lg.Println("calling error message handler")
		p.errFn(WithController(context.Background(), p), m, err)
	}
}

func convertToMsg(cb *tb.Callback) *tb.Message {
	msg := cb.Message
	msg.Sender = cb.Sender
	return msg
}

func (p *Picklist) nextHandler(cb *tb.Callback) {
	if p.next != nil {
		// this call is part of the pipeline
		p.next.Handler(convertToMsg(cb))
	}
}
