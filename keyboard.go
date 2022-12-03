package tbcomctl

import tb "gopkg.in/telebot.v3"

type BtnLabel string

type KbdOption func(k *Keyboard)

func KbdOptButtonsInRow(n int) KbdOption {
	return func(k *Keyboard) {
		if n > 0 {
			k.btnsInRow = n
		}
	}
}

type Keyboard struct {
	commonCtl
	cmds      KeyboardCommands
	btnsInRow int
}

type KeyboardCmd struct {
	Label   BtnLabel
	Handler tb.HandlerFunc
}

type KeyboardCommands []KeyboardCmd

// type KeyboardCommands map[BtnLabel]func(c tb.Context) error

func NewKeyboard(cmds KeyboardCommands, opts ...KbdOption) *Keyboard {
	kbd := &Keyboard{
		commonCtl: commonCtl{},
		cmds:      cmds,
		btnsInRow: defNumButtons,
	}
	for _, opt := range opts {
		opt(kbd)
	}
	return kbd
}

// Markup returns the markup to be sent to user.
func (k *Keyboard) Markup(b *tb.Bot, lang string) *tb.ReplyMarkup {
	m := &tb.ReplyMarkup{ResizeKeyboard: true}

	p := Printer(lang, k.fallbackLang)
	var btns []tb.Btn
	for _, kc := range k.cmds {
		btn := m.Text(p.Sprintf(string(kc.Label)))
		btns = append(btns, btn)
		b.Handle(&btn, kc.Handler)
	}
	m.Reply(organizeButtons(btns, k.btnsInRow)...)
	return m
}

// InitForLanguages initialises handlers for languages listed.
func (k *Keyboard) InitForLanguages(b *tb.Bot, lang ...string) {
	for _, l := range lang {
		k.Markup(b, l)
	}
}
