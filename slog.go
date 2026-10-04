package logmanager

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

// SlogController is a [LevelController] backed by a [*slog.LevelVar].
//
// slog has no global level, so the LevelVar must be the one passed as
// [slog.HandlerOptions.Level] to the handlers whose level should be
// controlled. Any number of handlers may share it.
//
// Accepted level names are those understood by [slog.Level.UnmarshalText],
// matched case-insensitively: "debug", "info", "warn", "error", optionally
// with an offset such as "debug-4" or "info+2". "warning" is accepted as an
// alias for "warn". Levels are reported in lower case, e.g. "info" or
// "debug-4".
type SlogController struct {
	level *slog.LevelVar
}

var _ LevelController = (*SlogController)(nil)

// NewSlogController returns a [LevelController] for v. It panics if v is nil.
func NewSlogController(v *slog.LevelVar) *SlogController {
	if v == nil {
		panic("logmanager: nil *slog.LevelVar")
	}
	return &SlogController{level: v}
}

// NewSlog returns a LogManager controlling v that, by default, logs level
// changes with [SlogChangeLogger](nil). Pass [WithOnChange] to override this.
// It panics if v is nil.
func NewSlog(v *slog.LevelVar, opts ...Option) *LogManager {
	opts = append([]Option{WithOnChange(SlogChangeLogger(nil))}, opts...)
	return New(NewSlogController(v), opts...)
}

// Level implements [LevelController].
func (s *SlogController) Level() string {
	return strings.ToLower(s.level.Level().String())
}

// SetLevel implements [LevelController].
func (s *SlogController) SetLevel(name string) error {
	if strings.EqualFold(name, "warning") {
		name = "warn"
	}
	var l slog.Level
	if err := l.UnmarshalText([]byte(name)); err != nil {
		return fmt.Errorf("%w: %q", ErrUnknownLevel, name)
	}
	s.level.Set(l)
	return nil
}

// SlogChangeLogger returns a [LevelChangeFunc] that logs each change at
// [slog.LevelInfo] with l, passing the request context. If l is nil,
// [slog.Default] is used, looked up on every change.
//
// A change to a level above info suppresses its own message.
func SlogChangeLogger(l *slog.Logger) LevelChangeFunc {
	return func(ctx context.Context, from, to string) {
		logger := l
		if logger == nil {
			logger = slog.Default()
		}
		logger.LogAttrs(ctx, slog.LevelInfo, ChangeMessage,
			slog.String(FieldOldLevel, from),
			slog.String(FieldNewLevel, to),
		)
	}
}
