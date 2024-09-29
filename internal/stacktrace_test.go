package internal

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestExtractMethodTrace(t *testing.T) {
	m := LineTrace()
	require.Contains(t, m, "testing/testing.go:")
	require.Contains(t, m, "tRunner")

	m = LineTrace()
	require.Contains(t, m, "testing/testing.go:")
	require.Contains(t, m, "tRunner")
}
