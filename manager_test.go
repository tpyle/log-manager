package logmanager

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// fakeController is a LevelController that accepts a fixed set of names and
// can be told to fail.
type fakeController struct {
	mu    sync.Mutex
	level string
	err   error // returned by SetLevel instead of applying, if non-nil
	calls []string
}

func (f *fakeController) Level() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.level
}

func (f *fakeController) SetLevel(name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, name)
	if f.err != nil {
		return f.err
	}
	switch name {
	case "debug", "info", "warn", "error":
		f.level = name
		return nil
	}
	return fmt.Errorf("%w: %q", ErrUnknownLevel, name)
}

type change struct{ from, to string }

// recorder returns an option recording every change and the recorded slice.
func recorder() (Option, *[]change) {
	var mu sync.Mutex
	var got []change
	return WithOnChange(func(_ context.Context, from, to string) {
		mu.Lock()
		defer mu.Unlock()
		got = append(got, change{from, to})
	}), &got
}

func do(t *testing.T, h http.Handler, method, path, contentType, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func decodeLevel(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	var m LevelMessage
	if err := json.Unmarshal(w.Body.Bytes(), &m); err != nil {
		t.Fatalf("bad JSON %q: %v", w.Body.String(), err)
	}
	return m.Level
}

func TestNewPanicsOnNil(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("New(nil) did not panic")
		}
	}()
	New(nil)
}

func TestGet(t *testing.T) {
	lm := New(&fakeController{level: "warn"})
	w := do(t, lm, http.MethodGet, "/config", "", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %q", w.Code, w.Body.String())
	}
	if got := decodeLevel(t, w); got != "warn" {
		t.Errorf("level = %q, want warn", got)
	}
}

func TestHeadIsServed(t *testing.T) {
	lm := New(&fakeController{level: "info"})
	if w := do(t, lm, http.MethodHead, "/config", "", ""); w.Code != http.StatusOK {
		t.Errorf("HEAD status = %d", w.Code)
	}
}

func TestSetSuccess(t *testing.T) {
	tests := []struct {
		name, contentType, body, want string
	}{
		{"plain", "application/json", `{"level":"debug"}`, "debug"},
		{"charset param", "application/json; charset=utf-8", `{"level":"error"}`, "error"},
		{"upper-case media type", "Application/JSON", `{"level":"error"}`, "error"},
		{"surrounding whitespace", "application/json", `{"level":"  warn\n"}`, "warn"},
		{"unknown fields ignored", "application/json", `{"level":"debug","reportCaller":true}`, "debug"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fc := &fakeController{level: "info"}
			w := do(t, New(fc), http.MethodPost, "/config", tt.contentType, tt.body)
			if w.Code != http.StatusOK {
				t.Fatalf("status = %d, body %q", w.Code, w.Body.String())
			}
			if got := decodeLevel(t, w); got != tt.want {
				t.Errorf("response level = %q, want %q", got, tt.want)
			}
			if fc.Level() != tt.want {
				t.Errorf("controller level = %q, want %q", fc.Level(), tt.want)
			}
		})
	}
}

// The response reports the controller's resulting level, not the request text.
func TestSetRespondsWithControllerLevel(t *testing.T) {
	c := NewSlogController(new(slog.LevelVar))
	w := do(t, New(c), http.MethodPost, "/config", "application/json", `{"level":"WARNING"}`)
	if got := decodeLevel(t, w); got != "warn" {
		t.Errorf("level = %q, want warn", got)
	}
}

func TestSetErrors(t *testing.T) {
	tests := []struct {
		name, contentType, body string
		ctrlErr                 error
		wantStatus              int
		wantBody                string
		wantSetCalled           bool
	}{
		{"missing content type", "", `{"level":"debug"}`, nil, http.StatusUnsupportedMediaType, "Content-Type must be application/json", false},
		{"wrong content type", "text/plain", `{"level":"debug"}`, nil, http.StatusUnsupportedMediaType, "Content-Type must be application/json", false},
		{"malformed content type", "application/json; =", `{"level":"debug"}`, nil, http.StatusUnsupportedMediaType, "Content-Type must be application/json", false},
		{"json suffix type", "application/problem+json", `{"level":"debug"}`, nil, http.StatusUnsupportedMediaType, "Content-Type must be application/json", false},
		{"empty body", "application/json", ``, nil, http.StatusBadRequest, "invalid JSON body", false},
		{"bad json", "application/json", `{"level":`, nil, http.StatusBadRequest, "invalid JSON body", false},
		{"wrong type", "application/json", `{"level":3}`, nil, http.StatusBadRequest, "invalid JSON body", false},
		{"missing level", "application/json", `{}`, nil, http.StatusBadRequest, "level is required", false},
		{"blank level", "application/json", `{"level":"   "}`, nil, http.StatusBadRequest, "level is required", false},
		{"unknown level", "application/json", `{"level":"verbose"}`, nil, http.StatusBadRequest, "unknown log level", true},
		{"controller failure", "application/json", `{"level":"debug"}`, errors.New("boom"), http.StatusInternalServerError, "failed to set log level", true},
		{"too large", "application/json", `{"level":"` + strings.Repeat("a", MaxRequestBytes) + `"}`, nil, http.StatusRequestEntityTooLarge, "request body too large", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fc := &fakeController{level: "info", err: tt.ctrlErr}
			opt, changes := recorder()
			w := do(t, New(fc, opt), http.MethodPost, "/config", tt.contentType, tt.body)
			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if got := strings.TrimSpace(w.Body.String()); got != tt.wantBody {
				t.Errorf("body = %q, want %q", got, tt.wantBody)
			}
			if called := len(fc.calls) > 0; called != tt.wantSetCalled {
				t.Errorf("SetLevel called = %v, want %v", called, tt.wantSetCalled)
			}
			if fc.Level() != "info" {
				t.Errorf("level changed to %q", fc.Level())
			}
			if len(*changes) != 0 {
				t.Errorf("OnChange called: %v", *changes)
			}
		})
	}
}

func TestRouting(t *testing.T) {
	lm := New(&fakeController{level: "info"})
	tests := []struct {
		method, path string
		want         int
	}{
		{http.MethodPut, "/config", http.StatusMethodNotAllowed},
		{http.MethodDelete, "/config", http.StatusMethodNotAllowed},
		{http.MethodGet, "/", http.StatusNotFound},
		{http.MethodGet, "/config/extra", http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			if w := do(t, lm, tt.method, tt.path, "", ""); w.Code != tt.want {
				t.Errorf("status = %d, want %d", w.Code, tt.want)
			}
		})
	}
}

func TestMountedWithStripPrefix(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle("/admin/log/", http.StripPrefix("/admin/log", New(&fakeController{level: "error"})))
	w := do(t, mux, http.MethodGet, "/admin/log/config", "", "")
	if w.Code != http.StatusOK || decodeLevel(t, w) != "error" {
		t.Errorf("status = %d, body %q", w.Code, w.Body.String())
	}
}

func TestHandlersOnCustomRoutes(t *testing.T) {
	fc := &fakeController{level: "info"}
	lm := New(fc)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /level", lm.GetLevelHandler)
	mux.HandleFunc("PUT /level", lm.SetLevelHandler)

	if w := do(t, mux, http.MethodPut, "/level", "application/json", `{"level":"debug"}`); w.Code != http.StatusOK {
		t.Fatalf("PUT status = %d", w.Code)
	}
	if got := decodeLevel(t, do(t, mux, http.MethodGet, "/level", "", "")); got != "debug" {
		t.Errorf("level = %q", got)
	}
}

func TestOnChange(t *testing.T) {
	type ctxKey struct{}
	fc := &fakeController{level: "info"}
	var gotCtxValue any
	var got []change
	lm := New(fc, WithOnChange(func(ctx context.Context, from, to string) {
		gotCtxValue = ctx.Value(ctxKey{})
		got = append(got, change{from, to})
	}))

	post := func(level string) {
		req := httptest.NewRequest(http.MethodPost, "/config", strings.NewReader(`{"level":"`+level+`"}`))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(context.WithValue(req.Context(), ctxKey{}, "marker"))
		lm.ServeHTTP(httptest.NewRecorder(), req)
	}
	post("debug")
	post("debug") // unchanged: no callback
	post("error")

	want := []change{{"info", "debug"}, {"debug", "error"}}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("changes = %v, want %v", got, want)
	}
	if gotCtxValue != "marker" {
		t.Errorf("callback did not receive request context")
	}
}

func TestWithOnChangeNilDisables(t *testing.T) {
	opt, changes := recorder()
	lm := New(&fakeController{level: "info"}, opt, WithOnChange(nil))
	w := do(t, lm, http.MethodPost, "/config", "application/json", `{"level":"debug"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	if len(*changes) != 0 {
		t.Errorf("callback still called: %v", *changes)
	}
}

// failingWriter is an http.ResponseWriter whose Write always fails.
type failingWriter struct{ header http.Header }

func (f *failingWriter) Header() http.Header       { return f.header }
func (f *failingWriter) WriteHeader(int)           {}
func (f *failingWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }

func TestWriteFailureDoesNotPanic(t *testing.T) {
	lm := New(&fakeController{level: "info"})
	lm.ServeHTTP(&failingWriter{header: http.Header{}}, httptest.NewRequest(http.MethodGet, "/config", nil))
}

// Run with -race. Each callback must see a consistent from/to pair: with
// changes serialized, every recorded "to" must be the next recorded "from".
func TestConcurrentChangesAreSerialized(t *testing.T) {
	fc := &fakeController{level: "info"}
	var mu sync.Mutex
	var got []change
	lm := New(fc, WithOnChange(func(_ context.Context, from, to string) {
		mu.Lock()
		defer mu.Unlock()
		got = append(got, change{from, to})
	}))

	levels := []string{"debug", "info", "warn", "error"}
	var wg sync.WaitGroup
	for i := range 200 {
		wg.Go(func() {
			if i%2 == 0 {
				do(t, lm, http.MethodGet, "/config", "", "")
				return
			}
			do(t, lm, http.MethodPost, "/config", "application/json", `{"level":"`+levels[i%len(levels)]+`"}`)
		})
	}
	wg.Wait()

	for i, c := range got {
		if c.from == c.to {
			t.Errorf("change %d is a no-op: %v", i, c)
		}
		if i > 0 && got[i-1].to != c.from {
			t.Errorf("change %d from %q does not follow previous to %q", i, c.from, got[i-1].to)
		}
	}
}

// The core package must not import anything outside the standard library, so
// that programs using only slog do not pull in zerolog or logrus.
func TestCoreImportsOnlyStdlib(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range f.Imports {
			path, _ := strconv.Unquote(imp.Path.Value)
			if first, _, _ := strings.Cut(path, "/"); strings.Contains(first, ".") {
				t.Errorf("%s imports non-stdlib package %q", name, path)
			}
		}
	}
}

// The root module must have no requirements, so that depending on it never
// adds zerolog, logrus or anything else to a program's module graph. Adapters
// with third-party dependencies are separate modules.
func TestRootModuleHasNoRequirements(t *testing.T) {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		t.Fatal(err)
	}
	for i, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "require") {
			t.Errorf("go.mod:%d: %s", i+1, line)
		}
	}
	if _, err := os.Stat("go.sum"); err == nil {
		t.Error("go.sum exists, so the root module has dependencies")
	}
}
