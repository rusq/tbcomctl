package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/rusq/tbcomctl/v3"
	tb "gopkg.in/tucnak/telebot.v3"
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
	p1 := tbcomctl.NewPicklist(
		"1",
		tbcomctl.NewStaticTVC("first picklist",
			[]string{"1", "2", "3", "4"},
			func(ctx context.Context, c tb.Context) error {
				fmt.Println(tbcomctl.Sdump(c.Callback()))
				return nil
			},
		),
	)
	p2 := tbcomctl.NewPicklist(
		"2",
		tbcomctl.NewStaticTVC(
			"second picklist",
			[]string{"5", "6", "7", "8", "back"},
			func(ctx context.Context, c tb.Context) error {
				fmt.Println(tbcomctl.Sdump(c.Callback()))
				if c.Data() == "back" {
					return tbcomctl.BackPressed
				}
				return nil
			},
		),
		tbcomctl.PickOptBtnPattern([]uint{1, 2, 1, 1}),
	)
	p3 := tbcomctl.NewPicklist(
		"3",
		tbcomctl.NewStaticTVC(
			"picklist with back button option",
			[]string{"9", "A", "B", "C"},
			func(ctx context.Context, c tb.Context) error {
				fmt.Println(tbcomctl.Sdump(c.Callback()))
				return nil
			},
		),
		tbcomctl.PickOptBtnPattern([]uint{1, 2, 1}),
		tbcomctl.PickOptBtnBack(tbcomctl.NewTexter("option back")),
	)
	m := tbcomctl.NewMessageText("msg", "all ok")
	form := tbcomctl.NewForm(p1, p2, p3, m).
		SetOverwrite(true).
		SetRemoveButtons(true)
	b.Handle("/picklist", form.Handler)

	log.Println("ready, send /picklist")
	b.Start()
}
