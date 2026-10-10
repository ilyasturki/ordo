// Package clipboard puts text on the system clipboard, working in terminal
// sessions where the usual autodetection gives up.
//
// github.com/atotto/clipboard picks its backend in an init() that only looks at
// wl-copy when WAYLAND_DISPLAY is non-empty. A tmux server first started from a
// TTY has an empty WAYLAND_DISPLAY, so on a Wayland box with no xclip or xsel
// installed the library reports "No clipboard utilities available" even though
// wl-copy is right there on PATH. Nothing the app does at startup can undo that
// decision: atotto's init() runs before ours. So instead of trying to repair the
// environment for the library, we run wl-copy ourselves with a WAYLAND_DISPLAY
// pointing at whatever socket the compositor actually left in XDG_RUNTIME_DIR.
package clipboard

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	atotto "github.com/atotto/clipboard"
)

// ErrNoBackend reports that no clipboard utility could be reached. Callers that
// have another way to hand text to the user — a TUI can emit OSC 52 and let the
// terminal do it — should treat this as "try that" rather than as a failure.
var ErrNoBackend = errors.New("no clipboard utility available")

// A read issued just after a write must see that write: the app decides what
// to paste by comparing the clipboard with what it last wrote.
var mu sync.Mutex

// Write puts text on the system clipboard.
func Write(text string) error {
	mu.Lock()
	defer mu.Unlock()
	if !atotto.Unsupported {
		if err := atotto.WriteAll(text); err == nil {
			return nil
		}
	}
	return wlWrite(text)
}

// Read returns the system clipboard's contents.
func Read() (string, error) {
	mu.Lock()
	defer mu.Unlock()
	if !atotto.Unsupported {
		if text, err := atotto.ReadAll(); err == nil {
			return text, nil
		}
	}
	return wlRead()
}

func wlWrite(text string) error {
	cmd, err := wlCommand("wl-copy")
	if err != nil {
		return err
	}
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

func wlRead() (string, error) {
	// --no-newline: wl-paste otherwise appends one that was never copied.
	cmd, err := wlCommand("wl-paste", "--no-newline")
	if err != nil {
		return "", err
	}
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// wlCommand builds a wl-clipboard invocation aimed at the running compositor,
// or reports ErrNoBackend when either the tool or the socket is missing.
func wlCommand(name string, args ...string) (*exec.Cmd, error) {
	path, err := exec.LookPath(name)
	if err != nil {
		return nil, ErrNoBackend
	}
	display, ok := waylandDisplay()
	if !ok {
		return nil, ErrNoBackend
	}
	cmd := exec.Command(path, args...)
	cmd.Env = append(os.Environ(), "WAYLAND_DISPLAY="+display)
	return cmd, nil
}

// waylandDisplay names the compositor's socket: the environment's answer when
// it has one, otherwise the first socket sitting in the runtime dir.
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
