package logmanager

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog"
	"github.com/spf13/viper"
)

type LogManager struct {
	http.ServeMux

	viperInstance *viper.Viper
}

func NewLogManager(v *viper.Viper) *LogManager {
	lm := &LogManager{
		viperInstance: v,
	}
	lm.HandleFunc("GET /config", lm.GetLogLevelHandler)
	lm.HandleFunc("POST /config", lm.SetLogLevelHandler)

	return lm
}

func (lm *LogManager) GetLogLevelHandler(w http.ResponseWriter, r *http.Request) {
	logMessage := LogMessage{
		Level: zerolog.GlobalLevel().String(),
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(logMessage); err != nil {
		zerolog.Ctx(r.Context()).Warn().Err(err).Msg("error encoding log level response")
	}
}

func (lm *LogManager) SetLogLevelHandler(w http.ResponseWriter, r *http.Request) {
	message := LogMessage{}

	if r.Header.Get("Content-Type") != "application/json" {
		zerolog.Ctx(r.Context()).Debug().Msg("invalid content type for log level request")
		http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
		return
	}

	if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
		zerolog.Ctx(r.Context()).Debug().Err(err).Msg("error unmarshalling log level request")
		http.Error(w, "invalid log level", http.StatusBadRequest)
		return
	}

	level, err := parseLevel(message.Level)
	if err != nil {
		zerolog.Ctx(r.Context()).Debug().Err(err).Msg("unknown log level")
		http.Error(w, "unknown log level", http.StatusBadRequest)
		return
	}

	zerolog.Ctx(r.Context()).Info().Str("new_level", level.String()).Msg("log level updated")
	zerolog.SetGlobalLevel(level)

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(message); err != nil {
		zerolog.Ctx(r.Context()).Warn().Err(err).Msg("error encoding log level response")
	}
}
