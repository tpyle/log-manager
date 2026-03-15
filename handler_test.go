package logmanager

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/spf13/viper"
)

// failingResponseWriter is an http.ResponseWriter whose Write calls always fail,
// used to exercise json-encoder error branches.
type failingResponseWriter struct {
	header http.Header
	code   int
}

func newFailingResponseWriter() *failingResponseWriter {
	return &failingResponseWriter{header: make(http.Header)}
}

func (fw *failingResponseWriter) Header() http.Header      { return fw.header }
func (fw *failingResponseWriter) WriteHeader(code int)     { fw.code = code }
func (fw *failingResponseWriter) Write([]byte) (int, error) { return 0, fmt.Errorf("write error") }

// newTestLogManager is a helper that creates a LogManager backed by a fresh viper instance.
func newTestLogManager() *LogManager {
	return NewLogManager(viper.New())
}

func TestNewLogManager_NotNil(t *testing.T) {
	v := viper.New()
	lm := NewLogManager(v)
	if lm == nil {
		t.Fatal("NewLogManager returned nil")
	}
	if lm.viperInstance != v {
		t.Error("viperInstance not stored correctly")
	}
}

func TestNewLogManager_RoutesRegistered(t *testing.T) {
	originalLevel := zerolog.GlobalLevel()
	defer zerolog.SetGlobalLevel(originalLevel)

	lm := newTestLogManager()

	// GET /config should be routed to GetLogLevelHandler
	req := httptest.NewRequest(http.MethodGet, "/config", nil)
	w := httptest.NewRecorder()
	lm.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("GET /config: expected 200, got %d", w.Code)
	}

	// POST /config with valid body should be routed to SetLogLevelHandler
	body, _ := json.Marshal(LogMessage{Level: "info"})
	req = httptest.NewRequest(http.MethodPost, "/config", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	lm.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("POST /config: expected 200, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// GetLogLevelHandler
// ---------------------------------------------------------------------------

func TestGetLogLevelHandler_ReturnsCurrentLevel(t *testing.T) {
	originalLevel := zerolog.GlobalLevel()
	defer zerolog.SetGlobalLevel(originalLevel)

	zerolog.SetGlobalLevel(zerolog.WarnLevel)

	lm := newTestLogManager()
	req := httptest.NewRequest(http.MethodGet, "/config", nil)
	w := httptest.NewRecorder()
	lm.GetLogLevelHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}

	var msg LogMessage
	if err := json.NewDecoder(w.Body).Decode(&msg); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if msg.Level != "warn" {
		t.Errorf("expected level %q, got %q", "warn", msg.Level)
	}
}

func TestGetLogLevelHandler_EncoderError(t *testing.T) {
	// Uses a failing writer to hit the json.Encoder error branch.
	lm := newTestLogManager()
	req := httptest.NewRequest(http.MethodGet, "/config", nil)
	fw := newFailingResponseWriter()
	// Should not panic; the error is logged internally and execution continues.
	lm.GetLogLevelHandler(fw, req)
}

// ---------------------------------------------------------------------------
// SetLogLevelHandler
// ---------------------------------------------------------------------------

func TestSetLogLevelHandler_InvalidContentType(t *testing.T) {
	lm := newTestLogManager()
	body, _ := json.Marshal(LogMessage{Level: "debug"})
	req := httptest.NewRequest(http.MethodPost, "/config", bytes.NewReader(body))
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	lm.SetLogLevelHandler(w, req)

	if w.Code != http.StatusUnsupportedMediaType {
		t.Errorf("expected 415, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Content-Type must be application/json") {
		t.Errorf("unexpected body: %q", w.Body.String())
	}
}

func TestSetLogLevelHandler_MissingContentType(t *testing.T) {
	lm := newTestLogManager()
	body, _ := json.Marshal(LogMessage{Level: "debug"})
	req := httptest.NewRequest(http.MethodPost, "/config", bytes.NewReader(body))
	w := httptest.NewRecorder()

	lm.SetLogLevelHandler(w, req)

	if w.Code != http.StatusUnsupportedMediaType {
		t.Errorf("expected 415, got %d", w.Code)
	}
}

func TestSetLogLevelHandler_InvalidJSON(t *testing.T) {
	lm := newTestLogManager()
	req := httptest.NewRequest(http.MethodPost, "/config", strings.NewReader("{invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	lm.SetLogLevelHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "invalid log level") {
		t.Errorf("unexpected body: %q", w.Body.String())
	}
}

func TestSetLogLevelHandler_UnknownLogLevel(t *testing.T) {
	lm := newTestLogManager()
	body, _ := json.Marshal(LogMessage{Level: "notalevel"})
	req := httptest.NewRequest(http.MethodPost, "/config", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	lm.SetLogLevelHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "unknown log level") {
		t.Errorf("unexpected body: %q", w.Body.String())
	}
}

func TestSetLogLevelHandler_ValidLevels(t *testing.T) {
	originalLevel := zerolog.GlobalLevel()
	defer zerolog.SetGlobalLevel(originalLevel)

	cases := []struct {
		input    string
		expected zerolog.Level
	}{
		{"panic", zerolog.PanicLevel},
		{"fatal", zerolog.FatalLevel},
		{"error", zerolog.ErrorLevel},
		{"warn", zerolog.WarnLevel},
		{"info", zerolog.InfoLevel},
		{"debug", zerolog.DebugLevel},
		{"trace", zerolog.TraceLevel},
	}

	lm := newTestLogManager()

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			body, _ := json.Marshal(LogMessage{Level: tc.input})
			req := httptest.NewRequest(http.MethodPost, "/config", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			lm.SetLogLevelHandler(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("level %q: expected 200, got %d", tc.input, w.Code)
			}
			if zerolog.GlobalLevel() != tc.expected {
				t.Errorf("level %q: expected global level %v, got %v", tc.input, tc.expected, zerolog.GlobalLevel())
			}

			var resp LogMessage
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("level %q: failed to decode response: %v", tc.input, err)
			}
			if resp.Level != tc.input {
				t.Errorf("level %q: expected response level %q, got %q", tc.input, tc.input, resp.Level)
			}
		})
	}
}

func TestSetLogLevelHandler_EncoderError(t *testing.T) {
	// Uses a failing writer to hit the json.Encoder error branch after a valid request.
	originalLevel := zerolog.GlobalLevel()
	defer zerolog.SetGlobalLevel(originalLevel)

	lm := newTestLogManager()
	body, _ := json.Marshal(LogMessage{Level: "info"})
	req := httptest.NewRequest(http.MethodPost, "/config", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	fw := newFailingResponseWriter()

	// Should not panic; the error is logged internally and execution continues.
	lm.SetLogLevelHandler(fw, req)
}
