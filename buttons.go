package tbcomctl

import (
	"errors"
	"fmt"

	tb "gopkg.in/tucnak/telebot.v2"
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
	b    Boter
	cbFn func(cb *tb.Callback)
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
func NewPostButtons(b Boter, callbackFn func(cb *tb.Callback), opts ...PBOption) *PostButtons {
	pb := &PostButtons{
		b:       b,
		cbFn:    callbackFn,
		buttons: &buttons{maxButtons: defNumButtons},
	}
	for _, o := range opts {
		o(pb)
	}
	return pb
}

// Markup returns the markup with buttons labeled with labels.
func (pb *PostButtons) Markup(labels []string) *tb.ReplyMarkup {
	return ButtonMarkup(pb.b, labels, pb.maxButtons, pb.cbFn)

}

// ButtonMarkup returns the button markup for the message.  It creates handlers
// for all the buttons assigning the cbFn callback function to each of them.
// Values must be unique.  maxRowButtons is maximum number of buttons in a row.
func ButtonMarkup(b Boter, values []string, maxRowButtons int, cbFn func(*tb.Callback)) *tb.ReplyMarkup {
	if maxRowButtons <= 0 || defNumButtons < maxRowButtons {
		maxRowButtons = defNumButtons
	}
	selector := new(tb.ReplyMarkup)
	var btns []tb.Btn
	for _, label := range values {
		btn := selector.Data(label, hash(label), label)
		btns = append(btns, btn)
		b.Handle(&btn, cbFn)
	}

	selector.Inline(organizeButtons(btns, maxRowButtons)...)
	return selector
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
