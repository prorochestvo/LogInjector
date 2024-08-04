package loginjector

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
)

func JsonEncode(w io.Writer, v any, indent ...int) error {
	encoder := json.NewEncoder(w)
	if len(indent) > 0 && indent[0] > 0 {
		encoder.SetIndent("", strings.Repeat(" ", indent[0]))
	}
	return encoder.Encode(v)
}

func JsonDecode(r io.Reader, v any) error {
	return json.NewDecoder(r).Decode(&v)
}

func JsonEncodeAndClose(w io.WriteCloser, v any, indent ...int) (err error) {
	defer func(c io.Closer) { errors.Join(err, c.Close()) }(w)
	err = JsonEncode(w, v, indent...)
	return
}

func JsonDecodeAndClose(r io.ReadCloser, v any) (err error) {
	defer func(c io.Closer) { errors.Join(err, c.Close()) }(r)
	err = JsonDecode(r, v)
	return
}

func JsonEncodeEx(w io.Writer, v any, indent ...int) ([]byte, error) {
	buf := bytes.NewBufferString("")
	encoder := json.NewEncoder(io.MultiWriter(w, buf))
	if len(indent) > 0 && indent[0] > 0 {
		encoder.SetIndent("", strings.Repeat(" ", indent[0]))
	}
	err := encoder.Encode(&v)
	return buf.Bytes(), err
}

func JsonDecodeEx(r io.Reader, v any) ([]byte, error) {
	buf := bytes.NewBufferString("")
	err := json.NewDecoder(io.TeeReader(r, buf)).Decode(&v)
	return buf.Bytes(), err
}

func JsonEncodeAndCloseEx(w io.WriteCloser, v any, indent ...int) (raw []byte, err error) {
	defer func(c io.Closer) { errors.Join(err, c.Close()) }(w)
	raw, err = JsonEncodeEx(w, v, indent...)
	return
}

func JsonDecodeAndCloseEx(r io.ReadCloser, v any) (raw []byte, err error) {
	defer func(c io.Closer) { errors.Join(err, c.Close()) }(r)
	raw, err = JsonDecodeEx(r, v)
	return
}
