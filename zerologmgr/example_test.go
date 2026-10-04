package zerologmgr_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"

	"github.com/rs/zerolog"

	logmanager "github.com/tpyle/log-manager/v3"
	"github.com/tpyle/log-manager/zerologmgr/v3"
)

func ExampleNew() {
	orig := zerolog.GlobalLevel()
	defer zerolog.SetGlobalLevel(orig)
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	logger := zerolog.New(os.Stdout)
	lm := zerologmgr.New(logmanager.WithOnChange(zerologmgr.ChangeLogger(&logger)))

	logger.Debug().Msg("not shown")

	req := httptest.NewRequest(http.MethodPost, "/config", strings.NewReader(`{"level":"debug"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	lm.ServeHTTP(w, req)
	fmt.Print(w.Body.String())

	logger.Debug().Msg("now shown")
	// Output:
	// {"level":"info","oldLevel":"info","newLevel":"debug","message":"log level changed"}
	// {"level":"debug"}
	// {"level":"debug","message":"now shown"}
}
