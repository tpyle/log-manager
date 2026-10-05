# Writing a Custom Controller

The handler talks to your logger through one interface:

```go
type LevelController interface {
    Level() string
    SetLevel(name string) error
}
```

- `Level` returns the current level's name. The handler sends it to clients
  without changing it, so choose names that `SetLevel` accepts.
- `SetLevel` parses `name` and applies it. If the name is not a valid level,
  return an error wrapping `logmanager.ErrUnknownLevel` and leave the level
  unchanged. The client gets 400 Bad Request. Any other error results in
  500 Internal Server Error.
- Both methods must be safe for concurrent use. The handler serializes its own
  changes, but your program may read the level from other goroutines.

The handler trims surrounding whitespace and rejects empty names before
calling `SetLevel`.

## Example

```go
type verboseFlag struct{ on atomic.Bool }

func (f *verboseFlag) Level() string {
    if f.on.Load() {
        return "verbose"
    }
    return "quiet"
}

func (f *verboseFlag) SetLevel(name string) error {
    switch name {
    case "verbose":
        f.on.Store(true)
    case "quiet":
        f.on.Store(false)
    default:
        return fmt.Errorf("%w: %q", logmanager.ErrUnknownLevel, name)
    }
    return nil
}

lm := logmanager.New(&verboseFlag{})
```

`logmanager.New` doesn't log changes. Add a callback with `WithOnChange` if you
want them logged (see [Logging and Auditing Changes](Change-Logging.md)).
`examples/custom` is a runnable version that also requires a bearer token.

## Testing

Because the controller is injected, you can test code that builds a
`LogManager` with a fake controller and `httptest`, without touching any
global logger state.
