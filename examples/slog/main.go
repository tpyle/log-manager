// Command slog runs an HTTP server whose slog level can be changed at runtime.
//
//	go run ./examples/slog
//	curl localhost:8080/hello
//	curl localhost:8081/log/config
//	curl -X POST localhost:8081/log/config -H 'Content-Type: application/json' -d '{"level":"debug"}'
//	curl localhost:8080/hello
package main

import (
	"log/slog"
	"net/http"
	"os"

	logmanager "github.com/tpyle/log-manager/v3"
)

func main() {
	// The same LevelVar must be given to the handler and the manager.
	level := new(slog.LevelVar)
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})))

	app := http.NewServeMux()
	app.HandleFunc("GET /hello", func(w http.ResponseWriter, r *http.Request) {
		slog.DebugContext(r.Context(), "debug detail", "path", r.URL.Path)
		slog.InfoContext(r.Context(), "hello")
		_, _ = w.Write([]byte("hello\n"))
	})

	// Serve the log manager on a separate, loopback-only port: it has no
	// authentication of its own.
	admin := http.NewServeMux()
	admin.Handle("/log/", http.StripPrefix("/log", logmanager.NewSlog(level)))
	go func() {
		if err := http.ListenAndServe("127.0.0.1:8081", admin); err != nil {
			slog.Error("admin server failed", "err", err)
			os.Exit(1)
		}
	}()

	slog.Info("listening", "app", ":8080", "admin", "127.0.0.1:8081")
	if err := http.ListenAndServe(":8080", app); err != nil {
		slog.Error("server failed", "err", err)
		os.Exit(1)
	}
}
