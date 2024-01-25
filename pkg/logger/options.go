package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/kralicky/gpkg/sync"
	slogmulti "github.com/samber/slog-multi"
	slogsampling "github.com/samber/slog-sampling"
)

var logSampler = &sampler{}

type sampler struct {
	dropped sync.Map[string, uint64]
}

func (s *sampler) onDroppedHook(_ context.Context, r slog.Record) {
	key := r.Message
	count, _ := s.dropped.LoadOrStore(key, 0)
	s.dropped.Store(key, count+1)
}

type LoggerOptions struct {
	Level          slog.Level
	AddSource      bool
	ReplaceAttr    func(groups []string, a slog.Attr) slog.Attr
	Writer         io.Writer
	ColorEnabled   bool
	Sampling       *slogsampling.ThresholdSamplingOption
	TimeFormat     string
	OmitLoggerName bool
}

func ParseLevel(lvl string) slog.Level {
	l := &slog.LevelVar{}
	l.UnmarshalText([]byte(lvl))
	return l.Level()
}

type LoggerOption func(*LoggerOptions)

func (o *LoggerOptions) apply(opts ...LoggerOption) {
	for _, op := range opts {
		op(o)
	}
}

func WithLogLevel(l slog.Level) LoggerOption {
	return func(o *LoggerOptions) {
		o.Level = slog.Level(l)
	}
}

func WithWriter(w io.Writer) LoggerOption {
	return func(o *LoggerOptions) {
		o.Writer = w
	}
}

func WithColor(color bool) LoggerOption {
	return func(o *LoggerOptions) {
		o.ColorEnabled = color
	}
}

func WithDisableCaller() LoggerOption {
	return func(o *LoggerOptions) {
		o.AddSource = false
	}
}

func WithTimeFormat(format string) LoggerOption {
	return func(o *LoggerOptions) {
		o.TimeFormat = format
	}
}

func WithSampling(cfg *slogsampling.ThresholdSamplingOption) LoggerOption {
	return func(o *LoggerOptions) {
		o.Sampling = &slogsampling.ThresholdSamplingOption{
			Tick:      cfg.Tick,
			Threshold: cfg.Threshold,
			Rate:      cfg.Rate,
			OnDropped: logSampler.onDroppedHook,
		}
		o.ReplaceAttr = func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.MessageKey {
				msg := a.Value.String()
				count, _ := logSampler.dropped.Load(msg)
				if count > 0 {
					numDropped, _ := logSampler.dropped.LoadAndDelete(msg)
					a.Value = slog.StringValue(fmt.Sprintf("x%d %s", numDropped+1, msg))
				}
			}
			return a
		}
	}
}

func WithOmitLoggerName() LoggerOption {
	return func(o *LoggerOptions) {
		o.OmitLoggerName = true
	}
}

func colorHandlerWithOptions(opts ...LoggerOption) slog.Handler {
	options := &LoggerOptions{
		Writer:         DefaultWriter,
		ColorEnabled:   ColorEnabled(),
		Level:          DefaultLogLevel,
		AddSource:      DefaultAddSource,
		TimeFormat:     DefaultTimeFormat,
		OmitLoggerName: true,
	}

	options.apply(opts...)

	if DefaultWriter == nil {
		DefaultWriter = os.Stderr
	}

	var middlewares []slogmulti.Middleware

	if options.Sampling != nil {
		middlewares = append(middlewares, options.Sampling.NewMiddleware())
	}
	var chain *slogmulti.PipeBuilder
	for i, middleware := range middlewares {
		if i == 0 {
			chain = slogmulti.Pipe(middleware)
		} else {
			chain = chain.Pipe(middleware)
		}
	}

	handler := newColorHandler(options.Writer, options)

	if chain != nil {
		handler = chain.Handler(handler)
	}

	return handler
}
