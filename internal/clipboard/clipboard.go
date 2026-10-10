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
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	atotto "github.com/atotto/clipboard"
)

// ErrNoBackend reports that no clipboard utility could be reached. Callers that
// have another way to hand text to the user — a TUI can emit OSC 52 and let the
// terminal do it — should treat this as "try that" rather than as a failure.
var ErrNoBackend = errors.New("no clipboard utility available")

var ErrTimeout = errors.New("clipboard read timed out")

// wl-paste on GNOME (no data-control protocol) waits for a focus an unfocused session never gets.
var readTimeout = 2 * time.Second

// Write puts text on the system clipboard.
func Write(text string) error {
	if !atotto.Unsupported {
		if err := atotto.WriteAll(text); err == nil {
			return nil
		}
	}
	return wlWrite(text)
}

// A hung atotto read can't be killed: its goroutine outlives the ErrTimeout.
func Read() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), readTimeout)
	defer cancel()
	type result struct {
		text string
		err  error
	}
	done := make(chan result, 1)
	go func() {
		text, err := read(ctx)
		done <- result{text, err}
	}()
	select {
	case r := <-done:
		if r.err != nil && ctx.Err() != nil {
			return "", ErrTimeout
		}
		return r.text, r.err
	case <-ctx.Done():
		return "", ErrTimeout
	}
}

func read(ctx context.Context) (string, error) {
	if !atotto.Unsupported {
		if text, err := atotto.ReadAll(); err == nil {
			return text, nil
		}
	}
	return wlRead(ctx)
}

func wlWrite(text string) error {
	cmd, err := wlCommand(context.Background(), "wl-copy")
	if err != nil {
		return err
	}
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

func wlRead(ctx context.Context) (string, error) {
	// --no-newline: wl-paste otherwise appends one that was never copied.
	cmd, err := wlCommand(ctx, "wl-paste", "--no-newline")
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
func wlCommand(ctx context.Context, name string, args ...string) (*exec.Cmd, error) {
	path, err := exec.LookPath(name)
	if err != nil {
		return nil, ErrNoBackend
	}
	display, ok := waylandDisplay()
	if !ok {
		return nil, ErrNoBackend
	}
	cmd := exec.CommandContext(ctx, path, args...)
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
