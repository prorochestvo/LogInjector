package loginjector

import (
	"bytes"
	"github.com/twinj/uuid"
	"strings"
	"testing"
)

func TestCreateLogEvent(t *testing.T) {
	b := bytes.NewBufferString("")
	l, err := NewLogger(logLevelInfo, b)
	if err != nil {
		t.Fatal(err)
	}

	UseAsDefault(l)

	event := CreateLogEvent(logLevelInfo, "pkg", "logevent.go")
	if event == nil {
		t.Fatal("event is nil")
	}

	m := uuid.NewV4().String()

	_, err = event.Write([]byte(m))
	if err != nil {
		t.Fatal(err)
	}

	if s := event.Error(); !strings.Contains(s, m) {
		t.Errorf("unexpected message: %s", s)
	}

	if s := event.MethodTrace(); !strings.Contains(s, "TestCreateLogEvent") {
		t.Errorf("unexpected stack trace: %s", s)
	}

	err = event.Close()
	if err != nil {
		t.Fatal(err)
	}

	if s := b.String(); !strings.Contains(s, m) {
		t.Errorf("unexpected message: %v", s)
	}
}

func TestCreateAndCloseLogEvent(t *testing.T) {
	b := bytes.NewBufferString("")
	l, err := NewLogger(logLevelInfo, b)
	if err != nil {
		t.Fatal(err)
	}

	UseAsDefault(l)

	m := uuid.NewV4().String()

	err = CreateAndCloseLogEvent(logLevelInfo, m, "pkg", "logevent.go")
	if err != nil {
		t.Fatal(err)
	}

	if s := b.String(); !strings.Contains(s, m) {
		t.Errorf("unexpected message: %v", s)
	}
}
