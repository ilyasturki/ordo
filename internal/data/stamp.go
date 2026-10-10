package data

import (
	"errors"
	"os"
)

// Returned only by snapshot saves; overwriting would discard another writer's edits.
var ErrStaleProject = errors.New("project changed on disk since it was loaded")

// Atomic rename changes the inode per write; size and mtime cover inode reuse.
func sameVersion(a, b os.FileInfo) bool {
	return os.SameFile(a, b) && a.Size() == b.Size() && a.ModTime().Equal(b.ModTime())
}

func (s *Store) adopt(id string, info os.FileInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stamps == nil {
		s.stamps = make(map[string]os.FileInfo)
	}
	s.stamps[id] = info
}

func (s *Store) forget(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.stamps, id)
}

// A project this process never loaded has no baseline and passes.
func (s *Store) checkFresh(id string) error {
	s.mu.Lock()
	want, known := s.stamps[id]
	s.mu.Unlock()
	if !known {
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
