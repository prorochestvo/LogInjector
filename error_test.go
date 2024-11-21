package loginjector

import (
	"errors"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

func TestHttpError(t *testing.T) {
	key1 := "KEY_1"
	key2 := "KEY_2"
	key3 := "KEY_3"

	var err error = NewHttpError(http.StatusInternalServerError)
	require.Contains(t, err.Error(), http.StatusText(http.StatusInternalServerError))
	require.NotContains(t, err.Error(), key1)
	require.NotContains(t, err.Error(), key2)
	require.NotContains(t, err.Error(), key3)

	err = errors.Join(err, errors.New(key1))
	err = errors.Join(err, NewHttpError(http.StatusBadRequest))
	require.Contains(t, err.Error(), http.StatusText(http.StatusBadRequest))
	require.Contains(t, err.Error(), key1)
	require.NotContains(t, err.Error(), key2)
	require.NotContains(t, err.Error(), key3)

	err = errors.Join(err, errors.New(key2))
	require.Contains(t, err.Error(), http.StatusText(http.StatusInternalServerError))
	require.Contains(t, err.Error(), key1)
	require.Contains(t, err.Error(), key2)
	require.NotContains(t, err.Error(), key3)

	var httpErr HttpError
	require.Equal(t, true, errors.As(err, &httpErr), "could not convert error to httpCode")
	require.Equal(t, http.StatusInternalServerError, httpErr.StatusCode(), "unexpected status code")
}

func TestFullTrace(t *testing.T) {
	key1 := "KEY_1"
	key2 := "KEY_2"
	key3 := "KEY_3"

	var err error = NewTraceError()
	require.Contains(t, err.Error(), "TestFullTrace")
	require.NotContains(t, err.Error(), key1)
	require.NotContains(t, err.Error(), key2)
	require.NotContains(t, err.Error(), key3)

	err = errors.Join(err, errors.New(key1))
	err = errors.Join(err, NewTraceError())
	require.Contains(t, err.Error(), "TestFullTrace")
	require.Contains(t, err.Error(), key1)
	require.NotContains(t, err.Error(), key2)
	require.NotContains(t, err.Error(), key3)

	err = errors.Join(err, errors.New(key2))
	require.Contains(t, err.Error(), "TestFullTrace")
	require.Contains(t, err.Error(), key1)
	require.Contains(t, err.Error(), key2)
	require.NotContains(t, err.Error(), key3)

	var traceErr TraceError
	require.Equal(t, errors.As(err, &traceErr), true, "could not convert error to fullTrace")
	require.Contains(t, traceErr.Line(), "error_test.go:", "unexpected status code")
	require.Contains(t, traceErr.Line(), "TestFullTrace", "unexpected status code")
}
