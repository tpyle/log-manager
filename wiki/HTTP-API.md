# HTTP API

`LogManager` serves one route, relative to wherever you mount it.

## GET /config

Returns the current level. `HEAD` is also accepted.

```http
HTTP/1.1 200 OK
Content-Type: application/json

{"level":"info"}
```

## POST /config

Sets the level.

```http
POST /config HTTP/1.1
Content-Type: application/json

{"level":"debug"}
```

The response contains the level now in effect, as the logger reports it. This
can differ from the requested name. For example, logrus reports `warn` as
`"warning"`, and slog reports `WARNING` as `"warn"`:

```http
HTTP/1.1 200 OK
Content-Type: application/json

{"level":"debug"}
```

- `Content-Type` must be `application/json`. Parameters such as
  `; charset=utf-8` are allowed.
- Unknown JSON fields are ignored.
- Whitespace around the level name is ignored. Case is ignored by all bundled
  controllers.
- Setting the level that is already in effect succeeds and doesn't log a
  change.

## Errors

Error responses have a `text/plain` body.

| Status | Body | Cause |
|--------|------|-------|
| 400 | `invalid JSON body` | Body is empty, not JSON, or `level` is not a string |
| 400 | `level is required` | `level` is missing or blank |
| 400 | `unknown log level` | The controller doesn't recognize the name |
| 404 | | Path other than `/config` |
| 405 | | Method other than GET, HEAD or POST (an `Allow` header lists them) |
| 413 | `request body too large` | Body exceeds `logmanager.MaxRequestBytes` (64 KiB) |
| 415 | `Content-Type must be application/json` | Missing or different `Content-Type` |
| 500 | `failed to set log level` | The controller returned an error other than `ErrUnknownLevel` |

## Level names by logger

| Name | slog | zerolog | logrus |
|------|------|---------|--------|
| `trace` | | yes | yes |
| `debug` | yes | yes | yes |
| `info` | yes | yes | yes |
| `warn` / `warning` | yes | yes | yes (reported as `warning`) |
| `error` | yes | yes | yes |
| `fatal` | | yes | yes |
| `panic` | | yes | yes |
| `disabled` | | yes | |
| offsets, e.g. `debug-4` | yes | | |
