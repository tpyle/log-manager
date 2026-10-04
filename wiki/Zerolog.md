# Using zerolog

```bash
go get github.com/tpyle/log-manager/zerologmgr/v3
```

```go
import "github.com/tpyle/log-manager/zerologmgr/v3"

lm := zerologmgr.New()
http.Handle("/log/", http.StripPrefix("/log", lm))
```

The adapter controls zerolog's **global** level (`zerolog.SetGlobalLevel`).
zerolog loggers are immutable values, so there is no way to change the level
of one logger after it is created.

## Interaction with per-logger levels

zerolog drops an event if it is below the global level **or** below the
logger's own level. The global level can therefore only make a logger quieter
than its own level, never more verbose:

```go
logger := zerolog.New(os.Stdout).Level(zerolog.InfoLevel)
// POST {"level":"debug"} sets the global level to debug, but this logger
// still drops debug events because its own level is info.
```

For full runtime control, leave loggers at their default level (trace) and
set the starting level globally:

```go
zerolog.SetGlobalLevel(zerolog.InfoLevel)
log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
```

## Level names

Matched case-insensitively:

| Name | Level |
|------|-------|
| `trace` | `zerolog.TraceLevel` |
| `debug` | `zerolog.DebugLevel` |
| `info` | `zerolog.InfoLevel` |
| `warn`, `warning` | `zerolog.WarnLevel` |
| `error` | `zerolog.ErrorLevel` |
| `fatal` | `zerolog.FatalLevel` |
| `panic` | `zerolog.PanicLevel` |
| `disabled` | `zerolog.Disabled` (turns all logging off) |

Levels are always reported with these names, even if you have customized
`zerolog.LevelWarnValue` and similar variables for your log output.

## Change logging

By default, `zerologmgr.New` logs each change with the logger attached to the
request context (`zerolog.Ctx`), such as one added by
[log-middleware](https://github.com/tpyle/log-middleware)'s `zerologmw`. If the
context has no logger, it uses the global `log.Logger`. To use a specific
logger:

```go
lm := zerologmgr.New(logmanager.WithOnChange(zerologmgr.ChangeLogger(&logger)))
```
