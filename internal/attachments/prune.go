package attachments

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// PruneAge spares a file another device may have just referenced in an edit
// not synced here yet: Syncthing would carry the delete to it.
const PruneAge = 30 * 24 * time.Hour

// mention finds a stored file's name in any form: a reference, an absolute
// path copied out of ordo, or JSON-escaped.
var mention = regexp.MustCompile(`[0-9a-f]{32}\.(?:png|jpg|gif|webp)`)

// leftover is the temp file of an Add that never finished.
var leftover = regexp.MustCompile(`^\.[0-9a-f]{32}\.(?:png|jpg|gif|webp)\.\d+\.tmp$`)

func prunable(name string, referenced map[string]bool) bool {
	return fileName.MatchString(name) && !referenced[name] || leftover.MatchString(name)
}

type Orphan struct {
	Name    string
	Size    int64
	ModTime time.Time
}

// Unreferenced lists the files no project file names, conflict copies and
// writes in progress included, and that are older than PruneAge at now.
func (s *Store) Unreferenced(now time.Time) ([]Orphan, error) {
	entries, err := os.ReadDir(s.dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	referenced, err := mentioned(filepath.Dir(s.dir))
	if err != nil {
		return nil, err
	}
	var orphans []Orphan
	for _, e := range entries {
		name := e.Name()
		if !prunable(name, referenced) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return nil, err
		}
		if !info.Mode().IsRegular() || now.Sub(info.ModTime()) < PruneAge {
			continue
		}
		orphans = append(orphans, Orphan{Name: name, Size: info.Size(), ModTime: info.ModTime()})
	}
	return orphans, nil
}

func mentioned(projectsDir string) (map[string]bool, error) {
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return nil, err
	}
	names := map[string]bool{}
	for _, e := range entries {
		if !strings.Contains(e.Name(), ".json") {
			continue
		}
		path := filepath.Join(projectsDir, e.Name())
		// Stat, not the entry's type: a symlinked project file counts too.
		info, err := os.Stat(path)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if !info.Mode().IsRegular() {
			continue
		}
		data, err := os.ReadFile(path)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, name := range mention.FindAll(data, -1) {
			names[string(name)] = true
		}
	}
	return names, nil
}

// Remove deletes the named files from the store, and nothing else.
func (s *Store) Remove(orphans []Orphan) error {
	for _, o := range orphans {
		if !fileName.MatchString(o.Name) && !leftover.MatchString(o.Name) {
			return errors.New("not a stored image: " + o.Name)
		}
		if err := os.Remove(filepath.Join(s.dir, o.Name)); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}
	return nil
}
