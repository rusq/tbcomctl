package tbcomctl

import (
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

const (
	MsgUnexpected  = "🤯 (500) Unexpected error occurred."
	MsgRetry       = "Incorrect choice."
	MsgChooseVal   = "Choose the value from the list:"
	MsgOK          = "✅"
	MsgVoteCounted = "✅ Vote counted."
	MsgSubCheck    = "？ Check subscription >>"
	MsgSubNoSub    = "❌ You're not subscribed to one or more of the required channels."
)

var translations = map[language.Tag][]i18nmsg{
	language.Russian: {
		{MsgUnexpected, "🤯 (500) Произошло недоразумение."},
		{MsgRetry, "Неверный выбор"},
		{MsgChooseVal, "Выберите значение из списка:"},
		{MsgVoteCounted, "✅ Голос учтен."},
		{MsgSubCheck, "？ Проверить подписку >>"},
		{MsgSubNoSub, "❌ Вы не подписались на один или более необходимых каналов."},
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
