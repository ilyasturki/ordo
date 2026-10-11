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
	fakeTools(t, `[ "$1" = --no-newline ] && printf 'from the browser'`)

	got, err := Read()

	require.NoError(t, err)
	assert.Equal(t, "from the browser", got)
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
