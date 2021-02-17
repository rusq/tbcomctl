package tbcomctl

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rusq/dlog"

	tb "gopkg.in/tucnak/telebot.v2"
)

const None = "<none>"
const notAvailable = "N/A"
const chatPrivate = "private"

func Userinfo(u *tb.User) string {
	if u == nil {
		return None
	}

	return fmt.Sprintf("<[ID %d] %s (%s)>", u.ID, u.Username, Nvlstring(u.LanguageCode, notAvailable))
}

func ChatInfo(ch *tb.Chat) string {
	if ch == nil {
		return None
	}
	if ch.Type == chatPrivate {
		return chatPrivate + ":" + Userinfo(&tb.User{
			ID:        int(ch.ID),
			FirstName: ch.FirstName,
			LastName:  ch.LastName,
			Username:  ch.Username,
		})
	}
	var title = ""
	if ch.Title != "" {
		title = fmt.Sprintf(" (%q)", ch.Title)
	}
	return fmt.Sprintf("<[%d] %s%s>", ch.ID, ch.Type, title)
}

func Sdump(m interface{}) string {
	var buf strings.Builder
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(m); err != nil {
		dlog.Println("failed to marshal")
	}
	return buf.String()
}
