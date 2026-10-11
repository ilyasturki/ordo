// Package clipboard reads and writes the system clipboard through the
// platform's tools (wl-clipboard, or pbcopy and pbpaste on macOS), and kills a
// tool still running when its timeout passes.
package clipboard

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"
)

// EnvFile names a file that stands in for the clipboard, so tests and tapes
// never reach the desktop.
const EnvFile = "ORDO_CLIPBOARD_FILE"

// ErrNoBackend reports that no clipboard utility could be reached.
var ErrNoBackend = errors.New("no clipboard utility available")

var ErrTimeout = errors.New("clipboard utility timed out")

// ErrNotText reports a clipboard holding bytes that are neither text nor an image.
var ErrNotText = errors.New("the clipboard holds no text")

var ErrTooLarge = errors.New("the clipboard's image is too large")

// On GNOME (no data-control protocol) wl-copy and wl-paste wait for a focus an unfocused session never gets.
var timeout = 2 * time.Second

// A multi-megabyte image takes longer to cross the pipe than text.
var imageTimeout = 10 * time.Second

// Above what attachments takes, so an image too large to attach is still
// reported as one.
const maxImage = 32 << 20

func Redirected() bool {
	return os.Getenv(EnvFile) != ""
}

func Write(text string) error {
	if path := os.Getenv(EnvFile); path != "" {
		return os.WriteFile(path, []byte(text), 0o600)
	}
	return run(strings.NewReader(text), nil, copyTool, timeout)
}

// Content is what the clipboard holds: text, the bytes of an image, or the
// files a file manager copied, which Text then lists one per line.
type Content struct {
	Text  string
	Image []byte
	Paths []string
}

// Read returns the clipboard's content, sniffed so binary bytes never pass as
// text. ErrNoBackend only when no tool ran: one that ran and failed most likely
// found the clipboard empty.
func Read() (Content, error) {
	if path := os.Getenv(EnvFile); path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return Content{}, err
		}
		return sniff(data)
	}
	// Not an OSC 52 read: tmux answers one from its own paste buffer, never the desktop clipboard.
	return readTool()
}

func readText() (Content, error) {
	var out bytes.Buffer
	if err := run(nil, &out, pasteTool, timeout); err != nil {
		return Content{}, err
	}
	return sniff(out.Bytes())
}

// FilePath is the local path a file:// URI names, percent-decoded.
func FilePath(uri string) (string, bool) {
	u, err := url.Parse(uri)
	if err != nil || u.Scheme != "file" || u.Path == "" || (u.Host != "" && u.Host != "localhost") {
		return "", false
	}
	return u.Path, true
}

// cappedBuffer fails a write past limit, which kills the tool behind it. Not
// an embedded bytes.Buffer: io.Copy would take its ReadFrom and skip the cap.
type cappedBuffer struct {
	buf   bytes.Buffer
	limit int
	over  bool
}

func (b *cappedBuffer) Write(p []byte) (int, error) {
	if b.buf.Len()+len(p) > b.limit {
		b.over = true
		return 0, ErrTooLarge
	}
	return b.buf.Write(p)
}

// Text goes first: BMP's "BM" signature starts ordinary words.
func sniff(data []byte) (Content, error) {
	if utf8.Valid(data) && bytes.IndexByte(data, 0) < 0 {
		return Content{Text: string(data)}, nil
	}
	if strings.HasPrefix(http.DetectContentType(data), "image/") {
		return Content{Image: data}, nil
	}
	return Content{}, ErrNotText
}

// run kills the tool's whole process group once the timeout passes.
func run(stdin io.Reader, stdout io.Writer, tool []string, timeout time.Duration) error {
	path, err := exec.LookPath(tool[0])
	if err != nil {
		return ErrNoBackend
	}
	env, ok := toolEnv()
	if !ok {
		return ErrNoBackend
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, tool[1:]...)
	cmd.Env = env
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	// A child that escaped the group could hold stdout open forever.
	cmd.WaitDelay = 100 * time.Millisecond
	err = cmd.Run()
	if err != nil && ctx.Err() != nil {
		return ErrTimeout
	}
	return err
}
