package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/rusq/tbcomctl/v4"
	tb "gopkg.in/telebot.v3"
)

var _ = godotenv.Load()

var (
	token = os.Getenv("TOKEN")
	// chat  = os.Getenv("CHAT")
)

func main() {
	b, err := tb.NewBot(tb.Settings{
		Token:  token,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		log.Fatal(err)
	}

	nameIp := tbcomctl.NewInputText("name", "Input your name:", processInput(b))
	ageIp := tbcomctl.NewInputText("age", "Input your age", processInput(b))

	form := tbcomctl.NewForm(nameIp, ageIp)
	b.Handle("/input", form.Handler)
	// b.Handle(tb.OnText, tbcomctl.NewMiddlewareChain(onText, nameIp.OnTextMw, ageIp.OnTextMw))
	b.Handle(tb.OnText, form.OnTextMiddleware(func(c tb.Context) error {
		log.Printf("onText is called: %q\nuser data: %v", c.Message().Text, form.Data(c.Sender()))
		return nil
	}))

	b.Start()
}

func processInput(b *tb.Bot) func(ctx context.Context, c tb.Context) error {
	return func(ctx context.Context, c tb.Context) error {
		val := c.Message().Text
		log.Println("msgCb function is called, input value:", val)
		switch val {
		case "error":
			return fmt.Errorf("error requested: %s", val)
		case "wrong":
			return tbcomctl.NewInputError("wrong input")
		}
		if ctrl, ok := tbcomctl.ControllerFromCtx(ctx); ok {
			log.Println("form values so far: ", ctrl.Form().Data(c.Sender()))
		}
		return nil
	}
}
