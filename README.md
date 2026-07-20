# LogInjector

A small, dependency-light Go logging library. One `Logger` fans each message out to
several sinks at once — size-rotated files, the console, a Telegram chat — filtered by
severity levels you define. Application code logs through the standard library
(`log.New` over an `io.Writer`); LogInjector supplies that writer and handles routing,
rotation, and alerts.

## Features

- **Fan-out** — one `Logger` writes to many `io.Writer` sinks at once; safe for
  concurrent use.
- **Your own levels** — no baked-in DEBUG/INFO; higher `int` = more severe. The optional
  `levels` sub-package ships a ready `Debug..Critical` ladder.
- **Ready handlers** — rotating files, timestamped console, Telegram, with an optional
  per-handler minimum level.
- **Level hooks** — tee alerts (e.g. to Telegram) on exact levels without lowering the
  global threshold.
- **Stdlib-native usage** — hand components a `*log.Logger` via `StdLog`; they never
  import loginjector.
- **Opt-in HTTP tapping** (`httptap`) — payload dump, access log, and panic-recover
  middleware, with always-on redaction of auth/cookie headers.
- **Dependency-light** — no third-party runtime dependencies.

## Install

```sh
go get github.com/prorochestvo/loginjector
```

Requires Go 1.22+.

## Quick start

```go
import (
	"github.com/prorochestvo/loginjector"
	"github.com/prorochestvo/loginjector/levels"
)

// size-rotated files under ./logs (5 MiB × 7) + timestamped console echo.
l, err := loginjector.NewFileLogger("./logs", "app", levels.Info)
if err != nil {
	panic(err)
}
l.Printf(levels.Info, "user %q signed in", "alice")
```

**Recommended:** build one `Logger` in `main`, then give each component its own
standard-library front logger with `StdLog(level, prefix)` — the component depends on
`*log.Logger`, not on loginjector:

```go
// everything at Error and above is also delivered to Telegram.
l.Hook(loginjector.TelegramHandler(botToken, chatID, "errors.log", "prod"),
	levels.Error, levels.Severe, levels.Critical)

dbLog := l.StdLog(levels.Info, "sqlite ")  // = log.New(l.WriterAs(Info), "sqlite ", log.Lmsgprefix)
aiErr := l.StdLog(levels.Error, "openai ")
svc := NewService(dbLog, aiErr)            // takes *log.Logger

// inside the component, call sites stay standard library:
s.log.Printf("request sent (model=%s)", model) // "sqlite request sent (model=gpt-4o)"
```

The full API — handlers, error / JSON / closer helpers, `httptap` middleware, and every
option — is on [pkg.go.dev](https://pkg.go.dev/github.com/prorochestvo/loginjector).

## Notes

- **`Panicf`/`Panic` `panic`, they do not call `os.Exit`** — a deferred `recover` can
  intercept them.
- **Emitter flags:** `StdLog` sets `log.Lmsgprefix` (prefix only) because the sink stamps
  the timestamp; a hand-built front logger with `log.LstdFlags` doubles it.
- **`httptap` redaction is always on:** `Authorization`, `Proxy-Authorization`, `Cookie`,
  and `Set-Cookie` are never logged. `NewRecoverHandler` buffers the response body, so keep
  it off SSE / WebSocket / streaming routes.
- **Stable-path rotation:** `RotatingFileHandler(dir, prefix, WithStableCurrentName())`
  keeps the live file at a fixed `prefix.log` path (backups stay indexed) so external
  tooling can follow it with `tail -F`; pair it with `WithMaxAge` and `WithCompress` for
  age-based retention and gzipped backups.

## License

MIT — see [LICENSE](LICENSE).
