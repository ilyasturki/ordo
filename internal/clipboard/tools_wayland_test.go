//go:build !darwin

package clipboard

import (
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWaylandDisplay_PrefersTheEnvironment(t *testing.T) {
	t.Setenv("WAYLAND_DISPLAY", "wayland-7")
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())

	got, ok := waylandDisplay()
	require.True(t, ok)
	assert.Equal(t, "wayland-7", got)
}

// A tmux server started from a TTY has no WAYLAND_DISPLAY, so the socket has
// to be found on disk instead.
func TestWaylandDisplay_FindsTheSocketWhenTheEnvironmentIsEmpty(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "wayland-0.lock"), nil, 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "wayland-0"), nil, 0o600))
	t.Setenv("WAYLAND_DISPLAY", "")
	t.Setenv("XDG_RUNTIME_DIR", dir)

	got, ok := waylandDisplay()
	require.True(t, ok)
	assert.Equal(t, "wayland-0", got, "the .lock file shadowing the socket must not be picked")
}

func TestWaylandDisplay_ReportsNothingWithoutASocket(t *testing.T) {
	t.Setenv("WAYLAND_DISPLAY", "")
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())

	_, ok := waylandDisplay()
	assert.False(t, ok)
}

func TestRun_ReportsErrNoBackendWithoutASocket(t *testing.T) {
	fakeTools(t, "exit 0")
	t.Setenv("WAYLAND_DISPLAY", "")
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())

	assert.ErrorIs(t, Write("x"), ErrNoBackend)
	_, err := Read()
	assert.ErrorIs(t, err, ErrNoBackend)
}

func TestRead_ReturnsWhatThePasteToolPrints(t *testing.T) {
	fakeTools(t, `case "$*" in
--list-types) printf 'text/plain;charset=utf-8\nUTF8_STRING\n' ;;
--no-newline) printf 'from the browser' ;;
*) exit 1 ;;
esac`)

	got, err := Read()

	require.NoError(t, err)
	assert.Equal(t, "from the browser", got.Text)
}

func TestRead_PrefersAnImageToTheTextBesideIt(t *testing.T) {
	img := filepath.Join(t.TempDir(), "shot.png")
	require.NoError(t, os.WriteFile(img, pngBytes(t), 0o600))
	fakeTools(t, `case "$*" in
--list-types) printf 'text/html\nimage/jpeg\nimage/png\n' ;;
"--no-newline --type image/png") cat `+img+` ;;
*) exit 1 ;;
esac`)

	got, err := Read()

	require.NoError(t, err)
	assert.Equal(t, pngBytes(t), got.Image)
	assert.Empty(t, got.Text)
}

func TestRead_RefusesAnImageTooLargeToHold(t *testing.T) {
	fakeTools(t, `case "$*" in
--list-types) echo image/png ;;
*) head -c 40000000 /dev/zero ;;
esac`)

	_, err := Read()

	assert.ErrorIs(t, err, ErrTooLarge)
}

func TestRead_CopiedFilesArePaths(t *testing.T) {
	fakeTools(t, `case "$*" in
--list-types) printf 'x-special/gnome-copied-files\ntext/uri-list\ntext/plain\n' ;;
"--no-newline --type text/uri-list") printf '# from Files\r\nfile:///home/me/My%%20Shots/a.png\r\nfile://localhost/tmp/b.png\r\n' ;;
*) exit 1 ;;
esac`)

	got, err := Read()

	require.NoError(t, err)
	assert.Equal(t, []string{"/home/me/My Shots/a.png", "/tmp/b.png"}, got.Paths)
	assert.Equal(t, "/home/me/My Shots/a.png\n/tmp/b.png", got.Text)
}

func TestRead_AURIListOfRemoteFilesIsText(t *testing.T) {
	fakeTools(t, `case "$*" in
--list-types) printf 'text/uri-list\ntext/plain\n' ;;
"--no-newline --type text/uri-list") printf 'https://example.com/a.png\n' ;;
--no-newline) printf 'https://example.com/a.png' ;;
*) exit 1 ;;
esac`)

	got, err := Read()

	require.NoError(t, err)
	assert.Nil(t, got.Paths)
	assert.Equal(t, "https://example.com/a.png", got.Text)
}

// wl-copy forks a server once the selection is set; that server must live on.
func TestWrite_LeavesTheSelectionServerRunning(t *testing.T) {
	pids := filepath.Join(t.TempDir(), "pids")
	fakeTools(t, "cat >/dev/null; sleep 30 </dev/null >/dev/null 2>&1 & echo $! > "+pids)

	start := time.Now()
	require.NoError(t, Write("x"))

	assert.Less(t, time.Since(start), time.Second)
	server := readPIDs(t, pids)
	require.Len(t, server, 1)
	t.Cleanup(func() { _ = syscall.Kill(server[0], syscall.SIGKILL) })
	assert.False(t, gone(server[0]))
}

func TestRuntimeDir_FallsBackToTheLogindPath(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", "")

	assert.Equal(t, filepath.Join("/run/user", strconv.Itoa(os.Getuid())), runtimeDir())
}
