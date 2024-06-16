package loginjector

import (
	"errors"
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
	hooks           []hook
	m               sync.RWMutex
}

// SetMinLevel sets the minimum log level
func (l *Logger) SetMinLevel(level LogLevel) {
	l.m.Lock()
	defer l.m.Unlock()

	l.minimumLogLevel = level
}

// Hook adds a hook to the logger
func (l *Logger) Hook(logLevel LogLevel, writers ...io.Writer) HookID {
	l.m.Lock()
	defer l.m.Unlock()

	h := hook{
		ID:              HookID(uuid.NewV4().String()),
		MinimumLogLevel: logLevel,
		Writers:         writers,
	}

	l.hooks = append(l.hooks, h)

	return h.ID
}

// Unhook removes a hook from the logger
func (l *Logger) Unhook(id HookID) {
	l.m.Lock()
	defer l.m.Unlock()

	for i, h := range l.hooks {
		if h.ID == id {
			l.hooks = append(l.hooks[:i], l.hooks[i+1:]...)
			break
		}
	}
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
		if level < h.MinimumLogLevel {
			continue
		}

		for _, w := range h.Writers {
			wg.Add(1)
			go func(w io.Writer) {
				defer wg.Done()
				if _, e := w.Write(message); e != nil {
					errs = append(errs, e)
				}
			}(w)
		}
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

// Write writes a log message with the minimum log level
func (l *Logger) Write(message []byte) (int, error) {
	return l.WriteLog(l.minimumLogLevel, message)
}

// LogLevel is a log level
type LogLevel int

// HookID is a unique identifier for a hook
type HookID string

// hook is a log hook
type hook struct {
	ID              HookID
	MinimumLogLevel LogLevel
	Writers         []io.Writer
}
