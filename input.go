package tbcomctl

import (
	"context"
	"fmt"
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

// interface assertions
var (
	_ Controller = &Input{}
	_ onTexter   = &Input{}
)

// MsgErrFunc is the function that processes the user input.  If the input is
// invalid, it should return InputError with the message, then the user is
// offered to retry the input.
type MsgErrFunc func(ctx context.Context, c tb.Context) error

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
func NewInput(name string, textFn TextFunc, onTextFn MsgErrFunc, opts ...InputOption) *Input {
	ip := &Input{
		commonCtl: newCommonCtl(name, textFn),
		OnTextFn:  onTextFn,
	}
	for _, opt := range opts {
		opt(ip)
	}
	return ip
}

func NewInputText(name string, text string, onTextFn MsgErrFunc, opts ...InputOption) *Input {
	return NewInput(name, TextFn(text), onTextFn, opts...)
}

func (ip *Input) Handler(c tb.Context) error {
	var opts []interface{}
	if !ip.noReply {
		opts = append(opts, tb.ForceReply)
	}
	pr := Printer(c.Sender().LanguageCode)
	text, err := ip.textFn(WithController(context.Background(), ip), c.Sender())
	if err != nil {
		c.Send(pr.Sprintf(MsgUnexpected))
		return fmt.Errorf("error while generating text for controller: %s: %w", ip.name, err)
	}
	outbound, err := c.Bot().Send(c.Sender(), text, opts...)
	if err != nil {
		return fmt.Errorf("Input.Handle: %w", err)
	}
	ip.waitFor(c.Sender(), outbound.ID)
	ip.register(c.Sender(), outbound.ID)
	ip.logOutgoingMsg(outbound)
	return nil
}

// NewInputError returns an input error with msg.
func NewInputError(msg string) error {
	return &Error{Msg: msg, Type: TInputError}
}

const nothing = 0

// OnTextMw returns the middleware that should wrap the OnText handler. It will
// process the message only if control awaits for this particular user input.
func (ip *Input) OnTextMw(fn tb.HandlerFunc) tb.HandlerFunc {
	return tb.HandlerFunc(func(c tb.Context) error {
		if !ip.isWaiting(c.Sender()) {
			// not waiting for input, proceed to the next handler, if it's present.
			if fn != nil {
				return fn(c)
			}
			return nil
		}

		valueErr := ip.OnTextFn(WithController(context.Background(), ip), c)
		if valueErr != nil {
			// wrong input or some other problem
			lg.Println(valueErr)
			if e, ok := valueErr.(*Error); ok {
				return ip.processError(c, e.Msg)
			} else {
				if err := c.Send(MsgUnexpected); err != nil {
					return err
				}
			}
		}

		ip.SetValue(c.Sender().Recipient(), c.Message().Text)

		ip.logCallbackMsg(c.Message())
		ip.unregister(c.Sender(), ip.stopWaiting(c.Sender()))

		if ip.next != nil && valueErr == nil {
			// if there are chained controls
			return ip.next.Handler(c)
		}
		return nil
	})
}

func (ip *Input) processError(c tb.Context, errmsg string) error {
	if err := c.Send(errmsg); err != nil {
		return err
	}
	c.Bot().Notify(c.Sender(), tb.Typing)
	time.Sleep(retryDelay)
	return ip.Handler(c)
}
