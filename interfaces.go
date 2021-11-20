package tbcomctl

import (
	"context"

	tb "gopkg.in/tucnak/telebot.v3"
)

// Controller is the interface that some common controls implement. Controllers
// can be chained together. Controller is everything that can interact with user,
// i.e. a message that waits for the user input, or a picklist that presents the
// user with a set of buttons. Caller may want to create a custom controller to
// include in the form.
type Controller interface {
	// Handler is the controller's message handler.
	Handler(c tb.Context) error
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
	// OutgoingID should return the value of the outgoing message ID for the
	// user and true if the message is present or false otherwise.
	OutgoingID(recipient string) (int, bool)
}

// HandleContextFunc is the callback function that is being called within
// controller callbacks. It should return the errors specific to controller
// requirements for the input to be processed.
type HandleContextFunc func(ctx context.Context, c tb.Context) error

// Texter is the interface that contains only the method to return the Message
// Text.
type Texter interface {
	// Text should return the message that is presented to the user and an error.
	Text(ctx context.Context, c tb.Context) (string, error)
}

// NewTexter wraps the msg returning a Texter.
func NewTexter(msg string) Texter {
	return NewStaticTVC(msg, nil, nil)
}

// Valuer is the interface
type Valuer interface {
	// Values should return a list of strings to present as choices to the user and an error.
	// TODO describe supported error return values.
	Values(ctx context.Context, c tb.Context) ([]string, error)
}

// Callbacker defines the interface for the callback function.
type Callbacker interface {
	// Callback should process the handler's callback.
	// TODO: describe supported error return values.
	Callback(ctx context.Context, c tb.Context) error
}

// ErrorHandler is an optional interface for some controls, that the caller might
// implement so that errors occurred while handling the Callback could be
// handled by the caller as well.
type ErrorHandler interface {
	// OnError should process the error.
	OnError(ctx context.Context, c tb.Context, err error)
}

// TextValuer combines Texter and Valuer interfaces.
type TextValuer interface {
	Texter
	Valuer
}

// TextCallbacker combines Texter and Callbacker interfaces.
type TextCallbacker interface {
	Texter
	Callbacker
}

// TextValueCallbacker combines Texter, Valuer and Callbacker interfaces.
type TextValueCallbacker interface {
	Texter
	Valuer
	Callbacker
}

// TVC is an implementation of TextValuerCallbacker.
type TVC struct {
	TextFn   func(context.Context, tb.Context) (string, error)
	ValuesFn func(context.Context, tb.Context) ([]string, error)
	CBfn     HandleContextFunc
}

// NewStaticTVC is a convenience constructor for TVC (TextValueCallbacker) with
// static text and values.
func NewStaticTVC(text string, values []string, callbackFn HandleContextFunc) *TVC {
	return &TVC{
		TextFn:   func(_ context.Context, _ tb.Context) (string, error) { return text, nil },
		ValuesFn: func(_ context.Context, _ tb.Context) ([]string, error) { return values, nil },
		CBfn:     callbackFn,
	}
}

// Text callse the TextFn with contexts.
// ctx is legacy from the times when there was not telebot.Context.
func (t TVC) Text(ctx context.Context, c tb.Context) (string, error) {
	return t.TextFn(ctx, c)
}

func (t TVC) Values(ctx context.Context, c tb.Context) ([]string, error) {
	return t.ValuesFn(ctx, c)
}

func (t TVC) Callback(ctx context.Context, c tb.Context) error {
	return t.CBfn(ctx, c)
}
