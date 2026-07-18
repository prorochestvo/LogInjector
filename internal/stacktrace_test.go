package internal

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// callFromGeneric is a top-level generic function. The Go compiler appends
// "[...]" to its runtime name (e.g. "internal.callFromGeneric[...]"), which
// contains dots and would confuse the last-dot split without the generic-strip fix.
func callFromGeneric[T any]() string { return LineTrace(1) }

func TestLineTrace(t *testing.T) {
	t.Parallel()

	// "resolves caller frame" must be run at function top level (not inside a
	// t.Run closure) to land on tRunner at skip=2 — a closure adds an extra frame.
	// We capture the value here and verify inside the subtest so t.Run still
	// owns the reporting, but the skip count is fixed at the top-level call site.
	callerFrameResult := LineTrace(2)

	t.Run("resolves caller frame", func(t *testing.T) {
		t.Parallel()
		require.Contains(t, callerFrameResult, "testing/testing.go:")
		require.Contains(t, callerFrameResult, "tRunner")
	})

	t.Run("returns empty on invalid skip", func(t *testing.T) {
		t.Parallel()
		result := LineTrace(1 << 20)
		require.Equal(t, "", result)
	})

	t.Run("resolves own frame on skip=0", func(t *testing.T) {
		t.Parallel()
		result := LineTrace(0)
		require.Contains(t, result, "stacktrace.go:")
		require.Contains(t, result, "LineTrace")
	})

	t.Run("strips generic instantiation suffix", func(t *testing.T) {
		t.Parallel()
		result := callFromGeneric[int]()
		// must contain the bare function name, not "]" from "[...]"
		require.Contains(t, result, "callFromGeneric")
		require.NotContains(t, result, "]")
	})
}
