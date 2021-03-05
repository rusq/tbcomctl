package main

import (
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

// [chatID][msgID]Buttons
var ratings = make(map[int64]map[string][2]tbcomctl.Button)

func main() {
	b, err := tb.NewBot(tb.Settings{
		Token:  token,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		log.Fatal(err)
	}

	rb := tbcomctl.NewRating(b,
		func(e tb.Editable, r *tb.User, idx int) ([2]tbcomctl.Button, error) {
			mid, cid := e.MessageSig()
			log.Printf("%s, %d: u: %s, idx %d", mid, cid, r.Recipient(), idx)
			btns := getButtons(cid, mid, r.Recipient())
			btns[idx].Value++
			ratings[cid][mid] = btns
			return btns, nil
		},
		tbcomctl.RBOptShowVoteCounter(true),
	)

	go func() {
		ch, err := b.ChatByID(chat)
		if err != nil {
			log.Fatal(err)
		}
		if _, err := b.Send(ch, "rating test", rb.Markup(ratingButtons())); err != nil {
			log.Fatal(err)
		}
	}()

	b.Start()
}

func getButtons(chatID int64, msgID string, userID string) [2]tbcomctl.Button {
	if _, ok := ratings[chatID]; !ok {
		ratings[chatID] = make(map[string][2]tbcomctl.Button)
	}
	if _, ok := ratings[chatID][msgID]; !ok {
		ratings[chatID][msgID] = ratingButtons()
	}
	return ratings[chatID][msgID]
}

func ratingButtons() [2]tbcomctl.Button {
	return [2]tbcomctl.Button{
		{Name: "up"},
		{Name: "dn"},
	}
}
