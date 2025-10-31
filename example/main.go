package main

import (
	"fmt"
	"net/http"

	"github.com/rs/zerolog/log"
	logmanager "github.com/tpyle/log-manager/v2"
)

func main() {
	// Test the log manager
	log.Info().Msg("Starting log manager example")

	// Set up the HTTP handler
	http.HandleFunc("/log", logmanager.HandleLogCall)

	fmt.Println("Log manager server starting on :8080")
	fmt.Println("Try these commands:")
	fmt.Println("curl -X GET  http://localhost:8080/log")
	fmt.Println("curl -X POST http://localhost:8080/log -H 'Content-Type: application/json' -d '{\"level\":\"debug\"}'")
	fmt.Println("curl -X POST http://localhost:8080/log -H 'Content-Type: application/json' -d '{\"level\":\"info\",\"reportCaller\":true}'")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal().Err(err).Msg("Server failed to start")
	}
}
