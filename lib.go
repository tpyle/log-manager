package logmanager

import (
	"encoding/json"
	"net/http"

	"github.com/sirupsen/logrus"
)

type logConfig struct {
	Level        string `json:"level,omitempty"`
	ReportCaller *bool  `json:"reportCaller,omitempty"`
}

func HandleLogCall(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		settings := logConfig{}
		if r.Header.Get("Content-Type") != "application/json" {
			http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
			return
		}

		if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
			logrus.Warningf("error unmarshalling log settings: %v", err)
			http.Error(w, "invalid log config", http.StatusBadRequest)
			return
		}

		if len(settings.Level) > 0 {
			level, error := logrus.ParseLevel(settings.Level)
			if error != nil {
				http.Error(w, "unknown log level", http.StatusBadRequest)
				return
			}

			logrus.SetLevel(level)
		}

		if settings.ReportCaller != nil {
			logrus.SetReportCaller(*settings.ReportCaller)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(logrus.GetLevel().String()))
	case http.MethodGet:
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(logrus.GetLevel().String()))
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
