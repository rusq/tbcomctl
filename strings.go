package tbcomctl

import (
	"golang.org/x/text/language"
	"golang.org/x/text/message"
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

func Printer(lang string, fallback ...string) *message.Printer {
	tag, err := language.Parse(lang)
	if err != nil {
		if len(fallback) > 0 && fallback[0] != "" {
			tag = language.MustParse(fallback[0])
		} else {
			tag = language.MustParse("en-US")
		}
	}
	return message.NewPrinter(tag)
}
