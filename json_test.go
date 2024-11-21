package loginjector

import (
	"bytes"
	"github.com/stretchr/testify/require"
	"github.com/twinj/uuid"
	"strings"
	"testing"
)

func TestJsonEncode(t *testing.T) {
	output := &writerReaderCloser{Buffer: bytes.NewBufferString(""), IsClosed: false}
	obj := struct {
		Key string `json:"key"`
	}{
		Key: uuid.NewV4().String(),
	}
	err := JsonEncode(output, obj)
	require.NoError(t, err, output.String())
	require.Equal(t, `{"key":"`+obj.Key+`"}`, strings.TrimSpace(output.String()))
	require.Equal(t, false, output.IsClosed)
}

func TestJsonEncode_indent(t *testing.T) {
	output := &writerReaderCloser{Buffer: bytes.NewBufferString(""), IsClosed: false}
	obj := struct {
		Key string `json:"key"`
	}{
		Key: uuid.NewV4().String(),
	}
	err := JsonEncode(output, obj, 2)
	require.NoError(t, err, output.String())
	require.Equal(t, `{
  "key": "`+obj.Key+`"
}`, strings.TrimSpace(output.String()))
	require.Equal(t, false, output.IsClosed)
}

func TestJsonDecode(t *testing.T) {
	key := uuid.NewV4().String()
	input := &writerReaderCloser{Buffer: bytes.NewBufferString(`{"key":"` + key + `"}`), IsClosed: false}
	obj := struct {
		Key string `json:"key"`
	}{}
	err := JsonDecode(input, &obj)
	require.NoError(t, err)
	require.Equal(t, key, obj.Key)
	require.Equal(t, false, input.IsClosed)
}

func TestJsonEncodeAndClose(t *testing.T) {
	output := &writerReaderCloser{Buffer: bytes.NewBufferString(""), IsClosed: false}
	obj := struct {
		Key string `json:"key"`
	}{
		Key: uuid.NewV4().String(),
	}
	err := JsonEncodeAndClose(output, obj)
	require.NoError(t, err, output.String())
	require.Equal(t, `{"key":"`+obj.Key+`"}`, strings.TrimSpace(output.String()))
	require.Equal(t, true, output.IsClosed)
}

func TestJsonEncodeAndClose_Indent(t *testing.T) {
	output := &writerReaderCloser{Buffer: bytes.NewBufferString(""), IsClosed: false}
	obj := struct {
		Key string `json:"key"`
	}{
		Key: uuid.NewV4().String(),
	}
	err := JsonEncodeAndClose(output, obj, 2)
	require.NoError(t, err, output.String())
	require.Equal(t, `{
  "key": "`+obj.Key+`"
}`, strings.TrimSpace(output.String()))
	require.Equal(t, true, output.IsClosed)
}

func TestJsonDecodeAndClose(t *testing.T) {
	key := uuid.NewV4().String()
	input := &writerReaderCloser{Buffer: bytes.NewBufferString(`{"key":"` + key + `"}`), IsClosed: false}
	obj := struct {
		Key string `json:"key"`
	}{}
	err := JsonDecodeAndClose(input, &obj)
	require.NoError(t, err)
	require.Equal(t, key, obj.Key)
	require.Equal(t, true, input.IsClosed)
}

func TestJsonEncodeEx(t *testing.T) {
	output := &writerReaderCloser{Buffer: bytes.NewBufferString(""), IsClosed: false}
	obj := struct {
		Key string `json:"key"`
	}{
		Key: uuid.NewV4().String(),
	}
	raw, err := JsonEncodeEx(output, obj)
	require.NoError(t, err, output.String())
	require.Equal(t, `{"key":"`+obj.Key+`"}`, strings.TrimSpace(output.String()))
	require.Equal(t, `{"key":"`+obj.Key+`"}`, strings.TrimSpace(string(raw)))
	require.Equal(t, false, output.IsClosed)
}

func TestJsonEncodeEx_Indent(t *testing.T) {
	output := &writerReaderCloser{Buffer: bytes.NewBufferString(""), IsClosed: false}
	obj := struct {
		Key string `json:"key"`
	}{
		Key: uuid.NewV4().String(),
	}
	raw, err := JsonEncodeEx(output, obj, 2)
	require.NoError(t, err, output.String())
	require.Equal(t, `{
  "key": "`+obj.Key+`"
}`, strings.TrimSpace(output.String()))
	require.Equal(t, `{
  "key": "`+obj.Key+`"
}`, strings.TrimSpace(string(raw)))
	require.Equal(t, false, output.IsClosed)
}

func TestJsonDecodeEx(t *testing.T) {
	key := uuid.NewV4().String()
	input := &writerReaderCloser{Buffer: bytes.NewBufferString(`{"key":"` + key + `"}`), IsClosed: false}
	obj := struct {
		Key string `json:"key"`
	}{}
	raw, err := JsonDecodeEx(input, &obj)
	require.NoError(t, err)
	require.Equal(t, key, obj.Key)
	require.Equal(t, `{"key":"`+key+`"}`, string(raw))
	require.Equal(t, false, input.IsClosed)
}

func TestJsonEncodeAndCloseEx(t *testing.T) {
	output := &writerReaderCloser{Buffer: bytes.NewBufferString(""), IsClosed: false}
	obj := struct {
		Key string `json:"key"`
	}{
		Key: uuid.NewV4().String(),
	}
	raw, err := JsonEncodeAndCloseEx(output, obj)
	require.NoError(t, err, output.String())
	require.Equal(t, `{"key":"`+obj.Key+`"}`, strings.TrimSpace(output.String()))
	require.Equal(t, `{"key":"`+obj.Key+`"}`, strings.TrimSpace(string(raw)))
	require.Equal(t, true, output.IsClosed)
}

func TestJsonEncodeAndCloseEx_Indent(t *testing.T) {
	output := &writerReaderCloser{Buffer: bytes.NewBufferString(""), IsClosed: false}
	obj := struct {
		Key string `json:"key"`
	}{
		Key: uuid.NewV4().String(),
	}
	raw, err := JsonEncodeAndCloseEx(output, obj, 2)
	require.NoError(t, err, output.String())
	require.Equal(t, `{
  "key": "`+obj.Key+`"
}`, strings.TrimSpace(output.String()))
	require.Equal(t, `{
  "key": "`+obj.Key+`"
}`, strings.TrimSpace(string(raw)))
	require.Equal(t, true, output.IsClosed)
}

func TestJsonDecodeAndCloseEx(t *testing.T) {
	key := uuid.NewV4().String()
	input := &writerReaderCloser{Buffer: bytes.NewBufferString(`{"key":"` + key + `"}`), IsClosed: false}
	obj := struct {
		Key string `json:"key"`
	}{}
	raw, err := JsonDecodeAndCloseEx(input, &obj)
	require.NoError(t, err)
	require.Equal(t, key, obj.Key)
	require.Equal(t, `{"key":"`+key+`"}`, string(raw))
	require.Equal(t, true, input.IsClosed)
}

type writerReaderCloser struct {
	IsClosed bool
	*bytes.Buffer
}

func (wrc *writerReaderCloser) Close() error {
	wrc.IsClosed = true
	return nil
}
