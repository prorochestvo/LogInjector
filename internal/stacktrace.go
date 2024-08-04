package internal

import (
	"path"
	"regexp"
	"runtime/debug"
	"strings"
)

var (
	rxFileRowNumber = regexp.MustCompile(`(\w+/\w+/\w+.go:\d+)\s+[A-Ba-z0-9+.]+$`)
	rxMethodName    = regexp.MustCompile(`/(\w+[^/]+\w+)[({]+.+$`)
)

// StackTrace returns the current stack trace as a string.
func StackTrace() string {
	result := string(debug.Stack())
	result = strings.TrimSpace(result)
	return result
}

// LineTrace extracts method trace information from a stack trace.
// It returns the last method name and the file name with the line number.
// The result is formatted as "file.go:line (method)".
func LineTrace() string {
	stacktrace := StackTrace()

	index := strings.LastIndex(stacktrace, path.Join("internal", "stacktrace.go"))
	if index <= 0 {
		return ""
	}
	index += strings.Index(stacktrace[index:], "\n") + 1
	lines := strings.Split(stacktrace[index:], "\n")

	method := ""
	position := ""

	if l := len(lines); l >= 4 {
		method = lines[2]
		position = lines[3]
	} else if l >= 2 {
		method = lines[0]
		position = lines[1]
	} else {
		return ""
	}

	// extract method name
	if n := path.Base(method); len(n) > 3 {
		if i := strings.LastIndex(n, "."); i > 0 {
			n = n[i+1:]
		}
		if i := strings.LastIndex(n, "("); i > 0 {
			n = n[:i]
		}
		method = n
	}

	// extract file path and line number
	if d, n := path.Split(position); len(n) > 3 {
		if i := strings.LastIndex(n, " "); i > 0 {
			n = n[:i]
		}
		if tmp, _ := path.Split(path.Dir(d)); tmp != "" {
			d = strings.ReplaceAll(d, tmp, "")
		}
		position = path.Join(d, n)
	}

	return position + " (" + method + ")"
}
