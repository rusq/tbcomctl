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
	noUpdate      bool
	msgChoose     bool

	vFn  ValuesFunc
	cbFn BtnCallbackFunc

	btnPattern []uint
}

var _ Controller = &Picklist{}

type PicklistOption func(p *Picklist)

// PickOptRemoveButtons set the Remove Buttons option.  If Remove Buttons is
// set, the inline buttons will be removed once the user make the choice.
func PickOptRemoveButtons(b bool) PicklistOption {
	return func(p *Picklist) {
		p.removeButtons = b
	}
}

func PickOptOverwrite(b bool) PicklistOption {
	return func(p *Picklist) {
		p.overwrite = b
	}
}

// PickOptNoUpdate sets the No Update option.  If No Update is set, the text is
// not updated once the user makes their choice.
func PickOptNoUpdate(b bool) PicklistOption {
	return func(p *Picklist) {
		p.noUpdate = b
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

// PickOptBtnPattern sets the inline markup button pattern.
// Each unsigned integer in the pattern represents the number
// of buttons shown on each of the rows.
//
// Example:
//
//   pattern: []uint{1, 2, 3}
//   will produce the following markup for the picklist choices
//
//   +-------------------+
//   | Picklist text     |
//   +-------------------+
//   |     button 1      |
//   +---------+---------+
//   | button 2| button 3|
//   +------+--+---+-----+
//   | btn4 | btn5 | btn6|
//   +------+------+-----+
func PickOptBtnPattern(pattern []uint) PicklistOption {
	return func(p *Picklist) {
		p.btnPattern = pattern
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
		TextFn(text),
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
	// if overwrite is true and prev is not nil - edit, otherwise - send.
	outbound, err := p.sendOrEdit(m, text, &tb.SendOptions{ReplyMarkup: markup, ParseMode: tb.ModeHTML})
	if err != nil {
		lg.Println(err)
		return
	}
	_ = p.register(m.Sender, outbound.ID)
	p.logOutgoingMsg(outbound, fmt.Sprintf("picklist: %q", strings.Join(values, "*")))
}

func (p *Picklist) Callback(cb *tb.Callback) {
	p.logCallback(cb)

	var resp tb.CallbackResponse
	err := p.cbFn(WithController(context.Background(), p), cb)
	if err != nil {
		if e, ok := err.(*Error); !ok {
			p.editMsg(cb)
			p.b.Respond(cb, &tb.CallbackResponse{Text: err.Error(), ShowAlert: true})
			p.unregister(cb.Sender, cb.Message.ID)
			return
		} else {
			switch e.Type {
			case TErrNoChange:
				resp = tb.CallbackResponse{}
			case TErrRetry:
				p.b.Respond(cb, &tb.CallbackResponse{Text: e.Msg, ShowAlert: e.Alert})
				return
			default:
				p.b.Respond(cb, &tb.CallbackResponse{Text: e.Msg, ShowAlert: e.Alert})
			}
		}
	} else { // err == nil
		resp = tb.CallbackResponse{Text: MsgOK}
	} // if err != nil

	p.SetValue(cb.Sender.Recipient(), cb.Data)
	// edit message
	p.editMsg(cb)
	p.b.Respond(cb, &resp)
	p.nextHandler(cb)
	p.unregister(cb.Sender, cb.Message.ID)
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
	if p.noUpdate {
		return true
	}

	values, err := p.vFn(WithController(context.Background(), p), cb.Sender)
	if err != nil {
		p.processErr(convertToMsg(cb), err)
		return false
	}

	markup := p.inlineMarkup(values)
	if _, err := p.b.Edit(cb.Message,
		p.format(cb.Sender, text),
		&tb.SendOptions{ParseMode: tb.ModeHTML, ReplyMarkup: markup},
	); err != nil {
		lg.Println(err)
		return false
	}

	return true
}

func (p *Picklist) format(u *tb.User, text string) string {
	if p.msgChoose {
		pr := Printer(u.LanguageCode, p.lang)
		text = pr.Sprintf("%s\n\n%s", text, pr.Sprintf(MsgChooseVal))
	}
	return text
}

func (p *Picklist) inlineMarkup(values []string) *tb.ReplyMarkup {
	if len(p.btnPattern) == 0 {
		return ButtonMarkup(p.b, values, p.maxButtons, p.Callback)
	}
	m, err := ButtonPatternMarkup(p.b, values, p.btnPattern, p.Callback)
	if err != nil {
		panic(err) // TODO handle this more gracefully.
	}
	return m
}

func (p *Picklist) processErr(m *tb.Message, err error) {
	pr := Printer(m.Sender.LanguageCode, p.lang)
	lg.Println(err)
	if p.errFn == nil {
		p.b.Send(m.Sender, pr.Sprintf(MsgUnexpected))
	} else {
		dlg.Println("calling error message handler")
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
