package tbcomctl

import "errors"

// ErrType is the type of error returned by the callback functions.
type ErrType int

const (
	TErrNoChange ErrType = iota // there has been no change to the selection
	TErrRetry                   // error of this type will ask user to retry the selection or input
	TInputError                 // tell user that there was an input error (user-error)
)

// Error is the type of error returned by the input-processing callback functions
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
	// BackPressed is a special type of error indicating that callback handler should call the previous handler.
	BackPressed = errors.New("back") //lint:ignore ST1012 it is what it is
)
