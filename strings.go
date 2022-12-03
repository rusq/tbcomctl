package tbcomctl

import (
	"crypto/rand"
	"io"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
	tb "gopkg.in/telebot.v3"
)

// Nvlstring returns the first non-empty string from the list.
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

const (
	randStringCharset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	randStringSz      = len(randStringCharset)
)

var randReader = rand.Reader

// randString returns a random string of n size.
func randString(n int) string {
	var buf = make([]byte, n)
	if x, err := randRead(buf); err != nil || x != n {
		panic("error reading from crypto source")
	}

	var ret = make([]byte, n)
	for i := range buf {
		ret[i] = randStringCharset[buf[i]%byte(randStringSz)]
	}
	return string(ret)
}

// randRead is a helper function that calls Reader.Read using io.ReadFull.
// On return, n == len(b) if and only if err == nil.
// (copy of crypto/rand.Read)
func randRead(b []byte) (n int, err error) {
	return io.ReadFull(randReader, b)
}
