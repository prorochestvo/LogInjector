# LogInjector

**Description:**

LogInjector is a lightweight logging utility written in Go designed to provide comprehensive and efficient log management applications.

**Features:**

~~- **Flexible Log Levels:** Easily categorize logs with predefined levels such as DEBUG, INFO, WARN, ERROR, and FATAL.
- **Structured Logging:** Support for structured logging, allowing you to include context-rich information in your logs.
- **Output Options:** Write logs to various outputs including console, files, and remote servers.
- **Log Rotation:** Automatically manage log file sizes and rotation, ensuring your log storage remains efficient.
- **Custom Formats:** Define custom log formats to suit your application's needs.
- **Performance Optimization:** Built with Go's concurrency model for high performance and minimal overhead.
- **Extensible:** Plugin architecture for adding custom log processors and outputs.
- **JSON Support:** Native support for logging in JSON format, perfect for integrating with log aggregation and analysis tools.
- **HTTP Middleware:** Easily integrate LogInjector with your Go web applications using the provided HTTP middleware.
- **Error Handling:** Built-in error handling and recovery mechanisms to ensure your application remains stable.~~

**Installation:**

```sh
go get github.com/prorochestvo/LogInjector
```

**Usage:**

```go
package main

import (
    "github.com/prorochestvo/LogInjector"
)

func main() {
    logger := LogInjector.NewLogger(LogInjector.Config{
        Level: LogInjector.DEBUG,
        Output: LogInjector.ConsoleOutput(),
    })

    logger.Info("Application started")
    logger.Debug("Debugging information")
    logger.Warn("This is a warning")
    logger.Error("An error has occurred")
}
```

**Contributing:**

We welcome contributions from the community! Please read our contributing guidelines and submit your pull requests or report issues on our GitHub repository.

**License:**

LogInjector is released under the MIT License. See the LICENSE file for more information.