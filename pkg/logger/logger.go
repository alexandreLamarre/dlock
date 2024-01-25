package logger

import (
	"io"
	"log/slog"
	"os"
	"time"
)

const (
	errKey = "err"
)

var (
	DefaultLogLevel   = slog.LevelDebug
	DefaultWriter     = os.Stdout
	DefaultAddSource  = true
	pluginGroupPrefix = "plugin"
	NoRepeatInterval  = 3600 * time.Hour // arbitrarily long time to denote one-time sampling
	DefaultTimeFormat = "2006 Jan 02 15:04:05"
)

type noAllocErr struct{ error }

func Err(e error) slog.Attr {
	if e != nil {
		e = noAllocErr{e}
	}
	return slog.Any(errKey, e)
}

func NewNop() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))
}

func New(opts ...LoggerOption) *slog.Logger {
	return slog.New(colorHandlerWithOptions(opts...))
}
