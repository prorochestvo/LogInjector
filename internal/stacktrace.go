package internal

import (
	"path"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
)

// StackTrace returns the full goroutine stack as a trimmed string via debug.Stack.
// It is intentionally kept on debug.Stack rather than reimplemented via
// runtime.Callers+CallersFrames: callers want the full multiline goroutine dump,
// the path is not hot (called only at error-construction time on failure paths),
// and the existing test assertions pin the multiline, human-readable format
// produced by debug.Stack — reformatting would break them for negligible gain.
//
// The returned string contains absolute build-host file paths; do not forward
// it to untrusted clients.
func StackTrace() string {
	result := string(debug.Stack())
	result = strings.TrimSpace(result)
	return result
}

// LineTrace returns a single-line string of the form "file.go:line (method)"
// identifying the call frame skip levels above LineTrace itself. skip=0 is the
// LineTrace frame; skip=1 is its immediate caller; skip=2 is that caller's
// caller, and so on. Returns "" when runtime.Caller reports ok == false.
//
// Note: //go:noinline is deliberately absent. LineTrace exceeds the inline
// budget by an order of magnitude, and runtime.Caller correctly handles
// inlined call sites since Go 1.12, so the skip count remains stable even
// if a caller is inlined.
func LineTrace(skip int) string {
	pc, file, line, ok := runtime.Caller(skip)
	if !ok {
		return ""
	}

	// shorten the absolute file path to "<parent>/<base>" so the output matches
	// the old debug.Stack parser: stdlib frames become "testing/testing.go",
	// module-root files get their immediate parent directory prefix.
	base := path.Base(file)
	parent := path.Base(path.Dir(file))
	shortFile := base
	if parent != "" && parent != "." && parent != "/" {
		shortFile = parent + "/" + base
	}

	method := "unknown"
	if fn := runtime.FuncForPC(pc); fn != nil {
		name := fn.Name()
		// strip generic instantiation shape suffix appended by the compiler to
		// top-level generic functions ("pkg.Foo[...]"). It must be removed before
		// the last-dot split because the brackets contain dots. Methods on generic
		// types ("pkg.(*T[...]).Method") are safe — their "[...]" is not at end-of-string.
		if strings.HasSuffix(name, "[...]") {
			name = name[:len(name)-len("[...]")]
		}
		// strip the package path prefix: the last dot separates package from function.
		if i := strings.LastIndex(name, "."); i >= 0 && i+1 < len(name) {
			name = name[i+1:]
		}
		if name != "" {
			method = name
		}
	}

	return shortFile + ":" + strconv.Itoa(line) + " (" + method + ")"
}
