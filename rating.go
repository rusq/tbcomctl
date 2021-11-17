package tbcomctl

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	tb "gopkg.in/tucnak/telebot.v3"
)

// Rating is a struct for attaching post rating.
type Rating struct {
	commonCtl

	hasRating  bool // show post rating between up/down vote buttons
	hasCounter bool // show counter of total upvotes-downvotes.

	rateFn RatingFunc //
}

// RatingFunc is the function called by callback, given the message, user
// and the button index it should update the records and return the new buttons
// with updated values for the posting, it must maintain count of votes inhouse.
type RatingFunc func(tb.Editable, *tb.User, int) ([2]Button, error)

type RBOption func(*Rating)

// RBOptShowVoteCounter enables post rating between up/down vote buttons
func RBOptShowVoteCounter(b bool) RBOption {
	return func(rb *Rating) {
		rb.hasCounter = b
	}
}

// RBOptShowPostRating enables counter of total upvotes/downvotes.
func RBOptShowPostRating(b bool) RBOption {
	return func(rb *Rating) {
		rb.hasRating = b
	}
}

type RatingType int

func NewRating(fn RatingFunc, opts ...RBOption) *Rating {
	rb := &Rating{
		commonCtl: newCommonCtl("rating"),
		rateFn:    fn,
	}
	for _, opt := range opts {
		opt(rb)
	}
	return rb
}

func (rb *Rating) Markup(b *tb.Bot, btns [2]Button) *tb.ReplyMarkup {
	const rbPrefix = "rating"
	return rb.multibuttonMarkup(b, btns[:], rb.hasCounter, rbPrefix, rb.callback)
}

var ErrAlreadyVoted = errors.New("already voted")

func (rb *Rating) callback(c tb.Context) error {
	respErr := tb.CallbackResponse{Text: MsgUnexpected}
	data := c.Data()

	btnIdx, err := strconv.Atoi(data)
	if err != nil {
		lg.Printf("failed to get the button index from data: %s", data)
		c.Respond(&respErr)
		return err
	}

	// get existing value for the post
	buttons, valErr := rb.rateFn(c.Message(), c.Sender(), btnIdx)
	if valErr != nil && valErr != ErrAlreadyVoted {
		lg.Printf("failed to get the data from the rating callback: %s", valErr)
		dlg.Printf("callback: %s", Sdump(c.Callback()))
		c.Respond(&respErr)
		return valErr
	}

	var msg string
	// update the post with new buttons
	if valErr != ErrAlreadyVoted {
		if err := c.Edit(rb.Markup(c.Bot(), buttons)); err != nil {
			if e, ok := err.(*tb.APIError); ok && e.Code == http.StatusBadRequest && strings.Contains(e.Description, "exactly the same") {
				// same button pressed - not an error.
				lg.Printf("%s: same button pressed", Userinfo(c.Sender()))
			} else {
				lg.Printf("failed to edit the message: %v: %s", c.Message(), err)
				c.Respond(&respErr)
				return err
			}
		}
		msg = PrinterContext(c, rb.fallbackLang).Sprint(MsgVoteCounted)
	}

	return c.Respond(&tb.CallbackResponse{Text: msg})
}
