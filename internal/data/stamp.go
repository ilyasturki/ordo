package data

import (
	"errors"
	"fmt"
	"os"
)

// Returned only by snapshot saves; overwriting would discard another writer's edits.
var ErrStaleProject = errors.New("project changed on disk since it was loaded")

// lineage is one in-memory copy of a project, from the load that made it. Its
// snapshots are checked against version, which only its own saves advance, so
// a snapshot taken before a reload can't pass the reloaded copy's check.
type lineage struct {
	version os.FileInfo
	watch   FileWatch // sees past the versions this copy wrote or already reported
	queued  int       // snapshots handed in so far
	saved   int       // the last of them that reached disk
	err     error
	origin  []byte // the version last taken from outside, what the copy's edits started from
	written []byte // the last snapshot of this copy that reached disk
}

// FileWatch tells when a file moved on from the version it last saw, by stat
// alone, so it can run on every tick. Not safe for concurrent use.
type FileWatch struct {
	path string
	seen os.FileInfo
}

// NewFileWatch starts from the file as it is now, missing or not.
func NewFileWatch(path string) *FileWatch {
	w := &FileWatch{path: path}
	w.Changed()
	return w
}

// Changed reports a new version once, a removal or a return counting as one.
// A caller that wrote the file itself calls it to see past its own write.
func (w *FileWatch) Changed() bool {
	info, err := os.Stat(w.path)
	if err != nil {
		info = nil
	}
	same := w.seen == nil && info == nil
	if w.seen != nil && info != nil {
		same = sameVersion(w.seen, info)
	}
	w.seen = info
	return !same
}

// UnsavedError names a project whose last edits never reached disk.
type UnsavedError struct {
	Name  string
	Edits int
	Err   error
}

func (e *UnsavedError) Error() string {
	edits := "edits"
	if e.Edits == 1 {
		edits = "edit"
	}
	return fmt.Sprintf("%d %s to %q not saved: %v", e.Edits, edits, e.Name, e.Err)
}

func (e *UnsavedError) Unwrap() error { return e.Err }

// Atomic rename changes the inode per write; size and mtime cover inode reuse.
func sameVersion(a, b os.FileInfo) bool {
	return os.SameFile(a, b) && a.Size() == b.Size() && a.ModTime().Equal(b.ModTime())
}

// adopt starts a new copy of id at the version just read or written.
func (s *Store) adopt(id string, info os.FileInfo, raw []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.copies == nil {
		s.copies = make(map[string]*lineage)
	}
	s.copies[id] = &lineage{version: info, watch: FileWatch{path: s.projectPath(id), seen: info}, origin: raw}
}

// wrote moves l to the version its own save just put on disk.
func (s *Store) wrote(id string, l *lineage, info os.FileInfo, raw []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	l.version = info
	l.written = raw
	if l.origin == nil {
		l.origin = raw
	}
	l.watch = FileWatch{path: s.projectPath(id), seen: info}
}

// OutsideChange reports, once per version, that another writer changed the file
// of the open copy of id. Edits the copy still holds are refused from then on,
// whatever stopped them before. A snapshot in flight holds the answer back: its
// rename lands before its version is noted.
func (s *Store) OutsideChange(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	l := s.copies[id]
	if l == nil || l.version == nil || (l.queued != l.saved && l.err == nil) {
		return false
	}
	if !l.watch.Changed() {
		return false
	}
	if l.queued != l.saved {
		l.err = ErrStaleProject
		if l.watch.seen == nil {
			l.err = ErrProjectNotFound
		}
	}
	return true
}

func (s *Store) forget(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.copies, id)
}

// queue counts one more snapshot of the current copy of id. A project this
// process never loaded gets a copy without a version, whose first save passes.
func (s *Store) queue(id string) (*lineage, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.copies == nil {
		s.copies = make(map[string]*lineage)
	}
	l := s.copies[id]
	if l == nil {
		l = &lineage{}
		s.copies[id] = l
	}
	l.queued++
	return l, l.queued
}

// settle records how snapshot seq of l went, reporting whether the outcome is
// worth telling: every failure, and the first success after one.
func (s *Store) settle(l *lineage, seq int, err error) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	failed := l.err != nil
	l.err = err
	if err == nil {
		l.saved = max(l.saved, seq)
	}
	return err != nil || failed
}

func (s *Store) checkFresh(id string, l *lineage) error {
	s.mu.Lock()
	want := l.version
	s.mu.Unlock()
	if want == nil {
		return nil
	}
	info, err := os.Stat(s.projectPath(id))
	if err != nil {
		return err
	}
	if !sameVersion(info, want) {
		return ErrStaleProject
	}
	return nil
}

// DropUnsaved writes off the open copy's edits that never reached disk, once
// the user chose to lose them. Its version stays, so its next save is still checked.
func (s *Store) DropUnsaved(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if l := s.copies[id]; l != nil {
		l.err = nil
		l.saved = l.queued
	}
}

// Refuse holds the open copy as unsaved, so ordo's exit names edits the user walked away from.
func (s *Store) Refuse(id string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if l := s.copies[id]; l != nil {
		l.queued++
		l.err = err
	}
}

// Unsaved reports how many snapshots of the open copy of id have not reached
// disk, and why the last one failed. Snapshots still queued count until they land.
func (s *Store) Unsaved(id string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	l := s.copies[id]
	if l == nil {
		return 0, nil
	}
	return l.queued - l.saved, l.err
}
