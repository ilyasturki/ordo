package data

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/ilyasturki/ordo/internal/domain"
	"github.com/ilyasturki/ordo/internal/fsutil"
)

const directoriesFile = ".directories.json"

// FilterNone is the filter value for the medium priority, which older builds called none, and the untagged bucket.
const FilterNone = "none"

// View is what every device shows of one project, kept in projects/<id>.view.json.
type View struct {
	Folds  []string `json:"folds,omitempty"`
	Filter Filter   `json:"filter,omitzero"`
}

// ViewChange names what one write changes; a nil field keeps what the file holds.
type ViewChange struct {
	Folds  *[]string `json:"folds,omitempty"`
	Filter *Filter   `json:"filter,omitempty"`
}

func (v View) IsZero() bool { return len(v.Folds) == 0 && v.Filter.IsZero() }

func cleanIDs(ids []string) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if domain.ValidateID(id) == nil {
			out = append(out, id)
		}
	}
	slices.Sort(out)
	return slices.Compact(out)
}

// cleaned keeps what a filter row offers, sorted so two devices write the same bytes.
func (f Filter) cleaned() Filter {
	keep := func(values []string, offered ...string) []string {
		out := make([]string, 0, len(values))
		for _, v := range values {
			if v == "" {
				v = FilterNone
			}
			if slices.Contains(offered, v) {
				out = append(out, v)
			}
		}
		slices.Sort(out)
		return slices.Compact(out)
	}
	tags := make([]string, len(f.Tags))
	for i, t := range f.Tags {
		tags[i] = domain.CanonicalTagColor(t)
	}
	priorities := make([]string, len(f.Priorities))
	for i, p := range f.Priorities {
		priorities[i] = domain.CanonicalPriority(p)
	}
	return Filter{
		Statuses:   keep(f.Statuses, domain.StatusTodo, domain.StatusInProgress, domain.StatusCompleted, domain.StatusCancelled),
		Priorities: keep(priorities, append([]string{FilterNone}, domain.PriorityOrder...)...),
		Categories: cleanIDs(f.Categories),
		Tags:       keep(tags, append([]string{FilterNone}, domain.TagColorCycle...)...),
	}
}

func (s *Store) viewPath(id string) string {
	return filepath.Join(s.Dir, id+".view.json")
}

// HasProject reports whether id names a project file on disk.
func (s *Store) HasProject(id string) bool {
	return domain.ValidateID(id) == nil && s.projectExists(id)
}

// View reads what the project's view file holds, ok false when there is none. An unreadable file reads as empty.
func (s *Store) View(id string) (view View, ok bool) {
	if domain.ValidateID(id) != nil {
		return View{}, false
	}
	raw, err := os.ReadFile(s.viewPath(id))
	if err != nil {
		return View{}, !errors.Is(err, fs.ErrNotExist)
	}
	if json.Unmarshal(raw, &view) != nil {
		return View{}, true
	}
	view.Folds, view.Filter = cleanIDs(view.Folds), view.Filter.cleaned()
	return view, true
}

// MergeView applies change onto the file's current content under the project's
// lock, so a fold toggled here keeps a filter set elsewhere a moment ago.
func (s *Store) MergeView(id string, change ViewChange) error {
	lock, err := s.acquireProjectLock(id)
	if err != nil {
		return err
	}
	defer lock.Close()
	return s.mergeViewLocked(id, change)
}

func (s *Store) mergeViewLocked(id string, change ViewChange) error {
	return mergeJSON(s.viewPath(id), func(doc map[string]json.RawMessage, _ bool) error {
		if change.Folds != nil {
			folds := cleanIDs(*change.Folds)
			if err := setField(doc, "folds", folds, len(folds) == 0); err != nil {
				return err
			}
		}
		if change.Filter != nil {
			filter := change.Filter.cleaned()
			if err := setField(doc, "filter", filter, filter.IsZero()); err != nil {
				return err
			}
		}
		return nil
	})
}

// seedView writes view only while the project has no view file: one already there came from another device, and wins.
func (s *Store) seedView(id string, view View) error {
	lock, err := s.acquireProjectLock(id)
	if err != nil {
		return err
	}
	defer lock.Close()
	return mergeJSON(s.viewPath(id), func(doc map[string]json.RawMessage, existed bool) error {
		if existed {
			return nil
		}
		folds := cleanIDs(view.Folds)
		if err := setField(doc, "folds", folds, len(folds) == 0); err != nil {
			return err
		}
		filter := view.Filter.cleaned()
		return setField(doc, "filter", filter, filter.IsZero())
	})
}

func setField(doc map[string]json.RawMessage, key string, value any, clear bool) error {
	if clear {
		delete(doc, key)
		return nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	doc[key] = raw
	return nil
}

// mergeJSON runs one reload, apply and save on a JSON object, keeping every key
// apply leaves alone and leaving the file untouched when nothing changed: a
// rewrite is what Syncthing turns into a conflict copy on another device. A file
// that doesn't parse is refused rather than overwritten. The caller holds the lock.
func mergeJSON(path string, apply func(doc map[string]json.RawMessage, existed bool) error) error {
	doc := make(map[string]json.RawMessage)
	raw, err := os.ReadFile(path)
	existed := err == nil
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if existed {
		if err := json.Unmarshal(raw, &doc); err != nil {
			return fmt.Errorf("%s: %w", filepath.Base(path), err)
		}
		// A file holding null leaves the map nil.
		if doc == nil {
			doc = make(map[string]json.RawMessage)
		}
	}
	before, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	if err := apply(doc, existed); err != nil {
		return err
	}
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	if bytes.Equal(out, before) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return fsutil.WriteAtomic(path, out, 0o644)
}

// directoryKey is the key of dir in .directories.json: under the home directory it
// starts with ~, so the same place matches across machines whatever $HOME is.
func directoryKey(dir string) string {
	dir = filepath.Clean(dir)
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return dir
	}
	home = filepath.Clean(home)
	if dir == home {
		return "~"
	}
	if rest, ok := strings.CutPrefix(dir, home+string(filepath.Separator)); ok {
		return "~/" + rest
	}
	return dir
}

func (s *Store) directoriesPath() string {
	return filepath.Join(s.Dir, directoriesFile)
}

// directoryProject reads the link fresh, since another device may have added one.
func (s *Store) directoryProject(dir string) string {
	raw, err := os.ReadFile(s.directoriesPath())
	if err != nil {
		return ""
	}
	var links map[string]string
	if json.Unmarshal(raw, &links) != nil {
		return ""
	}
	return links[directoryKey(dir)]
}

// mergeLink changes one directory's key and keeps every other machine's, under the
// lock beside the file. An empty id removes the key; keep leaves a key already set.
func (s *Store) mergeLink(dir, id string, keep bool) (prev string, err error) {
	if err := os.MkdirAll(s.Dir, 0o755); err != nil {
		return "", err
	}
	lock, err := fsutil.Lock(s.directoriesPath()+".lock", 0o644)
	if err != nil {
		return "", err
	}
	defer lock.Close()
	key := directoryKey(dir)
	err = mergeJSON(s.directoriesPath(), func(doc map[string]json.RawMessage, _ bool) error {
		held, set := doc[key]
		if set {
			_ = json.Unmarshal(held, &prev)
			if keep {
				return nil
			}
		}
		return setField(doc, key, id, id == "")
	})
	return prev, err
}
