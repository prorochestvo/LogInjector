package loginjector

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/prorochestvo/LogInjector/internal/stacktrace"
	"github.com/twinj/uuid"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestTelegramHandler(t *testing.T) {
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	chatID := os.Getenv("TELEGRAM_BOT_CHAT_ID")

	if len(botToken) == 0 || len(chatID) == 0 {
		t.Skipf("TELEGRAM_BOT_TOKEN or TELEGRAM_BOT_CHAT_ID not set")
	}

	m, s := stacktrace.ExtractMethodTrace()
	s = fmt.Sprintf("%s\n\nSTACKTRACE:\n%s", m, s)

	h := TelegramHandler(botToken, chatID, "test.log", "LogInjector", "<b>demo</b> of telegram handler")
	_, err := h.Write([]byte(s))
	if err != nil {
		t.Fatal(err)
	}
}

func TestCyclicOverwritingFilesHandler(t *testing.T) {
	tmpFolder := path.Join(os.TempDir(), fmt.Sprintf("log-%d", rand.Uint64()))
	err := os.MkdirAll(tmpFolder, 0777)
	if err != nil {
		t.Fatal(err)
	}
	defer func(path string) {
		_ = os.RemoveAll(path)
	}(tmpFolder)

	h := CyclicOverwritingFilesHandler(tmpFolder, "err", 7, 3)

	_, err = h.Write(bytes.Repeat([]byte("1"), 2))
	if err != nil {
		t.Fatal(err)
	}
	files, err := extractFilesOrFail(tmpFolder)
	if err != nil {
		t.Fatal(err)
	}
	if l := len(files); l != 1 {
		t.Fatalf("incorrect files count, got: %d, expected: %d", l, 1)
	}
	if f, ok := files["err.00000001."+defaultFileExtension]; !ok || f != "11\n" {
		t.Fatalf("incorrect file context, got: %s, expected: %s", f, "11\n")
	}

	_, err = h.Write(bytes.Repeat([]byte("2"), 2))
	if err != nil {
		t.Fatal(err)
	}
	files, err = extractFilesOrFail(tmpFolder)
	if err != nil {
		t.Fatal(err)
	}
	if l := len(files); l != 1 {
		t.Fatalf("incorrect files count, got: %d, expected: %d", l, 1)
	}
	if f, ok := files["err.00000001."+defaultFileExtension]; !ok || f != "11\n22\n" {
		t.Fatalf("incorrect file context, got: %s, expected: %s", f, "11\n22\n")
	}

	_, err = h.Write(bytes.Repeat([]byte("3"), 2))
	if err != nil {
		t.Fatal(err)
	}
	files, err = extractFilesOrFail(tmpFolder)
	if err != nil {
		t.Fatal(err)
	}
	if l := len(files); l != 1 {
		t.Fatalf("incorrect files count, got: %d, expected: %d", l, 2)
	}
	if f, ok := files["err.00000001."+defaultFileExtension]; !ok || f != "11\n22\n33\n" {
		t.Fatalf("incorrect file context, got: %s, expected: %s", f, "11\n22\n33\n")
	}

	_, err = h.Write([]byte("4"))
	if err != nil {
		t.Fatal(err)
	}
	files, err = extractFilesOrFail(tmpFolder)
	if err != nil {
		t.Fatal(err)
	}
	if l := len(files); l != 2 {
		t.Fatalf("incorrect files count, got: %d, expected: %d", l, 2)
	}
	if f, ok := files["err.00000001."+defaultFileExtension]; !ok || f != "11\n22\n33\n" {
		t.Fatalf("incorrect file context, got: %s, expected: %s", f, "11\n22\n33\n")
	}
	if f, ok := files["err.00000002."+defaultFileExtension]; !ok || f != "4\n" {
		t.Fatalf("incorrect file context, got: %s, expected: %s", f, "4\n")
	}

	_, err = h.Write(bytes.Repeat([]byte("5"), 5))
	if err != nil {
		t.Fatal(err)
	}
	files, err = extractFilesOrFail(tmpFolder)
	if err != nil {
		t.Fatal(err)
	}
	if l := len(files); l != 2 {
		t.Fatalf("incorrect files count, got: %d, expected: %d", l, 2)
	}
	if f, ok := files["err.00000001."+defaultFileExtension]; !ok || f != "11\n22\n33\n" {
		t.Fatalf("incorrect file context, got: %s, expected: %s", f, "11\n22\n33\n")
	}
	if f, ok := files["err.00000002."+defaultFileExtension]; !ok || f != "4\n55555\n" {
		t.Fatalf("incorrect file context, got: %s, expected: %s", f, "4\n55555\n")
	}

	_, err = h.Write(bytes.Repeat([]byte("6"), 10))
	if err != nil {
		t.Fatal(err)
	}
	_, err = h.Write(bytes.Repeat([]byte("7"), 10))
	if err != nil {
		t.Fatal(err)
	}
	files, err = extractFilesOrFail(tmpFolder)
	if err != nil {
		t.Fatal(err)
	}
	if l := len(files); l != 3 {
		t.Fatalf("incorrect files count, got: %d, expected: %d", l, 2)
	}
	if f, ok := files["err.00000002."+defaultFileExtension]; !ok || f != "4\n55555\n" {
		t.Fatalf("incorrect file context, got: %s, expected: %s", f, "4\n55555\n")
	}
	if f, ok := files["err.00000003."+defaultFileExtension]; !ok || f != "6666666666\n" {
		t.Fatalf("incorrect file context, got: %s, expected: %s", f, "6666666666\n")
	}
	if f, ok := files["err.00000004."+defaultFileExtension]; !ok || f != "7777777777\n" {
		t.Fatalf("incorrect file context, got: %s, expected: %s", f, "7777777777\n")
	}
}

func TestReinitCyclicOverwritingFilesHandler(t *testing.T) {
	// TODO: Implement reinit last file state after restart\recreate handler
	t.Skipf("test not implemented")
}

func TestCyclicOverwritingFilesHandlerForRaceCondition(t *testing.T) {
	tmpFolder := path.Join(os.TempDir(), fmt.Sprintf("log-%d", rand.Uint64()))
	err := os.MkdirAll(tmpFolder, 0777)
	if err != nil {
		t.Fatal(err)
	}
	defer func(path string) {
		_ = os.RemoveAll(path)
	}(tmpFolder)

	h := CyclicOverwritingFilesHandler(tmpFolder, "err", 70, 10)
	messages := make([]string, 0)
	for i := 0; i < 16; i++ {
		messages = append(messages, fmt.Sprintf("%0.3d->>%s", i, uuid.NewV4().String()))
	}

	wg := sync.WaitGroup{}
	for _, m := range messages {
		wg.Add(1)
		go func(txt string) {
			defer wg.Done()
			if _, e := h.Write([]byte(txt)); e != nil {
				err = errors.Join(err, e)
			}
		}(m)
	}
	wg.Wait()
	if err != nil {
		t.Fatal(err)
	}

	files, err := extractFilesOrFail(tmpFolder)
	if err != nil {
		t.Fatal(err)
	}

	allContext := ""
	for f, ctx := range files {
		allContext += fmt.Sprintf("\n%s:\n%s\n", f, ctx)
	}
	allContext = strings.TrimSpace(allContext)

	for _, m := range messages {
		if !strings.Contains(allContext, m) {
			t.Errorf("not found %s in %s", m, allContext)
		}
	}
}

func TestFilePerDaysHandler(t *testing.T) {
	// TODO: Implement
	t.Skipf("test not implemented")
}

func TestFilePerDaysHandlerForRaceCondition(t *testing.T) {
	// TODO: Implement
	t.Skipf("test not implemented")
}

func TestVerifyFiles(t *testing.T) {
	tmpFolder := path.Join(os.TempDir(), fmt.Sprintf("log-%d", rand.Uint64()))
	err := os.MkdirAll(tmpFolder, 0777)
	if err != nil {
		t.Fatal(err)
	}
	defer func(path string) {
		_ = os.RemoveAll(path)
	}(tmpFolder)

	err = verifyFiles(tmpFolder, 3)
	if err != nil {
		t.Fatal(err)
	}

	files, err := extractFilesOrFail(tmpFolder)
	if err != nil {
		t.Fatal(err)
	}
	if l := len(files); l != 0 {
		t.Fatalf("incorrect files count, got: %d, expected: %d", l, 0)
	}

	for i := 0; i < 4; i++ {
		err = os.WriteFile(path.Join(tmpFolder, fmt.Sprintf("%d.%s", rand.Int31(), defaultFileExtension)), []byte("-"), 0777)
		if err != nil {
			t.Fatal(err)
		}
	}

	err = verifyFiles(tmpFolder, 3)
	if err != nil {
		t.Fatal(err)
	}

	files, err = extractFilesOrFail(tmpFolder)
	if err != nil {
		t.Fatal(err)
	}
	if l := len(files); l != 3 {
		t.Fatalf("incorrect files count, got: %d, expected: %d", l, 3)
	}
}

func extractFilesOrFail(folder string) (map[string]string, error) {
	files, err := filepath.Glob(path.Join(folder, "*."+defaultFileExtension))
	if err != nil || len(files) == 0 {
		return nil, err
	}
	r := make(map[string]string, 0)
	for _, filePath := range files {
		b, e := os.ReadFile(filePath)
		_, filePath = path.Split(filePath)
		if e != nil {
			r[filePath] = e.Error()
		} else {
			r[filePath] = string(b)
		}
	}
	return r, nil
}
