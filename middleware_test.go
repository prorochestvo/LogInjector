package loginjector

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/require"
	"github.com/twinj/uuid"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewHttpPayloadHandler(t *testing.T) {
	b := bytes.NewBufferString("")
	l, _ := NewLogger(logLevelInfo, b)

	userAgent := uuid.NewV4().String()
	responseID := uuid.NewV4().String()
	request := uuid.NewV4().String() + strings.Repeat("Z", rand.Int()%16)
	response := uuid.NewV4().String()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/%s?q=%s", uuid.NewV4().String(), uuid.NewV4().String()), bytes.NewBufferString(request))
	req.Header.Set("User-Agent", userAgent)
	res := httptest.NewRecorder()
	httpStatusOK := func(w http.ResponseWriter, r *http.Request) {
		defer CloseOrPanic(r.Body)
		payload, _ := io.ReadAll(r.Body)

		w.WriteHeader(http.StatusOK)
		w.Header().Set("RESPONSE_ID", responseID)
		_, _ = w.Write([]byte(response))
		_, _ = w.Write([]byte("--"))
		_, _ = w.Write([]byte(fmt.Sprintf("PAYLOAD_%d", len(payload))))
	}

	h, err := NewHttpPayloadHandler(l, logLevelInfo, httpStatusOK)
	require.NoError(t, err)

	h(res, req)

	require.Equal(t, http.StatusOK, res.Code, res.Body.String())
	require.Contains(t, res.Body.String(), response, "incorrect response body")
	require.Contains(t, res.Body.String(), fmt.Sprintf("PAYLOAD_%d", len(request)), "incorrect response body")
	require.Greater(t, len(b.String()), 32, "incorrect log context")
	require.Contains(t, b.String(), response, "incorrect log context")
	require.Contains(t, b.String(), request, "incorrect log context")
	require.Contains(t, b.String(), userAgent, "incorrect log context")
	require.Contains(t, b.String(), responseID, "incorrect log context")
	require.Contains(t, b.String(), req.Method, "incorrect log context")
	require.Contains(t, b.String(), req.URL.Path, "incorrect log context")
	require.Contains(t, b.String(), req.URL.RawQuery, "incorrect log context")
	require.Contains(t, b.String(), req.Proto, "incorrect log context")
	require.Contains(t, b.String(), http.StatusText(res.Code), "incorrect log context")
}
