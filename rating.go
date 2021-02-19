package tbcomctl

import (
	"encoding/json"
	"errors"
	"strconv"
	"sync"

	"github.com/rusq/dlog"
	tb "gopkg.in/tucnak/telebot.v2"
)

// Rating is a customizable struct for attaching post rating.
type Rating struct {
	commonCtl

	hasRating   bool // show post rating between up/down vote buttons
	hasCounter  bool // show counter of total upvotes-downvotes.
	allowUnvote bool // allow user to revoke the vote, otherwise - do nothing.

	rateFn RatingFunc //

	btns []Button
	idx  map[string]int // button index by label
	mu   sync.Mutex
}

// RatingFunc is the function called by callback, given the message and
// button it should update the records and return the value and an error.
type RatingFunc func(tb.Editable, tb.Recipient, Button) (int, error)

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

// RBOptAllowUnvote allows user to revoke their vote.
func RBOptAllowUnvote(b bool) RBOption {
	return func(rb *Rating) {
		rb.allowUnvote = b
	}
}

var defRatingBtns = [2]Button{
	{"↑", 0},
	{"↓", 0},
}

func RBOptButtons(btns [2]Button) RBOption {
	return func(rb *Rating) {
		rb.btns = btns[:]
		// indexing buttons for fast updates
		for i, btn := range btns {
			rb.idx[btn.Name] = i
		}
	}
}

func NewRating(b Boter, fn RatingFunc, opts ...RBOption) *Rating {
	rb := &Rating{
		commonCtl: commonCtl{
			b: b,
		},
		rateFn: fn,
		idx:    make(map[string]int, len(defRatingBtns)),
	}
	RBOptButtons(defRatingBtns)(rb)
	for _, opt := range opts {
		opt(rb)
	}
	return rb
}

type Button struct {
	Name  string `json:"l"`
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
	data, err := json.Marshal(ri)
	if err != nil {
		panic(err)
	}
	return string(data)
}

func (rb *Rating) Markup() *tb.ReplyMarkup {
	const (
		prefix = "rb"
		sep    = ": "
	)
	markup := new(tb.ReplyMarkup)

	var buttons []tb.Btn
	for _, ri := range rb.btns {
		bn := markup.Data(ri.label(rb.hasCounter, sep), hash(prefix+ri.Name), strconv.Itoa(rb.idx[ri.Name]))
		buttons = append(buttons, bn)
		rb.b.Handle(&bn, rb.callback)
	}

	markup.Inline(organizeButtons(markup, buttons, defNumButtons)...)

	return markup
}

var ErrAlreadyVoted = errors.New("already voted")

func (rb *Rating) callback(cb *tb.Callback) {
	respErr := tb.CallbackResponse{Text: "something went wrong"}
	i, err := strconv.ParseInt(cb.Data, 10, 32)
	if err != nil {
		dlog.Printf("failed to get the button index from data: %s", cb.Data)
		rb.b.Respond(cb, &respErr)
	}

	rb.mu.Lock()
	defer rb.mu.Unlock()

	// get existing value for the post
	newVal, valErr := rb.rateFn(cb.Message, cb.Sender, rb.btns[i])
	if valErr != nil && valErr != ErrAlreadyVoted {
		dlog.Println("failed to get the data from the rating callback for msg %v: %s", cb.Message, err)
		rb.b.Respond(cb, &respErr)
		return
	}
	rb.btns[i].Value = newVal

	if _, err := rb.b.Edit(cb.Message, rb.Markup()); err != nil {
		dlog.Println("failed to edit the message: %v: %s", cb.Message, err)
		rb.b.Respond(cb, &respErr)
	}
	msg := "vote counted"
	if valErr == ErrAlreadyVoted && !rb.allowUnvote {
		msg = ""
	}
	rb.b.Respond(cb, &tb.CallbackResponse{Text: msg})
}
