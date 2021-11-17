package tbcomctl

import (
	"errors"
	"fmt"

	tb "gopkg.in/tucnak/telebot.v3"
)

type buttons struct {
	maxButtons int
}

func (b *buttons) SetMaxButtons(n int) {
	if n <= 0 || maxButtons < n {
		n = defNumButtons
	}
	b.maxButtons = n
}

type PostButtons struct {
	*buttons
	cbFn tb.HandlerFunc
}

type PBOption func(*PostButtons)

func PBOptMaxButtons(n int) PBOption {
	return func(pb *PostButtons) {
		pb.buttons.SetMaxButtons(n)
	}
}

// NewPostButtons creates an instance of PostButtons.  The callbackFunction is
// the function that will be assigned and called for each button press, so it
// should handle all possible values.
func NewPostButtons(callbackFn func(c tb.Context) error, opts ...PBOption) *PostButtons {
	pb := &PostButtons{
		cbFn:    callbackFn,
		buttons: &buttons{maxButtons: defNumButtons},
	}
	for _, o := range opts {
		o(pb)
	}
	return pb
}

// Markup returns the markup with buttons labeled with labels.
func (pb *PostButtons) Markup(c tb.Context, labels []string, pattern ...uint) (*tb.ReplyMarkup, error) {
	if len(pattern) == 0 {
		return ButtonMarkup(c, labels, pb.maxButtons, pb.cbFn), nil
	}
	markup, err := ButtonPatternMarkup(c, labels, pattern, pb.cbFn)
	if err != nil {
		panic(err)
	}
	return markup, nil
}

// ButtonMarkup returns the button markup for the message.  It creates handlers
// for all the buttons assigning the cbFn callback function to each of them.
// Values must be unique.  maxRowButtons is maximum number of buttons in a row.
func ButtonMarkup(c tb.Context, values []string, maxRowButtons int, cbFn func(c tb.Context) error) *tb.ReplyMarkup {
	if maxRowButtons <= 0 || defNumButtons < maxRowButtons {
		maxRowButtons = defNumButtons
	}
	markup, btns := createButtons(c, values, cbFn)
	markup.Inline(organizeButtons(btns, maxRowButtons)...)
	return markup
}

func ButtonPatternMarkup(c tb.Context, values []string, pattern []uint, cbFn tb.HandlerFunc) (*tb.ReplyMarkup, error) {
	markup, btns := createButtons(c, values, cbFn)
	rows, err := organizeButtonsPattern(btns, pattern)
	if err != nil {
		return nil, err
	}
	markup.Inline(rows...)
	return markup, nil
}

func createButtons(c tb.Context, values []string, cbFn func(c tb.Context) error) (*tb.ReplyMarkup, []tb.Btn) {
	markup := new(tb.ReplyMarkup)
	var btns []tb.Btn
	for _, label := range values {
		btn := markup.Data(label, hash(label), label)
		btns = append(btns, btn)
		c.Bot().Handle(&btn, cbFn)
	}
	return markup, btns
}

// organizeButtons organizes buttons in rows, at most btnInRow per row.
func organizeButtons(btns []tb.Btn, btnInRow int) []tb.Row {
	var rows []tb.Row
	var buttons []tb.Btn
	for i, btn := range btns {
		if i%btnInRow == 0 {
			if len(buttons) > 0 {
				rows = append(rows, buttons)
			}
			buttons = make([]tb.Btn, 0, btnInRow)
		}
		buttons = append(buttons, btn)
	}
	if 0 < len(buttons) && len(buttons) <= btnInRow {
		rows = append(rows, buttons)
	}
	return rows
}

func organizeButtonsPattern(btns []tb.Btn, pattern []uint) ([]tb.Row, error) {
	if len(btns) == 0 {
		return nil, errors.New("no buttons to organize")
	}
	// check total number, must not exceed sum of buttons in pattern
	sum := 0
	for i, perRow := range pattern {
		if perRow < 1 {
			return nil, fmt.Errorf("patterns can't have < 1 buttons in a row (row index: %d)", i)
		}
		sum += int(perRow)
	}
	if sum < len(btns) {
		return nil, fmt.Errorf("can't fit %d buttons in this pattern: %v", len(btns), pattern)
	}

	var rows []tb.Row
	var offset uint = 0
	for _, perRow := range pattern {
		var row tb.Row
		if offset >= uint(len(btns)) {
			break
		}
		for i := offset; i-offset < perRow; i++ {
			row = append(row, btns[i])
		}
		rows = append(rows, row)
		offset += uint(len(row))
	}
	return rows, nil
}
