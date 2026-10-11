package clipboard

import (
	"bytes"
	"errors"
	"image"
	"image/png"
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

func TestRun_ReportsErrNoBackendWithoutTheTool(t *testing.T) {
	t.Setenv(EnvFile, "")
	t.Setenv("PATH", t.TempDir())
	t.Setenv("WAYLAND_DISPLAY", "wayland-0")

	assert.ErrorIs(t, Write("x"), ErrNoBackend)
	_, err := Read()
	assert.ErrorIs(t, err, ErrNoBackend)
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
			fakeTools(t, "sleep 30 & echo $! $$ > "+pids+"; exec sleep 30")
			shortTimeout(t)

			start := time.Now()
			err := call()

			assert.ErrorIs(t, err, ErrTimeout)
			assert.Less(t, time.Since(start), 2*time.Second)
			for _, pid := range readPIDs(t, pids) {
				assert.Eventually(t, func() bool { return gone(pid) }, time.Second, 10*time.Millisecond,
					"pid %d outlived the call", pid)
			}
		})
	}
}

func TestRead_AFailingToolIsNotAMissingOne(t *testing.T) {
	fakeTools(t, "echo 'Nothing is copied' >&2; exit 1")

	_, err := Read()

	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrNoBackend, "an empty clipboard must not read as an unreachable one")
	assert.NotErrorIs(t, err, ErrTimeout)
}

func TestEnvFile_StandsInForTheClipboard(t *testing.T) {
	path := filepath.Join(t.TempDir(), "clipboard")
	t.Setenv(EnvFile, path)
	t.Setenv("PATH", t.TempDir())
	assert.True(t, Redirected())

	_, err := Read()
	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrNoBackend, "a file never written is an empty clipboard")

	require.NoError(t, Write("Alpha\n\nbody"))
	got, err := Read()
	require.NoError(t, err)
	assert.Equal(t, "Alpha\n\nbody", got.Text)
}

func TestRead_SniffsTheBytes(t *testing.T) {
	pngData := pngBytes(t)
	for name, tc := range map[string]struct {
		data      []byte
		wantText  string
		wantImage bool
		wantErr   error
	}{
		"text":                        {data: []byte("Alpha\n\nbody"), wantText: "Alpha\n\nbody"},
		"text starting like a bitmap": {data: []byte("BMW service"), wantText: "BMW service"},
		"a png":                       {data: pngData, wantImage: true},
		"other binary bytes":          {data: []byte("%PDF-1.7\n\x00\xe2\xe3"), wantErr: ErrNotText},
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "clipboard")
			require.NoError(t, os.WriteFile(path, tc.data, 0o600))
			t.Setenv(EnvFile, path)

			got, err := Read()

			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantText, got.Text)
			if tc.wantImage {
				assert.Equal(t, tc.data, got.Image)
			} else {
				assert.Nil(t, got.Image)
			}
		})
	}
}

func pngBytes(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 4, 3))))
	return buf.Bytes()
}

// fakeTools stands body in for the copy and paste tools, behind a socket name
// no compositor answers to.
func fakeTools(t *testing.T, body string) {
	t.Helper()
	sh, err := exec.LookPath("sh")
	require.NoError(t, err)
	sleep, err := exec.LookPath("sleep")
	require.NoError(t, err)
	dir := t.TempDir()
	script := "#!" + sh + "\nPATH=" + filepath.Dir(sleep) + ":$PATH\n" + body + "\n"
	for _, name := range []string{copyTool[0], pasteTool[0]} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(script), 0o755))
	}
	t.Setenv(EnvFile, "")
	t.Setenv("PATH", dir)
	t.Setenv("WAYLAND_DISPLAY", "wayland-test")
}

func shortTimeout(t *testing.T) {
	t.Helper()
	saved := timeout
	timeout = 500 * time.Millisecond
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
