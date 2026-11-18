package fsutil

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAtomicIfLeavesTheFileWhenTheCheckFails(t *testing.T) {
	path := filepath.Join(t.TempDir(), "p.json")
	if err := os.WriteFile(path, []byte("theirs"), 0o644); err != nil {
		t.Fatal(err)
	}
	changed := errors.New("changed")
	if _, err := WriteAtomicIf(path, []byte("mine"), 0o644, func() error { return changed }); !errors.Is(err, changed) {
		t.Fatalf("got %v, want the check's error", err)
	}
	if got, _ := os.ReadFile(path); string(got) != "theirs" {
		t.Fatalf("file holds %q", got)
	}
	if _, err := os.Stat(path + ".tmp"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("temp file left: %v", err)
	}
}

func TestWriteAtomicIfReturnsTheVersionItRenamedIn(t *testing.T) {
	path := filepath.Join(t.TempDir(), "p.json")
	info, err := WriteAtomicIf(path, []byte("mine"), 0o644, func() error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	now, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(info, now) || info.Size() != now.Size() || !info.ModTime().Equal(now.ModTime()) {
		t.Fatalf("returned %v, file is %v", info, now)
	}
}
