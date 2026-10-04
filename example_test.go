package logmanager_test

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"

	logmanager "github.com/tpyle/log-manager/v3"
)

func ExampleNewSlog() {
	// Share one LevelVar between the handler options and the manager.
	level := new(slog.LevelVar)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{} // stable output
			}
			return a
		},
	}))
	lm := logmanager.NewSlog(level, logmanager.WithOnChange(logmanager.SlogChangeLogger(logger)))

	logger.Debug("not shown")

	req := httptest.NewRequest(http.MethodPost, "/config", strings.NewReader(`{"level":"debug"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	lm.ServeHTTP(w, req)
	fmt.Print(w.Body.String())

	logger.Debug("now shown")
	// Output:
	// level=INFO msg="log level changed" oldLevel=info newLevel=debug
	// {"level":"debug"}
	// level=DEBUG msg="now shown"
}

// verbosity is a minimal custom LevelController for a logger with a numeric
// verbosity setting.
type verbosity struct{ v atomic.Int32 }

func (c *verbosity) Level() string { return fmt.Sprint(c.v.Load()) }

func (c *verbosity) SetLevel(name string) error {
	var n int32
	if _, err := fmt.Sscan(name, &n); err != nil || n < 0 || n > 9 {
		return fmt.Errorf("%w: %q", logmanager.ErrUnknownLevel, name)
	}
	c.v.Store(n)
	return nil
}

func ExampleNew() {
	lm := logmanager.New(&verbosity{})

	req := httptest.NewRequest(http.MethodPost, "/config", strings.NewReader(`{"level":"3"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	lm.ServeHTTP(w, req)
	fmt.Print(w.Body.String())
	// Output:
	// {"level":"3"}
}

func ExampleLogManager_mounted() {
	lm := logmanager.NewSlog(new(slog.LevelVar), logmanager.WithOnChange(nil))

	mux := http.NewServeMux()
	mux.Handle("/admin/log/", http.StripPrefix("/admin/log", lm))

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin/log/config", nil))
	fmt.Print(w.Body.String())
	// Output:
	// {"level":"info"}
}
