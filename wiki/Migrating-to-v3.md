# Migrating to v3

v3 moves logger support out of the core module so that it no longer requires
zerolog. The core module uses `log/slog`, and zerolog support moved to the
separate `github.com/tpyle/log-manager/zerologmgr/v3` module. The `viper`
dependency was removed.

```bash
go get github.com/tpyle/log-manager/v3 github.com/tpyle/log-manager/zerologmgr/v3
```

## From v2 `LogManager`

```diff
-import logmanager "github.com/tpyle/log-manager/v2"
+import "github.com/tpyle/log-manager/zerologmgr/v3"

-lm := logmanager.NewLogManager(viper.New())
+lm := zerologmgr.New()
```

The routes (`GET /config`, `POST /config`) and JSON format are unchanged.

| v2 | v3 |
|----|----|
| `NewLogManager(*viper.Viper)` | `zerologmgr.New(opts...)` |
| `(*LogManager).GetLogLevelHandler` | `(*LogManager).GetLevelHandler` |
| `(*LogManager).SetLogLevelHandler` | `(*LogManager).SetLevelHandler` |
| `LogMessage` | `LevelMessage` |
| embedded `http.ServeMux` methods (`Handle`, `HandleFunc`, ...) | removed. `LogManager` is only an `http.Handler`. Register extra routes on your own mux. |

### Behavior changes

- The `POST` response contains the level now in effect, not the request
  text. Sending `"WARNING"` returns `{"level":"warn"}` instead of
  `{"level":"WARNING"}`.
- `Content-Type: application/json; charset=utf-8` is now accepted. v2
  required the exact string `application/json`.
- An empty or missing `level` returns `level is required` instead of
  `unknown log level`. Both are still 400.
- Bodies over 64 KiB are rejected with 413.
- `disabled` is now an accepted zerolog level.
- Changes are logged at info level after they're applied. v2 logged them
  before. If the request context has no zerolog logger, v3 logs to the global
  `log.Logger`, where v2 logged nothing. Failed requests are no longer logged
  at debug level.
- The minimum Go version is now 1.26.

## From the v2 package-level handlers

`HandleLogCall`, `GetLogLevelHandler` and `SetLogLevelHandler` were removed.
They returned plain-text responses on a single route:

```diff
-http.HandleFunc("/api/log", logmanager.HandleLogCall)
+http.Handle("/api/log/", http.StripPrefix("/api/log", zerologmgr.New()))
```

Clients must now call `/api/log/config`. Requests are JSON as before, but
responses are now JSON (`{"level":"debug"}`) instead of plain text.

### `reportCaller` was removed

The v2 legacy handler accepted `"reportCaller": true`. To do this, it replaced
the global `log.Logger` with a new logger writing to stderr, which discarded
your logger's output, format and fields. v3 doesn't support it, and the field
is ignored if sent. Configure caller reporting when you build your logger
(`zerolog.New(w).With().Caller().Logger()`).

### No more global side effects on import

v2 overwrote zerolog's global `log.Logger` in `init()`, so importing the
package replaced your configured logger. v3 doesn't change any global state
until a level change request arrives.

## Switching to slog

```go
level := new(slog.LevelVar)
slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})))
lm := logmanager.NewSlog(level)
```

See [Using slog](Slog.md).
