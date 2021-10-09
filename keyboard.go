package tbcomctl

import tb "gopkg.in/tucnak/telebot.v3"

type BtnLabel string

type Keyboard struct {
	commonCtl
	cmds      KeyboardCommands
	btnsInRow int
}

type KeyboardCommands map[BtnLabel]func(c tb.Context) error

func NewKeyboard(b Boter, cmds KeyboardCommands) *Keyboard {
	kbd := &Keyboard{
		commonCtl: commonCtl{b: b},
		cmds:      cmds,
		btnsInRow: defNumButtons,
	}
	return kbd
}

// Markup returns the markup to be sent to user.
func (k *Keyboard) Markup(lang string) *tb.ReplyMarkup {
	m := &tb.ReplyMarkup{ResizeKeyboard: true}

	p := Printer(lang, k.lang)
	var btns []tb.Btn
	for lbl, h := range k.cmds {
		btn := m.Text(p.Sprintf(string(lbl)))
		btns = append(btns, btn)
		k.b.Handle(&btn, h)
	}
	m.Reply(organizeButtons(btns, k.btnsInRow)...)
	return m
}

// InitForLanguages initialises handlers for languages listed.
func (k *Keyboard) InitForLanguages(lang ...string) {
	for _, l := range lang {
		k.Markup(l)
	}
}
