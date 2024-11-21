package loginjector

import (
	"bytes"
	"github.com/stretchr/testify/require"
	"github.com/twinj/uuid"
	"io"
	"log"
	"strings"
	"sync"
	"testing"
)

const (
	logLevelDebug   LogLevel = 0x00
	logLevelInfo    LogLevel = 0x01
	logLevelWarning LogLevel = 0x02
	logLevelSevere  LogLevel = 0xF0
	//logLevelSilence = 0xFF
)

func TestNewLogger(t *testing.T) {
	l, err := NewLogger(logLevelDebug)
	require.NoError(t, err)
	require.NotEqual(t, nil, l)
	require.Equal(t, logLevelDebug, l.minimumLogLevel)
	require.Equal(t, len(l.hooks), 0, "unexpected hooks count")
	require.Equal(t, len(l.handlers), 1, "unexpected handlers count")
}

func TestLogger_SetMinLevel(t *testing.T) {
	l, err := NewLogger(logLevelDebug)
	require.NoError(t, err)
	require.NotEqual(t, nil, l)
	require.NotEqual(t, logLevelInfo, l.minimumLogLevel)

	l.SetMinLevel(logLevelInfo)
	require.Equal(t, logLevelInfo, l.minimumLogLevel)
}

func TestLogger_Hook(t *testing.T) {
	m := uuid.NewV4().String()
	b := bytes.NewBufferString("")
	l, err := NewLogger(logLevelInfo, SilenceHandler())
	require.NoError(t, err)
	require.NotEqual(t, nil, l)
	require.Equal(t, len(l.hooks), 0, "unexpected number of hooks")

	hookID := l.Hook(b, logLevelWarning)
	require.Equal(t, len(l.hooks), 1, "unexpected number of hooks")
	require.Equal(t, hookID, l.hooks[0].ID, "unexpected hook[0].id")
	require.Equal(t, logLevelWarning, l.hooks[0].Level, "unexpected hook[0].Level")
	n, err := l.WriteLog(logLevelWarning, []byte(m))
	require.NoError(t, err)
	require.Equal(t, len(m), n)
	require.Contains(t, b.String(), m)

	b.Reset()
	m = uuid.NewV4().String()
	n, err = l.WriteLog(logLevelDebug, []byte(m))
	require.NoError(t, err)
	require.Equal(t, 0, n)
	require.NotContains(t, b.String(), m)

	b.Reset()
	m = uuid.NewV4().String()
	n, err = l.WriteLog(logLevelSevere, []byte(m))
	require.NoError(t, err)
	require.Equal(t, len(m), n)
	require.NotContains(t, b.String(), m)
}

func TestLogger_Unhook(t *testing.T) {
	m := uuid.NewV4().String()
	b := bytes.NewBufferString("")
	l, err := NewLogger(logLevelInfo, SilenceHandler())
	require.NoError(t, err)
	require.NotEqual(t, nil, l)

	hookID := l.Hook(b, logLevelWarning, logLevelSevere, logLevelSevere)
	require.Len(t, l.hooks, 3, "unexpected number of hooks")

	l.Unhook(hookID)
	require.Len(t, l.hooks, 0, "unexpected number of hooks")

	n, err := l.WriteLog(logLevelWarning, []byte(m))
	require.NoError(t, err)
	require.Len(t, m, n, "method returned an unexpected length")
	require.Equal(t, len(b.String()), 0, "unexpected message")
}

func TestLogger_JoinAs(t *testing.T) {
	m := uuid.NewV4().String()
	b := bytes.NewBufferString("")
	l, err := NewLogger(logLevelInfo, b)
	require.NoError(t, err)
	require.NotEqual(t, nil, l)

	require.NoError(t, err)
	require.Len(t, l.hooks, 0)

	var w io.Writer = nil
	l.JoinAs(logLevelWarning, func(nW io.Writer) { w = nW })

	n, err := w.Write([]byte(m))
	require.NoError(t, err)
	require.Len(t, m, n)
	require.Equal(t, m, b.String())

	b.Reset()
	m = uuid.NewV4().String()

	n, err = w.Write([]byte(m))
	require.NoError(t, err)
	require.Len(t, m, n)
	require.Equal(t, m, b.String())
}

func TestLogger_WriteLog(t *testing.T) {
	m := uuid.NewV4().String()
	b := bytes.NewBufferString("")
	l, err := NewLogger(logLevelInfo, b)
	require.NoError(t, err)
	require.NotEqual(t, nil, l)

	n, err := l.WriteLog(logLevelSevere, []byte(m))
	require.NoError(t, err)
	require.Len(t, m, n)
	require.Equal(t, m, b.String())

	b.Reset()
	m = uuid.NewV4().String()

	n, err = l.WriteLog(logLevelDebug, []byte(m))
	require.NoError(t, err)
	require.Equal(t, 0, n)
	require.Len(t, b.String(), 0)

	b.Reset()
	m = uuid.NewV4().String()

	n, err = l.WriteLog(logLevelSevere, []byte(m))
	require.NoError(t, err)
	require.Len(t, m, n)
	require.Equal(t, m, b.String())
}

func TestLogger_Write(t *testing.T) {
	m := uuid.NewV4().String()
	b := bytes.NewBufferString("")
	l, err := NewLogger(logLevelInfo, b)
	require.NoError(t, err)
	require.NotEqual(t, nil, l)

	log.SetOutput(l)

	n, err := l.Write([]byte(m))
	require.NoError(t, err)
	require.Len(t, m, n)
	require.Equal(t, m, b.String())

	b.Reset()
	m = uuid.NewV4().String()

	n, err = l.Write([]byte(m))
	require.NoError(t, err)
	require.Len(t, m, n)
	require.Equal(t, m, b.String())

	b.Reset()
	m = uuid.NewV4().String()
	log.Println(m)

	require.Contains(t, b.String(), m)
}

func TestLogger_PrintAndFatal(t *testing.T) {
	m1 := "M1_" + uuid.NewV4().String()
	m2 := "M2_" + uuid.NewV4().String()
	m3 := "M3_" + uuid.NewV4().String()
	m4 := "M4_" + uuid.NewV4().String()
	m5 := "M5_" + uuid.NewV4().String()
	m6 := "M6_" + uuid.NewV4().String()
	m7 := "M7_" + uuid.NewV4().String()
	b := bytes.NewBufferString("")
	l, err := NewLogger(logLevelInfo, b)
	require.NoError(t, err)
	require.NotEqual(t, nil, l)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func(wg *sync.WaitGroup, l *Logger, txt string) {
		defer wg.Done()
		l.Printf(logLevelDebug, "%s", txt)
	}(&wg, l, m1)
	wg.Add(1)
	go func(wg *sync.WaitGroup, l *Logger, txt string) {
		defer wg.Done()
		l.Print(logLevelInfo, txt, uuid.NewV4().String(), uuid.NewV4().String())
	}(&wg, l, m2)
	wg.Add(1)
	go func(wg *sync.WaitGroup, l *Logger, txt string) {
		defer wg.Done()
		l.Printf(logLevelSevere, "%s", txt)
	}(&wg, l, m3)
	wg.Wait()
	wg.Add(1)
	go func(wg *sync.WaitGroup, l *Logger, txt string) {
		defer wg.Done()
		defer func() {
			_ = recover()
		}()
		l.Fatalf(logLevelDebug, "%s", txt)
	}(&wg, l, m4)
	wg.Add(1)
	go func(wg *sync.WaitGroup, l *Logger, txt1, txt2 string) {
		defer wg.Done()
		defer func() {
			_ = recover()
		}()
		l.Fatal(logLevelInfo, txt1, txt2, uuid.NewV4().String())
	}(&wg, l, m5, m6)
	wg.Wait()
	wg.Add(1)
	go func(wg *sync.WaitGroup, l *Logger, txt string) {
		defer wg.Done()
		defer func() {
			_ = recover()
		}()
		l.Fatalf(logLevelSevere, "%s", txt)
	}(&wg, l, m7)
	wg.Wait()

	require.NotEqual(t, 0, len(b.String()))
	require.NotContains(t, b.String(), m1)
	require.Contains(t, b.String(), m2)
	require.Contains(t, b.String(), m3)
	require.NotContains(t, b.String(), m4)
	require.Contains(t, b.String(), m5)
	require.Contains(t, b.String(), m6)
	require.Contains(t, b.String(), m7)
}

func TestLogger_WriterAs(t *testing.T) {
	b := bytes.NewBufferString("")
	l, err := NewLogger(logLevelInfo, b)
	require.NoError(t, err)
	require.NotEqual(t, nil, l)

	w := l.WriterAs(logLevelWarning)
	m := uuid.NewV4().String()
	_, _ = w.Write([]byte(m))
	if s := b.String(); !strings.Contains(s, m) {
		t.Errorf("Write method wrote an unexpected message: %v", s)
	}
	b.Reset()

	w = l.WriterAs(logLevelDebug)
	m = uuid.NewV4().String()
	_, _ = w.Write([]byte(m))
	if s := b.String(); strings.Contains(s, m) {
		t.Errorf("Write method wrote an unexpected message: %v", s)
	}
	b.Reset()

	w = l.WriterAs(logLevelSevere)
	m = uuid.NewV4().String()
	_, _ = w.Write([]byte(m))
	if s := b.String(); !strings.Contains(s, m) {
		t.Errorf("Write method wrote an unexpected message: %v", s)
	}
}
