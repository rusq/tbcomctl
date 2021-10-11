package tbcomctl

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	tb "gopkg.in/tucnak/telebot.v3"
)

func Test_organizeButtons(t *testing.T) {

	var btn tb.Btn

	type args struct {
		btns     []tb.Btn
		btnInRow int
	}
	tests := []struct {
		name string
		args args
		want []tb.Row
	}{
		{
			"no buttons",
			args{
				btns:     nil,
				btnInRow: defNumButtons,
			},
			nil,
		},
		{
			"1 btn",
			args{
				btns:     []tb.Btn{btn},
				btnInRow: defNumButtons,
			},
			[]tb.Row{
				[]tb.Btn{btn},
			},
		},
		{
			"2 btn 1 btn per row",
			args{
				btns:     []tb.Btn{btn, btn},
				btnInRow: 1,
			},
			[]tb.Row{
				[]tb.Btn{btn},
				[]tb.Btn{btn},
			},
		},
		{
			"2 btn 2 btn per row",
			args{
				btns:     []tb.Btn{btn, btn},
				btnInRow: 2,
			},
			[]tb.Row{
				[]tb.Btn{btn, btn},
			},
		},
		{
			"3 btn 2 btn per row",
			args{
				btns:     []tb.Btn{btn, btn, btn},
				btnInRow: 2,
			},
			[]tb.Row{
				[]tb.Btn{btn, btn},
				[]tb.Btn{btn},
			},
		},
		{
			"4 btn 2 btn per row",
			args{
				btns:     []tb.Btn{btn, btn, btn, btn},
				btnInRow: 2,
			},
			[]tb.Row{
				[]tb.Btn{btn, btn},
				[]tb.Btn{btn, btn},
			},
		},
		{
			"4 btn 3 btn per row",
			args{
				btns:     []tb.Btn{btn, btn, btn, btn},
				btnInRow: 3,
			},
			[]tb.Row{
				[]tb.Btn{btn, btn, btn},
				[]tb.Btn{btn},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := organizeButtons(tt.args.btns, tt.args.btnInRow); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("organizeButtons() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_organizeButtonsPattern(t *testing.T) {
	var btn tb.Btn

	type args struct {
		btns    []tb.Btn
		pattern []uint
	}
	tests := []struct {
		name    string
		args    args
		want    []tb.Row
		wantErr bool
	}{
		{
			"no buttons",
			args{
				btns:    nil,
				pattern: []uint{1},
			},
			nil,
			true,
		},
		{
			"1 btn, 1 button pattern",
			args{
				btns:    []tb.Btn{btn},
				pattern: []uint{1},
			},
			[]tb.Row{
				[]tb.Btn{btn},
			},
			false,
		},
		{
			"2 btn 1 btn per row",
			args{
				btns:    []tb.Btn{btn, btn},
				pattern: []uint{1, 1},
			},
			[]tb.Row{
				[]tb.Btn{btn},
				[]tb.Btn{btn},
			},
			false,
		},
		{
			"3 btn 1,2 pattern",
			args{
				btns:    []tb.Btn{btn, btn, btn},
				pattern: []uint{1, 2},
			},
			[]tb.Row{
				[]tb.Btn{btn},
				[]tb.Btn{btn, btn},
			},
			false,
		},
		{
			"4 btn, 1,2,1 pattern",
			args{
				btns:    []tb.Btn{btn, btn, btn, btn},
				pattern: []uint{1, 2, 1},
			},
			[]tb.Row{
				[]tb.Btn{btn},
				[]tb.Btn{btn, btn},
				[]tb.Btn{btn},
			},
			false,
		},
		{
			"4 btn, 2,2,1 pattern",
			args{
				btns:    []tb.Btn{btn, btn, btn, btn},
				pattern: []uint{2, 2, 1},
			},
			[]tb.Row{
				[]tb.Btn{btn, btn},
				[]tb.Btn{btn, btn},
			},
			false,
		},
		{
			"4 btn 3 button pattern",
			args{
				btns:    []tb.Btn{btn, btn, btn, btn},
				pattern: []uint{2, 1},
			},
			nil,
			true,
		},
		{
			"4 btn 4 button pattern, zero in pattern",
			args{
				btns:    []tb.Btn{btn, btn, btn, btn},
				pattern: []uint{2, 0, 2},
			},
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := organizeButtonsPattern(tt.args.btns, tt.args.pattern)
			if (err != nil) != tt.wantErr {
				t.Errorf("organizeButtonsPattern() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
