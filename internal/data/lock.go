package data

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"phasionary/internal/domain"
	"phasionary/internal/fsutil"
)

// globalLockName is the lock file used to serialize operations that span
// multiple projects (CreateProject, RenameProject). Lock-ordering rule:
// always acquire the global lock BEFORE any per-project lock to avoid
// deadlocks against per-project writers.
const globalLockName = ".global.lock"

func (s *Store) globalLockPath() string {
	return filepath.Join(s.Dir, globalLockName)
}

func (s *Store) acquireGlobalLock() (*os.File, error) {
	if err := os.MkdirAll(s.Dir, 0o755); err != nil {
		return nil, err
	}
	return fsutil.Lock(s.globalLockPath(), 0o644)
}

// SaveProjectLocked saves a project while holding an exclusive flock on a
// per-project lock file. Use this when multiple processes may write to the
// same data directory (e.g. TUI + CLI at the same time).
//
// The lock file lives next to the project JSON at `{id}.json.lock` and is
// kept on disk between calls; only the flock state matters. Closing the fd
// releases the advisory lock, so an explicit LOCK_UN isn't needed.
//
// Returns ErrProjectNotFound if the project's JSON file is missing — this
// prevents a stale in-memory snapshot from resurrecting a project that was
// just deleted by another process. Use saveNewProjectLocked for the
// create path.
func (s *Store) SaveProjectLocked(project domain.Project) error {
	data, err := s.marshalProject(project)
	if err != nil {
		return err
	}
	return s.WriteProjectLocked(project.ID, data)
}

// WriteProjectLocked persists pre-marshaled project bytes under the per-project
// flock, refusing to recreate a project whose JSON file was removed (same
// guarantee as SaveProjectLocked). It is the write half of the asynchronous
// save: the caller runs marshalProject to snapshot state on its own thread, and
// this absorbs the fsync off that thread. Bytes must come from marshalProject
// for the same id.
func (s *Store) WriteProjectLocked(id string, data []byte) error {
	f, err := s.acquireProjectLock(id)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := os.Stat(s.projectPath(id)); errors.Is(err, fs.ErrNotExist) {
		return ErrProjectNotFound
	}
	if err := s.checkFresh(id); err != nil {
		return err
	}
	return s.writeProjectBytes(id, data)
}

// saveNewProjectLocked writes a fresh project file under the per-project
// flock. Unlike SaveProjectLocked it does not require the file to already
// exist; only CreateProject should use it.
func (s *Store) saveNewProjectLocked(project domain.Project) error {
	f, err := s.acquireProjectLock(project.ID)
	if err != nil {
		return err
	}
	defer f.Close()
	return s.saveProjectAtomic(project)
}

// WithProjectLocked acquires the project flock, loads the project, runs fn,
// then saves — all under the same lock so concurrent processes cannot lose
// each other's updates between read and write.
//
// Returns ErrProjectNotFound without creating a .lock file when the project
// does not exist; this avoids unauthenticated 404 spam littering the data dir
// with empty lock files.
func (s *Store) WithProjectLocked(id string, fn func(*domain.Project) error) (domain.Project, error) {
	// Checked before the stat so a malformed ID never reaches the filesystem at
	// all, not even to probe whether a path outside the data directory exists.
	if err := domain.ValidateID(id); err != nil {
		return domain.Project{}, ErrProjectNotFound
	}
	if _, err := os.Stat(s.projectPath(id)); errors.Is(err, fs.ErrNotExist) {
		return domain.Project{}, ErrProjectNotFound
	}
	f, err := s.acquireProjectLock(id)
	if err != nil {
		return domain.Project{}, err
	}
	defer f.Close()
	// Re-check after locking: another process may have deleted the project
	// between the existence check and the flock acquisition.
	project, err := s.LoadProjectByID(id)
	if err != nil {
		return domain.Project{}, err
	}
	if err := fn(&project); err != nil {
		return project, err
	}
	if err := s.saveProjectAtomic(project); err != nil {
		return project, err
	}
	return project, nil
}

func (s *Store) acquireProjectLock(id string) (*os.File, error) {
	// Every write and delete takes this lock before touching a file, so
	// validating here covers the write half of the store the way
	// LoadProjectByID covers the read half — including any future caller that
	// forgets to check for itself. Without it, .lock creation would happily
	// follow "../" out of the data directory.
	if err := domain.ValidateID(id); err != nil {
		return nil, ErrProjectNotFound
	}
	if err := os.MkdirAll(s.Dir, 0o755); err != nil {
		return nil, err
	}
	return fsutil.Lock(s.lockPath(id), 0o644)
}
