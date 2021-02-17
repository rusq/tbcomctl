package tbcomctl

import (
	"crypto/sha1"
	"fmt"
	"log"
	"strings"

	"github.com/rusq/dlog"

	tb "gopkg.in/tucnak/telebot.v2"
)

const (
	defNumButtons = 4 // in a row.
	maxButtons    = 8 // in a row.
)

type Picklist struct {
	*commonCtl
	removeButtons bool
	maxButtons    int

	vFn  ValuesFunc
	cbFn CallbackFunc
}

type PicklistOption func(p *Picklist)

func PickOptRemoveButtons(b bool) PicklistOption {
	return func(p *Picklist) {
		p.removeButtons = b
	}
}

func PickOptPrivateOnly(b bool) PicklistOption {
	return func(p *Picklist) {
		optPrivateOnly(b)(p.commonCtl)
	}
}

func PickOptErrFunc(fn ErrFunc) PicklistOption {
	return func(p *Picklist) {
		optErrFunc(fn)(p.commonCtl)
	}
}

func PickOptFallbackLang(lang string) PicklistOption {
	return func(p *Picklist) {
		optFallbackLang(lang)(p.commonCtl)
	}
}

func PickOptMaxInlineButtons(n int) PicklistOption {
	return func(p *Picklist) {
		if n <= 0 || maxButtons < n {
			n = defNumButtons
		}
		p.maxButtons = n
	}
}

func NewPicklist(b Boter, textFn TextFunc, valuesFn ValuesFunc, callbackFn CallbackFunc, next MsgHandler, opts ...PicklistOption) *Picklist {
	if b == nil {
		panic("bot is required")
	}
	if textFn == nil || valuesFn == nil || callbackFn == nil {
		panic("one or more of the functions not set")
	}
	p := &Picklist{
		commonCtl: &commonCtl{
			b:      b,
			textFn: textFn,
			next:   next,
		},
		vFn:        valuesFn,
		cbFn:       callbackFn,
		maxButtons: defNumButtons,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

func (p *Picklist) Command() MsgHandler {
	return func(m *tb.Message) {
		if p.privateOnly && !m.Private() {
			return
		}
		values, err := p.vFn(m.Sender)
		if err != nil {
			p.processErr(p.b, m, err)
			return
		}

		// generate markup
		markup := p.inlineMarkup(values)
		// send message with markup
		pr := Printer(m.Sender.LanguageCode, p.lang)
		resp, err := p.b.Send(m.Sender,
			fmt.Sprintf("%s\n\n%s", p.textFn(m.Sender), pr.Sprintf(MsgChooseVal)),
			&tb.SendOptions{ReplyMarkup: markup, ParseMode: tb.ModeHTML},
		)
		if err != nil {
			dlog.Println(err)
			return
		}
		_ = p.reqRegister(resp.ID)
		p.logOutgoingMsg(resp, fmt.Sprintf("picklist: %q", strings.Join(values, "*")))
	}
}

func (p *Picklist) Callback() func(*tb.Callback) {
	return func(cb *tb.Callback) {
		p.logCallback(cb)

		var resp tb.CallbackResponse
		err := p.cbFn(cb)
		switch err {
		case nil:
			resp = tb.CallbackResponse{Text: MsgOK}
		case ErrNoChange:
			resp = tb.CallbackResponse{}
		case ErrRetry:
			p.b.Respond(cb, &tb.CallbackResponse{Text: MsgRetry})
			return
		default: //err !=nil
			p.editMsg(cb)
			p.b.Respond(cb, &tb.CallbackResponse{Text: err.Error()})
			p.reqUnregister(cb.Message.ID)
			return
		}
		// edit message
		p.editMsg(cb)
		p.b.Respond(cb, &resp)
		p.nextHandler(cb)
		p.reqUnregister(cb.Message.ID)
	}
}

func (p *Picklist) editMsg(cb *tb.Callback) bool {
	if p.removeButtons {
		if _, err := p.b.Edit(
			cb.Message,
			p.textFn(cb.Sender),
			&tb.SendOptions{ParseMode: tb.ModeHTML},
		); err != nil {
			dlog.Println(err)
			return false
		}
		return true
	}

	pr := Printer(cb.Sender.LanguageCode, p.lang)
	values, err := p.vFn(cb.Sender)
	if err != nil {
		p.processErr(p.b, callbackToMesg(cb), err)
		return false
	}

	markup := p.inlineMarkup(values)
	if _, err := p.b.Edit(cb.Message,
		fmt.Sprintf("%s\n\n%s", p.textFn(cb.Sender), pr.Sprintf(MsgChooseVal)),
		&tb.SendOptions{ParseMode: tb.ModeHTML, ReplyMarkup: markup},
	); err != nil {
		dlog.Println(err)
		return false
	}

	return true
}

var hasher = sha1.New

func hash(s string) string {
	h := hasher()
	h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func (p *Picklist) inlineMarkup(values []string) *tb.ReplyMarkup {
	selector := new(tb.ReplyMarkup)
	var btns []tb.Btn
	for _, label := range values {
		btn := selector.Data(label, hash(label), label)
		btns = append(btns, btn)
		p.b.Handle(&btn, p.Callback())
	}

	selector.Inline(organizeButtons(selector, btns, p.maxButtons)...)
	return selector
}

func (p *Picklist) processErr(b Boter, m *tb.Message, err error) {
	pr := Printer(m.Sender.LanguageCode, p.lang)
	log.Println(err)
	if p.errFn == nil {
		b.Send(m.Sender, pr.Sprintf(MsgUnexpected))
	} else {
		dlog.Println("calling error message handler")
		p.errFn(m, err)
	}
}

func callbackToMesg(cb *tb.Callback) *tb.Message {
	msg := cb.Message
	msg.Sender = cb.Sender
	return msg
}

func (p *Picklist) nextHandler(cb *tb.Callback) {
	if p.next != nil {
		// this call is part of the pipeline
		p.next(callbackToMesg(cb))
	}
}
