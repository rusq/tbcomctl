package main

import (
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/rusq/dlog"
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

	rb := tbcomctl.NewRating(b,
		func(e tb.Editable, r tb.Recipient, b tbcomctl.Button) (int, error) {
			mid, cid := e.MessageSig()
			dlog.Printf("%s, %d: u: %s, btn %s", mid, cid, r.Recipient(), b.String())
			return b.Value + 1, nil
		},
		tbcomctl.RBOptShowVoteCounter(true),
	)

	go func() {
		ch, err := b.ChatByID(chat)
		if err != nil {
			log.Fatal(err)
		}
		if _, err := b.Send(ch, "rating test", rb.Markup()); err != nil {
			log.Fatal(err)
		}
	}()

	b.Start()
}
