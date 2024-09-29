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
	"runtime"
	"strconv"
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
	tmpFolder := t.TempDir()
	h := CyclicOverwritingFilesHandler(tmpFolder, "err", 7, 3)

	_, err := h.Write(bytes.Repeat([]byte("1"), 2))
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
	tmpFolder := t.TempDir()

	h := CyclicOverwritingFilesHandler(tmpFolder, "err", 3, 3)
	_, err := h.Write(bytes.Repeat([]byte("1"), 3))
	require.NoError(t, err)
	_, err = h.Write(bytes.Repeat([]byte("2"), 3))
	require.NoError(t, err)
	_, err = h.Write(bytes.Repeat([]byte("3"), 3))
	require.NoError(t, err)

	files, err := extractFilesOrFail(tmpFolder)
	require.NoError(t, err)
	require.Len(t, files, 3, "incorrect files count")
	f, ok := files["err.00000001."+defaultFileExtension]
	require.Equal(t, true, ok, "file context not found")
	require.Equal(t, "111\n", f, "incorrect file context")
	f, ok = files["err.00000002."+defaultFileExtension]
	require.Equal(t, true, ok, "file context not found")
	require.Equal(t, "222\n", f, "incorrect file context")
	f, ok = files["err.00000003."+defaultFileExtension]
	require.Equal(t, true, ok, "file context not found")
	require.Equal(t, "333\n", f, "incorrect file context")

	_, err = h.Write(bytes.Repeat([]byte("4"), 3))
	require.NoError(t, err)

	files, err = extractFilesOrFail(tmpFolder)
	require.NoError(t, err)
	require.Len(t, files, 3, "incorrect files count")
	f, ok = files["err.00000001."+defaultFileExtension]
	require.Equal(t, false, ok, "file context not found")
	f, ok = files["err.00000002."+defaultFileExtension]
	require.Equal(t, true, ok, "file context not found")
	require.Equal(t, "222\n", f, "incorrect file context")
	f, ok = files["err.00000003."+defaultFileExtension]
	require.Equal(t, true, ok, "file context not found")
	require.Equal(t, "333\n", f, "incorrect file context")
	f, ok = files["err.00000004."+defaultFileExtension]
	require.Equal(t, true, ok, "file context not found")
	require.Equal(t, "444\n", f, "incorrect file context")

}

func TestCyclicOverwritingFilesHandlerForRaceCondition(t *testing.T) {
	tmpFolder := t.TempDir()
	h := CyclicOverwritingFilesHandler(tmpFolder, "err", 70, 10)
	messages := make([]string, 0)
	for i := 0; i < 16; i++ {
		messages = append(messages, fmt.Sprintf("%0.3d->>%s", i, uuid.NewV4().String()))
	}
	var err error
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
	tmpFolder := t.TempDir()

	m := sync.Mutex{}
	fileNumber := -1
	fileNameGenerator := func() string {
		startingDay := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
		m.Lock()
		defer m.Unlock()
		fileNumber++
		return startingDay.AddDate(0, 0, fileNumber).Format("2006-01-02")
	}
	handler := FileByFormatHandler(tmpFolder, 4, fileNameGenerator)
	expectedFileContexts := []string{
		"f1:i0001", "f2:i0001",
		"f3:i0001", "f4:i0001",
		"f5:i0001", "f6:i0001",
		"f7:i0001", "f8:i0001",
		"f9:i0001", "f0:i0001",
	}

	for _, fileContext := range expectedFileContexts {
		_, err := handler.Write([]byte(fileContext))
		require.NoError(t, err)
	}

	expectedDataset := map[string]string{
		"2000-01-10.log": "f0:i0001\n",
		"2000-01-09.log": "f9:i0001\n",
		"2000-01-08.log": "f8:i0001\n",
		"2000-01-07.log": "f7:i0001\n",
	}

	files, err := extractFilesOrFail(tmpFolder)
	require.NoError(t, err)
	require.Len(t, files, 4, "incorrect files count")

	for expectedFileName, expectedFileContext := range expectedDataset {
		actualContext, exists := files[expectedFileName]
		require.True(t, exists)
		require.Equal(t, actualContext, expectedFileContext, expectedFileName)
	}
}

func TestFileByFormatHandlerV2(t *testing.T) {
	startedAt := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	tmpFolder := t.TempDir()

	dataset := []string{
		"f1:i0001", "f1:i0002",
		"f2:i0001", "f2:i0002",
		"f3:i0001", "f3:i0002",
		"f4:i0001", "f4:i0002",
		"f5:i0001", "f5:i0002",
	}
	expectedDataset := map[string]string{
		"2000-01-03.log": "f3:i0001\nf3:i0002\n",
		"2000-01-04.log": "f4:i0001\nf4:i0002\n",
		"2000-01-05.log": "f5:i0001\nf5:i0002\n",
	}

	fileIndexMutex := sync.Mutex{}
	fileIndex := 0
	handler := FileByFormatHandler(tmpFolder, 3, func() string {
		fileIndexMutex.Lock()
		defer fileIndexMutex.Unlock()
		d := startedAt.Add(time.Hour * 24 * time.Duration(fileIndex>>1))
		fileIndex++
		return d.Format(time.DateOnly)
	})

	for _, d := range dataset {
		_, err := handler.Write([]byte(d))
		require.NoError(t, err)
	}

	files, err := extractFilesOrFail(tmpFolder)
	require.NoError(t, err)
	require.Len(t, files, 3, "incorrect files count")
	for fileName, expectedFileContext := range expectedDataset {
		if strings.HasPrefix(fileName, "ignored") {
			continue
		}
		actualData, fExists := files[fileName]
		require.True(t, fExists, fileName)
		require.Equal(t, expectedFileContext, actualData, fileName)
	}
}

func TestFileByFormatHandlerForRaceCondition(t *testing.T) {
	tmpFolder := t.TempDir()

	handlerFileName := "2000-01-10"
	handler := FileByFormatHandler(tmpFolder, 1, func() string { return handlerFileName })

	expectedFileContexts := make([]string, 100)
	for i := range expectedFileContexts {
		expectedFileContexts[i] = strconv.Itoa(i) + ":" + uuid.NewV4().String()
	}

	var wg sync.WaitGroup
	for _, fileContext := range expectedFileContexts {
		wg.Add(1)
		go func(w io.Writer) {
			defer wg.Done()
			_, err := w.Write([]byte(fileContext))
			require.NoError(t, err)
		}(handler)
	}
	wg.Wait() // waiting when all jobs will be done

	files, err := extractFilesOrFail(tmpFolder)
	require.NoError(t, err)
	require.Len(t, files, 1, "incorrect files count")

	fileContext, ok := files[handlerFileName+".log"]
	require.True(t, ok)
	require.NotEmpty(t, fileContext)

	for _, expectedContext := range expectedFileContexts {
		require.Contains(t, fileContext, expectedContext)
	}
}

func TestVerifyFiles(t *testing.T) {
	tmpFolder := t.TempDir()

	err := verifyFiles(tmpFolder, 3)
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
		if runtime.GOOS == "windows" && strings.Contains(filePath, "\\") {
			// in some cases the filepath library is not able to handle the backslashes correctly
			// so we need to replace them with forward slashes
			filePath = strings.ReplaceAll(filePath, "\\", "/")
		}
		filePath = path.Base(filePath)
		if e != nil {
			r[filePath] = e.Error()
		} else {
			r[filePath] = string(b)
		}
	}
	return r, nil
}
