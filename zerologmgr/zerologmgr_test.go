package zerologmgr

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	logmanager "github.com/tpyle/log-manager/v3"
)

// restoreGlobals resets zerolog's global level and logger after the test.
func restoreGlobals(t *testing.T) {
	t.Helper()
	level, logger := zerolog.GlobalLevel(), log.Logger
	t.Cleanup(func() {
		zerolog.SetGlobalLevel(level)
		log.Logger = logger
	})
}

func post(t *testing.T, h http.Handler, ctx context.Context, level string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/config", strings.NewReader(`{"level":"`+level+`"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func TestSetLevel(t *testing.T) {
	restoreGlobals(t)
	tests := []struct {
		in        string
		want      zerolog.Level
		wantLevel string
	}{
		{"trace", zerolog.TraceLevel, "trace"},
		{"debug", zerolog.DebugLevel, "debug"},
		{"DEBUG", zerolog.DebugLevel, "debug"},
		{"info", zerolog.InfoLevel, "info"},
		{"warn", zerolog.WarnLevel, "warn"},
		{"warning", zerolog.WarnLevel, "warn"},
		{"Warning", zerolog.WarnLevel, "warn"},
		{"error", zerolog.ErrorLevel, "error"},
		{"fatal", zerolog.FatalLevel, "fatal"},
		{"panic", zerolog.PanicLevel, "panic"},
		{"disabled", zerolog.Disabled, "disabled"},
	}
	c := NewController()
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			zerolog.SetGlobalLevel(zerolog.NoLevel)
			if err := c.SetLevel(tt.in); err != nil {
				t.Fatalf("SetLevel(%q): %v", tt.in, err)
			}
			if zerolog.GlobalLevel() != tt.want {
				t.Errorf("global level = %v, want %v", zerolog.GlobalLevel(), tt.want)
			}
			if got := c.Level(); got != tt.wantLevel {
				t.Errorf("Level() = %q, want %q", got, tt.wantLevel)
			}
		})
	}
}

func TestSetLevelUnknown(t *testing.T) {
	restoreGlobals(t)
	for _, in := range []string{"", "verbose", "0", "1", "-1", "nolevel", "debug-4", " info"} {
		t.Run(in, func(t *testing.T) {
			zerolog.SetGlobalLevel(zerolog.WarnLevel)
			if err := NewController().SetLevel(in); !errors.Is(err, logmanager.ErrUnknownLevel) {
				t.Errorf("SetLevel(%q) = %v, want ErrUnknownLevel", in, err)
			}
			if zerolog.GlobalLevel() != zerolog.WarnLevel {
				t.Errorf("level changed to %v", zerolog.GlobalLevel())
			}
		})
	}
}

func TestLevelIgnoresCustomLevelNames(t *testing.T) {
	restoreGlobals(t)
	orig := zerolog.LevelWarnValue
	t.Cleanup(func() { zerolog.LevelWarnValue = orig })
	zerolog.LevelWarnValue = "WRN"

	zerolog.SetGlobalLevel(zerolog.WarnLevel)
	if got := NewController().Level(); got != "warn" {
		t.Errorf("Level() = %q, want warn", got)
	}
}

func TestLevelUnnamed(t *testing.T) {
	restoreGlobals(t)
	zerolog.SetGlobalLevel(zerolog.NoLevel)
	if got := NewController().Level(); got != "6" {
		t.Errorf("Level() = %q, want 6", got)
	}
}

func TestControlsLogger(t *testing.T) {
	restoreGlobals(t)
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	lm := New(logmanager.WithOnChange(nil))

	logger.Debug().Msg("hidden")
	if w := post(t, lm, context.Background(), "debug"); w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %q", w.Code, w.Body.String())
	}
	logger.Debug().Msg("shown")

	if out := buf.String(); strings.Contains(out, "hidden") || !strings.Contains(out, "shown") {
		t.Errorf("unexpected output: %q", out)
	}
}

func decode(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("bad JSON %q: %v", buf.String(), err)
	}
	return m
}

func checkChangeLine(t *testing.T, m map[string]any, from, to string) {
	t.Helper()
	want := map[string]any{
		"level":                  "info",
		"message":                logmanager.ChangeMessage,
		logmanager.FieldOldLevel: from,
		logmanager.FieldNewLevel: to,
	}
	for k, v := range want {
		if m[k] != v {
			t.Errorf("%s = %v, want %v", k, m[k], v)
		}
	}
}

func TestDefaultChangeLoggerUsesContextLogger(t *testing.T) {
	restoreGlobals(t)
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	var global, scoped bytes.Buffer
	log.Logger = zerolog.New(&global)
	ctxLogger := zerolog.New(&scoped).With().Str("requestId", "req-1").Logger()

	post(t, New(), ctxLogger.WithContext(context.Background()), "debug")

	if global.Len() != 0 {
		t.Errorf("global logger used: %q", global.String())
	}
	m := decode(t, &scoped)
	checkChangeLine(t, m, "info", "debug")
	if m["requestId"] != "req-1" {
		t.Errorf("context logger fields missing: %v", m)
	}
}

func TestDefaultChangeLoggerFallsBackToGlobal(t *testing.T) {
	restoreGlobals(t)
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	var global bytes.Buffer
	log.Logger = zerolog.New(&global)

	post(t, New(), context.Background(), "trace")
	checkChangeLine(t, decode(t, &global), "info", "trace")
}

func TestChangeLoggerExplicit(t *testing.T) {
	restoreGlobals(t)
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	var buf bytes.Buffer
	l := zerolog.New(&buf)
	ChangeLogger(&l)(context.Background(), "warn", "info")
	checkChangeLine(t, decode(t, &buf), "warn", "info")
}

func TestChangeLoggerPassesContextToHooks(t *testing.T) {
	restoreGlobals(t)
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	type key struct{}
	var seen any
	var buf bytes.Buffer
	l := zerolog.New(&buf).Hook(zerolog.HookFunc(func(e *zerolog.Event, _ zerolog.Level, _ string) {
		seen = e.GetCtx().Value(key{})
	}))
	ChangeLogger(&l)(context.WithValue(context.Background(), key{}, "v"), "info", "debug")
	if seen != "v" {
		t.Errorf("hook saw context value %v", seen)
	}
}
