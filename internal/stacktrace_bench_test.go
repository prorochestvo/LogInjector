package internal

import "testing"

// lineTraceSink prevents the compiler from eliding benchmark calls via dead-code elimination.
var lineTraceSink string

// lineTraceFromHelper is a non-inlineable indirection so that BenchmarkLineTrace
// exercises skip=2 — the same depth as production callers of LineTrace.
//
//go:noinline
func lineTraceFromHelper() string { return LineTrace(2) }

// BenchmarkLineTrace measures the cost of capturing one call-site line via LineTrace.
// It calls through lineTraceFromHelper so that skip=2 matches the production call depth.
func BenchmarkLineTrace(b *testing.B) {
	b.ReportAllocs()
	var s string
	for i := 0; i < b.N; i++ {
		s = lineTraceFromHelper()
	}
	lineTraceSink = s
}

// BenchmarkStackTrace measures the cost of the full debug.Stack() path so readers
// have a reference point for why StackTrace was left unchanged (not a hot path).
func BenchmarkStackTrace(b *testing.B) {
	b.ReportAllocs()
	var s string
	for i := 0; i < b.N; i++ {
		s = StackTrace()
	}
	lineTraceSink = s
}
