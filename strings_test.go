package tbcomctl

import (
	"bytes"
	"io"
	"sync"
	"testing"
)

// testRand is the
type testRand struct {
	oldRR  io.Reader
	randMu sync.Mutex // mutex to guard the global randReader during tests.
}

func (tr *testRand) setRandReader(t *testing.T, r io.Reader) {
	tr.randMu.Lock()
	defer tr.randMu.Unlock()
	if tr.oldRR != nil {
		t.Fatal("called setRandReader more than once")
	}
	tr.oldRR = randReader
	randReader = r
}

func (tr *testRand) restore() {
	tr.randMu.Lock()
	defer tr.randMu.Unlock()

	randReader = tr.oldRR
	tr.oldRR = nil
}

func Test_randString(t *testing.T) {
	type args struct {
		n int
	}
	tests := []struct {
		name       string
		randReader io.Reader
		args       args
		want       string
	}{
		{"abcde", bytes.NewReader([]byte{0, 1, 2, 3, 4}), args{len("abcde")}, "abcde"},
		{"<empty>", bytes.NewReader([]byte{0, 1, 2, 3, 4}), args{0}, ""},
		{"reader has more than needed", bytes.NewReader([]byte{0, 1, 2, 3, 4, 5, 6}), args{len("abcde")}, "abcde"},
	}
	var tr testRand

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr.setRandReader(t, tt.randReader)
			defer tr.restore()

			if got := randString(tt.args.n); got != tt.want {
				t.Errorf("randString() = %v, want %v", got, tt.want)
			}
		})
	}
}
