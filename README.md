# LogInjector

**Description:**

LogInjector is a lightweight logging utility written in Go designed to provide comprehensive and efficient log management applications.

**Features:**

- **Flexible Log Levels:** Easily categorize logs with predefined levels such as DEBUG, INFO, WARN, ERROR, and FATAL.
- **Structured Logging:** Support for structured logging, allowing you to include context-rich information in your logs.
- **Output Options:** Write logs to various outputs including console, files, and remote servers.
- **Log Rotation:** Automatically manage log file sizes and rotation, ensuring your log storage remains efficient.
- **Custom Formats:** Define custom log formats to suit your application's needs.
- **Performance Optimization:** Built with Go's concurrency model for high performance and minimal overhead.
- **Extensible:** Plugin architecture for adding custom log processors and outputs.
- **JSON Support:** Native support for logging in JSON format, perfect for integrating with log aggregation and analysis tools.
- **HTTP Middleware:** Easily integrate LogInjector with your Go web applications using the provided HTTP middleware.
- **Error Handling:** Built-in error handling and recovery mechanisms to ensure your application remains stable.

**Installation:**

```sh
go get github.com/prorochestvo/loginjector
```

**Usage:**

```go
package main

import (
	"bytes"
	"github.com/prorochestvo/loginjector"
)

const (
	LogLevelDebug loginjector.LogLevel = 1
	LogLevelInfo  loginjector.LogLevel = 2
	LogLevelWarn  loginjector.LogLevel = 3
	LogLevelError loginjector.LogLevel = 4
)

func main() {
	b := bytes.NewBufferString("")

	l, err := loginjector.NewLogger(LogLevelWarn, b)
	if err != nil {
		panic(err)
	}

	_, err = l.WriteLog(LogLevelInfo, []byte("Hello, World!"))
	if err != nil {
		panic(err)
	}

	_, err = l.WriteLog(LogLevelWarn, []byte("log message: warning"))
	if err != nil {
		panic(err)
	}

	println(b.String())
}

```

**Contributing:**

We welcome contributions from the community! Please read our contributing guidelines and submit your pull requests or report issues on our GitHub repository.

**License:**

LogInjector is released under the MIT License. See the LICENSE file for more information.