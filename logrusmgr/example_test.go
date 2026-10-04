package logrusmgr_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/tpyle/log-manager/v3/logrusmgr"
)

func ExampleNew() {
	logger := logrus.New()
	logger.SetOutput(os.Stdout)
	logger.SetFormatter(&logrus.TextFormatter{DisableTimestamp: true, DisableColors: true})

	lm := logrusmgr.New(logger)

	logger.Debug("not shown")

	req := httptest.NewRequest(http.MethodPost, "/config", strings.NewReader(`{"level":"debug"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	lm.ServeHTTP(w, req)
	fmt.Print(w.Body.String())

	logger.Debug("now shown")
	// Output:
	// level=info msg="log level changed" newLevel=debug oldLevel=info
	// {"level":"debug"}
	// level=debug msg="now shown"
}
