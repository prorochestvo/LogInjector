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
	"path/filepath"
	"sort"
	"strconv"
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

// RotatingFileOption configures RotatingFileHandler.
type RotatingFileOption func(*rotatingFileConfig)

// WithMaxFileSize overrides the maximum size of a single log file in bytes before
// the handler rotates to the next index. The default is 5 MiB (5 << 20).
func WithMaxFileSize(n uint32) RotatingFileOption {
	return func(c *rotatingFileConfig) { c.maxFileSize = n }
}

// WithMaxFiles overrides the maximum number of rotated log files retained in the
// folder. Older files beyond the limit are pruned on rotation. The default is 7.
func WithMaxFiles(n int) RotatingFileOption {
	return func(c *rotatingFileConfig) { c.maxFiles = n }
}

// WithFreshStart makes the handler remove every existing prefix.<8 hex>.log file in
// folder once at construction, before the first write, so each process run begins with a
// genuinely empty ring at index 1. Existing rotated content is permanently destroyed —
// this is the intended behaviour; omit the option to resume and preserve prior content
// across restarts. Files in folder that do not match the prefix.<8 hex>.log shape are
// left untouched.
//
// A missing file or folder is a no-op: no file is created at construction, and the first
// write creates index-1. Any removal error is stashed and surfaced on the first Write's
// return value, then cleared. WithFreshStart forces the starting index to 1 and the
// seeded size to 0, overriding the resume-at-highest-index behaviour.
func WithFreshStart() RotatingFileOption {
	return func(c *rotatingFileConfig) { c.freshStart = true }
}

// RotatingFileHandler saves messages to files by number. The file name is
// generated from prefix and an incrementing index (e.g. prefix.00000001.log).
// When a file exceeds the maximum size the handler moves to the next index and
// prunes the oldest files so that no more than the configured maximum remain in
// folder. The handler appends to files and ring-prunes the oldest ones; it never
// overwrites a file's existing contents unless WithFreshStart is passed.
//
// On construction the handler resumes at the highest existing index for prefix in
// folder (index 1 when none exist or WithFreshStart is set) and seeds the
// in-memory size counter from that file's on-disk size, so the first write after
// a restart accounts for existing content instead of writing into the oldest file
// and letting pruning delete the newest data. Filenames in folder that do not
// match the exact prefix.<8 hex>.log shape are ignored. Any stat/glob error
// encountered while resolving the resume point is stashed and joined into the
// first Write's return value.
//
// Defaults: max file size 5 MiB, max files 7. Override with WithMaxFileSize and
// WithMaxFiles; use WithFreshStart to begin each run from a truncated index-1
// file. FileByFormatHandler is the sibling for filename-generator-driven rotation
// and has no fresh-start variant (its target name is not known until the first
// write).
//
// The returned writer is mutex-guarded; the logger never calls its Write
// concurrently.
func RotatingFileHandler(folder, prefix string, opts ...RotatingFileOption) io.Writer {
	cfg := rotatingFileConfig{
		maxFileSize: 5 << 20,
		maxFiles:    7,
	}
	for _, o := range opts {
		o(&cfg)
	}

	// reject a prefix that contains path separators — it would escape the folder.
	if filepath.Base(prefix) != prefix {
		return &writer{
			h: func([]byte) (int, error) {
				return 0, fmt.Errorf("loginjector: file name prefix %q must not contain path separators", prefix)
			},
		}
	}

	index := 1
	var fileSize uint64
	var seedErr error
	if cfg.freshStart {
		// overwrite-on-start forces a clean start regardless of what is on disk, so the
		// resume scan is skipped and every existing prefix.<8 hex>.log file is removed —
		// not just index-1 truncated — so the ring truly begins empty. Leaving stale
		// higher-index files behind would let verifyFiles prune the fresh files ahead of
		// them and let a later rotation append onto old content. Index/size stay 1/0; a
		// missing file or folder is a no-op. Any remove error is surfaced on first Write.
		seedErr = resetRotation(folder, prefix)
	} else {
		// resume at the highest existing index so the handler keeps appending to the
		// newest file across a process restart, instead of restarting at index 1 —
		// which is the lexicographically-oldest file that verifyFiles prunes first,
		// destroying the newest data.
		index, fileSize, seedErr = resumeRotation(folder, prefix)
	}

	fileName := rotatingFileName(prefix, index)

	w := &writer{
		h: func(msg []byte) (int, error) {
			// surface any construction-time stat/truncate error on the first Write,
			// then clear it.
			err := seedErr
			seedErr = nil

			f, openErr := os.OpenFile(filepath.Join(folder, fileName), os.O_WRONLY|os.O_CREATE|os.O_APPEND, defaultFilePermissions)
			if openErr != nil {
				return 0, errors.Join(err, openErr)
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

			if fileSize > uint64(cfg.maxFileSize) {
				index++
				fileName = rotatingFileName(prefix, index)
				// seed the new target's size from disk rather than assuming 0: when the
				// handler rotates into a pre-existing higher-index file (from a prior
				// run) an assumed-0 counter would let that file grow to ~2× the cap
				// before rotating again.
				fileSize = 0
				if fi, e := os.Stat(filepath.Join(folder, fileName)); e == nil {
					fileSize = uint64(fi.Size())
				} else if !os.IsNotExist(e) {
					err = errors.Join(err, e)
				}
				err = errors.Join(err, verifyFiles(folder, cfg.maxFiles))
			}

			return int(l), err
		},
	}
	return w
}

// FileByFormatHandler save messages to files by format.
// The file name is generated by fileNameGenerator.
// The folder is the directory where the files are saved.
// The maxFilesInFolder is the maximum number of files in the folder.
// No fresh-start variant exists for this handler; see RotatingFileHandler's
// WithFreshStart for an analogous feature on the index-based handler (there the
// construction-time target is known, whereas here the name is generated per-write
// by fileNameGenerator).
func FileByFormatHandler(folder string, maxFilesInFolder int, fileNameGenerator func() string) io.Writer {
	lastFileName := ""
	w := &writer{
		h: func(msg []byte) (int, error) {
			fileName := fileNameGenerator() + "." + defaultFileExtension

			f, err := os.OpenFile(filepath.Join(folder, fileName), os.O_WRONLY|os.O_CREATE|os.O_APPEND, defaultFilePermissions)
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

			if lastFileName != fileName {
				lastFileName = fileName
				err = errors.Join(err, verifyFiles(folder, maxFilesInFolder))
			}

			return int(l), err
		},
	}
	return w
}

// PrintOption configures TimestampedPrintHandler.
type PrintOption func(*printConfig)

// WithTimeLayout overrides the timestamp layout used by TimestampedPrintHandler.
// The default layout is "2006/01/02 15:04:05".
func WithTimeLayout(layout string) PrintOption {
	return func(c *printConfig) { c.layout = layout }
}

// WithOutput overrides the sink used by TimestampedPrintHandler. The default is
// os.Stdout. This is primarily intended to redirect output for testing, but it is a
// legitimate consumer need (e.g. writing timestamped messages to a file instead of
// stdout).
func WithOutput(w io.Writer) PrintOption {
	return func(c *printConfig) { c.out = w }
}

// withClock is an unexported test seam that injects a fixed clock into
// TimestampedPrintHandler to make timestamp-sensitive assertions deterministic.
func withClock(fn func() time.Time) PrintOption {
	return func(c *printConfig) { c.clock = fn }
}

// printConfig holds the resolved configuration for TimestampedPrintHandler.
type printConfig struct {
	layout string
	out    io.Writer
	clock  func() time.Time
}

// TimestampedHandler wraps inner so each message is prefixed with a leading
// timestamp before being forwarded to inner, indenting the continuation lines of a
// multi-line message so the body reads as a block. It is the composable form of
// TimestampedPrintHandler: any io.Writer (a file handler, a buffer, another sink)
// can be given sink-side timestamps. The returned writer is mutex-guarded; the
// logger never calls its Write concurrently.
//
// The timestamp layout defaults to "2006/01/02 15:04:05". The first line of a
// message is prefixed with "<timestamp> "; subsequent lines are indented by the
// same width so multi-line payloads read as a visual block. Trailing whitespace and
// newlines are trimmed from the input before formatting; the output always ends
// with exactly one newline. An empty or whitespace-only input produces a single
// "<timestamp>\n" line.
//
// WithTimeLayout and withClock apply; WithOutput is ignored because the sink is the
// explicit inner argument. Use TimestampedPrintHandler when you want the os.Stdout
// instantiation with WithOutput support.
func TimestampedHandler(inner io.Writer, opts ...PrintOption) io.Writer {
	cfg := printConfig{
		layout: "2006/01/02 15:04:05",
		out:    inner,
		clock:  time.Now,
	}
	for _, o := range opts {
		o(&cfg)
	}
	// the sink is the explicit inner argument; WithOutput must not redirect it.
	cfg.out = inner
	return newTimestampedWriter(cfg)
}

// TimestampedPrintHandler prints each message to stdout with a leading timestamp,
// indenting the continuation lines of a multi-line message so the body reads as a
// block. Unlike PrintHandler, which writes to stdout via a plain writer, this
// handler adds a timestamp; the sink is os.Stdout by default and overridable with
// WithOutput. The returned writer is mutex-guarded; the logger never calls its
// Write concurrently. It is the os.Stdout instantiation of TimestampedHandler.
//
// The timestamp layout defaults to "2006/01/02 15:04:05". The first line of a message
// is prefixed with "<timestamp> "; subsequent lines are indented by the same width so
// multi-line payloads read as a visual block. Trailing whitespace and newlines are
// trimmed from the input before formatting; the output always ends with exactly one
// newline. An empty or whitespace-only input produces a single "<timestamp>\n" line.
func TimestampedPrintHandler(opts ...PrintOption) io.Writer {
	cfg := printConfig{
		layout: "2006/01/02 15:04:05",
		out:    os.Stdout,
		clock:  time.Now,
	}
	for _, o := range opts {
		o(&cfg)
	}
	return newTimestampedWriter(cfg)
}

// newTimestampedWriter builds the mutex-guarded writer shared by TimestampedHandler
// and TimestampedPrintHandler: it renders each message with a leading timestamp and
// continuation-line indent, then forwards the rendered bytes to cfg.out.
func newTimestampedWriter(cfg printConfig) io.Writer {
	// precompute the continuation indent once — the rendered width of a fixed-width
	// layout is constant, so formatting a reference time gives the exact width.
	const sep = " "
	indent := strings.Repeat(" ", len(time.Time{}.Format(cfg.layout))+len(sep))

	return &writer{
		h: func(msg []byte) (int, error) {
			ts := cfg.clock().Format(cfg.layout)

			trimmed := bytes.TrimSpace(msg)

			var b strings.Builder

			// fast-path: single-line message (no '\n' in trimmed body).
			if bytes.IndexByte(trimmed, '\n') < 0 {
				b.WriteString(ts)
				if len(trimmed) > 0 {
					b.WriteString(sep)
					b.Write(trimmed)
				}
				b.WriteByte('\n')
			} else {
				body := string(trimmed)
				lines := strings.Split(body, "\n")
				b.WriteString(ts)
				if lines[0] != "" {
					b.WriteString(sep)
					b.WriteString(lines[0])
				}
				b.WriteByte('\n')
				for _, ln := range lines[1:] {
					b.WriteString(indent)
					b.WriteString(ln)
					b.WriteByte('\n')
				}
			}

			if _, err := io.WriteString(cfg.out, b.String()); err != nil {
				return 0, err
			}
			return len(msg), nil
		},
	}
}

// PrintHandler writes each message to os.Stdout as a plain line, trimming
// surrounding whitespace and appending a single newline. It adds no timestamp; for
// the timestamped console sink (which is the default when NewLogger is given no
// handlers) use TimestampedPrintHandler. The returned writer is mutex-guarded; the
// logger never calls its Write concurrently.
func PrintHandler() io.Writer {
	w := &writer{
		h: func(msg []byte) (int, error) {
			if _, err := fmt.Fprintln(os.Stdout, string(bytes.TrimSpace(msg))); err != nil {
				return 0, err
			}
			return len(msg), nil
		},
	}
	return w
}

// rotatingFileConfig holds the resolved configuration for RotatingFileHandler.
type rotatingFileConfig struct {
	maxFileSize uint32
	maxFiles    int
	freshStart  bool
}

// rotatingFileName renders the on-disk name for a given rotation index, e.g.
// "prefix.00000001.log". The index is formatted as fixed-width uppercase hex so
// names sort lexicographically in index order.
func rotatingFileName(prefix string, index int) string {
	return fmt.Sprintf("%s.%08X.%s", prefix, index, defaultFileExtension)
}

// resumeRotation resolves the starting rotation index and seed size for prefix in
// folder by scanning for existing prefix.<8 hex>.log files and picking the highest
// index. It returns (1, 0, nil) when none are found. The returned size is the
// on-disk size of the resolved file, so the first write accounts for existing
// content. Filenames that do not match the exact shape are ignored. A glob error
// or an unexpected stat error is returned as seedErr for the caller to surface on
// the first Write; index and size then fall back to the best values parsed so far.
func resumeRotation(folder, prefix string) (index int, size uint64, seedErr error) {
	index = 1

	// glob only the extension (a literal pattern) so metacharacters in prefix cannot
	// corrupt the match set, then filter to prefix.<8 hex>.log explicitly. The prefix
	// guard is load-bearing: without it a foreign "XXXXXXXX.log" would survive the
	// TrimPrefix no-op and be misparsed as a valid index.
	matches, err := filepath.Glob(filepath.Join(folder, "*."+defaultFileExtension))
	if err != nil {
		return index, 0, err
	}

	found := false
	for _, p := range matches {
		base := filepath.Base(p)
		if !strings.HasPrefix(base, prefix+".") {
			// not our prefix; skip before the TrimPrefix no-op can misparse it.
			continue
		}
		mid := strings.TrimSuffix(strings.TrimPrefix(base, prefix+"."), "."+defaultFileExtension)
		if len(mid) != 8 {
			// reject non-8-hex middles: foreign files or malformed indices.
			continue
		}
		n, e := strconv.ParseUint(mid, 16, 32)
		if e != nil {
			continue
		}
		if !found || int(n) > index {
			index = int(n)
			found = true
		}
	}

	if !found {
		return 1, 0, nil
	}

	// seed size from the resolved highest-index file (0 if it does not exist yet).
	if fi, e := os.Stat(filepath.Join(folder, rotatingFileName(prefix, index))); e == nil {
		size = uint64(fi.Size())
	} else if !os.IsNotExist(e) {
		seedErr = e
	}
	return index, size, seedErr
}

// resetRotation removes every prefix.<8 hex>.log file in folder so a WithFreshStart
// handler begins with a genuinely empty ring instead of only truncating index-1 — which
// would leave stale higher-index files for verifyFiles to prune ahead of the fresh ones
// and for a later rotation to append onto. It globs the literal "*.log" (so prefix
// metacharacters cannot corrupt the match set) and filters to the strict shape, mirroring
// resumeRotation. Foreign filenames are left untouched. Remove errors are joined and
// returned for the caller to surface on the first Write; a missing file is not an error.
func resetRotation(folder, prefix string) error {
	matches, err := filepath.Glob(filepath.Join(folder, "*."+defaultFileExtension))
	if err != nil {
		return err
	}
	for _, p := range matches {
		base := filepath.Base(p)
		if !strings.HasPrefix(base, prefix+".") {
			continue
		}
		mid := strings.TrimSuffix(strings.TrimPrefix(base, prefix+"."), "."+defaultFileExtension)
		if len(mid) != 8 {
			continue
		}
		if _, e := strconv.ParseUint(mid, 16, 32); e != nil {
			continue
		}
		if e := os.Remove(p); e != nil && !os.IsNotExist(e) {
			err = errors.Join(err, e)
		}
	}
	return err
}

// verifyFiles removes older files if the number of files exceeds limit
func verifyFiles(folder string, limit int) error {
	// read files by format
	files, err := filepath.Glob(filepath.Join(folder, "*."+defaultFileExtension))
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

const defaultFilePermissions = 0640
const defaultFileExtension = "log"

// WithMinLevel wraps handler so that Logger treats it as a LeveledHandler with
// the given minimum level. The returned writer forwards Write calls to handler
// unchanged; the only added behaviour is the MinLevel() method that Logger.WriteLog
// reads via interface assertion. Use this to give one handler a threshold
// distinct from the parent Logger's minimumLogLevel — for example, to send only
// high-severity messages to the console while sending everything to a file:
//
//	file    := loginjector.RotatingFileHandler("./logs", "app")
//	console := loginjector.WithMinLevel(levels.Warning, loginjector.TimestampedPrintHandler())
//	logger  := loginjector.NewLogger(levels.Info, file, console)
//
// The returned writer is itself thread-safe via Logger's existing ensureThreadSafe
// wrapping at registration. Calling WithMinLevel on an already-leveled writer
// replaces the inner MinLevel().
func WithMinLevel(level LogLevel, handler io.Writer) io.Writer {
	return &leveledWriter{inner: handler, level: level}
}

// leveledWriter is the concrete LeveledHandler implementation behind WithMinLevel.
type leveledWriter struct {
	inner io.Writer
	level LogLevel
}

func (lw *leveledWriter) Write(p []byte) (int, error) { return lw.inner.Write(p) }
func (lw *leveledWriter) MinLevel() LogLevel          { return lw.level }

var _ LeveledHandler = (*leveledWriter)(nil)

// writer is a thread-safe writer
type writer struct {
	m        sync.Mutex
	h        func(msg []byte) (n int, err error)
	original io.Writer // the unwrapped sink; set by ensureThreadSafe for unwrapLeveled.
}

// Write writes the message to the handler
func (w *writer) Write(p []byte) (n int, err error) {
	w.m.Lock()
	defer w.m.Unlock()
	return w.h(p)
}
