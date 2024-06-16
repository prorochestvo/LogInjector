package loginjector

import (
	"bytes"
	"fmt"
	"github.com/prorochestvo/LogInjector/internal/stacktrace"
	"io"
	"strings"
	"sync"
)

type LogEvent interface {
	StackTrace() string
	io.WriteCloser
	error
}

// CreateLogEvent creates a new record with the specified log level and default logger.
// The message is written to the logger when the record is closed.
// The record is thread-safe.
func CreateLogEvent(level LogLevel, message ...string) LogEvent {
	return newRecord(level, defaultLogger, message...)
}

// CreateAndCloseLogEvent creates a new record with the specified log level and default logger.
// The message is written immediately to the logger.
func CreateAndCloseLogEvent(level LogLevel, message ...string) {
	r := newRecord(level, defaultLogger, message...)
	defer CloseOrLog(r)
}

// newRecord creates a new record with the specified log level and logger.
// The message is written to the logger when the record is closed.
// The record is thread-safe.
func newRecord(level LogLevel, logger *Logger, message ...string) LogEvent {
	method, trace := stacktrace.ExtractMethodTrace("pkg/logevent.go")
	r := &record{
		level:       level,
		buffer:      bytes.NewBufferString(strings.Join(message, "\n")),
		logger:      logger,
		methodTrace: method,
		stackTrace:  trace,
	}
	return r
}

// record is a log record
type record struct {
	level       LogLevel
	buffer      *bytes.Buffer
	logger      *Logger
	methodTrace string
	stackTrace  string
	m           sync.Mutex
}

// Write accumulates the message in the record
func (r *record) Write(m []byte) (int, error) {
	r.m.Lock()
	defer r.m.Unlock()

	n, err := r.buffer.Write(m)
	if err == nil {
		err = r.buffer.WriteByte('\n')
	}

	return n, err
}

// Close writes the message to the logger
func (r *record) Close() error {
	r.m.Lock()
	defer r.m.Unlock()

	if r.logger == nil {
		return fmt.Errorf("logger is not set")
	}

	details := make([]byte, 0, r.buffer.Len()+len(r.stackTrace)+12)
	details = append(details, r.buffer.Bytes()...)
	details = append(details, []byte("STACKTRACE:\n")...)
	details = append(details, r.stackTrace...)

	_, err := r.logger.WriteLog(r.level, details)
	if err == nil {
		r.buffer.Reset()
	}
	return err
}

// Error returns the accumulated message
func (r *record) Error() string {
	r.m.Lock()
	defer r.m.Unlock()

	return r.buffer.String()
}

// StackTrace returns the stack trace of the record creation point
func (r *record) StackTrace() string {
	return r.methodTrace
}

// Default logger
var defaultLogger *Logger = nil

// UseAsDefault sets the logger as the default logger
func UseAsDefault(l *Logger) {
	defaultLogger = l
}
