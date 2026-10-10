package clipboard

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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

// The whole point of the package: a tmux server started from a TTY has no
// WAYLAND_DISPLAY, so the socket has to be found on disk instead.
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

// Callers distinguish "nothing to write with" from a real failure so they can
// fall back to OSC 52; that hinges on ErrNoBackend coming back unwrapped.
func TestWlCommand_ReportsErrNoBackendWithoutASocket(t *testing.T) {
	t.Setenv("WAYLAND_DISPLAY", "")
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())

	_, err := wlCommand(context.Background(), "wl-copy")
	assert.ErrorIs(t, err, ErrNoBackend)
}

func TestWlCommand_ReportsErrNoBackendWithoutTheTool(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	t.Setenv("WAYLAND_DISPLAY", "wayland-0")

	_, err := wlCommand(context.Background(), "wl-copy")
	assert.ErrorIs(t, err, ErrNoBackend)
}

func TestRead_GivesUpOnAHungTool(t *testing.T) {
	sleep, err := exec.LookPath("sleep")
	require.NoError(t, err)
	dir := t.TempDir()
	script := "#!/bin/sh\nexec " + sleep + " 2\n"
	for _, name := range []string{"wl-paste", "xclip", "xsel"} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(script), 0o755))
	}
	t.Setenv("PATH", dir)
	t.Setenv("WAYLAND_DISPLAY", "wayland-0")
	defer func(d time.Duration) { readTimeout = d }(readTimeout)
	readTimeout = 50 * time.Millisecond

	start := time.Now()
	_, err = Read()

	assert.ErrorIs(t, err, ErrTimeout)
	assert.Less(t, time.Since(start), time.Second)
}

func TestRead_AFailingToolIsNotAMissingOne(t *testing.T) {
	dir := t.TempDir()
	script := "#!/bin/sh\necho 'Nothing is copied' >&2\nexit 1\n"
	for _, name := range []string{"wl-paste", "xclip", "xsel"} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(script), 0o755))
	}
	t.Setenv("PATH", dir)
	t.Setenv("WAYLAND_DISPLAY", "wayland-0")

	_, err := Read()

	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrNoBackend, "an empty clipboard must not read as an unreachable one")
	assert.NotErrorIs(t, err, ErrTimeout)
}

func TestRuntimeDir_FallsBackToTheLogindPath(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", "")

	assert.Equal(t, filepath.Join("/run/user", strconv.Itoa(os.Getuid())), runtimeDir())
}
