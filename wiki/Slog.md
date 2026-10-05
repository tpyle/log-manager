# Using slog

slog has no global level setting. A handler's level is fixed when it is
created, unless you pass a `*slog.LevelVar` as `HandlerOptions.Level`. The
manager changes that `LevelVar`, so you must use the same one for every
handler you want to control:

```go
level := new(slog.LevelVar)
level.Set(slog.LevelInfo) // the zero value is already info

opts := &slog.HandlerOptions{Level: level}
appLogger := slog.New(slog.NewJSONHandler(os.Stdout, opts))
auditLogger := slog.New(slog.NewTextHandler(auditFile, opts)) // also controlled

lm := logmanager.NewSlog(level)
```

Handlers created without the `LevelVar` keep their own level.

## Level names

Names are matched case-insensitively, using `slog.Level.UnmarshalText`:

| Name | Level |
|------|-------|
| `debug` | `slog.LevelDebug` (-4) |
| `info` | `slog.LevelInfo` (0) |
| `warn`, `warning` | `slog.LevelWarn` (4) |
| `error` | `slog.LevelError` (8) |
| `debug-4`, `info+2`, ... | a named level plus or minus an offset |

The manager reports levels in lower case, such as `"info"` or `"debug-4"`.
slog has no `trace`, `fatal` or `panic` levels, so those names are rejected. If
your program uses a custom trace level such as `slog.LevelDebug - 4`, set it
with `"debug-4"`.

## Change logging

`NewSlog` logs each change through `slog.Default()`, looking it up at the time
of the change. To use a specific logger:

```go
lm := logmanager.NewSlog(level, logmanager.WithOnChange(logmanager.SlogChangeLogger(appLogger)))
```

The request context is passed to the logger, so a context-aware handler
(such as `logmiddleware.ContextHandler` from
[log-middleware](https://github.com/tpyle/log-middleware)) adds the request ID
to the line.

## Using a controller directly

`logmanager.NewSlogController(level)` returns the `LevelController` without
the HTTP handler. This is useful if you change levels from somewhere other
than HTTP, such as a signal handler.
