package main

import (
	"net/http"

	logmanager "github.com/tpyle/log-manager/v2"
)

func main() {
	logManagerHandler := logmanager.NewLogManager(nil)

	http.Handle("/logmanager/", http.StripPrefix("/logmanager", logManagerHandler))

	http.ListenAndServe(":8080", nil)
}
