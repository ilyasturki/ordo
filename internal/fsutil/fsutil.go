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
	_, err := WriteAtomicIf(path, data, perm, nil)
	return err
}

// WriteAtomicIf is WriteAtomic with check run last before the rename, so a writer can refuse
// to replace a file that changed while it wrote; a check that fails leaves path as it was. It
// returns the version written, stat'ed before the rename that keeps it: a stat after could see
// a file another writer renamed over it meanwhile. That version is nil when the stat fails.
func WriteAtomicIf(path string, data []byte, perm os.FileMode, check func() error) (os.FileInfo, error) {
	tmp := path + ".tmp"
	if err := writeSync(tmp, data, perm); err != nil {
		return nil, err
	}
	info, _ := os.Stat(tmp)
	if check != nil {
		if err := check(); err != nil {
			_ = os.Remove(tmp)
			return nil, err
		}
	}
	if err := renameSynced(tmp, path); err != nil {
		return nil, err
	}
	return info, nil
}

// WriteAtomicUnique is WriteAtomic through a temp file of its own, still named
// "*.tmp", so two processes writing the same path never share one.
func WriteAtomicUnique(path string, data []byte, perm os.FileMode) error {
	f, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	err = f.Chmod(perm)
	if err == nil {
		_, err = f.Write(data)
	}
	if err == nil {
		err = f.Sync()
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return renameSynced(tmp, path)
}

func renameSynced(tmp, path string) error {
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
