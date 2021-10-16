package tbcomctl

import (
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	tb "gopkg.in/tucnak/telebot.v3"
)

func Nvlstring(s string, ss ...string) string {
	if s != "" {
		return s
	}
	for _, alt := range ss {
		if alt != "" {
			return alt
		}
	}
	return ""
}

// PrinterContext returns the Message Printer set to the language of the sender.
// It is a convenience wrapper around Printer.
func PrinterContext(c tb.Context, fallback ...string) *message.Printer {
	return Printer(c.Sender().LanguageCode, fallback...)
}

// Printer returns the Message Printer for the desired lang.  If the lang is not
// valid, the fallback languages will be used, if set.
func Printer(lang string, fallback ...string) *message.Printer {
	tag, err := language.Parse(lang)
	if err != nil {
		if len(fallback) > 0 && fallback[0] != "" {
			tag = language.MustParse(fallback[0])
		} else {
			tag = language.MustParse(FallbackLang)
		}
	}
	return message.NewPrinter(tag)
}
