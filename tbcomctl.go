// Package tbcomctl provides common controls for telegram bots.
//
package tbcomctl

import (
	"crypto/sha1"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/text/language"

	"github.com/rusq/dlog"

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

// Standard message and Callback handlers signatures.
type (
	MsgHandler      func(m *tb.Message)
	CallbackHandler func(cb *tb.Callback)
)

type StoredMessage struct {
	MessageID string
	ChatID    int64
}

func (m StoredMessage) MessageSig() (string, int64) {
	return m.MessageID, m.ChatID
}

// TextFunc returns values for inline buttons, possibly personalised for user u.
type ValuesFunc func(u *tb.User) ([]string, error)

// TextFunc returns formatted text, possibly personalised for user u.
type TextFunc func(u *tb.User) string

type MiddlewareFunc func(func(m *tb.Message)) func(m *tb.Message)

type ErrFunc func(m *tb.Message, err error)

// BtnCallbackFunc is being called once the user picks the value, it should return error if the value is incorrect, or
// ErrRetry if the retry should be performed.
type BtnCallbackFunc func(cb *tb.Callback) error

var (
	// ErrRetry should be returned by CallbackFunc if the retry should be performed.
	ErrRetry = errors.New("retry")
	// ErrNoChange should be returned if the user picked the same value as before, and no update needed.
	ErrNoChange = errors.New("no change")
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

type commonCtl struct {
	b Boter

	textFn TextFunc
	next   MsgHandler

	privateOnly bool
	errFn       ErrFunc

	reqCache map[int]uuid.UUID // requests cache, maps message ID to request.
	mu       sync.RWMutex

	lang string
}

// PrivateOnly is the middleware that restricts the handler to only private
// messages.
func PrivateOnly(fn func(m *tb.Message)) func(*tb.Message) {
	return func(m *tb.Message) {
		if !m.Private() {
			return
		}
		fn(m)
	}
}

// reqRegister registers message in cache assigning it a request id.
func (c *commonCtl) reqRegister(msgID int) uuid.UUID {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.reqCache == nil {
		c.reqCache = make(map[int]uuid.UUID)
	}
	reqID := uuid.Must(uuid.NewUUID())
	c.reqCache[msgID] = reqID
	return reqID
}

// requestFor returns a request id for message ID and a bool. Bool will be true if
// message is registered and false otherwise.
func (c *commonCtl) requestFor(msgID int) (uuid.UUID, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.reqCache == nil {
		return uuid.UUID{}, false
	}
	reqID, ok := c.reqCache[msgID]
	return reqID, ok
}

func (c *commonCtl) reqUnregister(msgID int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.reqCache, msgID)
}

// organizeButtons organizes buttons in rows.
func organizeButtons(markup *tb.ReplyMarkup, btns []tb.Btn, btnInRow int) []tb.Row {
	var rows []tb.Row
	var buttons []tb.Btn
	for i, btn := range btns {
		if i%btnInRow == 0 {
			if len(buttons) > 0 {
				rows = append(rows, markup.Row(buttons...))
			}
			buttons = make([]tb.Btn, 0, btnInRow)
		}
		buttons = append(buttons, btn)
	}
	if 0 < len(buttons) && len(buttons) < btnInRow {
		rows = append(rows, buttons)
	}
	return rows
}

// logCallback logs callback data.
func (c *commonCtl) logCallback(cb *tb.Callback) {
	dlog.Debugf("%s: callback dump: %s", Userinfo(cb.Sender), Sdump(cb))

	reqID, at := c.reqIDInfo(cb.Message.ID)
	dlog.Printf("%s> %s: msg sent at %s, user response in: %s, callback data: %q", reqID, Userinfo(cb.Sender), at, time.Since(at), cb.Data)
}

// logOutgoingMsg logs the outgoing message and any additional string info passed in s.
func (c *commonCtl) logOutgoingMsg(m *tb.Message, s ...string) {
	dlog.Debugf("%s: message dump: %s", Userinfo(m.Sender), Sdump(m))

	reqID, at := c.reqIDInfo(m.ID)
	dlog.Printf("%s> msg to chat: %s, req time: %s: %s", reqID, ChatInfo(m.Chat), at, strings.Join(s, " "))
}

// reqIDInfo returns a request ID (or <unknown) and a time of the request (or zero time).
func (c *commonCtl) reqIDInfo(msgID int) (string, time.Time) {
	reqID, ok := c.requestFor(msgID)
	if !ok {
		return unknown, time.Time{}
	}
	return reqID.String(), time.Unix(reqID.Time().UnixTime())
}

// multibuttonMarkup returns a markup containing a bunch of buttons.  If
// showCounter is true, will show a counter beside each of the labels. each
// telegram button will have a button index pressed by the user in the
// callback.Data.
func (c *commonCtl) multibuttonMarkup(btns []Button, showCounter bool, cbFn func(*tb.Callback)) *tb.ReplyMarkup {
	const (
		prefix = "mb"
		sep    = ": "
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

	markup.Inline(organizeButtons(markup, buttons, defNumButtons)...)

	return markup
}
