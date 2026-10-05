// Command custom shows a LevelController for a logger the library has no
// adapter for: a stdlib *log.Logger with a hand-rolled verbose flag. It also
// requires a bearer token for the admin endpoint.
//
//	go run ./examples/custom
//	curl localhost:8080/hello
//	curl -X POST localhost:8080/log/config -H 'Authorization: Bearer secret' \
//	    -H 'Content-Type: application/json' -d '{"level":"verbose"}'
//	curl localhost:8080/hello
package main

import (
	"context"
	"crypto/subtle"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	logmanager "github.com/tpyle/log-manager/v3"
)

// verboseFlag is a two-level logger switch: "quiet" or "verbose".
type verboseFlag struct{ on atomic.Bool }

func (f *verboseFlag) Level() string {
	if f.on.Load() {
		return "verbose"
	}
	return "quiet"
}

func (f *verboseFlag) SetLevel(name string) error {
	switch name {
	case "verbose":
		f.on.Store(true)
	case "quiet":
		f.on.Store(false)
	default:
		return fmt.Errorf("%w: %q", logmanager.ErrUnknownLevel, name)
	}
	return nil
}

// requireToken rejects requests without the given bearer token.
func requireToken(token string, next http.Handler) http.Handler {
	want := []byte("Bearer " + token)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), want) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	logger := log.New(os.Stdout, "", log.LstdFlags)
	verbose := &verboseFlag{}

	lm := logmanager.New(verbose, logmanager.WithOnChange(func(_ context.Context, from, to string) {
		logger.Printf("log level changed from %s to %s", from, to)
	}))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /hello", func(w http.ResponseWriter, r *http.Request) {
		if verbose.on.Load() {
			logger.Printf("verbose: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
		}
		logger.Print("hello")
		_, _ = w.Write([]byte("hello\n"))
	})
	mux.Handle("/log/", requireToken("secret", http.StripPrefix("/log", lm)))

	logger.Print("listening on :8080")
	logger.Fatal(http.ListenAndServe(":8080", mux))
}
