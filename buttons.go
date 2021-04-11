package tbcomctl

import tb "gopkg.in/tucnak/telebot.v2"

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

	selector.Inline(organizeButtons(selector, btns, maxRowButtons)...)
	return selector
}

// organizeButtons organizes buttons in rows.
func organizeButtons(markup *tb.ReplyMarkup, btns []tb.Btn, btnInRow int) []tb.Row {
	var rows []tb.Row
	var buttons []tb.Btn
	for i, btn := range btns {
		if i%btnInRow == 0 {
			if len(buttons) > 0 {
				rows = append(rows, markup.Row(buttons...))
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
