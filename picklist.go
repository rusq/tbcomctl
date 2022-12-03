package tbcomctl

import (
	"context"
	"errors"
	"fmt"
	"runtime/trace"
	"strings"

	tb "gopkg.in/telebot.v3"
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
	backBtn       bool

	tvc        TextValueCallbacker
	backBtnTxt Texter

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
		p.commonCtl.setOverwrite(b)
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

func PickOptErrHandler(h ErrorHandler) PicklistOption {
	return func(p *Picklist) {
		optErrFunc(h)(&p.commonCtl)
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
//	pattern: []uint{1, 2, 3}
//	will produce the following markup for the picklist choices
//
//	+-------------------+
//	| Picklist text     |
//	+-------------------+
//	|     button 1      |
//	+---------+---------+
//	| button 2| button 3|
//	+------+--+---+-----+
//	| btn4 | btn5 | btn6|
//	+------+------+-----+
func PickOptBtnPattern(pattern []uint) PicklistOption {
	return func(p *Picklist) {
		p.btnPattern = pattern
	}
}

func PickOptBtnBack(texter Texter) PicklistOption {
	return func(p *Picklist) {
		p.backBtn = true
		p.backBtnTxt = texter
	}
}

// PickOptDefaultSendOptions allows to set the default send options
func PickOptDefaultSendOptions(opts *tb.SendOptions) PicklistOption {
	return func(p *Picklist) {
		optDefaultSendOpts(opts)(&p.commonCtl)
	}
}

// NewPicklist creates a new picklist.
func NewPicklist(name string, tvc TextValueCallbacker, opts ...PicklistOption) *Picklist {
	p := &Picklist{
		commonCtl: newCommonCtl(name),
		tvc:       tvc,
		buttons:   &buttons{maxButtons: defNumButtons},
	}
	for _, opt := range opts {
		opt(p)
	}
	if p.backBtn {
		p.btnPattern = append(p.btnPattern, 1)
	}
	return p
}

// Handler is a handler function to use with telebot.Handle.
func (p *Picklist) Handler(c tb.Context) error {
	m := c.Message()
	if p.privateOnly && !m.Private() {
		// skips handling if we're set to operate in private mode.
		return nil
	}

	ctrlCtx := WithController(context.Background(), p)

	values, err := p.tvc.Values(ctrlCtx, c)
	if err != nil {
		p.processErr(c, err)
		return err
	}

	// send message with markup
	text, err := p.tvc.Text(ctrlCtx, c)
	if err != nil {
		c.Send(unexpectedErrorText(c))
		return fmt.Errorf("error while generating text for controller: %s: %w", p.name, err)
	}

	outbound, err := p.sendOrEdit(c, text, p.withMarkup(p.inlineMarkup(c, values)))
	if err != nil {
		return err
	}
	_ = p.reg.Register(c.Sender(), outbound.ID)

	p.logOutgoingMsg(outbound, fmt.Sprintf("picklist: %q", strings.Join(values, "*")))

	return nil
}

// withMarkup adds a markup to default send options.
func (cc *commonCtl) withMarkup(markup *tb.ReplyMarkup) *tb.SendOptions {
	var opts tb.SendOptions
	opts = *cc.sendOpts
	opts.ReplyMarkup = markup
	return &opts
}

// callback is the callback function that will be registered for the buttons.
func (p *Picklist) callback(c tb.Context) error {
	ctx, task := trace.NewTask(context.Background(), "Picklist.Callback")
	defer task.End()

	cb := c.Callback()
	p.logCallback(cb)

	var resp tb.CallbackResponse

	if p.backBtn {
		// back button is enabled, check if the callback data contains back button text.
		txt, err := p.backBtnTxt.Text(ctx, c)
		if err != nil {
			trace.Logf(ctx, "back button", "err=%s", err)
		}
		if c.Data() == txt {
			trace.Log(ctx, "callback", "back is pressed (option)")
			return p.handleBackButton(ctx, c)
		}
	}

	err := p.tvc.Callback(WithController(ctx, p), c)
	if err != nil {
		if errors.Is(err, BackPressed) {
			// user callback function might return "back button is pressed" as well
			trace.Log(ctx, "callback", "back is pressed (user)")
			return p.handleBackButton(ctx, c)
		}
		if e, ok := err.(*Error); !ok {
			p.editMsg(ctx, c)
			pr := PrinterContext(c, p.fallbackLang)
			if err := c.Respond(&tb.CallbackResponse{Text: pr.Sprintf(MsgUnexpected), ShowAlert: true}); err != nil {
				trace.Log(ctx, "respond", err.Error())
			}
			p.reg.Unregister(c.Sender(), cb.Message.ID)
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
	if err := p.editMsg(ctx, c); err != nil {
		lg.Printf("%s: error editing message: %s", caller(0), err)
	}
	if err := c.Respond(&resp); err != nil {
		trace.Log(ctx, "respond", err.Error())
	}
	err = p.nextHandler(c)
	p.reg.Unregister(c.Sender(), cb.Message.ID)
	return err
}

// editMsg edits the existing message, returning true, if the message was edited without errors.
func (p *Picklist) editMsg(ctx context.Context, c tb.Context) error {
	text, err := p.tvc.Text(WithController(ctx, p), c)
	if err != nil {
		trace.Log(ctx, "editMsg", err.Error())
		return err
	}

	if p.removeButtons {
		if err := c.Edit(
			text,
			p.sendOpts,
		); err != nil {
			trace.Log(ctx, "Edit", err.Error())
			return err
		}
		return nil
	}
	if p.noUpdate {
		return nil
	}

	values, err := p.tvc.Values(WithController(ctx, p), c)
	if err != nil {
		trace.Log(ctx, "vFn", err.Error())
		p.processErr(c, err)
		return err
	}

	if err := c.Edit(
		p.format(c.Sender(), text),
		p.commonCtl.withMarkup(p.inlineMarkup(c, values)),
	); err != nil {
		return err
	}

	return nil
}

// format formats the text for the user.
func (p *Picklist) format(u *tb.User, text string) string {
	if p.msgChoose {
		pr := Printer(u.LanguageCode, p.fallbackLang)
		text = pr.Sprintf("%s\n\n%s", text, pr.Sprintf(MsgChooseVal))
	}
	return text
}

// inlineMarkup generates the inline markup for the values.
func (p *Picklist) inlineMarkup(c tb.Context, values []string) *tb.ReplyMarkup {
	if p.backBtn {
		txt, err := p.backBtnTxt.Text(context.Background(), c)
		if err != nil {
			dlg.Println("backTextFn returned an error: %s", err)
		}
		values = append(values, txt)
	}
	if len(p.btnPattern) == 0 {
		return ButtonMarkup(c, values, p.maxButtons, p.callback)
	}
	m, err := ButtonPatternMarkup(c, values, p.btnPattern, p.callback)
	if err != nil {
		panic(err) // TODO handle this more gracefully.
	}
	return m
}

// processErr logs the error, and if the error handling function errFn is not
// nil, invokes it.
func (p *Picklist) processErr(c tb.Context, err error) {
	pr := PrinterContext(c, p.fallbackLang)
	lg.Printf("processing error: %s", err)
	if eh, ok := p.tvc.(ErrorHandler); ok {
		dlg.Println("calling error message handler")
		eh.OnError(WithController(context.Background(), p), c, err)
	} else {
		c.Send(pr.Sprintf(MsgUnexpected))
	}
}

// // convertToMsg extracts the user message from the callback.
// // TODO: remove
// func convertToMsg(cb *tb.Callback) *tb.Message {
// 	msg := cb.Message
// 	msg.Sender = cb.Sender
// 	return msg
// }

// nextHandler runs the next handler, if it's available.
func (p *Picklist) nextHandler(c tb.Context) error {
	if p.next != nil {
		return p.next.Handler(c)
	}
	return nil
}

// handleBackButton sends the empty response to telegram to acknowledge button
// action and runs the previous handler if it's available. It will not return the
// error on the response.
func (p *Picklist) handleBackButton(ctx context.Context, c tb.Context) error {
	if err := c.Respond(&tb.CallbackResponse{}); err != nil {
		lg.Printf("%s: %s", caller(0), err)
		trace.Log(ctx, "respond", err.Error())
	}
	p.setBackPressed(c)
	if p.prev != nil {
		return p.prev.Handler(c)
	}
	return nil
}
