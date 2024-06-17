package loginjector

import (
	"bytes"
	"fmt"
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
	if err != nil {
		t.Fatal(err)
	}

	h(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("incorrect response code: got %d; expected %d", res.Code, http.StatusOK)
	}
	if s := res.Body.String(); strings.Contains(s, response) == false || strings.Contains(s, fmt.Sprintf("PAYLOAD_%d", len(request))) == false {
		t.Fatalf("incorrect response body: %s", s)
	}
	if s := b.String(); len(s) == 0 {
		t.Fatal("log context is empty")
	} else if !strings.Contains(s, response) {
		t.Fatalf("incorrect log context, response payload not found: %s", s)
	} else if !strings.Contains(s, request) {
		t.Fatalf("incorrect log context, request payload not found: %s", s)
	} else if !strings.Contains(s, userAgent) {
		t.Fatalf("incorrect log context, request headers not found: %s", s)
	} else if !strings.Contains(s, responseID) {
		t.Fatalf("incorrect log context, response headers not found: %s", s)
	} else if !strings.Contains(s, req.Method) || !strings.Contains(s, req.URL.Path) || !strings.Contains(s, req.URL.RawQuery) {
		t.Fatalf("incorrect log context, request head not found: %s", s)
	} else if !strings.Contains(s, req.Proto) || !strings.Contains(s, http.StatusText(res.Code)) {
		t.Fatalf("incorrect log context, response head not found: %s", s)
	}
}
