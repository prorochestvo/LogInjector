package loginjector

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twinj/uuid"
	"io"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestTelegramHandler(t *testing.T) {
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	chatID := os.Getenv("TELEGRAM_BOT_CHAT_ID")

	if len(botToken) == 0 || len(chatID) == 0 {
		t.Skipf("TELEGRAM_BOT_TOKEN or TELEGRAM_BOT_CHAT_ID not set")
	}

	m := time.Now().UTC().Format(time.RFC3339) + ": 14C225CB-9BE0-40D8-8FB3-6218FE17AE53"

	h := TelegramHandler(botToken, chatID, "test.log", "LogInjector", "<b>demo</b> of telegram handler")
	_, err := h.Write([]byte(m))
	require.NoError(t, err)
}

func TestCyclicOverwritingFilesHandler(t *testing.T) {
	tmpFolder := path.Join(os.TempDir(), fmt.Sprintf("log-%d", rand.Uint64()))
	err := os.MkdirAll(tmpFolder, os.ModePerm)
	require.NoError(t, err)
	defer func(path string) { _ = os.RemoveAll(path) }(tmpFolder)

	h := CyclicOverwritingFilesHandler(tmpFolder, "err", 7, 3)

	_, err = h.Write(bytes.Repeat([]byte("1"), 2))
	require.NoError(t, err)
	files, err := extractFilesOrFail(tmpFolder)
	require.NoError(t, err)
	require.Len(t, files, 1, "incorrect files count")
	f, ok := files["err.00000001."+defaultFileExtension]
	require.Equal(t, true, ok, "file context not found")
	require.Equal(t, "11\n", f, "incorrect file context")

	_, err = h.Write(bytes.Repeat([]byte("2"), 2))
	require.NoError(t, err)
	files, err = extractFilesOrFail(tmpFolder)
	require.NoError(t, err)
	require.Len(t, files, 1, "incorrect files count")
	f, ok = files["err.00000001."+defaultFileExtension]
	require.Equal(t, true, ok, "file context not found")
	require.Equal(t, "11\n22\n", f, "incorrect file context")

	_, err = h.Write(bytes.Repeat([]byte("3"), 2))
	require.NoError(t, err)
	files, err = extractFilesOrFail(tmpFolder)
	require.NoError(t, err)
	require.Len(t, files, 1, "incorrect files count")
	f, ok = files["err.00000001."+defaultFileExtension]
	require.Equal(t, true, ok, "file context not found")
	require.Equal(t, "11\n22\n33\n", f, "incorrect file context")

	_, err = h.Write([]byte("4"))
	require.NoError(t, err)
	files, err = extractFilesOrFail(tmpFolder)
	require.NoError(t, err)
	require.Len(t, files, 2, "incorrect files count")
	f, ok = files["err.00000001."+defaultFileExtension]
	require.Equal(t, true, ok, "file context not found")
	require.Equal(t, "11\n22\n33\n", f, "incorrect file context")
	f, ok = files["err.00000002."+defaultFileExtension]
	require.Equal(t, true, ok, "file context not found")
	require.Equal(t, "4\n", f, "incorrect file context")

	_, err = h.Write(bytes.Repeat([]byte("5"), 5))
	require.NoError(t, err)
	files, err = extractFilesOrFail(tmpFolder)
	require.NoError(t, err)
	require.Len(t, files, 2, "incorrect files count")
	f, ok = files["err.00000001."+defaultFileExtension]
	require.Equal(t, true, ok, "file context not found")
	require.Equal(t, "11\n22\n33\n", f, "incorrect file context")
	f, ok = files["err.00000002."+defaultFileExtension]
	require.Equal(t, true, ok, "file context not found")
	require.Equal(t, "4\n55555\n", f, "incorrect file context")

	_, err = h.Write(bytes.Repeat([]byte("6"), 10))
	require.NoError(t, err)
	_, err = h.Write(bytes.Repeat([]byte("7"), 10))
	require.NoError(t, err)
	files, err = extractFilesOrFail(tmpFolder)
	require.NoError(t, err)
	require.Len(t, files, 3, "incorrect files count")
	f, ok = files["err.00000002."+defaultFileExtension]
	require.Equal(t, true, ok, "file context not found")
	require.Equal(t, "4\n55555\n", f, "incorrect file context")
	f, ok = files["err.00000003."+defaultFileExtension]
	require.Equal(t, true, ok, "file context not found")
	require.Equal(t, "6666666666\n", f, "incorrect file context")
	f, ok = files["err.00000004."+defaultFileExtension]
	require.Equal(t, true, ok, "file context not found")
	require.Equal(t, "7777777777\n", f, "incorrect file context")
}

func TestReinitCyclicOverwritingFilesHandler(t *testing.T) {
	// TODO: Implement reinit last file state after restart\recreate handler
	t.Skipf("test not implemented")
}

func TestCyclicOverwritingFilesHandlerForRaceCondition(t *testing.T) {
	tmpFolder := path.Join(os.TempDir(), fmt.Sprintf("log-%d", rand.Uint64()))
	err := os.MkdirAll(tmpFolder, os.ModePerm)
	require.NoError(t, err)
	defer func(path string) { _ = os.RemoveAll(path) }(tmpFolder)

	h := CyclicOverwritingFilesHandler(tmpFolder, "err", 70, 10)
	messages := make([]string, 0)
	for i := 0; i < 16; i++ {
		messages = append(messages, fmt.Sprintf("%0.3d->>%s", i, uuid.NewV4().String()))
	}

	wg := sync.WaitGroup{}
	for _, m := range messages {
		wg.Add(1)
		go func(wg *sync.WaitGroup, w io.Writer, txt string) {
			defer wg.Done()
			if _, e := w.Write([]byte(txt)); e != nil {
				err = errors.Join(err, e)
			}
		}(&wg, h, m)
	}
	wg.Wait()
	require.NoError(t, err)

	files, err := extractFilesOrFail(tmpFolder)
	require.NoError(t, err)

	allContext := ""
	for f, ctx := range files {
		allContext += fmt.Sprintf("\n%s:\n%s\n", f, ctx)
	}
	allContext = strings.TrimSpace(allContext)

	for _, m := range messages {
		if !strings.Contains(allContext, m) {
			assert.Containsf(t, allContext, m, "not found %s in %s", m, allContext)
		}
	}
}

func TestFileByFormatHandler(t *testing.T) {

	tmpFolder := path.Join(os.TempDir(), fmt.Sprintf("log-%d", rand.Uint64()))
	err := os.MkdirAll(tmpFolder, os.ModePerm)
	require.NoError(t, err)
	defer func(path string) { _ = os.RemoveAll(path) }(tmpFolder)

	m := sync.Mutex{}
	fileNumber := 0
	fileNameGenerator := func() string {
		m.Lock()
		defer m.Unlock()
		fileNumber++
		return time.Date(2000, 1, fileNumber, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
	}
	for i := 0; i < 40; i++ {
		file := FileByFormatHandler(tmpFolder, 4, fileNameGenerator)
		file.Write([]byte("Hello, world"))
	}

	myDaysToKeep := []string{"2000-02-09", "2000-0-08", "2000-02-07", "2000-02-06"}
	var myFilesToKeep []string
	for _, dateString := range myDaysToKeep {
		file_name := path.Join(tmpFolder, fmt.Sprintf("%s.%s", dateString, defaultFileExtension))
		//myFilesToKeep = append(myFilesToKeep, file_name)
		myFilesToKeep = append(myFilesToKeep, strings.ReplaceAll(file_name, "/", "\\"))
	}

	files, err := extractFilesOrFail(tmpFolder)
	require.NoError(t, err)
	require.Len(t, files, 4, "incorrect files count")

	err = validateFileNames(files, myFilesToKeep)
	require.NoError(t, err)
}

func validateFileNames(m map[string]string, keys []string) error {
	for _, key := range keys {
		if _, exists := m[key]; !exists {
			return fmt.Errorf("key %q from slice is not found in the map", key)
		}
	}
	return nil
}

func TestFilePerDaysHandlerForRaceCondition(t *testing.T) {
	// TODO: Implement
	t.Skipf("test not implemented")
}

func TestVerifyFiles(t *testing.T) {
	tmpFolder := path.Join(os.TempDir(), fmt.Sprintf("log-%d", rand.Uint64()))
	err := os.MkdirAll(tmpFolder, os.ModePerm)
	require.NoError(t, err)
	defer func(path string) { _ = os.RemoveAll(path) }(tmpFolder)

	err = verifyFiles(tmpFolder, 3)
	require.NoError(t, err)

	files, err := extractFilesOrFail(tmpFolder)
	require.NoError(t, err)
	require.Len(t, files, 0, "incorrect files count")

	for i := 0; i < 4; i++ {
		err = os.WriteFile(path.Join(tmpFolder, fmt.Sprintf("%d.%s", rand.Int31(), defaultFileExtension)), []byte("-"), os.ModePerm)
		require.NoError(t, err)
	}

	err = verifyFiles(tmpFolder, 3)
	require.NoError(t, err)

	files, err = extractFilesOrFail(tmpFolder)
	require.NoError(t, err)
	require.Len(t, files, 3, "incorrect files count")
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
