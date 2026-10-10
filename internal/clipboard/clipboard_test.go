package clipboard

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
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

func TestRun_ReportsErrNoBackendWithoutTheTool(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	t.Setenv("WAYLAND_DISPLAY", "wayland-0")

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

func TestWrite_HandsTheTextToTheCopyTool(t *testing.T) {
	out := filepath.Join(t.TempDir(), "copied")
	fakeTools(t, "cat > "+out)

	require.NoError(t, Write("Alpha\n\nbody"))

	got, err := os.ReadFile(out)
	require.NoError(t, err)
	assert.Equal(t, "Alpha\n\nbody", string(got))
}

// A tool waiting on a focus it never gets is killed with its children, not orphaned.
func TestRun_KillsAHungTool(t *testing.T) {
	for name, call := range map[string]func() error{
		"read":  func() error { _, err := Read(); return err },
		"write": func() error { return Write("x") },
	} {
		t.Run(name, func(t *testing.T) {
			pids := filepath.Join(t.TempDir(), "pids")
			fakeTools(t, "sleep 30 & echo $! >> "+pids+"; echo $$ >> "+pids+"; exec sleep 30")
			shortTimeout(t)

			start := time.Now()
			err := call()

			assert.ErrorIs(t, err, ErrTimeout)
			assert.Less(t, time.Since(start), time.Second)
			for _, pid := range readPIDs(t, pids) {
				assert.Eventually(t, func() bool { return gone(pid) }, time.Second, 10*time.Millisecond,
					"pid %d outlived the call", pid)
			}
		})
	}
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

func TestRead_AFailingToolIsNotAMissingOne(t *testing.T) {
	fakeTools(t, "echo 'Nothing is copied' >&2; exit 1")

	_, err := Read()

	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrNoBackend, "an empty clipboard must not read as an unreachable one")
	assert.NotErrorIs(t, err, ErrTimeout)
}

func TestRuntimeDir_FallsBackToTheLogindPath(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", "")

	assert.Equal(t, filepath.Join("/run/user", strconv.Itoa(os.Getuid())), runtimeDir())
}

// fakeTools stands body in for wl-copy and wl-paste, behind a socket name no
// compositor answers to.
func fakeTools(t *testing.T, body string) {
	t.Helper()
	sh, err := exec.LookPath("sh")
	require.NoError(t, err)
	sleep, err := exec.LookPath("sleep")
	require.NoError(t, err)
	dir := t.TempDir()
	script := "#!" + sh + "\nPATH=" + filepath.Dir(sleep) + ":$PATH\n" + body + "\n"
	for _, name := range []string{"wl-copy", "wl-paste"} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(script), 0o755))
	}
	t.Setenv("PATH", dir)
	t.Setenv("WAYLAND_DISPLAY", "wayland-test")
}

func shortTimeout(t *testing.T) {
	t.Helper()
	saved := timeout
	timeout = 100 * time.Millisecond
	t.Cleanup(func() { timeout = saved })
}

func readPIDs(t *testing.T, path string) []int {
	t.Helper()
	var data []byte
	require.Eventually(t, func() bool {
		var err error
		data, err = os.ReadFile(path)
		return err == nil && len(data) > 0
	}, time.Second, 10*time.Millisecond)
	var pids []int
	for _, field := range strings.Fields(string(data)) {
		pid, err := strconv.Atoi(field)
		require.NoError(t, err)
		pids = append(pids, pid)
	}
	return pids
}

// A killed orphan can linger as a zombie until its new parent reaps it.
func gone(pid int) bool {
	if err := syscall.Kill(pid, 0); errors.Is(err, syscall.ESRCH) {
		return true
	}
	stat, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return true
	}
	fields := strings.Fields(string(stat[bytes.LastIndexByte(stat, ')')+1:]))
	return len(fields) > 0 && fields[0] == "Z"
}
