package tbcomctl

import (
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

const (
	MsgUnexpected  = "Unexpected error occurred."
	MsgRetry       = "Incorrect choice."
	MsgChooseVal   = "Choose value from the list:"
	MsgOK          = "Changed successfully."
	MsgVoteCounted = "Vote counted."
)

var translations = map[language.Tag][]i18nmsg{
	language.Russian: {
		{MsgUnexpected, "Произошло недоразумение."},
		{MsgRetry, "Неверный выбор"},
		{MsgChooseVal, "Сделайте выбор из списка:"},
		{MsgOK, "Успешно изменено."},
		{MsgVoteCounted, "Голос учтен."},
	},
}

type i18nmsg struct {
	key         string
	translation string
}

func init() {
	initMessages()
}

func initMessages() {
	for l, tt := range translations {
		for _, t := range tt {
			must(message.SetString(l, t.key, t.translation))
		}
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
