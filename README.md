# Go Log Manager v3

An HTTP handler for reading and changing a Go program's log level at runtime,
without a restart.

The core module has no dependencies outside the standard library and controls
[`log/slog`](https://pkg.go.dev/log/slog). Adapters for
[zerolog](https://github.com/rs/zerolog) and
[logrus](https://github.com/sirupsen/logrus) are separate modules, so those
libraries never appear in your `go.mod` unless you add the adapter. Any other
logger can be plugged in by implementing a two-method interface.

## Install

```bash
go get github.com/tpyle/log-manager/v3              # core and slog
go get github.com/tpyle/log-manager/zerologmgr/v3   # zerolog adapter
go get github.com/tpyle/log-manager/logrusmgr/v3    # logrus adapter
```

Requires Go 1.26+. v2 (zerolog only) remains available at
`github.com/tpyle/log-manager/v2`.

## Usage

### slog

slog has no global level, so share one `*slog.LevelVar` between your handler
options and the manager:

```go
import logmanager "github.com/tpyle/log-manager/v3"

level := new(slog.LevelVar)
slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})))

http.Handle("/log/", http.StripPrefix("/log", logmanager.NewSlog(level)))
```

### zerolog

Controls zerolog's global level:

```go
import "github.com/tpyle/log-manager/zerologmgr/v3"

http.Handle("/log/", http.StripPrefix("/log", zerologmgr.New()))
```

### logrus

Controls one `*logrus.Logger` (`nil` means `logrus.StandardLogger()`):

```go
import "github.com/tpyle/log-manager/logrusmgr/v3"

http.Handle("/log/", http.StripPrefix("/log", logrusmgr.New(logger)))
```

### Any other logger

Implement `logmanager.LevelController` and pass it to `logmanager.New`:

```go
type LevelController interface {
    Level() string
    SetLevel(name string) error // wrap logmanager.ErrUnknownLevel for bad names
}
```

See [wiki/Custom-Loggers.md](wiki/Custom-Loggers.md).

## HTTP API

```bash
$ curl localhost:8081/log/config
{"level":"info"}

$ curl -X POST localhost:8081/log/config -H 'Content-Type: application/json' -d '{"level":"debug"}'
{"level":"debug"}
```

| Status | When |
|--------|------|
| 200 | Success. The body holds the level now in effect. |
| 400 | Invalid JSON, or the level is missing or unknown |
| 405 | Method other than GET, HEAD or POST |
| 413 | Body larger than 64 KiB |
| 415 | `Content-Type` is not `application/json` |
| 500 | The controller failed for a reason other than an unknown level |

Accepted level names depend on the logger. See
[wiki/HTTP-API.md](wiki/HTTP-API.md).

Each change is logged at info level as `log level changed` with `oldLevel` and
`newLevel` fields. Use `logmanager.WithOnChange` to replace or disable this.

> **Security:** the handler does no authentication. Serve it on an internal
> port, or wrap it in your own auth middleware. See
> [wiki/Security.md](wiki/Security.md).

## Documentation

- [wiki/](wiki/Home.md): guides for each logger, the HTTP API, security, and
  [migrating from v2](wiki/Migrating-to-v3.md)
- [examples/](examples/): runnable servers for slog, zerolog, logrus and a
  custom logger (`go run ./examples/slog`)
- API reference: <https://pkg.go.dev/github.com/tpyle/log-manager/v3>

## Development

The repository holds four Go modules: the root, `zerologmgr`, `logrusmgr` and
`examples`. Run commands in each one:

```bash
for m in . zerologmgr logrusmgr examples; do
  (cd $m && go mod tidy -diff && go vet ./... && go test -race -cover ./... && golangci-lint run ./...)
done
```

See [wiki/Development.md](wiki/Development.md) for the module layout and how to
release. All three modules are released together with the same version.

## License

MIT. See [LICENSE](LICENSE).
