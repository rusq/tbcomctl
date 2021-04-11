package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/rusq/tbcomctl"
	tb "gopkg.in/tucnak/telebot.v2"
)

var _ = godotenv.Load()

var (
	token = os.Getenv("TOKEN")
	chat  = os.Getenv("CHAT")
)

func main() {
	b, err := tb.NewBot(tb.Settings{
		Token:  token,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		log.Fatal(err)
	}
	p1 := tbcomctl.NewPicklistText(
		b,
		"1",
		"first picklist",
		[]string{"1", "2", "3", "4"},
		func(ctx context.Context, cb *tb.Callback) error {
			fmt.Println(tbcomctl.Sdump(cb))
			return nil
		},
	)
	p2 := tbcomctl.NewPicklistText(
		b,
		"2",
		"second picklist",
		[]string{"5", "6", "7", "8"},
		func(ctx context.Context, cb *tb.Callback) error {
			fmt.Println(tbcomctl.Sdump(cb))
			return nil
		},
		tbcomctl.PickOptBtnPattern([]uint{1, 2, 1}),
	)
	m := tbcomctl.NewMessageText(b, "msg", "all ok")
	form := tbcomctl.NewForm(p1, p2, m).
		SetOverwrite(true).
		SetRemoveButtons(true)
	b.Handle("/picklist", form.Handler)

	b.Start()
}
