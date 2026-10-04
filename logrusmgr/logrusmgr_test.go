package logrusmgr

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"

	logmanager "github.com/tpyle/log-manager/v3"
)

func newLogger(buf *bytes.Buffer) *logrus.Logger {
	l := logrus.New()
	l.SetOutput(buf)
	l.SetFormatter(&logrus.JSONFormatter{DisableTimestamp: true})
	return l
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
	tests := []struct {
		in        string
		want      logrus.Level
		wantLevel string
	}{
		{"trace", logrus.TraceLevel, "trace"},
		{"debug", logrus.DebugLevel, "debug"},
		{"DEBUG", logrus.DebugLevel, "debug"},
		{"info", logrus.InfoLevel, "info"},
		{"warn", logrus.WarnLevel, "warning"},
		{"warning", logrus.WarnLevel, "warning"},
		{"error", logrus.ErrorLevel, "error"},
		{"fatal", logrus.FatalLevel, "fatal"},
		{"panic", logrus.PanicLevel, "panic"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			l := logrus.New()
			c := NewController(l)
			if err := c.SetLevel(tt.in); err != nil {
				t.Fatalf("SetLevel(%q): %v", tt.in, err)
			}
			if l.GetLevel() != tt.want {
				t.Errorf("level = %v, want %v", l.GetLevel(), tt.want)
			}
			if got := c.Level(); got != tt.wantLevel {
				t.Errorf("Level() = %q, want %q", got, tt.wantLevel)
			}
		})
	}
}

func TestSetLevelUnknown(t *testing.T) {
	for _, in := range []string{"", "verbose", "disabled", "4", " info"} {
		t.Run(in, func(t *testing.T) {
			l := logrus.New()
			l.SetLevel(logrus.WarnLevel)
			if err := NewController(l).SetLevel(in); !errors.Is(err, logmanager.ErrUnknownLevel) {
				t.Errorf("SetLevel(%q) = %v, want ErrUnknownLevel", in, err)
			}
			if l.GetLevel() != logrus.WarnLevel {
				t.Errorf("level changed to %v", l.GetLevel())
			}
		})
	}
}

func TestNilUsesStandardLogger(t *testing.T) {
	std := logrus.StandardLogger()
	orig := std.GetLevel()
	t.Cleanup(func() { std.SetLevel(orig) })

	if err := NewController(nil).SetLevel("trace"); err != nil {
		t.Fatal(err)
	}
	if std.GetLevel() != logrus.TraceLevel {
		t.Errorf("standard logger level = %v", std.GetLevel())
	}
}

func TestControlsDerivedEntries(t *testing.T) {
	var buf bytes.Buffer
	l := newLogger(&buf)
	entry := l.WithField("component", "db")
	lm := New(l, logmanager.WithOnChange(nil))

	entry.Debug("hidden")
	if w := post(t, lm, context.Background(), "debug"); w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %q", w.Code, w.Body.String())
	}
	entry.Debug("shown")

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
		"msg":                    logmanager.ChangeMessage,
		logmanager.FieldOldLevel: from,
		logmanager.FieldNewLevel: to,
	}
	for k, v := range want {
		if m[k] != v {
			t.Errorf("%s = %v, want %v", k, m[k], v)
		}
	}
}

// ctxHook copies a context value into each entry, standing in for hooks such
// as log-middleware's logrusmw.ContextHook.
type ctxHook struct{}

type ctxKey struct{}

func (ctxHook) Levels() []logrus.Level { return logrus.AllLevels }

func (ctxHook) Fire(e *logrus.Entry) error {
	if e.Context != nil {
		if v := e.Context.Value(ctxKey{}); v != nil {
			e.Data["requestId"] = v
		}
	}
	return nil
}

func TestDefaultChangeLoggerUsesManagedLoggerAndContext(t *testing.T) {
	var buf bytes.Buffer
	l := newLogger(&buf)
	l.AddHook(ctxHook{})

	post(t, New(l), context.WithValue(context.Background(), ctxKey{}, "req-1"), "debug")

	m := decode(t, &buf)
	checkChangeLine(t, m, "info", "debug")
	if m["requestId"] != "req-1" {
		t.Errorf("context not passed to hooks: %v", m)
	}
}

func TestChangeLoggerWithEntry(t *testing.T) {
	var buf bytes.Buffer
	entry := newLogger(&buf).WithField("component", "admin")
	ChangeLogger(entry)(context.Background(), "warning", "info")

	m := decode(t, &buf)
	checkChangeLine(t, m, "warning", "info")
	if m["component"] != "admin" {
		t.Errorf("entry fields missing: %v", m)
	}
}

func TestChangeLoggerNilUsesStandardLogger(t *testing.T) {
	std := logrus.StandardLogger()
	out, formatter := std.Out, std.Formatter
	t.Cleanup(func() {
		std.SetOutput(out)
		std.SetFormatter(formatter)
	})
	var buf bytes.Buffer
	std.SetOutput(&buf)
	std.SetFormatter(&logrus.JSONFormatter{DisableTimestamp: true})

	ChangeLogger(nil)(context.Background(), "info", "error")
	checkChangeLine(t, decode(t, &buf), "info", "error")
}
