package logmanager

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"testing"
)

func TestNewSlogControllerPanicsOnNil(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("NewSlogController(nil) did not panic")
		}
	}()
	NewSlogController(nil)
}

func TestSlogSetLevel(t *testing.T) {
	tests := []struct {
		in        string
		want      slog.Level
		wantLevel string
	}{
		{"debug", slog.LevelDebug, "debug"},
		{"DEBUG", slog.LevelDebug, "debug"},
		{"info", slog.LevelInfo, "info"},
		{"Info", slog.LevelInfo, "info"},
		{"warn", slog.LevelWarn, "warn"},
		{"warning", slog.LevelWarn, "warn"},
		{"WARNING", slog.LevelWarn, "warn"},
		{"error", slog.LevelError, "error"},
		{"debug-4", slog.LevelDebug - 4, "debug-4"},
		{"info+2", slog.LevelInfo + 2, "info+2"},
		{"ERROR+8", slog.LevelError + 8, "error+8"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			v := new(slog.LevelVar)
			v.Set(slog.LevelError + 100)
			c := NewSlogController(v)
			if err := c.SetLevel(tt.in); err != nil {
				t.Fatalf("SetLevel(%q): %v", tt.in, err)
			}
			if v.Level() != tt.want {
				t.Errorf("LevelVar = %v, want %v", v.Level(), tt.want)
			}
			if got := c.Level(); got != tt.wantLevel {
				t.Errorf("Level() = %q, want %q", got, tt.wantLevel)
			}
			// The reported name must be accepted back.
			if err := c.SetLevel(c.Level()); err != nil || v.Level() != tt.want {
				t.Errorf("round trip of %q failed: %v, %v", c.Level(), err, v.Level())
			}
		})
	}
}

func TestSlogSetLevelUnknown(t *testing.T) {
	for _, in := range []string{"", "trace", "fatal", "panic", "verbose", "info+", "4", "warning+1"} {
		t.Run(in, func(t *testing.T) {
			v := new(slog.LevelVar)
			v.Set(slog.LevelWarn)
			err := NewSlogController(v).SetLevel(in)
			if !errors.Is(err, ErrUnknownLevel) {
				t.Errorf("SetLevel(%q) = %v, want ErrUnknownLevel", in, err)
			}
			if v.Level() != slog.LevelWarn {
				t.Errorf("level changed to %v", v.Level())
			}
		})
	}
}

func TestSlogControlsHandler(t *testing.T) {
	var buf bytes.Buffer
	v := new(slog.LevelVar)
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: v}))
	lm := NewSlog(v, WithOnChange(nil))

	logger.Debug("hidden")
	do(t, lm, http.MethodPost, "/config", "application/json", `{"level":"debug"}`)
	logger.Debug("shown")

	if out := buf.String(); strings.Contains(out, "hidden") || !strings.Contains(out, "shown") {
		t.Errorf("unexpected output: %q", out)
	}
}

func TestSlogChangeLogger(t *testing.T) {
	var buf bytes.Buffer
	l := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	}))
	SlogChangeLogger(l)(context.Background(), "info", "debug")
	want := `level=INFO msg="log level changed" oldLevel=info newLevel=debug` + "\n"
	if buf.String() != want {
		t.Errorf("got %q, want %q", buf.String(), want)
	}
}

func TestNewSlogDefaultLogsToSlogDefault(t *testing.T) {
	orig := slog.Default()
	t.Cleanup(func() { slog.SetDefault(orig) })

	v := new(slog.LevelVar)
	lm := NewSlog(v)

	// Set after construction to check the default logger is looked up lazily.
	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
	do(t, lm, http.MethodPost, "/config", "application/json", `{"level":"warn"}`)

	out := buf.String()
	for _, s := range []string{`"msg":"log level changed"`, `"oldLevel":"info"`, `"newLevel":"warn"`} {
		if !strings.Contains(out, s) {
			t.Errorf("output %q missing %s", out, s)
		}
	}
}
