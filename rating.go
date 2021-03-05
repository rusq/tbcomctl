package tbcomctl

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	tb "gopkg.in/tucnak/telebot.v2"
)

// Rating is a customizable struct for attaching post rating.
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

func NewRating(b Boter, fn RatingFunc, opts ...RBOption) *Rating {
	rb := &Rating{
		commonCtl: commonCtl{b: b},
		rateFn:    fn,
	}
	for _, opt := range opts {
		opt(rb)
	}
	return rb
}

type Button struct {
	Name  string `json:"n"`
	Value int    `json:"v"`
}

// label outputs the label for the ratingInfo.  If counter is set, will output a
// decimal representation of value after a separator sep.
func (ri *Button) label(counter bool, sep string) string {
	if !counter {
		return ri.Name
	}
	return ri.Name + sep + strconv.FormatInt(int64(ri.Value), 10)
}

func (ri *Button) String() string {
	return fmt.Sprintf("<Button name: %s, value: %d>", ri.Name, ri.Value)
}

func (rb *Rating) Markup(btns [2]Button) *tb.ReplyMarkup {
	const rbPrefix = "rating"
	return rb.multibuttonMarkup(btns[:], rb.hasCounter, rbPrefix, rb.callback)
}

var ErrAlreadyVoted = errors.New("already voted")

func (rb *Rating) callback(cb *tb.Callback) {
	respErr := tb.CallbackResponse{Text: MsgUnexpected}
	i, err := strconv.Atoi(cb.Data)
	if err != nil {
		lg.Printf("failed to get the button index from data: %s", cb.Data)
		rb.b.Respond(cb, &respErr)
		return
	}

	// get existing value for the post
	buttons, valErr := rb.rateFn(cb.Message, cb.Sender, i)
	if valErr != nil && valErr != ErrAlreadyVoted {
		lg.Printf("failed to get the data from the rating callback: %s", valErr)
		dlg.Printf("callback: %s", Sdump(cb))
		rb.b.Respond(cb, &respErr)
		return
	}

	var msg string
	// update the post with new buttons
	if valErr != ErrAlreadyVoted {
		if _, err := rb.b.Edit(cb.Message, rb.Markup(buttons)); err != nil {
			if e, ok := err.(*tb.APIError); ok && e.Code == 400 && strings.Contains(e.Description, "exactly the same") {
				lg.Printf("%s: same button pressed", Userinfo(cb.Sender))
			} else {
				lg.Printf("failed to edit the message: %v: %s", cb.Message, err)
				rb.b.Respond(cb, &respErr)
				return
			}
		}
		msg = MsgVoteCounted
	}

	rb.b.Respond(cb, &tb.CallbackResponse{Text: msg})
}
