package loginjector

import (
	"errors"
	"fmt"
	"github.com/twinj/uuid"
	"io"
	"sync"
)

// NewLogger creates a new logger with the given minimum log level and handlers
// If no handlers are provided, a default print handler is added
func NewLogger(minLogLevel LogLevel, handlers ...io.Writer) (*Logger, error) {
	if handlers == nil || len(handlers) == 0 {
		handlers = []io.Writer{PrintHandler()}
	}

	l := &Logger{
		minimumLogLevel: minLogLevel,
		handlers:        handlers,
	}

	return l, nil
}

// Logger is describing the logger structure and its methods
type Logger struct {
	minimumLogLevel LogLevel
	handlers        []io.Writer
	hooks           []*hook
	m               sync.RWMutex
}

// SetMinLevel sets the minimum log level
func (l *Logger) SetMinLevel(level LogLevel) {
	l.m.Lock()
	defer l.m.Unlock()

	l.minimumLogLevel = level
}

// Hook adds a hook to the logger
func (l *Logger) Hook(writer io.Writer, level LogLevel, additional ...LogLevel) HookID {
	l.m.Lock()
	defer l.m.Unlock()

	hID := HookID(uuid.NewV4().String())

	l.hooks = append(l.hooks, &hook{
		ID:     hID,
		Level:  level,
		Writer: writer,
	})

	for _, logLevel := range additional {
		l.hooks = append(l.hooks, &hook{
			ID:     hID,
			Level:  logLevel,
			Writer: writer,
		})
	}

	return hID
}

// Unhook removes a hook from the logger
func (l *Logger) Unhook(id HookID) {
	l.m.Lock()
	defer l.m.Unlock()

	items := make([]*hook, 0, len(l.hooks)-1)

	for _, h := range l.hooks {
		if h.ID != id {
			items = append(items, h)
		}
	}

	l.hooks = items
}

// JoinAs joins the logger as a writer to the given outputs
func (l *Logger) JoinAs(logLevel LogLevel, outputs ...func(w io.Writer)) {
	l.m.Lock()
	defer l.m.Unlock()

	w := func(level LogLevel) io.Writer {
		w := &writer{
			h: func(msg []byte) (int, error) {
				return l.WriteLog(level, msg)
			},
		}
		return w
	}(logLevel)

	for _, output := range outputs {
		output(w)
	}
}

// WriteLog writes a log message
func (l *Logger) WriteLog(level LogLevel, message []byte) (int, error) {
	n := len(message)
	errs := make([]error, 0)
	wg := sync.WaitGroup{}

	l.m.RLock()
	defer l.m.RUnlock()

	for _, h := range l.hooks {
		if level != h.Level {
			continue
		}
		wg.Add(1)
		go func(w io.Writer) {
			defer wg.Done()
			if _, e := w.Write(message); e != nil {
				errs = append(errs, e)
			}
		}(h.Writer)
	}

	if level < l.minimumLogLevel {
		return 0, nil
	}

	for _, h := range l.handlers {
		wg.Add(1)
		go func(w io.Writer) {
			defer wg.Done()
			if _, e := w.Write(message); e != nil {
				errs = append(errs, e)
			}
		}(h)
	}

	wg.Wait()

	return n, errors.Join(errs...)
}

// Printf writes a formatted log message
func (l *Logger) Printf(level LogLevel, format string, args ...any) {
	s := fmt.Sprintf(format, args...)
	_, err := l.WriteLog(level, []byte(s+"\n"))
	if err != nil {
		println(err.Error(), StackTrace())
	}
}

// Print writes a log message
func (l *Logger) Print(level LogLevel, args ...any) {
	s := fmt.Sprint(args...)
	_, err := l.WriteLog(level, []byte(s+"\n"))
	if err != nil {
		println(err.Error(), StackTrace())
	}
}

// Fatalf writes a formatted log message and exits the program
func (l *Logger) Fatalf(level LogLevel, format string, args ...any) {
	s := fmt.Sprintf(format, args...)
	_, err := l.WriteLog(level, []byte(s+"\n"))
	if err != nil {
		println(err.Error(), StackTrace())
	}
	panic(s)
}

// Fatal writes a log message and exits the program
func (l *Logger) Fatal(level LogLevel, args ...any) {
	s := fmt.Sprint(args...)
	_, err := l.WriteLog(level, []byte(s+"\n"))
	if err != nil {
		println(err.Error(), StackTrace())
	}
	panic(s)
}

// Write writes a log message with the minimum log level
func (l *Logger) Write(message []byte) (int, error) {
	return l.WriteLog(l.minimumLogLevel, message)
}

// WriterAs returns a writer that writes to the logger as the given log level
func (l *Logger) WriterAs(level LogLevel) io.Writer {
	w := &writer{
		h: func(msg []byte) (int, error) {
			return l.WriteLog(level, msg)
		},
	}
	return w
}

// LogLevel is a log level
type LogLevel int

// HookID is a unique identifier for a hook
type HookID string

// hook is a log hook
type hook struct {
	ID     HookID
	Level  LogLevel
	Writer io.Writer
}
