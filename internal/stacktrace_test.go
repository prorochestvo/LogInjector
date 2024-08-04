package internal

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestExtractMethodTrace(t *testing.T) {
	m := LineTrace()
	require.Contains(t, "testing/testing.go:", m)
	require.Contains(t, "tRunner", m)

	m = LineTrace()
	require.Contains(t, "testing/testing.go:", m)
	require.Contains(t, "tRunner", m)
}
