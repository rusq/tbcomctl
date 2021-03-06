package tbcomctl

import (
	"time"

	tb "gopkg.in/tucnak/telebot.v2"
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

	await map[string]bool

	noReply bool
}

var _ Controller = &Input{}

// MsgErrFunc is the function that processes the user input.  If the input is
// invalid, it should return InputError with the message, then the user is
// offered to retry the input.
type MsgErrFunc func(m *tb.Message) error

type InputOption func(*Input)

func IOptNoReply(b bool) InputOption {
	return func(ip *Input) {
		ip.noReply = true
	}
}

// NewInput text creates a new text input, optionally chaining with the `next`
// handler. One must use Handle as a handler for bot endpoint, and then hook the
// OnText to OnTextMw.  msgFn is the function that should produce the text that
// user initially sees, onTextFn is the function that should process the user
// input.  It should return an error if the user input is not accepted, and then
// user is offered to retry.  It can format the return error with fmt.Errorf, as
// this is what user will see.  next is allowed to be nil.
func NewInput(b Boter, msgFn TextFunc, onTextFn MsgErrFunc) *Input {
	return &Input{
		commonCtl: commonCtl{
			b:           b,
			textFn:      msgFn,
			privateOnly: false,
		},
		OnTextFn: onTextFn,
		await:    make(map[string]bool),
	}
}

func NewInputText(b Boter, txt string, onTextFn MsgErrFunc) *Input {
	return NewInput(b, func(u *tb.User) string { return txt }, onTextFn)
}

func (ip *Input) Handler(m *tb.Message) {
	var opts []interface{}
	if !ip.noReply {
		opts = append(opts, tb.ForceReply)
	}
	if _, err := ip.b.Send(m.Sender, ip.textFn(m.Sender), opts...); err != nil {
		lg.Println("Input.Handle:", err)
		return
	}
	ip.await[m.Sender.Recipient()] = true
}

type InputError struct {
	Message string
}

func (e *InputError) Error() string {
	return "input error: " + e.Message
}

func (ip *Input) OnTextMw(fn func(m *tb.Message)) func(*tb.Message) {
	return func(m *tb.Message) {
		if !ip.await[m.Sender.Recipient()] {
			// not waiting for input, proceed to the next handler, if it's present.
			if fn != nil {
				fn(m)
			}
			return
		}

		valueErr := ip.OnTextFn(m)
		if valueErr != nil {
			// wrong input or some other problem
			lg.Println(valueErr)
			if e, ok := valueErr.(*InputError); ok {
				ip.processError(m, e.Message)
				return
			} else {
				if _, err := ip.b.Send(m.Sender, MsgUnexpected); err != nil {
					lg.Println(err)
					return
				}
			}
		}

		ip.await[m.Sender.Recipient()] = false

		if ip.next != nil && valueErr == nil {
			// if there are chained controls
			ip.next(m)
		} else {
			// run the initial handler
			fn(m)
		}
	}
}

func (ip *Input) processError(m *tb.Message, errmsg string) {
	if _, err := ip.b.Send(m.Sender, errmsg); err != nil {
		return
	}
	if err := ip.b.Notify(m.Sender, tb.Typing); err != nil {
		lg.Println(err)
		return
	}
	time.Sleep(retryDelay)
	ip.Handler(m)
}
