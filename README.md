# Go Log Manager v2

A lightweight HTTP-based log configuration manager for Go applications using [zerolog](https://github.com/rs/zerolog). This library provides a simple HTTP endpoint to dynamically adjust logging settings in your running Go applications without requiring restarts.

## Features

- **Dynamic Log Level Control**: Change log levels at runtime via HTTP API
- **Report Caller Configuration**: Toggle caller information in log entries
- **RESTful Interface**: Simple GET/POST endpoints for configuration
- **JSON Configuration**: Easy-to-use JSON payload for settings
- **Zero Downtime**: Modify logging behavior without application restarts
- **Full Test Coverage**: Comprehensive unit tests with 100% coverage

## Installation

```bash
go get github.com/tpyle/log-manager/v2
```

## Quick Start

### Basic Usage

```go
package main

import (
    "fmt"
    "net/http"

    logmanager "github.com/tpyle/log-manager/v2"
    "github.com/rs/zerolog/log"
)

func main() {
    // Set up the log management endpoint
    http.HandleFunc("/api/log", logmanager.HandleLogCall)

    // Your application routes
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        log.Info().Msg("Hello World endpoint accessed")
        w.Write([]byte("Hello World!"))
    })

    log.Info().Msg("Server starting on :8080")
    if err := http.ListenAndServe(":8080", nil); err != nil {
        log.Fatal().Err(err).Msg("Server failed to start")
    }
}
```

## API Reference

### Get Current Log Level

**GET** `/api/log`

Returns the current log level as a plain text response.

**Example:**
```bash
curl http://localhost:8080/api/log
```

**Response:**
```
info
```

### Update Log Configuration

**POST** `/api/log`

Updates the logging configuration with a JSON payload.

**Headers:**
- `Content-Type: application/json`

**Request Body:**
```json
{
  "level": "debug",
  "reportCaller": true
}
```

**Parameters:**
- `level` (string, optional): Log level to set. Valid values: `panic`, `fatal`, `error`, `warn`, `warning`, `info`, `debug`, `trace`
- `reportCaller` (boolean, optional): Whether to include caller information in log entries

**Example:**
```bash
curl -X POST http://localhost:8080/api/log \
  -H "Content-Type: application/json" \
  -d '{"level": "debug", "reportCaller": true}'
```

**Response:**
```
debug
```

## Log Levels

The log levels come from the [zerolog](https://github.com/rs/zerolog) package and include:

- **panic**: Highest level of severity. Logs and then calls panic.
- **fatal**: Logs and then calls `os.Exit(1)`.
- **error**: Error conditions.
- **warn/warning**: Warning conditions.
- **info**: General informational messages.
- **debug**: Debug-level messages.
- **trace**: Most verbose level.

## Error Handling

The API returns appropriate HTTP status codes:

- **200 OK** - Configuration updated successfully
- **400 Bad Request** - Invalid JSON or unknown log level
- **405 Method Not Allowed** - Unsupported HTTP method
- **415 Unsupported Media Type** - Missing or incorrect Content-Type header

**Example error response:**
```bash
curl -X POST http://localhost:8080/api/log \
  -H "Content-Type: application/json" \
  -d '{"level": "invalid"}'

# Response: 400 Bad Request
# Body: unknown log level
```

## Testing

Run the test suite:

```bash
go test -v
```

Run with coverage:

```bash
go test -cover
```

Generate coverage report:

```bash
go test -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
