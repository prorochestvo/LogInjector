package loginjector

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// CyclicOverwritingFilesHandler creates a new file when the current file exceeds maxFileCapacity and removes older files if the number of files exceeds maxFilesInFolder
func CyclicOverwritingFilesHandler(folder, fileNamePrefix string, maxFileCapacity uint32, maxFilesInFolder int) io.Writer {
	var fileSize uint64 = 0
	index := 1
	fileName := fileNamePrefix + "." + fmt.Sprintf("%0.8X", index) + "." + defaultFileExtension
	w := &writer{
		h: func(msg []byte) (int, error) {
			f, err := os.OpenFile(path.Join(folder, fileName), os.O_WRONLY|os.O_CREATE|os.O_APPEND, defaultFilePermissions)
			if err != nil {
				return 0, err
			}
			defer func(f *os.File) {
				if e := f.Close(); e != nil {
					err = errors.Join(err, e)
				}
			}(f)

			var l uint64 = 0

			if n, e := f.Write(bytes.TrimSpace(msg)); e != nil {
				err = errors.Join(err, e)
			} else {
				l += uint64(n)
			}

			if n, e := f.Write([]byte{'\n'}); e != nil {
				err = errors.Join(err, e)
			} else {
				l += uint64(n)
			}

			fileSize += l

			if fileSize > uint64(maxFileCapacity) {
				fileSize = 0
				index++
				fileName = fileNamePrefix + "." + fmt.Sprintf("%0.8X", index) + "." + defaultFileExtension
				err = errors.Join(err, verifyFiles(folder, maxFilesInFolder))
			}

			return int(l), err
		},
	}
	return w
}

// FilePerDaysHandler creates a new file every day and removes older files if the number of files exceeds maxFilesInFolder
func FilePerDaysHandler(folder string, maxFilesInFolder int) io.Writer {
	lastFileName := ""
	w := &writer{
		h: func(msg []byte) (int, error) {
			fileName := time.Now().Format("2006-01-02") + "." + defaultFileExtension

			f, err := os.OpenFile(path.Join(folder, fileName), os.O_WRONLY|os.O_CREATE|os.O_APPEND, defaultFilePermissions)
			if err != nil {
				return 0, err
			}
			defer func(f *os.File) {
				if e := f.Close(); e != nil {
					err = errors.Join(err, e)
				}
			}(f)

			var l uint64 = 0

			if n, e := f.Write(bytes.TrimSpace(bytes.TrimSpace(msg))); e != nil {
				err = errors.Join(err, e)
			} else {
				l += uint64(n)
			}

			if n, e := f.Write([]byte{'\n'}); e != nil {
				err = errors.Join(err, e)
			} else {
				l += uint64(n)
			}

			if lastFileName != fileName {
				lastFileName = fileName
				err = errors.Join(err, verifyFiles(folder, maxFilesInFolder))
			}

			return int(l), err
		},
	}
	return w
}

// defaultPrintHandler prints the message to the console
func defaultPrintHandler() io.Writer {
	w := &writer{
		h: func(msg []byte) (int, error) {

			msg = bytes.TrimSpace(msg)
			println(msg)

			return len(msg), nil
		},
	}
	return w
}

// verifyFiles removes older files if the number of files exceeds limit
func verifyFiles(folder string, limit int) error {
	// read files by format
	files, err := filepath.Glob(path.Join(folder, "*."+defaultFileExtension))
	if err != nil || len(files) == 0 {
		return err
	}
	sort.Strings(files)
	// remove older files
	for f, lFiles := 0, len(files); f < lFiles && (lFiles-f) > limit; f++ {
		err = errors.Join(err, os.Remove(files[f]))
	}
	return err
}

const defaultFilePermissions = 0666
const defaultFileExtension = "log"

// writer is a thread-safe writer
type writer struct {
	m sync.Mutex
	h handler
}

// Write writes the message to the handler
func (w *writer) Write(p []byte) (n int, err error) {
	w.m.Lock()
	defer w.m.Unlock()
	return w.h(p)
}

// handler is a function that handles messages
type handler func(msg []byte) (n int, err error)
