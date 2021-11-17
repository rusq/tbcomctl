// Package tbcomctl provides common controls for telegram bots.
//
package tbcomctl

import (
	"context"
	"crypto/sha1"
	"fmt"
	"strconv"

	"golang.org/x/text/language"
	tb "gopkg.in/tucnak/telebot.v3"

	"github.com/rusq/tbcomctl/v3/internal/registry"
)

const (
	// FallbackLang is the default fallback language.
	FallbackLang = "en-US"
)

type overwriter interface {
	setOverwrite(b bool)
}

type commonCtl struct {
	name string // name of the control, must be unique if used within chained controls.

	prev Controller // if nil - this is the first controller in the chain
	next Controller // if nil - this is the last controller in the chain.
	form *Form      // if not nil, controller is part of the form.

	hError ErrorHandler // custom error handler.

	privateOnly bool // should handle only private messages
	overwrite   bool // overwrite the previous message sent by control.

	fallbackLang string          // fallback language for i18n
	sendOpts     *tb.SendOptions // default send options.

	reg *registry.Memory
}

// PrivateOnly is the middleware that restricts the handler to only private
// messages.
func PrivateOnly(fn tb.HandlerFunc) tb.HandlerFunc {
	return PrivateOnlyMsg("", fn)
}

// PrivateOnlyMsg returns the handler that will reject non-private messages (eg.
// sent in groups) with i18n formatted message.
func PrivateOnlyMsg(msg string, fn tb.HandlerFunc) tb.HandlerFunc {
	return func(c tb.Context) error {
		if !c.Message().Private() {
			if msg != "" {
				pr := Printer(c.Sender().LanguageCode)
				return c.Send(pr.Sprintf(msg))
			}
			return nil
		}
		return fn(c)
	}
}

type controllerKey int    // controller key type for context
var ctrlKey controllerKey // controller key for context.

// WithController adds the controller to the context.
func WithController(ctx context.Context, ctrl Controller) context.Context {
	return context.WithValue(ctx, ctrlKey, ctrl)
}

// ControllerFromCtx returns the controller from the context.
func ControllerFromCtx(ctx context.Context) (Controller, bool) {
	ctrl, ok := ctx.Value(ctrlKey).(Controller)
	return ctrl, ok
}

// StoredMessage represents the stored message in the database.
type StoredMessage struct {
	MessageID string
	ChatID    int64
}

func (m StoredMessage) MessageSig() (string, int64) {
	return m.MessageID, m.ChatID
}

var hasher = sha1.New

// hash returns the hash of the s, using the hasher function.
func hash(s string) string {
	h := hasher()
	h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// option is the function signature for options that are common to all the
// controls. Concrete control implementations should use these options, if they
// must implement this functionality.
type option func(ctl *commonCtl)

// optPrivateOnly sets the control to be operatable only in the private mode.
func optPrivateOnly(b bool) option {
	return func(ctl *commonCtl) {
		ctl.privateOnly = b
	}
}

// optErrFunc sets the error handler.
func optErrFunc(h ErrorHandler) option {
	return func(ctl *commonCtl) {
		ctl.hError = h
	}
}

// optFallbackLang sets the default fallback language for the control.
func optFallbackLang(lang string) option {
	return func(ctl *commonCtl) {
		_ = language.MustParse(lang) // will panic if wrong.
		ctl.fallbackLang = lang
	}
}

// optDefaultSendOpts allows to set the default send options.  If this option is
// not in the option list, the built-in defaults are used.
func optDefaultSendOpts(opts *tb.SendOptions) option {
	return func(ctl *commonCtl) {
		ctl.sendOpts = opts
	}
}

// newCommonCtl creates a new commonCtl instance.  It gives most of the
// functions that satisfy Controller interface for free.
func newCommonCtl(name string) commonCtl {
	return commonCtl{
		name:     name,
		reg:      registry.NewMemRegistry(),
		sendOpts: &tb.SendOptions{ParseMode: tb.ModeHTML},
	}
}

// multibuttonMarkup returns a markup containing a bunch of buttons.  If
// showCounter is true, will show a counter beside each of the labels. each
// telegram button will have a button index pressed by the user in the
// callback.Data. Prefix is the prefix that will be prepended to the unique
// before hash is called to form the Control-specific unique fields.
func (cc *commonCtl) multibuttonMarkup(b *tb.Bot, btns []Button, showCounter bool, prefix string, cbFn func(tb.Context) error) *tb.ReplyMarkup {
	const (
		sep = ": "
	)
	if cbFn == nil {
		panic("internal error: callback function is empty")
	}
	markup := new(tb.ReplyMarkup)

	var buttons []tb.Btn
	for i, ri := range btns {
		bn := markup.Data(ri.label(showCounter, sep), hash(prefix+ri.Name), strconv.Itoa(i))
		buttons = append(buttons, bn)
		b.Handle(&bn, cbFn)
	}

	markup.Inline(organizeButtons(buttons, defNumButtons)...)

	return markup
}

// SetNext sets next controller in the chain.
func (cc *commonCtl) SetNext(ctrl Controller) {
	if ctrl != nil {
		cc.next = ctrl
	}
}

// SetPrev sets the previous controller in the chain.
func (cc *commonCtl) SetPrev(ctrl Controller) {
	if ctrl != nil {
		cc.prev = ctrl
	}
}

// NewControllerChain returns the controller chain.
//
// Deprecated: use NewForm instead.  NewControllerChain will be removed in the next versions.
func NewControllerChain(first Controller, cc ...Controller) tb.HandlerFunc {
	var chain Controller
	for i := len(cc) - 1; i >= 0; i-- {
		cc[i].SetNext(chain)
		chain = cc[i]
	}
	first.SetNext(chain)
	return first.Handler
}

// Name returns the controller name.
func (cc *commonCtl) Name() string {
	return cc.name
}

// SetForm links the controller to the form.
func (cc *commonCtl) SetForm(fm *Form) {
	cc.form = fm
}

// Form returns the form.
func (cc *commonCtl) Form() *Form {
	return cc.form
}

// setOverwite sets overwrite flag to b.
func (cc *commonCtl) setOverwrite(b bool) {
	cc.overwrite = b
}

// sendOrEdit sends the message or edits the previous one if the overwrite flag is true.  It returns the outbound
// message and an error.
func (cc *commonCtl) sendOrEdit(c tb.Context, txt string, sendOpts ...interface{}) (*tb.Message, error) {
	var outbound *tb.Message
	var err error
	msgID, ok := cc.getPreviousMsgID(c)
	if cc.overwrite && ok {
		prevMsg := tb.Message{ID: msgID, Chat: c.Chat()}
		outbound, err = c.Bot().Edit(&prevMsg,
			txt,
			sendOpts...,
		)
	} else {
		outbound, err = c.Bot().Send(c.Chat(), txt, sendOpts...)
	}
	return outbound, err
}

// getPreviousMsgID returns the ID of the previous outbound message.
func (cc *commonCtl) getPreviousMsgID(ct tb.Context) (int, bool) {
	if cc.isBackPressed(ct) {
		cc.resetBackPressed(ct)
		if cc.next == nil {
			// internal error
			return 0, false
		}
		return cc.next.OutgoingID(ct.Sender().Recipient())
	}
	// back not pressed
	if cc.prev == nil {
		return 0, false
	}
	return cc.prev.OutgoingID(ct.Sender().Recipient())
}

// isBackPressed returns true if the "back" button was pressed.
func (cc *commonCtl) isBackPressed(ct tb.Context) bool {
	backPressed, ok := ct.Get(BackPressed.Error()).(bool)
	return ok && backPressed
}

func (cc *commonCtl) setBackPressed(ct tb.Context) {
	ct.Set(BackPressed.Error(), true)
}

func (cc *commonCtl) resetBackPressed(ct tb.Context) {
	ct.Set(BackPressed.Error(), false) // reset the context value
}

func unexpectedErrorText(c tb.Context, fallbackLang ...string) string {
	pr := PrinterContext(c, fallbackLang...)
	return pr.Sprintf(MsgUnexpected)
}

// OutgoingID returns the controller's outgoing message ID for the user.
func (cc *commonCtl) OutgoingID(recipient string) (int, bool) {
	return cc.reg.OutgoingID(recipient)
}

// Value returns the Controller value for the recipient.
func (cc *commonCtl) Value(recipient string) (string, bool) {
	return cc.reg.Value(recipient)
}

// SetValue sets the Controller value.
func (cc *commonCtl) SetValue(recipient string, value string) {
	cc.reg.SetValue(recipient, value)
}
