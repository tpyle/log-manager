# Logging and Auditing Changes

After a request changes the level, the manager calls its `LevelChangeFunc`:

```go
type LevelChangeFunc func(ctx context.Context, from, to string)
```

`ctx` is the request context. `from` and `to` are the level names before and
after the change. The function isn't called if the request fails or sets the
level that is already in effect.

## Defaults

| Constructor | Default |
|-------------|---------|
| `logmanager.New` | none |
| `logmanager.NewSlog` | `logmanager.SlogChangeLogger(nil)` (uses `slog.Default()`) |
| `zerologmgr.New` | `zerologmgr.ChangeLogger(nil)` (uses `zerolog.Ctx(ctx)`, then `log.Logger`) |
| `logrusmgr.New` | `logrusmgr.ChangeLogger(logger)` (uses the managed logger) |

Each logs at info level with the message `log level changed` and the fields
`oldLevel` and `newLevel`, which are available as `logmanager.ChangeMessage`,
`logmanager.FieldOldLevel` and `logmanager.FieldNewLevel`.

## Replacing or disabling

```go
// Log somewhere else:
lm := logmanager.NewSlog(level, logmanager.WithOnChange(logmanager.SlogChangeLogger(auditLogger)))

// Custom audit trail:
lm := zerologmgr.New(logmanager.WithOnChange(func(ctx context.Context, from, to string) {
    audit.Record(ctx, "log_level", from, to)
}))

// No logging:
lm := logrusmgr.New(logger, logmanager.WithOnChange(nil))
```

## Things to know

- **Raising the level above info hides its own message.** The default loggers
  log at info level after the change is applied. For example, a change to
  `error` filters out the line that reports it. If you need a record of every
  change, send it somewhere that isn't filtered by the level you're
  controlling, such as a separate audit logger.
- **Calls are serialized and in order.** The function runs while the manager
  holds its lock, so callbacks observe changes in the order they were applied.
  Other change requests wait for it to return, so keep it fast.
