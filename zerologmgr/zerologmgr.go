// Package zerologmgr adapts [github.com/rs/zerolog] to logmanager.
//
// It controls zerolog's global level ([zerolog.SetGlobalLevel]). zerolog drops
// an event if it is below either the global level or the logger's own level,
// so the global level can only make a logger quieter than its own level. For
// full runtime control, leave loggers at their default level (trace) instead
// of calling [zerolog.Logger.Level].
package zerologmgr

import (
	"context"
	"fmt"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	logmanager "github.com/tpyle/log-manager/v3"
)

// levelNames maps each controllable level to the name reported by
// [Controller.Level]. Fixed names are used instead of [zerolog.Level.String]
// because the latter follows the configurable zerolog.Level*Value variables.
var levelNames = map[zerolog.Level]string{
	zerolog.TraceLevel: "trace",
	zerolog.DebugLevel: "debug",
	zerolog.InfoLevel:  "info",
	zerolog.WarnLevel:  "warn",
	zerolog.ErrorLevel: "error",
	zerolog.FatalLevel: "fatal",
	zerolog.PanicLevel: "panic",
	zerolog.Disabled:   "disabled",
}

// Controller is a [logmanager.LevelController] for zerolog's global level.
//
// Accepted level names, matched case-insensitively, are "trace", "debug",
// "info", "warn" (or "warning"), "error", "fatal", "panic" and "disabled".
type Controller struct{}

var _ logmanager.LevelController = Controller{}

// NewController returns a [logmanager.LevelController] for zerolog's global
// level.
func NewController() Controller {
	return Controller{}
}

// New returns a LogManager controlling zerolog's global level that, by
// default, logs level changes with [ChangeLogger](nil). Pass
// [logmanager.WithOnChange] to override this.
func New(opts ...logmanager.Option) *logmanager.LogManager {
	opts = append([]logmanager.Option{logmanager.WithOnChange(ChangeLogger(nil))}, opts...)
	return logmanager.New(NewController(), opts...)
}

// Level implements [logmanager.LevelController]. A global level set outside
// this package to a value without a name (such as [zerolog.NoLevel]) is
// reported as its number.
func (Controller) Level() string {
	l := zerolog.GlobalLevel()
	if name, ok := levelNames[l]; ok {
		return name
	}
	return fmt.Sprint(int8(l))
}

// SetLevel implements [logmanager.LevelController].
func (Controller) SetLevel(name string) error {
	l, ok := parseLevel(name)
	if !ok {
		return fmt.Errorf("%w: %q", logmanager.ErrUnknownLevel, name)
	}
	zerolog.SetGlobalLevel(l)
	return nil
}

func parseLevel(name string) (zerolog.Level, bool) {
	name = strings.ToLower(name)
	if name == "warning" {
		return zerolog.WarnLevel, true
	}
	for l, n := range levelNames {
		if n == name {
			return l, true
		}
	}
	return zerolog.NoLevel, false
}

// ChangeLogger returns a [logmanager.LevelChangeFunc] that logs each change
// at info level with l. If l is nil, the logger attached to the request
// context ([zerolog.Ctx]) is used, or the global [log.Logger] if the context
// has none.
//
// A change to a level above info suppresses its own message.
func ChangeLogger(l *zerolog.Logger) logmanager.LevelChangeFunc {
	return func(ctx context.Context, from, to string) {
		logger := l
		if logger == nil {
			logger = zerolog.Ctx(ctx)
			if logger.GetLevel() == zerolog.Disabled {
				logger = &log.Logger
			}
		}
		logger.Info().Ctx(ctx).
			Str(logmanager.FieldOldLevel, from).
			Str(logmanager.FieldNewLevel, to).
			Msg(logmanager.ChangeMessage)
	}
}
