package loginjector

import (
	"bytes"
	"github.com/twinj/uuid"
	"io"
	"log"
	"strings"
	"testing"
)

const (
	logLevelDebug   = 0x00
	logLevelInfo    = 0x01
	logLevelWarning = 0x02
	logLevelSevere  = 0xF0
	//logLevelSilence = 0xFF
)

func TestNewLogger(t *testing.T) {
	l, err := NewLogger(logLevelDebug)
	if err != nil {
		t.Fatalf("NewLogger method returned an error: %v", err)
	}

	if l == nil {
		t.Fatal("NewLogger method returned nil")
	}

	if l.minimumLogLevel != logLevelDebug {
		t.Fatalf("NewLogger method returned a logger with an unexpected minimum log level: %v", l.minimumLogLevel)
	}

	if n := len(l.hooks); n != 0 {
		t.Fatalf("NewLogger method returned a logger with unexpected hooks: %d", n)
	}

	if n := len(l.handlers); n != 1 {
		t.Fatalf("NewLogger method returned a logger with unexpected handlers: %d", n)
	}
}

func TestLogger_SetMinLevel(t *testing.T) {
	l, _ := NewLogger(logLevelDebug)

	if l.minimumLogLevel != logLevelDebug {
		t.Fatalf("SetMinLevel method set an unexpected minimum log level: %v", l.minimumLogLevel)
	}

	l.SetMinLevel(logLevelInfo)

	if l.minimumLogLevel != logLevelInfo {
		t.Fatalf("SetMinLevel method set an unexpected minimum log level: %v", l.minimumLogLevel)
	}
}

func TestLogger_Hook(t *testing.T) {
	m := uuid.NewV4().String()
	b := bytes.NewBufferString("")
	l, _ := NewLogger(logLevelInfo, SilenceHandler())

	if n := len(l.hooks); n != 0 {
		t.Fatalf("Hook method added an unexpected number of hooks: %d", n)
	}

	hookID := l.Hook(b, logLevelWarning)
	if n := len(l.hooks); n != 1 {
		t.Fatalf("Hook method added an unexpected number of hooks: %d", n)
	}
	if l.hooks[0].ID != hookID {
		t.Fatalf("Hook method added a hook with an unexpected ID: %v", l.hooks[0].ID)
	}
	if l.hooks[0].Level != logLevelWarning {
		t.Fatalf("Hook method added a hook with an unexpected minimum log level: %v", l.hooks[0].Level)
	}
	n, err := l.WriteLog(logLevelWarning, []byte(m))
	if err != nil {
		t.Fatalf("Write method returned an error: %v", err)
	}
	if n != len(m) {
		t.Errorf("Write method returned an unexpected length: %d", n)
	}
	if s := b.String(); s != m {
		t.Errorf("Write method wrote an unexpected message: %v", s)
	}

	b.Reset()
	m = uuid.NewV4().String()
	n, err = l.WriteLog(logLevelDebug, []byte(m))
	if err != nil {
		t.Fatalf("Write method returned an error: %v", err)
	}
	if n != 0 {
		t.Errorf("Write method returned an unexpected length: %d", n)
	}
	if s := b.String(); strings.Contains(s, m) {
		t.Errorf("Write method wrote an unexpected message: %v", s)
	}

	b.Reset()
	m = uuid.NewV4().String()
	n, err = l.WriteLog(logLevelSevere, []byte(m))
	if err != nil {
		t.Fatalf("Write method returned an error: %v", err)
	}
	if n == 0 {
		t.Errorf("Write method returned an unexpected length: %d", n)
	}
	if s := b.String(); strings.Contains(s, m) {
		t.Errorf("Write method wrote an unexpected message: %v", s)
	}
}

func TestLogger_Unhook(t *testing.T) {
	m := uuid.NewV4().String()
	b := bytes.NewBufferString("")
	l, _ := NewLogger(logLevelInfo, SilenceHandler())

	hookID := l.Hook(b, logLevelWarning, logLevelSevere, logLevelSevere)

	if n := len(l.hooks); n != 3 {
		t.Fatalf("Hook method added an unexpected number of hooks: %d", n)
	}

	l.Unhook(hookID)

	if n := len(l.hooks); n != 0 {
		t.Fatalf("Unhook method removed an unexpected number of hooks: %d", n)
	}
	n, err := l.WriteLog(logLevelWarning, []byte(m))
	if err != nil {
		t.Fatalf("Write method returned an error: %v", err)
	}
	if n != len(m) {
		t.Errorf("Write method returned an unexpected length: %d", n)
	}
	if s := b.String(); len(s) != 0 {
		t.Errorf("Write method wrote an unexpected message: %v", s)
	}
}

func TestLogger_JoinAs(t *testing.T) {
	m := uuid.NewV4().String()
	b := bytes.NewBufferString("")
	l, _ := NewLogger(logLevelInfo, b)

	if n := len(l.hooks); n != 0 {
		t.Fatalf("JoinAs method added an unexpected number of hooks: %d", n)
	}

	var w io.Writer = nil
	l.JoinAs(logLevelWarning, func(nW io.Writer) {
		w = nW
	})

	n, err := w.Write([]byte(m))
	if err != nil {
		t.Fatalf("Write method returned an error: %v", err)
	}
	if n != len(m) {
		t.Errorf("Write method returned an unexpected length: %d", n)
	}
	if s := b.String(); s != m {
		t.Errorf("Write method wrote an unexpected message: %v", s)
	}

	b.Reset()
	m = uuid.NewV4().String()

	n, err = w.Write([]byte(m))
	if err != nil {
		t.Fatalf("Write method returned an error: %v", err)
	}
	if n != len(m) {
		t.Errorf("Write method returned an unexpected length: %d", n)
	}
	if s := b.String(); s != m {
		t.Errorf("Write method wrote an unexpected message: %v", s)
	}
}

func TestLogger_WriteLog(t *testing.T) {
	m := uuid.NewV4().String()
	b := bytes.NewBufferString("")
	l, _ := NewLogger(logLevelInfo, b)

	n, err := l.WriteLog(logLevelSevere, []byte(m))
	if err != nil {
		t.Fatalf("Write method returned an error: %v", err)
	}
	if n != len(m) {
		t.Errorf("Write method returned an unexpected length: %d", n)
	}
	if s := b.String(); s != m {
		t.Errorf("Write method wrote an unexpected message: %v", s)
	}

	b.Reset()
	m = uuid.NewV4().String()

	n, err = l.WriteLog(logLevelDebug, []byte(m))
	if err != nil {
		t.Fatalf("Write method returned an error: %v", err)
	}
	if n != 0 {
		t.Errorf("Write method returned an unexpected length: %d", n)
	}
	if s := b.String(); len(s) != 0 {
		t.Errorf("Write method wrote an unexpected message: %v", s)
	}

	b.Reset()
	m = uuid.NewV4().String()

	n, err = l.WriteLog(logLevelSevere, []byte(m))
	if err != nil {
		t.Fatalf("Write method returned an error: %v", err)
	}
	if n != len(m) {
		t.Errorf("Write method returned an unexpected length: %d", n)
	}
	if s := b.String(); s != m {
		t.Errorf("Write method wrote an unexpected message: %v", s)
	}
}

func TestLogger_Write(t *testing.T) {
	m := uuid.NewV4().String()
	b := bytes.NewBufferString("")
	l, _ := NewLogger(logLevelInfo, b)

	log.SetOutput(l)

	n, err := l.Write([]byte(m))
	if err != nil {
		t.Fatalf("Write method returned an error: %v", err)
	}
	if n != len(m) {
		t.Errorf("Write method returned an unexpected length: %d", n)
	}
	if s := b.String(); s != m {
		t.Errorf("Write method wrote an unexpected message: %v", s)
	}

	b.Reset()
	m = uuid.NewV4().String()

	n, err = l.Write([]byte(m))
	if err != nil {
		t.Fatalf("Write method returned an error: %v", err)
	}
	if n != len(m) {
		t.Errorf("Write method returned an unexpected length: %d", n)
	}
	if s := b.String(); s != m {
		t.Errorf("Write method wrote an unexpected message: %v", s)
	}

	b.Reset()
	m = uuid.NewV4().String()
	log.Println(m)

	if s := b.String(); !strings.Contains(s, m) {
		t.Errorf("Write method wrote an unexpected message: %v", s)
	}
}
