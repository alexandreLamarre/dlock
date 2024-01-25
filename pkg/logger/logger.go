package logger

import "log/slog"

const (
	errKey = "err"
)

type noAllocErr struct{ error }

func Err(e error) slog.Attr {
	if e != nil {
		e = noAllocErr{e}
	}
	return slog.Any(errKey, e)
}
