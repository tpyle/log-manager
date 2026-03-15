package logmanager

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type logConfig struct {
	Level        string `json:"level,omitempty"`
	ReportCaller *bool  `json:"reportCaller,omitempty"`
}

var globalLogger zerolog.Logger

func init() {
	// Initialize the global logger with console output
	globalLogger = zerolog.New(os.Stderr).With().Timestamp().Logger()
	log.Logger = globalLogger
}

// parseLevel converts a string level to zerolog.Level
func parseLevel(levelStr string) (zerolog.Level, error) {
	switch strings.ToLower(levelStr) {
	case "panic":
		return zerolog.PanicLevel, nil
	case "fatal":
		return zerolog.FatalLevel, nil
	case "error":
		return zerolog.ErrorLevel, nil
	case "warn", "warning":
		return zerolog.WarnLevel, nil
	case "info":
		return zerolog.InfoLevel, nil
	case "debug":
		return zerolog.DebugLevel, nil
	case "trace":
		return zerolog.TraceLevel, nil
	default:
		return zerolog.InfoLevel, errors.New("invalid log level")
	}
}

// levelToString converts a zerolog.Level to its string representation
func levelToString(level zerolog.Level) string {
	switch level {
	case zerolog.PanicLevel:
		return "panic"
	case zerolog.FatalLevel:
		return "fatal"
	case zerolog.ErrorLevel:
		return "error"
	case zerolog.WarnLevel:
		return "warn"
	case zerolog.InfoLevel:
		return "info"
	case zerolog.DebugLevel:
		return "debug"
	case zerolog.TraceLevel:
		return "trace"
	default:
		return "info"
	}
}

// getCurrentLevel returns the current global log level as a string
func getCurrentLevel() string {
	return levelToString(zerolog.GlobalLevel())
}

func GetLogLevelHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(getCurrentLevel()))
}

func SetLogLevelHandler(w http.ResponseWriter, r *http.Request) {
	settings := logConfig{}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
		return
	}

	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		log.Warn().Err(err).Msg("error unmarshalling log settings")
		http.Error(w, "invalid log config", http.StatusBadRequest)
		return
	}

	if len(settings.Level) > 0 {
		level, err := parseLevel(settings.Level)
		if err != nil {
			http.Error(w, "unknown log level", http.StatusBadRequest)
			return
		}

		zerolog.SetGlobalLevel(level)
	}

	if settings.ReportCaller != nil {
		if *settings.ReportCaller {
			globalLogger = globalLogger.With().Caller().Logger()
		} else {
			globalLogger = zerolog.New(os.Stderr).With().Timestamp().Logger()
		}
		log.Logger = globalLogger
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(getCurrentLevel()))
}

func HandleLogCall(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		SetLogLevelHandler(w, r)
	case http.MethodGet:
		GetLogLevelHandler(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
