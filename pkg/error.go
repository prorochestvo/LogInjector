package loginjector

import (
	"github.com/prorochestvo/loginjector/internal"
	"net/http"
)

// TraceError is an error that provides a stack trace.
type TraceError interface {
	Line() string
	error
}

// NewTraceError creates a new TraceError with the current stack trace.
func NewTraceError() TraceError {
	return &traceError{line: internal.LineTrace()}
}

// traceError is an error that provides a stack trace.
type traceError struct {
	line string
}

func (e *traceError) Line() string {
	return e.line
}

func (e *traceError) Error() string {
	return e.line
}

// HttpError is an error that provides an HTTP status code.
type HttpError interface {
	StatusCode() int
	error
}

// NewHttpError creates a new HttpError with the given status code.
func NewHttpError(code int) HttpError {
	return &httpError{code: code}
}

type httpError struct {
	code int
}

// StatusCode returns the HTTP status code.
func (e *httpError) StatusCode() int {
	return e.code
}

// Error returns the HTTP status text.
func (e *httpError) Error() string {
	return http.StatusText(e.code)
}
