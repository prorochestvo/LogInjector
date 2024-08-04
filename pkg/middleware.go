package loginjector

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// NewStopwatchHTTPHandler - create new http middleware handler.
//func NewStopwatchHTTPHandler(level LogLevel, nextFunc http.HandlerFunc) (http.HandlerFunc, error) {
//
//}

// NewHttpPayloadHandler - create new http middleware handler.
// This middleware logs the request and response payloads.
// The log is written to the logger.
func NewHttpPayloadHandler(logger *Logger, level LogLevel, nextFunc http.HandlerFunc) (http.HandlerFunc, error) {
	if logger == nil {
		return nil, fmt.Errorf("logger is nil")
	}
	return func(w http.ResponseWriter, r *http.Request) {
		iw, ir, i := newInterceptor(w, r)

		defer func(i *interceptor, inboundedAt time.Time) {
			elapsed := time.Since(inboundedAt)
			payloadSuze := i.payload.Len()
			responseSize := i.response.Len()
			responseCode := i.code

			wg := sync.WaitGroup{}
			wg.Add(1)
			go func(wg *sync.WaitGroup, i *interceptor) {
				defer wg.Done()
				_, err := logger.WriteLog(level, i.Bytes())
				if err != nil {
					println(err.Error())
				}
			}(&wg, i)
			defer wg.Wait()

			fmt.Printf("[%d] %s %s: %0.3f msec; ↓%0.2fKb; ↑%0.2fKb;\n",
				responseCode,
				r.Method,
				r.URL.Path,
				float64(elapsed)/1000000,
				float64(payloadSuze)/1024,
				float64(responseSize)/1024,
			)
		}(i, time.Now().UTC())

		nextFunc(iw, ir)

		if f, ok := w.(http.Flusher); ok && f != nil {
			f.Flush()
		}
	}, nil
}

// newInterceptor - create new http interceptor, that recreate request and response http implementations.
func newInterceptor(w http.ResponseWriter, r *http.Request) (http.ResponseWriter, *http.Request, *interceptor) {
	i := &interceptor{
		payload:        bytes.NewBufferString(""),
		response:       bytes.NewBufferString(""),
		request:        r,
		ResponseWriter: w,
	}

	r.Body = struct {
		io.Reader
		io.Closer
	}{
		Reader: io.TeeReader(r.Body, i.payload),
		Closer: r.Body,
	}

	return i, r, i
}

// interceptor - http interceptor, that includes request and response payloads.
type interceptor struct {
	code     int
	payload  *bytes.Buffer
	response *bytes.Buffer
	request  *http.Request
	http.ResponseWriter
}

// Write - override http.ResponseWriter.Write method for save response payload.
func (i *interceptor) Write(b []byte) (int, error) {
	i.response.Write(b)
	return i.ResponseWriter.Write(b)
}

// WriteHeader - override http.ResponseWriter.WriteHeader method for save http status code.
func (i *interceptor) WriteHeader(statusCode int) {
	i.code = statusCode
	i.ResponseWriter.WriteHeader(statusCode)
}

// Bytes - convert http details to byte array.
func (i *interceptor) Bytes() []byte {
	requestRawQuery := i.request.URL.RawQuery
	if requestRawQuery != "" {
		requestRawQuery = "?" + requestRawQuery
	}
	requestHead := fmt.Sprintf("%s %s%s\n", i.request.Method, i.request.URL.Path, requestRawQuery)
	requestHeadersAsString := headerToString(&i.request.Header)
	requestHeadersAsString = strings.TrimSpace(requestHeadersAsString)
	responseHead := fmt.Sprintf("%s %d %s\n", i.request.Proto, i.code, http.StatusText(i.code))
	responseHeaders := i.Header()
	responseHeadersAsString := headerToString(&responseHeaders)
	responseHeadersAsString = strings.TrimSpace(responseHeadersAsString)

	l := len(requestHead) + len(requestHeadersAsString) + i.payload.Len() + i.response.Len() + len(responseHead) + len(responseHeadersAsString) + 3
	dataset := make([]byte, 0, l)

	dataset = append(dataset, []byte(requestHead)...)
	dataset = append(dataset, []byte(requestHeadersAsString)...)
	dataset = append(dataset, byte('\n'))
	dataset = append(dataset, bytes.TrimSpace(i.payload.Bytes())...)
	dataset = append(dataset, byte('\n'))
	dataset = append(dataset, []byte(responseHead)...)
	dataset = append(dataset, []byte(responseHeadersAsString)...)
	dataset = append(dataset, byte('\n'))
	dataset = append(dataset, bytes.TrimSpace(i.response.Bytes())...)

	return dataset
}

// headerToString - convert http header map to string.
func headerToString(header *http.Header) string {
	res := ""
	for key, v := range *header {
		key = strings.TrimSpace(key)
		val := strings.Join(v, "; ")
		val = strings.TrimSpace(val)
		if key == "Authorization" || key == "Cookie" || key == "Set-Cookie" || key == "" {
			continue
		}
		res += key + ": " + val + "\n"
	}
	return res
}
