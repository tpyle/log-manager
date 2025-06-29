package logmanager

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestHandleLogCall_POST_ValidLogLevel(t *testing.T) {
	// Save original log level to restore after test
	originalLevel := logrus.GetLevel()
	defer logrus.SetLevel(originalLevel)

	config := logConfig{Level: "debug"}
	body, _ := json.Marshal(config)

	req := httptest.NewRequest(http.MethodPost, "/log", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	HandleLogCall(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	responseBody, _ := io.ReadAll(resp.Body)
	if string(responseBody) != "debug" {
		t.Errorf("Expected response body 'debug', got '%s'", string(responseBody))
	}

	if logrus.GetLevel() != logrus.DebugLevel {
		t.Errorf("Expected log level to be debug, got %s", logrus.GetLevel())
	}
}

func TestHandleLogCall_POST_ValidReportCaller(t *testing.T) {
	// Save original report caller setting to restore after test
	originalReportCaller := logrus.StandardLogger().ReportCaller
	defer logrus.SetReportCaller(originalReportCaller)

	reportCaller := true
	config := logConfig{ReportCaller: &reportCaller}
	body, _ := json.Marshal(config)

	req := httptest.NewRequest(http.MethodPost, "/log", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	HandleLogCall(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	if !logrus.StandardLogger().ReportCaller {
		t.Error("Expected ReportCaller to be true")
	}
}

func TestHandleLogCall_POST_ValidLevelAndReportCaller(t *testing.T) {
	// Save original settings to restore after test
	originalLevel := logrus.GetLevel()
	originalReportCaller := logrus.StandardLogger().ReportCaller
	defer func() {
		logrus.SetLevel(originalLevel)
		logrus.SetReportCaller(originalReportCaller)
	}()

	reportCaller := false
	config := logConfig{
		Level:        "error",
		ReportCaller: &reportCaller,
	}
	body, _ := json.Marshal(config)

	req := httptest.NewRequest(http.MethodPost, "/log", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	HandleLogCall(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	responseBody, _ := io.ReadAll(resp.Body)
	if string(responseBody) != "error" {
		t.Errorf("Expected response body 'error', got '%s'", string(responseBody))
	}

	if logrus.GetLevel() != logrus.ErrorLevel {
		t.Errorf("Expected log level to be error, got %s", logrus.GetLevel())
	}

	if logrus.StandardLogger().ReportCaller {
		t.Error("Expected ReportCaller to be false")
	}
}

func TestHandleLogCall_POST_EmptyBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/log", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	HandleLogCall(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Should return current log level since no changes were made
	responseBody, _ := io.ReadAll(resp.Body)
	currentLevel := logrus.GetLevel().String()
	if string(responseBody) != currentLevel {
		t.Errorf("Expected response body '%s', got '%s'", currentLevel, string(responseBody))
	}
}

func TestHandleLogCall_POST_InvalidContentType(t *testing.T) {
	config := logConfig{Level: "debug"}
	body, _ := json.Marshal(config)

	req := httptest.NewRequest(http.MethodPost, "/log", bytes.NewReader(body))
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	HandleLogCall(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnsupportedMediaType {
		t.Errorf("Expected status %d, got %d", http.StatusUnsupportedMediaType, resp.StatusCode)
	}

	responseBody, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(responseBody), "Content-Type must be application/json") {
		t.Errorf("Expected error message about Content-Type, got '%s'", string(responseBody))
	}
}

func TestHandleLogCall_POST_MissingContentType(t *testing.T) {
	config := logConfig{Level: "debug"}
	body, _ := json.Marshal(config)

	req := httptest.NewRequest(http.MethodPost, "/log", bytes.NewReader(body))
	// Don't set Content-Type header
	w := httptest.NewRecorder()

	HandleLogCall(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnsupportedMediaType {
		t.Errorf("Expected status %d, got %d", http.StatusUnsupportedMediaType, resp.StatusCode)
	}
}

func TestHandleLogCall_POST_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/log", strings.NewReader("{invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	HandleLogCall(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
	}

	responseBody, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(responseBody), "invalid log config") {
		t.Errorf("Expected error message about invalid log config, got '%s'", string(responseBody))
	}
}

func TestHandleLogCall_POST_InvalidLogLevel(t *testing.T) {
	config := logConfig{Level: "invalid_level"}
	body, _ := json.Marshal(config)

	req := httptest.NewRequest(http.MethodPost, "/log", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	HandleLogCall(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
	}

	responseBody, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(responseBody), "unknown log level") {
		t.Errorf("Expected error message about unknown log level, got '%s'", string(responseBody))
	}
}

func TestHandleLogCall_POST_AllValidLogLevels(t *testing.T) {
	// Save original log level to restore after test
	originalLevel := logrus.GetLevel()
	defer logrus.SetLevel(originalLevel)

	validLevels := []string{"panic", "fatal", "error", "warn", "warning", "info", "debug", "trace"}

	for _, level := range validLevels {
		t.Run("level_"+level, func(t *testing.T) {
			config := logConfig{Level: level}
			body, _ := json.Marshal(config)

			req := httptest.NewRequest(http.MethodPost, "/log", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			HandleLogCall(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status %d for level %s, got %d", http.StatusOK, level, resp.StatusCode)
			}

			responseBody, _ := io.ReadAll(resp.Body)
			parsedLevel, _ := logrus.ParseLevel(level)
			expectedResponse := parsedLevel.String()
			if string(responseBody) != expectedResponse {
				t.Errorf("Expected response body '%s' for level %s, got '%s'", expectedResponse, level, string(responseBody))
			}
		})
	}
}

func TestHandleLogCall_GET(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/log", nil)
	w := httptest.NewRecorder()

	HandleLogCall(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	responseBody, _ := io.ReadAll(resp.Body)
	currentLevel := logrus.GetLevel().String()
	if string(responseBody) != currentLevel {
		t.Errorf("Expected response body '%s', got '%s'", currentLevel, string(responseBody))
	}
}

func TestHandleLogCall_UnsupportedMethod(t *testing.T) {
	methods := []string{http.MethodPut, http.MethodDelete, http.MethodPatch, "CUSTOM"}

	for _, method := range methods {
		t.Run("method_"+method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/log", nil)
			w := httptest.NewRecorder()

			HandleLogCall(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusMethodNotAllowed {
				t.Errorf("Expected status %d for method %s, got %d", http.StatusMethodNotAllowed, method, resp.StatusCode)
			}

			responseBody, _ := io.ReadAll(resp.Body)
			if !strings.Contains(string(responseBody), "Method not allowed") {
				t.Errorf("Expected error message about method not allowed for %s, got '%s'", method, string(responseBody))
			}
		})
	}
}

func TestHandleLogCall_POST_ReaderError(t *testing.T) {
	// Create a reader that will fail after reading some bytes
	failingReader := &failingReader{
		data:      []byte("{\"level\":"),
		failAfter: 5,
	}

	req := httptest.NewRequest(http.MethodPost, "/log", failingReader)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	HandleLogCall(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
	}

	responseBody, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(responseBody), "invalid log config") {
		t.Errorf("Expected error message about invalid log config, got '%s'", string(responseBody))
	}
}

// failingReader is a helper type that simulates a reader that fails after reading a certain number of bytes
type failingReader struct {
	data      []byte
	pos       int
	failAfter int
}

func (f *failingReader) Read(p []byte) (n int, err error) {
	if f.pos >= f.failAfter {
		return 0, io.ErrUnexpectedEOF
	}

	remaining := len(f.data) - f.pos
	if remaining == 0 {
		return 0, io.EOF
	}

	n = len(p)
	if n > remaining {
		n = remaining
	}

	if f.pos+n > f.failAfter {
		n = f.failAfter - f.pos
	}

	copy(p, f.data[f.pos:f.pos+n])
	f.pos += n

	if f.pos >= f.failAfter {
		return n, io.ErrUnexpectedEOF
	}

	return n, nil
}

func TestHandleLogCall_POST_NilReportCaller(t *testing.T) {
	// Test with nil ReportCaller (should not change the setting)
	originalReportCaller := logrus.StandardLogger().ReportCaller
	defer logrus.SetReportCaller(originalReportCaller)

	config := logConfig{
		Level:        "info",
		ReportCaller: nil, // explicitly nil
	}
	body, _ := json.Marshal(config)

	req := httptest.NewRequest(http.MethodPost, "/log", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	HandleLogCall(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// ReportCaller should remain unchanged
	if logrus.StandardLogger().ReportCaller != originalReportCaller {
		t.Error("Expected ReportCaller to remain unchanged when nil")
	}
}

func TestHandleLogCall_POST_EmptyLevel(t *testing.T) {
	// Test with empty string level (should not change the level)
	originalLevel := logrus.GetLevel()
	defer logrus.SetLevel(originalLevel)

	config := logConfig{
		Level: "", // empty string
	}
	body, _ := json.Marshal(config)

	req := httptest.NewRequest(http.MethodPost, "/log", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	HandleLogCall(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Log level should remain unchanged
	if logrus.GetLevel() != originalLevel {
		t.Errorf("Expected log level to remain unchanged when empty string, got %s", logrus.GetLevel())
	}
}
