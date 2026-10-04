// Command logrus runs an HTTP server whose logrus level can be changed at
// runtime.
//
//	go run ./examples/logrus
//	curl localhost:8080/hello
//	curl localhost:8081/log/config
//	curl -X POST localhost:8081/log/config -H 'Content-Type: application/json' -d '{"level":"debug"}'
//	curl localhost:8080/hello
package main

import (
	"net/http"
	"os"

	"github.com/sirupsen/logrus"

	"github.com/tpyle/log-manager/logrusmgr/v3"
)

func main() {
	logger := logrus.New()
	logger.SetOutput(os.Stdout)
	logger.SetFormatter(&logrus.JSONFormatter{})

	app := http.NewServeMux()
	app.HandleFunc("GET /hello", func(w http.ResponseWriter, r *http.Request) {
		entry := logger.WithContext(r.Context())
		entry.WithField("path", r.URL.Path).Debug("debug detail")
		entry.Info("hello")
		_, _ = w.Write([]byte("hello\n"))
	})

	// Serve the log manager on a separate, loopback-only port: it has no
	// authentication of its own.
	admin := http.NewServeMux()
	admin.Handle("/log/", http.StripPrefix("/log", logrusmgr.New(logger)))
	go func() {
		if err := http.ListenAndServe("127.0.0.1:8081", admin); err != nil {
			logger.WithError(err).Fatal("admin server failed")
		}
	}()

	logger.WithFields(logrus.Fields{"app": ":8080", "admin": "127.0.0.1:8081"}).Info("listening")
	if err := http.ListenAndServe(":8080", app); err != nil {
		logger.WithError(err).Fatal("server failed")
	}
}
