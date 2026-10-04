# log-manager

`github.com/tpyle/log-manager/v3` is an HTTP handler that lets you read and
change a running Go program's log level:

- `GET /config` returns `{"level":"<current>"}`
- `POST /config` with `{"level":"<new>"}` changes it and returns the level now
  in effect

The core package depends only on the standard library and controls `log/slog`.
zerolog and logrus are supported through adapter subpackages. Their
dependencies are downloaded and compiled only if you import the adapter.

| Logger | Import | Constructor | What it controls |
|--------|--------|-------------|------------------|
| `log/slog` | `github.com/tpyle/log-manager/v3` | `logmanager.NewSlog(levelVar)` | a `*slog.LevelVar` shared with your handlers |
| zerolog | `github.com/tpyle/log-manager/v3/zerologmgr` | `zerologmgr.New()` | `zerolog.SetGlobalLevel` |
| logrus | `github.com/tpyle/log-manager/v3/logrusmgr` | `logrusmgr.New(logger)` | `(*logrus.Logger).SetLevel` |
| anything else | `github.com/tpyle/log-manager/v3` | `logmanager.New(controller)` | your `LevelController` |

## Pages

- [Getting Started](Getting-Started.md)
- [Using slog](Slog.md)
- [Using zerolog](Zerolog.md)
- [Using logrus](Logrus.md)
- [Writing a Custom Controller](Custom-Loggers.md)
- [HTTP API](HTTP-API.md)
- [Logging and Auditing Changes](Change-Logging.md)
- [Security](Security.md)
- [Migrating to v3](Migrating-to-v3.md)

Runnable programs for each logger are in the repository's `examples/` directory.
