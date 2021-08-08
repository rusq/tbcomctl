// Package tbcomctl provides common controls for telegram bots.
//
package tbcomctl

import (
	"context"
	"crypto/sha1"
	"fmt"
	"strconv"
	"sync"
	"time"

	"golang.org/x/text/language"

	"github.com/google/uuid"

	tb "gopkg.in/tucnak/telebot.v2"
)

const (
	FallbackLang = "en-US"
)
const (
	unknown = "<unknown>"
)

// Boter is the interface to send messages.
type Boter interface {
	Handle(endpoint interface{}, handler interface{})
	Send(to tb.Recipient, what interface{}, options ...interface{}) (*tb.Message, error)
	Edit(msg tb.Editable, what interface{}, options ...interface{}) (*tb.Message, error)
	Respond(c *tb.Callback, resp ...*tb.CallbackResponse) error
}

type BotChecker interface {
	Boter
	ChatMemberOf(chat *tb.Chat, user *tb.User) (*tb.ChatMember, error)
	ChatByID(id string) (*tb.Chat, error)
}

type BotNotifier interface {
	Boter
	Notify(to tb.Recipient, action tb.ChatAction) error
}

// Controller is the interface that some of the common controls implement.  Controllers can
// be chained together
type Controller interface {
	// Handler is the controller's message handler.
	Handler(m *tb.Message)
	// Name returns the name of the control assigned to it on creation.  When
	// Controller is a part of a form, one can call Form.Controller(name) method
	// to get the controller.
	Name() string
	// SetNext sets the next handler, when control is part of a form.
	SetNext(Controller)
	// SetPrev sets the previous handler.
	SetPrev(Controller)
	// SetForm assigns the form to the controller, this will allow controller to
	// address other controls in a form by name.
	SetForm(*Form)
	// Form returns the form associated with the controller.
	Form() *Form
	// Value returns the value stored in the controller for the recipient.
	Value(recipient string) (string, bool)
	// Bot returns the bot that is used to inialise the controller.
	Bot() Boter
	// OutgoingID should return the value of the outgoing message ID for the
	// user and true if the message is present or false otherwise.
	OutgoingID(recipient string) (int, bool)
}

type overwriter interface {
	setOverwrite(b bool)
}

type commonCtl struct {
	b Boter

	name string // name of the control, must be unique if used within chained controls
	prev Controller
	next Controller
	form *Form // if not nil, controller is part of the form.

	textFn TextFunc
	errFn  ErrFunc

	privateOnly bool
	overwrite   bool // overwrite the previous message sent by control.

	reqCache map[string]map[int]uuid.UUID // requests cache, maps message ID to request.
	await    map[string]int               // await maps userID to the messageID and indicates that we're waiting for user to reply.
	values   map[string]string            // values entered, maps userID to the value
	messages map[string]int               // messages sent, maps userID to the message_id
	mu       sync.RWMutex

	lang string
}

// PrivateOnly is the middleware that restricts the handler to only private
// messages.
func PrivateOnly(fn func(m *tb.Message)) func(*tb.Message) {
	return PrivateOnlyMsg(nil, "", fn)
}

func PrivateOnlyMsg(b Boter, msg string, fn func(m *tb.Message)) func(*tb.Message) {
	return func(m *tb.Message) {
		if !m.Private() {
			if !(b == nil || msg == "") {
				pr := Printer(m.Sender.LanguageCode)
				b.Send(m.Chat, pr.Sprintf(msg))
			}
			return
		}
		fn(m)
	}
}

type controllerKey int

var ctrlKey controllerKey

func WithController(ctx context.Context, ctrl Controller) context.Context {
	return context.WithValue(ctx, ctrlKey, ctrl)
}

func ControllerFromCtx(ctx context.Context) (Controller, bool) {
	ctrl, ok := ctx.Value(ctrlKey).(Controller)
	return ctrl, ok
}

type StoredMessage struct {
	MessageID string
	ChatID    int64
}

func (m StoredMessage) MessageSig() (string, int64) {
	return m.MessageID, m.ChatID
}

// TextFunc returns values for inline buttons, possibly personalised for user u.
type ValuesFunc func(ctx context.Context, u *tb.User) ([]string, error)

// TextFunc returns formatted text, possibly personalised for user u.
type TextFunc func(ctx context.Context, u *tb.User) (string, error)

type MiddlewareFunc func(func(m *tb.Message)) func(m *tb.Message)

type ErrFunc func(ctx context.Context, m *tb.Message, err error)

// BtnCallbackFunc is being called once the user picks the value, it should return error if the value is incorrect, or
// ErrRetry if the retry should be performed.
type BtnCallbackFunc func(ctx context.Context, cb *tb.Callback) error

type ErrType int

const (
	TErrNoChange ErrType = iota
	TErrRetry
	TInputError
)

type Error struct {
	Alert bool
	Msg   string
	Type  ErrType
}

func (e *Error) Error() string { return e.Msg }

var (
	// ErrRetry should be returned by CallbackFunc if the retry should be performed.
	ErrRetry = &Error{Type: TErrRetry, Msg: "retry", Alert: true}
	// ErrNoChange should be returned if the user picked the same value as before, and no update needed.
	ErrNoChange = &Error{Type: TErrNoChange, Msg: "no change"}
)

var hasher = sha1.New

func hash(s string) string {
	h := hasher()
	h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum(nil))
}

type option func(ctl *commonCtl)

func optPrivateOnly(b bool) option {
	return func(ctl *commonCtl) {
		ctl.privateOnly = b
	}
}

func optErrFunc(fn ErrFunc) option {
	return func(ctl *commonCtl) {
		ctl.errFn = fn
	}
}

func optFallbackLang(lang string) option {
	return func(ctl *commonCtl) {
		_ = language.MustParse(lang) // will panic if wrong.
		ctl.lang = lang
	}
}

// newCommonCtl creates a new commonctl instance.
func newCommonCtl(b Boter, name string, textFn TextFunc) commonCtl {
	return commonCtl{
		b:        b,
		name:     name,
		textFn:   textFn,
		messages: make(map[string]int),
	}
}

// register registers message in cache assigning it a request id.
func (c *commonCtl) register(r tb.Recipient, msgID int) uuid.UUID {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.requestEnsure(r)

	reqID := uuid.Must(uuid.NewUUID())
	c.reqCache[r.Recipient()][msgID] = reqID
	c.messages[r.Recipient()] = msgID
	return reqID
}

// requestEnsure initialises that request cache is initialised.
func (c *commonCtl) requestEnsure(r tb.Recipient) {
	if c.reqCache == nil {
		c.reqCache = make(map[string]map[int]uuid.UUID)
	}
	if c.reqCache[r.Recipient()] == nil {
		c.reqCache[r.Recipient()] = make(map[int]uuid.UUID)
	}
}

// requestFor returns a request id for message ID and a bool. Bool will be true if
// message is registered and false otherwise.
func (c *commonCtl) requestFor(r tb.Recipient, msgID int) (uuid.UUID, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	c.requestEnsure(r)
	reqID, ok := c.reqCache[r.Recipient()][msgID]
	return reqID, ok
}

func (c *commonCtl) unregister(r tb.Recipient, msgID int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.reqCache[r.Recipient()], msgID)
}

// reqIDInfo returns a request ID (or <unknown) and a time of the request (or zero time).
func (c *commonCtl) reqIDInfo(r tb.Recipient, msgID int) (string, time.Time) {
	reqID, ok := c.requestFor(r, msgID)
	if !ok {
		return unknown, time.Time{}
	}
	return reqID.String(), time.Unix(reqID.Time().UnixTime())
}

// multibuttonMarkup returns a markup containing a bunch of buttons.  If
// showCounter is true, will show a counter beside each of the labels. each
// telegram button will have a button index pressed by the user in the
// callback.Data. Prefix is the prefix that will be prepended to the unique
// before hash is called to form the Control-specific unique fields.
func (c *commonCtl) multibuttonMarkup(btns []Button, showCounter bool, prefix string, cbFn func(*tb.Callback)) *tb.ReplyMarkup {
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
		c.b.Handle(&bn, cbFn)
	}

	markup.Inline(organizeButtons(buttons, defNumButtons)...)

	return markup
}

// SetNext sets next controller in the chain.
func (c *commonCtl) SetNext(ctrl Controller) {
	if ctrl != nil {
		c.next = ctrl
	}
}

// SetPrev sets the previous controller in the chain.
func (c *commonCtl) SetPrev(ctrl Controller) {
	if ctrl != nil {
		c.prev = ctrl
	}
}

func NewControllerChain(first Controller, cc ...Controller) func(m *tb.Message) {
	var chain Controller
	for i := len(cc) - 1; i >= 0; i-- {
		cc[i].SetNext(chain)
		chain = cc[i]
	}
	first.SetNext(chain)
	return first.Handler
}

// Value returns the Controller value for the recipient.
func (c *commonCtl) Value(recipient string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.values == nil {
		c.values = make(map[string]string)
	}
	v, ok := c.values[recipient]
	return v, ok
}

// SetValue sets the Controller value.
func (c *commonCtl) SetValue(recipient string, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.values == nil {
		c.values = make(map[string]string)
	}
	c.values[recipient] = value
}

//
// waiting function
//

// waitFor places the outbound message ID to the waiting list.  Message ID in
// outbound waiting list means that we expect the user to respond.
func (c *commonCtl) waitFor(r tb.Recipient, outboundID int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.await == nil {
		c.await = make(map[string]int)
	}
	c.await[r.Recipient()] = outboundID
}

func (c *commonCtl) stopWaiting(r tb.Recipient) int {
	outboundID := c.await[r.Recipient()]
	c.await[r.Recipient()] = nothing
	return outboundID
}

func (c *commonCtl) outboundID(r tb.Recipient) int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.await[r.Recipient()]
}

func (c *commonCtl) isWaiting(r tb.Recipient) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.await[r.Recipient()] != nothing
}

func (c *commonCtl) Name() string {
	return c.name
}

func (c *commonCtl) SetForm(fm *Form) {
	c.form = fm
}

func (c *commonCtl) Form() *Form {
	return c.form
}

// OutgoingID returns the controller's outgoing message ID for the user.
func (c *commonCtl) OutgoingID(recipient string) (int, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	id, ok := c.messages[recipient]
	return id, ok
}

// TextFn wraps the message in a TextFunc.
func TextFn(msg string) TextFunc {
	return func(ctx context.Context, u *tb.User) (string, error) {
		return msg, nil
	}
}

func (c *commonCtl) setOverwrite(b bool) {
	c.overwrite = b
}

func (c *commonCtl) sendOrEdit(userMsg *tb.Message, txt string, sendOpts ...interface{}) (*tb.Message, error) {
	var outbound *tb.Message
	var err error
	if c.overwrite && c.prev != nil {
		msgID, ok := c.prev.OutgoingID(userMsg.Sender.Recipient())
		if !ok {
			return nil, fmt.Errorf("%s can't find previous message ID for %s", caller(2), Userinfo(userMsg.Sender))

		}
		prevMsg := tb.Message{ID: msgID, Chat: userMsg.Chat}
		outbound, err = c.b.Edit(&prevMsg,
			txt,
			sendOpts...,
		)
	} else {
		outbound, err = c.b.Send(userMsg.Chat, txt, sendOpts...)
	}
	return outbound, err
}

func (c *commonCtl) Bot() Boter {
	return c.b
}
