# Using logrus

```go
import "github.com/tpyle/log-manager/v3/logrusmgr"

logger := logrus.New()
lm := logrusmgr.New(logger) // nil means logrus.StandardLogger()
http.Handle("/log/", http.StripPrefix("/log", lm))
```

The adapter controls one `*logrus.Logger` through `SetLevel`. Entries created
from that logger (`logger.WithField(...)`, `logger.WithContext(...)`) follow
its level, including entries created before the change. If your program uses
several `*logrus.Logger` instances, each needs its own manager.

## Level names

Names are parsed with `logrus.ParseLevel`, case-insensitively: `trace`,
`debug`, `info`, `warn` or `warning`, `error`, `fatal`, `panic`.

Levels are reported the way logrus names them, so the warn level is reported
as `"warning"`.

## Change logging

By default, changes are logged at info level through the managed logger,
bound to the request context with `WithContext`. Hooks that read the
context, such as [log-middleware](https://github.com/tpyle/log-middleware)'s
`logrusmw.ContextHook`, apply to the line. To log through a different
logger or entry:

```go
lm := logrusmgr.New(logger, logmanager.WithOnChange(logrusmgr.ChangeLogger(auditEntry)))
```
