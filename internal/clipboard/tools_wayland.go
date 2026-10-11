//go:build !darwin

package clipboard

import (
	"os"
	"path/filepath"
	"strconv"
)

var (
	// wl-copy forks the process serving the selection only once the selection is set, so killing a hung one leaves nothing behind.
	copyTool = []string{"wl-copy"}
	// --no-newline: wl-paste otherwise appends one that was never copied.
	pasteTool = []string{"wl-paste", "--no-newline"}
)

func toolEnv() ([]string, bool) {
	display, ok := waylandDisplay()
	if !ok {
		return nil, false
	}
	return append(os.Environ(), "WAYLAND_DISPLAY="+display), true
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
