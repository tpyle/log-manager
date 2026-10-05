# Getting Started

## Install

```bash
go get github.com/tpyle/log-manager/v3
```

For zerolog or logrus, also add the adapter. Each adapter is its own Go
module, so these loggers never enter your dependency graph unless you use them:

```bash
go get github.com/tpyle/log-manager/zerologmgr/v3
go get github.com/tpyle/log-manager/logrusmgr/v3
```

The core and both adapters share the same version number, so use matching
versions. Go 1.26 or newer is required.

## Add the handler

A `*logmanager.LogManager` is an `http.Handler` that serves `/config`. Mount
it under a prefix with `http.StripPrefix`:

```go
package main

import (
    "log/slog"
    "net/http"
    "os"

    logmanager "github.com/tpyle/log-manager/v3"
)

func main() {
    level := new(slog.LevelVar) // starts at info
    slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})))

    admin := http.NewServeMux()
    admin.Handle("/log/", http.StripPrefix("/log", logmanager.NewSlog(level)))
    go http.ListenAndServe("127.0.0.1:8081", admin)

    // ... your application server ...
}
```

The admin server listens only on loopback because the handler has no
authentication of its own. See [Security](Security.md).

## Change the level

```bash
curl localhost:8081/log/config
# {"level":"info"}

curl -X POST localhost:8081/log/config \
  -H 'Content-Type: application/json' \
  -d '{"level":"debug"}'
# {"level":"debug"}
```

The program logs the change:

```json
{"level":"INFO","msg":"log level changed","oldLevel":"info","newLevel":"debug"}
```

## Using your own routes

If you don't want the `/config` route, register the handler methods directly.
They don't check the request method, so choose it in the pattern:

```go
lm := logmanager.NewSlog(level)
mux.HandleFunc("GET /admin/loglevel", lm.GetLevelHandler)
mux.HandleFunc("PUT /admin/loglevel", lm.SetLevelHandler)
```

## Next steps

- Using [zerolog](Zerolog.md) or [logrus](Logrus.md) instead of slog
- Supporting [another logger](Custom-Loggers.md)
- The full [HTTP API](HTTP-API.md)
