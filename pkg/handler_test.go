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
		startingDay := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
		m.Lock()
		defer m.Unlock()
		fileNumber++
		// TODO: REVIEW: 0-39 iterations will out of range of days for January.
		// TODO: REVIEW: time.Date is smart enough to handle this,
		// TODO: REVIEW: but it's better to use a more realistic date range
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
		_, err = handler.Write([]byte(fileContext))
		require.NoError(t, err)
	}

	// TODO: REVIEW: here we are creating a process handler, non required recreate it

	myFilesToKeep := map[string]string{
		"2000-01-10.log": "f0:i0001\n",
		"2000-01-09.log": "f9:i0001\n",
		"2000-01-08.log": "f8:i0001\n",
		"2000-01-07.log": "f7:i0001\n",
	}

	files, err := extractFilesOrFail(tmpFolder)
	require.NoError(t, err)
	require.Len(t, files, 4, "incorrect files count")

	for fileName, context := range files {
		fileNameParts := strings.Split(fileName, "\\")
		fileNameShort := fileNameParts[len(fileNameParts)-1]
		expectedContext, exists := myFilesToKeep[fileNameShort]
		require.True(t, exists)
		require.Equalf(t, expectedContext, context, "file: %s", fileNameShort)
	}
}

func validateFileNames(m map[string]string, keys []string) error {
	for _, key := range keys {
		if _, exists := m[key]; !exists {
			return fmt.Errorf("key %q from slice is not found in the map", key)
		}
	}
	return nil
}

// TODO: REVIEW: example how it can be done
func TestFileByFormatHandlerV2(t *testing.T) {
	startedAt := time.Now().UTC()

	tmpFolder := path.Join(os.TempDir(), fmt.Sprintf("log-%d", rand.Uint64()))
	err := os.MkdirAll(tmpFolder, os.ModePerm)
	require.NoError(t, err)
	defer func(path string) { require.NoError(t, os.RemoveAll(path)) }(tmpFolder)

	expectedFileNames := []string{
		startedAt.Add(time.Hour*24*5).Format(time.DateOnly) + ".log",
		startedAt.Add(time.Hour*24*4).Format(time.DateOnly) + ".log",
		startedAt.Add(time.Hour*24*3).Format(time.DateOnly) + ".log",
	}
	expectedFileContexts := []string{
		"f1:i0001", "f1:i0002",
		"f2:i0001", "f2:i0002",
		"f3:i0001", "f3:i0002",
		"f4:i0001", "f4:i0002",
		"f5:i0001", "f5:i0002",
	}

	fileIndexMutex := sync.Mutex{}
	fileIndex := 0
	handler := FileByFormatHandler(tmpFolder, 3, func() string {
		fileIndexMutex.Lock()
		defer fileIndexMutex.Unlock()
		fileIndex++
		return startedAt.Add(time.Hour * 12 * time.Duration(fileIndex)).Format(time.DateOnly)
	})

	for _, fileContext := range expectedFileContexts {
		_, err = handler.Write([]byte(fileContext))
		require.NoError(t, err)
	}

	files, err := extractFilesOrFail(tmpFolder)
	require.NoError(t, err)
	require.Len(t, files, 3, "incorrect files count")
	for i, n := range expectedFileNames {
		index := len(expectedFileContexts) - (i << 1) - 1 // index values: 9, 7, 5
		expectedFileContext := strings.Join(expectedFileContexts[index-1:index+1], "\n") + "\n"
		fData, fExists := files[n]
		fmt.Print(n)
		fmt.Print(files)
		fmt.Print(fExists)
		require.True(t, fExists)
		require.Equalf(t, expectedFileContext, fData, "index: %d", i)
	}
}

func TestFileByFormatHandlerForRaceCondition(t *testing.T) {
	// TODO: Implement
	//handler := FileByFormatHandler(tmpFolder, 4, fileNameGenerator)
	//expectedFileContexts := []string{
	//   "f1:i0001", "f2:i0001",
	//   "f3:i0001", "f4:i0001",
	//   "f5:i0001", "f6:i0001",
	//   "f7:i0001", "f8:i0001",
	//   "f9:i0001", "f0:i0001",
	//}
	//
	//var wg sync.WaitGroup
	//
	//for _, fileContext := range expectedFileContexts {
	//   wg.Add(1)
	//   go func(handler io.Writer) {
	//       defer wg.Done()
	//       _, err = handler.Write([]byte(fileContext))
	//       require.NoError(t, err)
	//   }(handler)
	//}
	//
	////Waiting for all goroutines to finish
	//wg.Wait()
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
		filePath = path.Base(filePath)
		if e != nil {
			r[filePath] = e.Error()
		} else {
			r[filePath] = string(b)
		}
	}
	return r, nil
}
