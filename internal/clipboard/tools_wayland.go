//go:build !darwin

package clipboard

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

var (
	// wl-copy forks the process serving the selection only once the selection is set, so killing a hung one leaves nothing behind.
	copyTool = []string{"wl-copy"}
	// --no-newline: wl-paste otherwise appends one that was never copied.
	pasteTool = []string{"wl-paste", "--no-newline"}
	listTool  = []string{"wl-paste", "--list-types"}
)

var imageTypes = []string{"image/png", "image/jpeg", "image/webp", "image/gif"}

func typedPasteTool(mime string) []string {
	return []string{"wl-paste", "--no-newline", "--type", mime}
}

// readTool asks for an image, then copied files, before text: a bare wl-paste
// picks text, such as the HTML a browser offers beside a copied image.
func readTool() (Content, error) {
	var list strings.Builder
	if err := run(nil, &list, listTool, timeout); err != nil {
		return Content{}, err
	}
	offered := strings.Fields(list.String())
	for _, mime := range imageTypes {
		if !slices.Contains(offered, mime) {
			continue
		}
		out := &cappedBuffer{limit: maxImage}
		err := run(nil, out, typedPasteTool(mime), imageTimeout)
		if out.over {
			return Content{}, ErrTooLarge
		}
		if err != nil {
			return Content{}, err
		}
		return sniff(out.buf.Bytes())
	}
	if slices.Contains(offered, "text/uri-list") {
		var out strings.Builder
		if err := run(nil, &out, typedPasteTool("text/uri-list"), timeout); err != nil {
			return Content{}, err
		}
		if paths := localPaths(out.String()); paths != nil {
			return Content{Text: strings.Join(paths, "\n"), Paths: paths}, nil
		}
	}
	return readText()
}

// localPaths reads a text/uri-list (RFC 2483), or nil when an entry isn't a
// local file.
func localPaths(list string) []string {
	var paths []string
	for _, line := range strings.Split(list, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		path, ok := FilePath(line)
		if !ok {
			return nil
		}
		paths = append(paths, path)
	}
	return paths
}

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
