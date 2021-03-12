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

	nameIp := tbcomctl.NewInputText(b, "name", "Input your name:", processInput(b))
	ageIp := tbcomctl.NewInputText(b, "age", "Input your age", processInput(b))

	handler := tbcomctl.NewControllerChain(nameIp, ageIp)
	form := tbcomctl.NewForm(nameIp, ageIp)

	b.Handle("/input", handler)
	b.Handle("/form", form.Handler)
	// b.Handle(tb.OnText, tbcomctl.NewMiddlewareChain(onText, nameIp.OnTextMw, ageIp.OnTextMw))
	b.Handle(tb.OnText, form.OnTextMiddleware(func(m *tb.Message) {
		log.Printf("onText is called: %q\nuser data: %v", m.Text, form.Data(m.Sender))
	}))

	b.Start()
}

func processInput(b *tb.Bot) func(context.Context, *tb.Message) error {
	return func(ctx context.Context, m *tb.Message) error {
		val := m.Text
		log.Println("msgCb function is called, input value:", val)
		switch val {
		case "error":
			return fmt.Errorf("error requested: %s", val)
		case "wrong":
			return &tbcomctl.InputError{Message: "wrong input"}
		}
		if ctrl, ok := tbcomctl.ControllerFromCtx(ctx); ok {
			log.Println("form values so far: ", ctrl.Form().Data(m.Sender))
		}
		return nil
	}
}

func onText(m *tb.Message) {
	log.Println("onText is called: ", m.Text)
}
