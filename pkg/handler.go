package loginjector

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

func TelegramHandler(botToken, chatID, fileName string, labels ...string) io.Writer {
	const telegramAPI = "https://api.telegram.org/bot"
	url := fmt.Sprintf("%s%s/sendDocument", telegramAPI, botToken)
	w := &writer{
		h: func(msg []byte) (int, error) {
			payload := &bytes.Buffer{}
			parts := multipart.NewWriter(payload)

			if filePart, err := parts.CreateFormFile("document", fileName); err != nil {
				return 0, fmt.Errorf("could not create request form file, details: %s", err)
			} else if _, err = filePart.Write(msg); err != nil {
				return 0, fmt.Errorf("could not write file part to request, details: %s", err)
			}

			if err := parts.WriteField("chat_id", chatID); err != nil {
				return 0, fmt.Errorf("could not write chat_id field to request: %s", err)
			}

			caption := strings.Join(labels, "\n")
			caption = fmt.Sprintf("%s %s", time.Now().UTC().Format("2006-01-02 15:04:05"), caption)
			caption = strings.TrimSpace(caption)
			if err := parts.WriteField("caption", caption); err != nil {
				return 0, fmt.Errorf("could not write caption field to request: %s", err)
			}

			if err := parts.WriteField("parse_mode", "HTML"); err != nil {
				return 0, fmt.Errorf("could not write parse_mode field to request: %s", err)
			}

			contentType := parts.FormDataContentType()

			if err := parts.Close(); err != nil {
				return 0, fmt.Errorf("could not close request: %s", err)
			}

			request, err := http.NewRequest("POST", url, payload)
			if err != nil {
				return 0, fmt.Errorf("could not create HTTP request: %v", err)
			}
			request.Header.Set("Content-Type", contentType)

			r, err := (&http.Client{Timeout: time.Second * 20}).Do(request)
			if err != nil {
				return 0, fmt.Errorf("could not send HTTP request: %v", err)
			}
			defer CloseOrLog(r.Body)

			if r.StatusCode != http.StatusOK {
				return 0, fmt.Errorf("could not to deliver message, status code: %v", r.StatusCode)
			}

			var response struct {
				Ok bool `json:"ok"`
			}
			rawResponse := bytes.NewBufferString("")
			err = json.NewDecoder(io.TeeReader(r.Body, rawResponse)).Decode(&response)
			if err != nil || !response.Ok {
				return 0, fmt.Errorf("could not decode response: %v\n%s", err, rawResponse.String())
			}

			return len(msg), nil
		},
	}
	return w
}

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

// SilenceHandler does nothing that is ignores any messages and returns the length of the message without error
func SilenceHandler() io.Writer {
	return &writer{
		h: func(msg []byte) (int, error) {
			return len(msg), nil
		},
	}
}

// PrintHandler prints the message to the console
func PrintHandler() io.Writer {
	w := &writer{
		h: func(msg []byte) (int, error) {
			println(string(bytes.TrimSpace(msg)))
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
	h func(msg []byte) (n int, err error)
}

// Write writes the message to the handler
func (w *writer) Write(p []byte) (n int, err error) {
	w.m.Lock()
	defer w.m.Unlock()
	return w.h(p)
}
