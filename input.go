package tbcomctl

import (
	"context"
	"time"

	tb "gopkg.in/tucnak/telebot.v3"
)

const retryDelay = 500 * time.Millisecond

type Input struct {
	commonCtl

	// UniqName is the unique name of the field (used to create pipelines, not
	// shown to the user)
	UniqName string
	// OnTextFn is the message callback function called when user responds.  If
	// it returns the error, user will be informed about it.
	OnTextFn MsgErrFunc

	noReply bool
}

var _ Controller = &Input{}

// MsgErrFunc is the function that processes the user input.  If the input is
// invalid, it should return InputError with the message, then the user is
// offered to retry the input.
type MsgErrFunc func(ctx context.Context, m *tb.Message) error

type InputOption func(*Input)

func IOptNoReply(b bool) InputOption {
	return func(ip *Input) {
		ip.noReply = true
	}
}

func IOptPrivateOnly(b bool) InputOption {
	return func(ip *Input) {
		optPrivateOnly(b)(&ip.commonCtl)
	}
}

// NewInput text creates a new text input, optionally chaining with the `next`
// handler. One must use Handle as a handler for bot endpoint, and then hook the
// OnText to OnTextMw.  msgFn is the function that should produce the text that
// user initially sees, onTextFn is the function that should process the user
// input.  It should return an error if the user input is not accepted, and then
// user is offered to retry.  It can format the return error with fmt.Errorf, as
// this is what user will see.  next is allowed to be nil.
func NewInput(b BotNotifier, name string, textFn TextFunc, onTextFn MsgErrFunc, opts ...InputOption) *Input {
	ip := &Input{
		commonCtl: newCommonCtl(b, name, textFn),
		OnTextFn:  onTextFn,
	}
	for _, opt := range opts {
		opt(ip)
	}
	return ip
}

func NewInputText(b BotNotifier, name string, text string, onTextFn MsgErrFunc, opts ...InputOption) *Input {
	return NewInput(b, name, TextFn(text), onTextFn, opts...)
}

func (ip *Input) Handler(m *tb.Message) {
	var opts []interface{}
	if !ip.noReply {
		opts = append(opts, tb.ForceReply)
	}
	pr := Printer(m.Sender.LanguageCode)
	text, err := ip.textFn(WithController(context.Background(), ip), m.Sender)
	if err != nil {
		lg.Printf("error while generating text for controller: %s: %s", ip.name, err)
		ip.b.Send(m.Sender, pr.Sprintf(MsgUnexpected))
		return
	}
	outbound, err := ip.b.Send(m.Sender, text, opts...)
	if err != nil {
		lg.Println("Input.Handle:", err)
		return
	}
	ip.waitFor(m.Sender, outbound.ID)
	ip.register(m.Sender, outbound.ID)
	ip.logOutgoingMsg(outbound)
}

// NewInputError returns an input error with msg.
func NewInputError(msg string) error {
	return &Error{Msg: msg, Type: TInputError}
}

const nothing = 0

// OnTextMw returns the middleware that should wrap the OnText handler. It will
// process the message only if control awaits for this particular user input.
func (ip *Input) OnTextMw(fn func(m *tb.Message)) func(*tb.Message) {
	return func(m *tb.Message) {
		if !ip.isWaiting(m.Sender) {
			// not waiting for input, proceed to the next handler, if it's present.
			if fn != nil {
				fn(m)
			}
			return
		}

		valueErr := ip.OnTextFn(WithController(context.Background(), ip), m)
		if valueErr != nil {
			// wrong input or some other problem
			lg.Println(valueErr)
			if e, ok := valueErr.(*Error); ok {
				ip.processError(m, e.Msg)
				return
			} else {
				if _, err := ip.b.Send(m.Sender, MsgUnexpected); err != nil {
					lg.Println(err)
					return
				}
			}
		}

		ip.SetValue(m.Sender.Recipient(), m.Text)

		ip.logCallbackMsg(m)
		ip.unregister(m.Sender, ip.stopWaiting(m.Sender))

		if ip.next != nil && valueErr == nil {
			// if there are chained controls
			ip.next.Handler(m)
		}
	}
}

func (ip *Input) processError(m *tb.Message, errmsg string) {
	if _, err := ip.b.Send(m.Sender, errmsg); err != nil {
		return
	}
	if b, ok := ip.b.(BotNotifier); ok {
		b.Notify(m.Sender, tb.Typing)
	}
	time.Sleep(retryDelay)
	ip.Handler(m)
}
