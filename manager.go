// Package logmanager provides an HTTP handler for reading and changing a
// program's log level at runtime.
//
// The package depends only on the standard library and controls
// [log/slog] levels out of the box (see [NewSlog]). Adapters for third-party
// loggers live in subpackages so their dependencies are only compiled into
// programs that import them:
//
//   - github.com/tpyle/log-manager/v3/zerologmgr for zerolog
//   - github.com/tpyle/log-manager/v3/logrusmgr for logrus
//
// Any other logger can be supported by implementing [LevelController].
//
// The handler performs no authentication. Serve it on an internal
// administration port or wrap it with your own authorization middleware.
package logmanager

import (
	"context"
	"encoding/json"
	"errors"
	"mime"
	"net/http"
	"strings"
	"sync"
)

// ErrUnknownLevel is returned (possibly wrapped) by
// [LevelController.SetLevel] when the level name is not recognized. The
// handler responds to it with 400 Bad Request.
var ErrUnknownLevel = errors.New("unknown log level")

// MaxRequestBytes is the largest request body the handler accepts. Larger
// bodies are rejected with 413 Request Entity Too Large.
const MaxRequestBytes = 64 << 10

// ChangeMessage is the message logged when the level changes. Adapters should
// use it so output is consistent across loggers.
const ChangeMessage = "log level changed"

// Field names used by the bundled adapters when logging a level change.
const (
	FieldOldLevel = "oldLevel"
	FieldNewLevel = "newLevel"
)

// LevelController is the integration point between the handler and a logging
// library.
//
// Level names are the logging library's own; the handler passes them through
// without interpretation. Implementations must be safe for concurrent use.
type LevelController interface {
	// Level returns the name of the current level.
	Level() string
	// SetLevel parses name and makes it the current level. It returns an
	// error wrapping [ErrUnknownLevel] if name is not a valid level, in
	// which case the level must be left unchanged. Any other error is
	// reported to the client as 500 Internal Server Error.
	SetLevel(name string) error
}

// LevelMessage is the JSON body of requests to and responses from the
// handler: {"level":"<name>"}.
type LevelMessage struct {
	Level string `json:"level"`
}

// LevelChangeFunc is called after the level changes. ctx is the request
// context, and from and to are the level names reported by
// [LevelController.Level] before and after the change.
type LevelChangeFunc func(ctx context.Context, from, to string)

// LogManager is an [http.Handler] serving the following routes:
//
//	GET  /config   responds with {"level":"<current>"}
//	POST /config   accepts {"level":"<new>"} and responds with the resulting level
//
// Mount it under a prefix with [http.StripPrefix], or register
// [LogManager.GetLevelHandler] and [LogManager.SetLevelHandler] on routes of
// your choosing. Create one with [New], [NewSlog], or a constructor from an
// adapter subpackage.
type LogManager struct {
	controller LevelController
	onChange   LevelChangeFunc
	mux        *http.ServeMux
	// mu serializes level changes so that each from/to pair passed to
	// onChange describes a single change, and calls arrive in order.
	mu sync.Mutex
}

// Option configures a [LogManager].
type Option func(*LogManager)

// WithOnChange sets the function called after a successful request changes
// the level. It is not called when the requested level equals the current one.
// Passing nil disables the callback, including any default installed by an
// adapter constructor.
//
// Calls are serialized and made in the order the changes were applied, so fn
// should return promptly; other requests to change the level wait for it.
func WithOnChange(fn LevelChangeFunc) Option {
	return func(lm *LogManager) {
		lm.onChange = fn
	}
}

// New returns a LogManager controlling c. It panics if c is nil.
//
// Unlike the adapter constructors, New installs no [LevelChangeFunc]; use
// [WithOnChange] to log or audit changes.
func New(c LevelController, opts ...Option) *LogManager {
	if c == nil {
		panic("logmanager: nil LevelController")
	}
	lm := &LogManager{controller: c}
	for _, opt := range opts {
		opt(lm)
	}
	lm.mux = http.NewServeMux()
	lm.mux.HandleFunc("GET /config", lm.GetLevelHandler)
	lm.mux.HandleFunc("POST /config", lm.SetLevelHandler)
	return lm
}

// ServeHTTP implements [http.Handler].
func (lm *LogManager) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	lm.mux.ServeHTTP(w, r)
}

// GetLevelHandler responds with the current level as {"level":"<name>"}. It
// does not check the request method.
func (lm *LogManager) GetLevelHandler(w http.ResponseWriter, _ *http.Request) {
	writeLevel(w, lm.controller.Level())
}

// SetLevelHandler applies the level in a {"level":"<name>"} request body and
// responds with the resulting level in the same format. It does not check the
// request method.
//
// It responds with 415 if the Content-Type is not application/json, 413 if
// the body exceeds [MaxRequestBytes], 400 if the body is not valid JSON or the
// level is missing or unknown, and 500 if the controller fails otherwise.
// Error bodies are plain text.
func (lm *LogManager) SetLevelHandler(w http.ResponseWriter, r *http.Request) {
	if !isJSON(r.Header.Get("Content-Type")) {
		http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
		return
	}

	var msg LevelMessage
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, MaxRequestBytes)).Decode(&msg); err != nil {
		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(msg.Level)
	if name == "" {
		http.Error(w, "level is required", http.StatusBadRequest)
		return
	}

	to, err := lm.setLevel(r.Context(), name)
	switch {
	case errors.Is(err, ErrUnknownLevel):
		http.Error(w, "unknown log level", http.StatusBadRequest)
		return
	case err != nil:
		http.Error(w, "failed to set log level", http.StatusInternalServerError)
		return
	}
	writeLevel(w, to)
}

// setLevel applies name and returns the resulting level. The change callback
// runs under the lock so that callbacks observe changes in order.
func (lm *LogManager) setLevel(ctx context.Context, name string) (string, error) {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	from := lm.controller.Level()
	if err := lm.controller.SetLevel(name); err != nil {
		return "", err
	}
	to := lm.controller.Level()
	if lm.onChange != nil && from != to {
		lm.onChange(ctx, from, to)
	}
	return to, nil
}

// isJSON reports whether contentType is application/json, ignoring
// parameters such as charset.
func isJSON(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	return err == nil && mediaType == "application/json"
}

func writeLevel(w http.ResponseWriter, level string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	// The status is already sent, so an encoding error (a failed write to
	// the client) cannot be reported.
	_ = json.NewEncoder(w).Encode(LevelMessage{Level: level})
}
