package loginjector

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"time"
)

type Handler func(string) error

// FilePerDaysHandler creates a new file every day and removes older files if the number of files exceeds maxFilesInFolder
func FilePerDaysHandler(folder string, maxFilesInFolder int) Handler {
	lastFileName := ""
	verify := func() error {
		// read files by format
		files, err := filepath.Glob(path.Join(folder, "*.log"))
		if err != nil || len(files) == 0 {
			return err
		}
		// sort files by name
		for i := 0; i < len(files); i++ {
			for j := i; j < len(files); j++ {
				if files[i] > files[j] {
					files[i], files[j] = files[j], files[i]
				}
			}
		}
		// remove older files
		for f, lFiles := 0, len(files); f < lFiles && (lFiles-f) > maxFilesInFolder; f++ {
			err = errors.Join(os.Remove(path.Join(folder, files[f])))
		}
		return err
	}
	return func(message string) error {
		fileName := fmt.Sprintf("%s.log", time.Now().Format("2006-01-02"))

		fPath := path.Join(folder, fileName)

		err := os.WriteFile(fPath, []byte(message), 0664)

		if lastFileName != fileName {
			lastFileName = fileName
			err = errors.Join(verify())
		}

		return err
	}
}
