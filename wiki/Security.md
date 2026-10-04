# Security

The handler has **no authentication or authorization**. Anyone who can reach
it can change your log level. That can:

- turn on `debug` or `trace` logging, which may write sensitive data (request
  bodies, tokens, personal data) to your logs and increase log volume and cost;
- turn logging down or off (`error`, or `disabled` with zerolog), hiding
  activity.

## Recommendations

**Serve it on a separate, internal listener.** This is the simplest option:

```go
admin := http.NewServeMux()
admin.Handle("/log/", http.StripPrefix("/log", lm))
go http.ListenAndServe("127.0.0.1:8081", admin)
```

In Kubernetes, use a port that isn't exposed by your Service or Ingress, and
reach it with `kubectl port-forward`.

**Or wrap it with your own auth middleware** if it must share the public
listener:

```go
mux.Handle("/log/", requireAdmin(http.StripPrefix("/log", lm)))
```

`examples/custom` shows a bearer-token check that uses a constant-time
comparison.

## Built-in protections

- Request bodies are limited to 64 KiB (`logmanager.MaxRequestBytes`).
- `POST` requires `Content-Type: application/json`. Browsers can't send
  this type in a cross-origin request without a CORS preflight, so a
  malicious page can't change the level through a visitor's browser.
- Each change is logged with the request context (see
  [Logging and Auditing Changes](Change-Logging.md)). With request logging
  middleware, the line carries the request ID, so you can match it to the
  request that made the change.
