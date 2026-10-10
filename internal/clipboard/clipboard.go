// Package clipboard reads and writes the Wayland clipboard through
// wl-clipboard, and kills a tool still running when its timeout passes.
package clipboard

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// EnvFile names a file that stands in for the clipboard, so tests and tapes
// never reach the desktop.
const EnvFile = "PHASIONARY_CLIPBOARD_FILE"

// ErrNoBackend reports that no clipboard utility could be reached.
var ErrNoBackend = errors.New("no clipboard utility available")

var ErrTimeout = errors.New("clipboard utility timed out")

// On GNOME (no data-control protocol) wl-copy and wl-paste wait for a focus an unfocused session never gets.
var timeout = 2 * time.Second

func Redirected() bool {
	return os.Getenv(EnvFile) != ""
}

// Write puts text on the clipboard with wl-copy.
func Write(text string) error {
	if path := os.Getenv(EnvFile); path != "" {
		return os.WriteFile(path, []byte(text), 0o600)
	}
	// wl-copy forks the process serving the selection only once the selection is set, so killing a hung one leaves nothing behind.
	return run(strings.NewReader(text), nil, "wl-copy")
}

// Read returns the clipboard's text. ErrNoBackend only when no tool ran: one
// that ran and failed most likely found the clipboard empty.
func Read() (string, error) {
	if path := os.Getenv(EnvFile); path != "" {
		data, err := os.ReadFile(path)
		return string(data), err
	}
	var out bytes.Buffer
	// --no-newline: wl-paste otherwise appends one that was never copied.
	if err := run(nil, &out, "wl-paste", "--no-newline"); err != nil {
		return "", err
	}
	return out.String(), nil
}

// run kills the tool's whole process group once the timeout passes.
func run(stdin io.Reader, stdout io.Writer, name string, args ...string) error {
	path, err := exec.LookPath(name)
	if err != nil {
		return ErrNoBackend
	}
	display, ok := waylandDisplay()
	if !ok {
		return ErrNoBackend
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Env = append(os.Environ(), "WAYLAND_DISPLAY="+display)
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

// waylandDisplay names the compositor's socket: the environment's answer when
// it has one, otherwise the first socket sitting in the runtime dir. A tmux
// server first started from a TTY has an empty WAYLAND_DISPLAY.
func waylandDisplay() (string, bool) {
	if d := os.Getenv("WAYLAND_DISPLAY"); d != "" {
		return d, true
	}
	dir := runtimeDir()
	if dir == "" {
		return "", false
	}
	matches, err := filepath.Glob(filepath.Join(dir, "wayland-*"))
	if err != nil {
		return "", false
	}
	for _, m := range matches {
		// Every socket is shadowed by a wayland-N.lock file; skip those.
		if filepath.Ext(m) == ".lock" {
			continue
		}
		return filepath.Base(m), true
	}
	return "", false
}

func runtimeDir() string {
	if dir := os.Getenv("XDG_RUNTIME_DIR"); dir != "" {
		return dir
	}
	// A login shell that never set XDG_RUNTIME_DIR still gets the directory
	// systemd-logind created for the user.
	return filepath.Join("/run/user", strconv.Itoa(os.Getuid()))
}
