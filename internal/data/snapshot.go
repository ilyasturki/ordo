package data

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"strings"

	"github.com/ilyasturki/ordo/internal/domain"
)

// Snapshot is a project as one read of its file found it. Project holds only the ID when
// Held is set or Err is.
type Snapshot struct {
	Project domain.Project
	Stamp   string
	// Held reports that the caller already holds this stamp, so the file wasn't parsed.
	Held bool
	Err  error
}

// StampOf names a version of a project file: it changes whenever the file's bytes do.
func StampOf(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:8])
}

// Stamp is the stamp of the project's file as it is now, empty when there is none.
func (s *Store) Stamp(id string) string {
	if domain.ValidateID(id) != nil {
		return ""
	}
	raw, err := os.ReadFile(s.projectPath(id))
	if err != nil {
		return ""
	}
	return StampOf(raw)
}

// Snapshots reads every project file once, so each stamp names the bytes its project came
// from, and parses only the ones whose stamp isn't have's for that ID.
func (s *Store) Snapshots(have map[string]string) ([]Snapshot, error) {
	entries, err := os.ReadDir(s.Dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []Snapshot
	for _, entry := range entries {
		id, ok := strings.CutSuffix(entry.Name(), ".json")
		if entry.IsDir() || !ok || domain.ValidateID(id) != nil {
			continue
		}
		raw, err := os.ReadFile(s.projectPath(id))
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		snap := Snapshot{Project: domain.Project{ID: id}, Stamp: StampOf(raw), Err: err}
		snap.Held = err == nil && have[id] == snap.Stamp
		if err == nil && !snap.Held {
			p, err := decodeProject(raw, entry.Name())
			switch {
			case err != nil:
				snap.Err = err
			case p.ID != id:
				continue
			default:
				snap.Project = p
			}
		}
		out = append(out, snap)
	}
	return out, nil
}
