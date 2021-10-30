package tbcomctl

import (
	"context"
	"errors"
	"fmt"
	"runtime/trace"
	"strings"

	tb "gopkg.in/tucnak/telebot.v3"
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
func NewPicklist(name string, textFn TextFunc, valuesFn ValuesFunc, callbackFn BtnCallbackFunc, opts ...PicklistOption) *Picklist {
	if textFn == nil || valuesFn == nil || callbackFn == nil {
		panic("one or more of the functions not set")
	}
	p := &Picklist{
		commonCtl: newCommonCtl(name, textFn),
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
func NewPicklistText(name string, text string, values []string, callbackFn BtnCallbackFunc, opts ...PicklistOption) *Picklist {
	return NewPicklist(
		name,
		TextFn(text),
		func(ctx context.Context, u *tb.User) ([]string, error) { return values, nil },
		callbackFn,
		opts...,
	)
}

func (p *Picklist) Handler(c tb.Context) error {
	m := c.Message()
	if p.privateOnly && !m.Private() {
		return nil
	}
	ctrlCtx := WithController(context.Background(), p)
	values, err := p.vFn(ctrlCtx, c.Sender())
	if err != nil {
		p.processErr(c, err)
		return err
	}

	// generate markup
	markup := p.inlineMarkup(c, values)
	// send message with markup
	pr := Printer(c.Sender().LanguageCode, p.lang)
	text, err := p.textFn(WithController(context.Background(), p), c.Sender())
	if err != nil {
		c.Send(pr.Sprintf(MsgUnexpected))
		return fmt.Errorf("error while generating text for controller: %s: %w", p.name, err)
	}
	// if overwrite is true and prev is not nil - edit, otherwise - send.
	outbound, err := p.sendOrEdit(c, text, &tb.SendOptions{ReplyMarkup: markup, ParseMode: tb.ModeHTML})
	if err != nil {
		return err
	}
	_ = p.register(c.Sender(), outbound.ID)
	p.logOutgoingMsg(outbound, fmt.Sprintf("picklist: %q", strings.Join(values, "*")))
	return nil
}

func (p *Picklist) Callback(c tb.Context) error {
	ctx, task := trace.NewTask(context.Background(), "Picklist.Callback")
	defer task.End()

	cb := c.Callback()
	p.logCallback(cb)

	var resp tb.CallbackResponse
	err := p.cbFn(WithController(ctx, p), c)
	if err != nil {
		if errors.Is(err, BackPressed) {
			// Back button is pressed.
			if err := c.Respond(&tb.CallbackResponse{}); err != nil {
				trace.Log(ctx, "respond", err.Error())
			}
			if p.prev != nil {
				p.prev.Handler(c)
			}
			return nil
		}
		if e, ok := err.(*Error); !ok {
			p.editMsg(ctx, c)
			if err := c.Respond(&tb.CallbackResponse{Text: err.Error(), ShowAlert: true}); err != nil {
				trace.Log(ctx, "respond", err.Error())
			}
			p.unregister(c.Sender(), cb.Message.ID)
			return e
		} else {
			switch e.Type {
			case TErrNoChange:
				resp = tb.CallbackResponse{}
			case TErrRetry:
				c.Respond(&tb.CallbackResponse{Text: e.Msg, ShowAlert: e.Alert})
				return e
			default:
				c.Respond(&tb.CallbackResponse{Text: e.Msg, ShowAlert: e.Alert})
			}
		}
	} else {
		resp = tb.CallbackResponse{Text: MsgOK}
	}

	p.SetValue(c.Sender().Recipient(), cb.Data)
	// edit message
	p.editMsg(ctx, c)
	if err := c.Respond(&resp); err != nil {
		trace.Log(ctx, "respond", err.Error())
	}
	err = p.nextHandler(c)
	p.unregister(c.Sender(), cb.Message.ID)
	return err
}

func (p *Picklist) editMsg(ctx context.Context, c tb.Context) bool {
	text, err := p.textFn(WithController(ctx, p), c.Sender())
	if err != nil {
		lg.Println(err)
		trace.Log(ctx, "textFn", err.Error())
		return false
	}

	if p.removeButtons {
		if err := c.Edit(
			text,
			&tb.SendOptions{ParseMode: tb.ModeHTML},
		); err != nil {
			lg.Println(err)
			trace.Log(ctx, "Edit", err.Error())
			return false
		}
		return true
	}
	if p.noUpdate {
		return true
	}

	values, err := p.vFn(WithController(ctx, p), c.Sender())
	if err != nil {
		trace.Log(ctx, "vFn", err.Error())
		p.processErr(c, err)
		return false
	}

	markup := p.inlineMarkup(c, values)
	if err := c.Edit(
		p.format(c.Sender(), text),
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

func (p *Picklist) inlineMarkup(c tb.Context, values []string) *tb.ReplyMarkup {
	if len(p.btnPattern) == 0 {
		return ButtonMarkup(c, values, p.maxButtons, p.Callback)
	}
	m, err := ButtonPatternMarkup(c, values, p.btnPattern, p.Callback)
	if err != nil {
		panic(err) // TODO handle this more gracefully.
	}
	return m
}

// processErr logs the error, and if the error handling function errFn is not
// nil, invokes it.
func (p *Picklist) processErr(c tb.Context, err error) {
	var m *tb.Message
	if cb := c.Callback(); cb != nil {
		m = convertToMsg(cb)
	} else {
		m = c.Message()
	}
	pr := Printer(c.Sender().LanguageCode, p.lang)
	lg.Println(err)
	if p.errFn == nil {
		c.Send(pr.Sprintf(MsgUnexpected))
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

func (p *Picklist) nextHandler(c tb.Context) error {
	if p.next != nil {
		return p.next.Handler(c)
	}
	return nil
}
