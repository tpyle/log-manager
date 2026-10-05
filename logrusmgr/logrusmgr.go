// Package logrusmgr adapts [github.com/sirupsen/logrus] to logmanager.
//
// It controls the level of a single [*logrus.Logger]
// ([logrus.Logger.SetLevel]). Entries derived from that logger, including
// those created with WithField or WithContext, follow it.
package logrusmgr

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	logmanager "github.com/tpyle/log-manager/v3"
)

// Controller is a [logmanager.LevelController] for a [*logrus.Logger].
//
// Accepted level names are those understood by [logrus.ParseLevel], matched
// case-insensitively: "trace", "debug", "info", "warn" or "warning", "error",
// "fatal" and "panic". Levels are reported as [logrus.Level.String] returns
// them, so the warn level is reported as "warning".
type Controller struct {
	logger *logrus.Logger
}

var _ logmanager.LevelController = (*Controller)(nil)

// NewController returns a [logmanager.LevelController] for l. If l is nil,
// [logrus.StandardLogger] is used.
func NewController(l *logrus.Logger) *Controller {
	if l == nil {
		l = logrus.StandardLogger()
	}
	return &Controller{logger: l}
}

// New returns a LogManager controlling l (or [logrus.StandardLogger] if l is
// nil) that, by default, logs level changes with [ChangeLogger] through the
// same logger. Pass [logmanager.WithOnChange] to override this.
func New(l *logrus.Logger, opts ...logmanager.Option) *logmanager.LogManager {
	c := NewController(l)
	opts = append([]logmanager.Option{logmanager.WithOnChange(ChangeLogger(c.logger))}, opts...)
	return logmanager.New(c, opts...)
}

// Level implements [logmanager.LevelController].
func (c *Controller) Level() string {
	return c.logger.GetLevel().String()
}

// SetLevel implements [logmanager.LevelController].
func (c *Controller) SetLevel(name string) error {
	l, err := logrus.ParseLevel(name)
	if err != nil {
		return fmt.Errorf("%w: %q", logmanager.ErrUnknownLevel, name)
	}
	c.logger.SetLevel(l)
	return nil
}

// ChangeLogger returns a [logmanager.LevelChangeFunc] that logs each change
// at info level with l, bound to the request context so context hooks apply.
// If l is nil, [logrus.StandardLogger] is used.
//
// A change to a level above info suppresses its own message.
func ChangeLogger(l logrus.FieldLogger) logmanager.LevelChangeFunc {
	if l == nil {
		l = logrus.StandardLogger()
	}
	return func(ctx context.Context, from, to string) {
		l.WithFields(logrus.Fields{
			logmanager.FieldOldLevel: from,
			logmanager.FieldNewLevel: to,
		}).WithContext(ctx).Info(logmanager.ChangeMessage)
	}
}
