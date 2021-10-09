package main

import (
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/rusq/tbcomctl"
	tb "gopkg.in/tucnak/telebot.v3"
)

var _ = godotenv.Load()

var (
	token = os.Getenv("TOKEN")
	chat  = os.Getenv("CHAT")
)

func main() {
	b, err := tb.NewBot(tb.Settings{
		URL:    "http://localhost:8081",
		Token:  token,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		log.Fatal(err)
	}

	sc := tbcomctl.NewSubChecker(b, "sc", tbcomctl.TextFn("test sub"), []string{chat})

	b.Handle("/subcheck", sc.Handler)

	b.Start()
}
