package loginjector

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type Handler func(string) error

func CyclicOverwritingFilesHandler(folder, fileNamePrefix string, maxFileCapacity uint32, maxFilesInFolder int) Handler {
	var fileSize uint64 = 0
	var m sync.Mutex
	index := 1
	fileName := fileNamePrefix + "." + fmt.Sprintf("%0.8X", index) + "." + defaultFileExtension
	return func(message string) (err error) {
		m.Lock()
		defer m.Unlock()

		f, err := os.OpenFile(path.Join(folder, fileName), os.O_WRONLY|os.O_CREATE|os.O_APPEND, defaultFilePermissions)
		if err != nil {
			return
		}
		defer func(f *os.File) {
			if e := f.Close(); e != nil {
				err = errors.Join(err, e)
			}
		}(f)

		n, err := f.WriteString(strings.TrimSpace(message) + "\n")
		l := uint64(n)

		fileSize += l

		if fileSize > uint64(maxFileCapacity) {
			fileSize = 0
			index++
			fileName = fileNamePrefix + "." + fmt.Sprintf("%0.8X", index) + "." + defaultFileExtension
			err = errors.Join(err, verifyFiles(folder, maxFilesInFolder))
		}

		return
	}
}

// FilePerDaysHandler creates a new file every day and removes older files if the number of files exceeds maxFilesInFolder
func FilePerDaysHandler(folder string, maxFilesInFolder int) Handler {
	var m sync.Mutex
	lastFileName := ""
	return func(message string) error {
		m.Lock()
		defer m.Unlock()

		fileName := time.Now().Format("2006-01-02") + "." + defaultFileExtension

		fPath := path.Join(folder, fileName)

		err := os.WriteFile(fPath, []byte(message), defaultFilePermissions)

		if lastFileName != fileName {
			lastFileName = fileName
			err = errors.Join(err, verifyFiles(folder, maxFilesInFolder))
		}

		return err
	}
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
