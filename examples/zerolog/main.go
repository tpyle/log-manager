// Command zerolog runs an HTTP server whose zerolog global level can be
// changed at runtime.
//
//	go run ./examples/zerolog
//	curl localhost:8080/hello
//	curl localhost:8081/log/config
//	curl -X POST localhost:8081/log/config -H 'Content-Type: application/json' -d '{"level":"debug"}'
//	curl localhost:8080/hello
package main

import (
	"net/http"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/tpyle/log-manager/zerologmgr/v3"
)

func main() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	// Leave the logger at its default (trace) level so the global level is
	// the only filter.
	log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()

	app := http.NewServeMux()
	app.HandleFunc("GET /hello", func(w http.ResponseWriter, r *http.Request) {
		log.Debug().Str("path", r.URL.Path).Msg("debug detail")
		log.Info().Msg("hello")
		_, _ = w.Write([]byte("hello\n"))
	})

	// Serve the log manager on a separate, loopback-only port: it has no
	// authentication of its own.
	admin := http.NewServeMux()
	admin.Handle("/log/", http.StripPrefix("/log", zerologmgr.New()))
	go func() {
		if err := http.ListenAndServe("127.0.0.1:8081", admin); err != nil {
			log.Fatal().Err(err).Msg("admin server failed")
		}
	}()

	log.Info().Str("app", ":8080").Str("admin", "127.0.0.1:8081").Msg("listening")
	if err := http.ListenAndServe(":8080", app); err != nil {
		log.Fatal().Err(err).Msg("server failed")
	}
}
