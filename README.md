# LogInjector

LogInjector is a small logging library for Go services that need plain-text logs
routed to several destinations at once — size-rotated files, the console, a Telegram
chat — filtered by severity levels you control. Application code keeps logging through
the standard library (`log.New` over an `io.Writer`); LogInjector supplies that writer
and takes care of routing, rotation, and alerts. A `Logger` is safe for concurrent use.

## Install

```sh
go get github.com/prorochestvo/loginjector
```

Requires Go 1.22+.

## Quick start

```go
package main

import (
	"github.com/prorochestvo/loginjector"
	"github.com/prorochestvo/loginjector/levels"
)

func main() {
	// size-rotated files under ./logs (5 MiB × 7 files) + timestamped console echo.
	l, err := loginjector.NewFileLogger("./logs", "app", levels.Info)
	if err != nil {
		panic(err)
	}

	l.Printf(levels.Info, "user %q signed in", "alice")
	l.Printf(levels.Debug, "dropped: below the minimum level")
}
```

## Recommended usage

Create one `Logger` in `main`, then hand every component its own standard-library
front logger via `Logger.StdLog(level, prefix)` — a fixed severity and a component
prefix over the shared sink. Components depend on `*log.Logger` (or plain `io.Writer`)
and never import loginjector.

```go
l, err := loginjector.NewFileLogger("./logs", "api", levels.Info,
	loginjector.WithPrinterMinLevel(levels.Warning), // file: Info and up, console: Warning and up
	loginjector.WithStdLogRedirect(levels.Warning),  // capture the global `log` package output
)
if err != nil {
	panic(err)
}

// everything at Error and above is additionally delivered to a Telegram chat.
l.Hook(
	loginjector.TelegramHandler(botToken, chatID, "errors.log", "myapp prod"),
	levels.Error, levels.Severe, levels.Critical,
)

// per-component front loggers: same sink, different prefix and severity.
// StdLog wires log.New(l.WriterAs(level), prefix, log.Lmsgprefix) for you.
dbLog := l.StdLog(levels.Info, "sqlite ")
aiLog := l.StdLog(levels.Info, "openai ")
aiErr := l.StdLog(levels.Error, "openai ")

svc := NewCompletionService(aiLog, aiErr) // takes *log.Logger, not loginjector types
```

Inside a component the call sites stay standard:

```go
s.log.Printf("request sent (model=%s)", s.model) // → "openai request sent (model=gpt-4o)"
s.err.Printf("request failed: %v", err)          // severity Error → file, console, Telegram
```

Two front loggers per component (regular + error) is the recommended split: it keeps
per-message severity without coupling the component to any logging library.

`StdLog` sets `log.Lmsgprefix` (prefix only, no emitter timestamp) precisely because the
sink stamps every line itself — so each line carries exactly one timestamp. Build a front
logger by hand with `log.LstdFlags` and you get doubled timestamps.

## Levels

`LogLevel` is a plain `int`: higher means more severe, and messages below the logger's
minimum are dropped. The `levels` sub-package ships a ready ladder — `Debug`, `Info`,
`Warning`, `Error`, `Severe`, `Critical` (1..6) — plus `levels.Parse("warning")` for
reading a level from config. The ladder is optional: declare your own constants if it
does not fit. `WithMinLevel(level, handler)` gives one handler its own threshold;
`SetMinLevel` changes the logger's minimum at runtime.

## Handlers

A handler is any `io.Writer`. Built-ins:

- `RotatingFileHandler(folder, prefix, opts...)` — size-rotated log files, oldest pruned. Options: `WithMaxFileSize` (default 5 MiB), `WithMaxFiles` (default 7), `WithFreshStart` (empty first file each run).
- `FileByFormatHandler(folder, maxFiles, nameFunc)` — one file per `nameFunc()` value (e.g. per day).
- `TimestampedPrintHandler(opts...)` — stdout echo with timestamps and multi-line indenting; the default when `NewLogger` gets no handlers.
- `TimestampedHandler(inner, opts...)` — wraps any `io.Writer` to prepend timestamps (the reusable core of `TimestampedPrintHandler`).
- `TelegramHandler(botToken, chatID, fileName, labels...)` — delivers each message as a document to a chat.
- `PrintHandler()` — bare stdout output, no timestamps.

Compose them yourself when `NewFileLogger` is not enough:

```go
l := loginjector.NewLogger(
	levels.Info,
	loginjector.RotatingFileHandler("./logs", "app"), // 5 MiB × 7 files by default
	loginjector.WithMinLevel(levels.Warning, loginjector.TimestampedPrintHandler()),
)
```

## Writing logs

| Method | Behaviour |
|--------|-----------|
| `Printf(level, format, args...)` / `Print(level, args...)` | format and write a line |
| `Panicf(level, ...)` / `Panic(level, ...)` | write a line, then **`panic`** (not `os.Exit`) |
| `WriteLog(level, msg) (int, error)` | low-level write; returns joined sink errors |
| `Write(msg) (int, error)` | `io.Writer` writing at the minimum level |
| `WriterAs(level) io.Writer` | an `io.Writer` bound to a fixed level |

## Hooks

A hook taps one or more exact levels independently of the minimum level — useful for
teeing alerts somewhere without lowering the global threshold. The Telegram hook in
the example above keeps firing even for levels the handlers would drop. `Hook` returns
an id; `Unhook(id)` removes the hook.

## HTTP middleware: `httptap`

HTTP traffic tapping lives in the `httptap` sub-package
(`import "github.com/prorochestvo/loginjector/httptap"`):

- `httptap.NewPayloadHandler(l, level, next)` — debug dump of the full request and
  response through the logger. `Authorization`, `Proxy-Authorization`, `Cookie`, and
  `Set-Cookie` are always redacted. `NewPayloadHandlerWithOptions` adds body-size
  caps, extra redacted headers, and summary control (see godoc).
- `httptap.NewAccessHandler(out, next)` — production access log: one line per request
  (timestamp, status, method, path, duration) into any `io.Writer`.
- `httptap.NewRecoverHandler(l, level, next, opts...)` — converts a panic into a
  logged stack trace plus a clean HTTP 500. It buffers the response body, so do not
  wrap SSE, WebSocket, or other streaming routes with it.

All three take and return `http.HandlerFunc` and chain freely, e.g.
`recover(payload(access(next)))`. `http.Flusher`, `http.Hijacker`, and `http.Pusher`
are preserved when the underlying `ResponseWriter` supports them.

## Helpers

- **Errors** — constructors that attach context to errors: `NewTraceError` (call
  site), `NewHttpError(code)` (HTTP status), `NewPublicError(public, cause)`
  (client-safe message separated from the internal cause), `NewStackTraceError` (full
  stack plus runtime info). `Wrap*` and `*f` variants integrate with
  `errors.Is/As/Unwrap`.
- **Closers** — `CloseOrLog`, `CloseOrPanic`, `CloseOrJoin(&err, c)` and friends for
  `defer`-ing `io.Closer` cleanup.

## Concurrency

Every sink is wrapped so the logger never calls its `Write` concurrently — logging
from multiple goroutines is safe even when a handler itself is not thread-safe.

## License

LogInjector is released under the MIT License. See [LICENSE](LICENSE).
