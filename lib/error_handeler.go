package lib

import (
	"errors"
	"runtime"
)

var ErrEmptyConverter = errors.New("converter file is empty or missing header")

func CallerName() string {
	pc, _, _, _ := runtime.Caller(2)
	return runtime.FuncForPC(pc).Name()
}
