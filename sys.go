package tbcomctl

import (
	"path/filepath"
	"runtime"
)

// caller returns the name of the calling function from the stack with steps
// depth.
func caller(steps int) string {
	name := "?"
	if pc, _, _, ok := runtime.Caller(steps + 1); ok {
		name = filepath.Base(runtime.FuncForPC(pc).Name())
	}
	return name
}
