package tbcomctl

import (
	"testing"

	tb "gopkg.in/tucnak/telebot.v3"
)

func TestChatInfo(t *testing.T) {
	type args struct {
		ch *tb.Chat
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"supergroup", args{&tb.Chat{ID: 12345, Type: "supergroup", Title: "title"}}, "<[12345] supergroup (\"title\")>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ChatInfo(tt.args.ch); got != tt.want {
				t.Errorf("ChatInfo() = %v, want %v", got, tt.want)
			}
		})
	}
}
