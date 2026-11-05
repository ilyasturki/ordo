package data

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/ilyasturki/ordo/internal/domain"
	"github.com/ilyasturki/ordo/internal/merge"
)

// A copy older than this many writes falls back to comparing contents.
const writeLogLength = 64

type Outside struct {
	Theirs domain.Project
	Parent domain.Project
}

// LoadOutside adopts the file as a load does. Parent is what the copy last wrote when the file was
// built on it, else what its edits started from, so a swapped-in file can't lose them.
func (s *Store) LoadOutside(id string) (Outside, error) {
	s.mu.Lock()
	var origin, written []byte
	if l := s.copies[id]; l != nil {
		origin, written = l.origin, l.written
	}
	s.mu.Unlock()

	theirs, info, raw, err := s.readProject(id)
	if err != nil {
		return Outside{}, err
	}
	out := Outside{Theirs: theirs, Parent: theirs}
	if origin != nil {
		if out.Parent, err = decodeProject(origin, id); err != nil {
			return Outside{}, err
		}
	}
	if written != nil {
		wrote, err := decodeProject(written, id)
		if err != nil {
			return Outside{}, err
		}
		if s.builtOn(id, written, raw) || merge.Holds(out.Parent, wrote, theirs) {
			out.Parent = wrote
		}
	}
	s.adopt(id, info, raw)
	return out, nil
}

// Outside projects/ so it never syncs: it only speaks for writers under this machine's lock.
func (s *Store) writeLogPath(id string) string {
	return filepath.Join(filepath.Dir(s.Dir), "writes", id)
}

func (s *Store) readWriteLog(id string) []string {
	raw, err := os.ReadFile(s.writeLogPath(id))
	if err != nil {
		return nil
	}
	return strings.Fields(string(raw))
}

// logWrite chains the versions local writers made under the lock; any other version breaks the chain.
func (s *Store) logWrite(id string, replaced, wrote []byte) {
	log := s.readWriteLog(id)
	from := StampOf(replaced)
	if len(log) == 0 || log[len(log)-1] != from {
		log = []string{from}
	}
	log = append(log, StampOf(wrote))
	log = log[max(0, len(log)-writeLogLength):]
	path := s.writeLogPath(id)
	if os.MkdirAll(filepath.Dir(path), 0o755) == nil {
		_ = os.WriteFile(path, []byte(strings.Join(log, "\n")+"\n"), 0o644)
	}
}

func (s *Store) builtOn(id string, wrote, theirs []byte) bool {
	log := s.readWriteLog(id)
	return len(log) > 1 && log[len(log)-1] == StampOf(theirs) && slices.Contains(log[:len(log)-1], StampOf(wrote))
}

func (s *Store) WriteRecovery(project domain.Project, side string) (string, error) {
	raw, err := json.MarshalIndent(project, "", "  ")
	if err != nil {
		return "", err
	}
	dir := filepath.Join(filepath.Dir(s.Dir), "recovery")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	name := fmt.Sprintf("%s.%s.%s.json", project.ID, time.Now().UTC().Format("20060102T150405Z"), side)
	path := filepath.Join(dir, name)
	return path, os.WriteFile(path, raw, 0o644)
}
