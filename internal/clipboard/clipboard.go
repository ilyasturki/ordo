// Package clipboard reads and writes the system clipboard through the
// platform's tools (wl-clipboard, or pbcopy and pbpaste on macOS), and kills a
// tool still running when its timeout passes.
package clipboard

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
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

func Write(text string) error {
	if path := os.Getenv(EnvFile); path != "" {
		return os.WriteFile(path, []byte(text), 0o600)
	}
	return run(strings.NewReader(text), nil, copyTool)
}

// Read returns the clipboard's text. ErrNoBackend only when no tool ran: one
// that ran and failed most likely found the clipboard empty.
func Read() (string, error) {
	if path := os.Getenv(EnvFile); path != "" {
		data, err := os.ReadFile(path)
		return string(data), err
	}
	var out bytes.Buffer
	// Not an OSC 52 read: tmux answers one from its own paste buffer, never the desktop clipboard.
	if err := run(nil, &out, pasteTool); err != nil {
		return "", err
	}
	return out.String(), nil
}

// run kills the tool's whole process group once the timeout passes.
func run(stdin io.Reader, stdout io.Writer, tool []string) error {
	path, err := exec.LookPath(tool[0])
	if err != nil {
		return ErrNoBackend
	}
	env, ok := toolEnv()
	if !ok {
		return ErrNoBackend
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, tool[1:]...)
	cmd.Env = env
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
