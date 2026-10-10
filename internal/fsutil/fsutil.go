package fsutil

import (
	"os"
	"path/filepath"
	"syscall"
)

// WriteAtomic writes data to path via "<path>.tmp": an fsync'd temp file,
// renamed into place, then a directory fsync so the rename survives a crash. A
// reader therefore sees either the old file or the new one, never a
// half-written one. It performs no locking or existence check; callers layer
// those on. os.WriteFile alone only returns once the bytes are in the page
// cache, which is not crash-safe.
func WriteAtomic(path string, data []byte, perm os.FileMode) error {
	tmp := path + ".tmp"
	if err := writeSync(tmp, data, perm); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if dir, derr := os.Open(filepath.Dir(path)); derr == nil {
		_ = dir.Sync()
		_ = dir.Close()
	}
	return nil
}

func writeSync(path string, data []byte, perm os.FileMode) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

// Lock creates path if needed and takes an exclusive advisory flock on it.
// Closing the returned file releases the lock, so no explicit LOCK_UN is
// needed. The caller is responsible for the parent directory.
func Lock(path string, perm os.FileMode) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, perm)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		_ = f.Close()
		return nil, err
	}
	return f, nil
}
