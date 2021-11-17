package main

import (
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/rusq/osenv"
	"github.com/rusq/tbcomctl/v3"
	tb "gopkg.in/tucnak/telebot.v3"
)

var _ = godotenv.Load()

var (
	token = os.Getenv("TOKEN")
	chat  = osenv.Int64("CHAT", 0)
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

	sc := tbcomctl.NewSubChecker("sc", tbcomctl.NewTexter("test sub"), []int64{chat})

	b.Handle("/subcheck", sc.Handler)

	b.Start()
}
