package data

import (
	"encoding/json"
	"errors"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"sync"

	"github.com/ilyasturki/ordo/internal/fsutil"
)

// Cursor records which row the TUI was focused on, addressed by stable IDs
// rather than by row index: the list is rebuilt from scratch every session, and
// the CLI may have added or removed rows in between, so an index would point
// at an unrelated task. Kind is a plain string rather than the TUI's
// FocusKind so state.json stays readable and survives a reorder of that enum.
type Cursor struct {
	Kind       string `json:"kind"`
	CategoryID string `json:"category_id,omitempty"`
	TaskID     string `json:"task_id,omitempty"`
}

func (c Cursor) IsZero() bool {
	return c.Kind == ""
}

// Filter is the task filter of one project. A list holds the values to
// keep, any of which matches; a task must match every non-empty list.
type Filter struct {
	Statuses   []string `json:"statuses,omitempty"`
	Priorities []string `json:"priorities,omitempty"`
	Categories []string `json:"categories,omitempty"`
	Tags       []string `json:"tags,omitempty"`
}

func (f Filter) IsZero() bool {
	return len(f.Statuses) == 0 && len(f.Priorities) == 0 && len(f.Categories) == 0 && len(f.Tags) == 0
}

type State struct {
	// Older builds kept these three here: Load moves them to projects/, and nothing writes them any more.
	DirectoryProjects map[string]string   `json:"directory_projects,omitempty"`
	FoldedCategories  map[string][]string `json:"folded_categories,omitempty"`
	Filters           map[string]Filter   `json:"filters,omitempty"`

	LastOpened map[string]string `json:"last_opened,omitempty"`
	Cursors    map[string]Cursor `json:"cursors,omitempty"`
}

// StateRepository abstracts persistence of UI state. Implemented by *StateManager.
type StateRepository interface {
	GetProjectForDir() string
	SetProjectForDir(id string) error
	UnlinkDir() (string, error)
	GetLastOpened() map[string]string
	SetLastOpened(projectID, at string) error
	DeleteLastOpened(projectID string) error
	GetFoldedCategories(projectID string) []string
	SetFoldedCategories(projectID string, categoryIDs []string) error
	GetCursor(projectID string) Cursor
	SetCursor(projectID string, cursor Cursor) error
	DeleteCursor(projectID string) error
	GetFilter(projectID string) Filter
	SetFilter(projectID string, filter Filter) error
	ViewChanged(projectID string) bool
}

// StateManager reads and writes state.json, and the view files and directory links
// beside the projects. Multiple processes may share that
// file — the TUI, the CLI, a second TUI on the same data dir — and every save
// rewrites it whole, so writes go through update: reload, apply, save. The
// mutex covers concurrent goroutines within one process.
type StateManager struct {
	mu         sync.Mutex
	path       string
	currentDir string
	state      State
	views      *Store

	watchedView string
	viewWatch   *FileWatch
	viewOutside bool
}

var _ StateRepository = (*StateManager)(nil)

// NewStateManager creates a state manager that persists to <stateDir>/state.json,
// with the view files and directory links in <stateDir>/projects.
// currentDir is the working directory that owns the linked project; pass "" for no directory.
func NewStateManager(stateDir, currentDir string) *StateManager {
	return &StateManager{
		path:       filepath.Join(stateDir, "state.json"),
		currentDir: currentDir,
		views:      NewStore(filepath.Join(stateDir, "projects")),
	}
}

func (m *StateManager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.load(); err != nil {
		return err
	}
	m.migrate()
	return nil
}

// migrate moves what older builds kept in state.json. The file is saved last, so a
// crash on the way repeats the move; an entry that can't move yet, for a project
// not on disk or a write that failed, waits for the next start.
func (m *StateManager) migrate() {
	s := &m.state
	if len(s.DirectoryProjects)+len(s.FoldedCategories)+len(s.Filters) == 0 {
		return
	}
	ids := make(map[string]bool)
	for id := range s.FoldedCategories {
		ids[id] = true
	}
	for id := range s.Filters {
		ids[id] = true
	}
	moved := false
	for id := range ids {
		if !m.views.HasProject(id) || m.views.seedView(id, View{Folds: s.FoldedCategories[id], Filter: s.Filters[id]}) != nil {
			continue
		}
		delete(s.FoldedCategories, id)
		delete(s.Filters, id)
		moved = true
	}
	for dir, id := range s.DirectoryProjects {
		// "" came from a last_project_id of long ago: no directory ever read it.
		if dir != "" {
			if _, err := m.views.mergeLink(dir, id, true); err != nil {
				continue
			}
		}
		delete(s.DirectoryProjects, dir)
		moved = true
	}
	if moved {
		_ = m.save()
	}
}

// load refreshes the in-memory state from disk. Callers hold m.mu. On any error
// the cached state is left untouched, so a transient read failure degrades to
// stale data rather than to empty data.
func (m *StateManager) load() error {
	data, err := os.ReadFile(m.path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			m.state = State{}
			return nil
		}
		return err
	}

	var fresh State
	if err := json.Unmarshal(data, &fresh); err != nil {
		return err
	}
	m.state = fresh
	return nil
}

func (m *StateManager) Save() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.save()
}

// save writes the whole cached state to disk. Callers hold m.mu.
func (m *StateManager) save() error {
	data, err := json.MarshalIndent(m.state, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(m.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	// Same atomic write the project store uses. state.json is rewritten whole on
	// every cursor move and project switch, so a crash mid-write would truncate
	// it and lose the cursors and the open times — cheap to prevent, and
	// the projects beside it already get this treatment.
	return fsutil.WriteAtomic(m.path, data, 0o644)
}

// update re-reads state.json, applies fn to the fresh state, and writes it back
// when fn reports a change. The reload is what keeps two processes from
// clobbering each other: without it, a save built from a cache loaded at
// startup would drop keys another process wrote in the meantime, because each
// save rewrites the entire file.
func (m *StateManager) update(fn func(*State) bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.load(); err != nil {
		return err
	}
	if !fn(&m.state) {
		return nil
	}
	return m.save()
}

// GetProjectForDir returns the project ID owned by the manager's current
// directory, or "" if none is linked.
func (m *StateManager) GetProjectForDir() string {
	if m.currentDir == "" {
		return ""
	}
	return m.views.directoryProject(m.currentDir)
}

// SetProjectForDir links the manager's current directory to the given project ID.
func (m *StateManager) SetProjectForDir(id string) error {
	if m.currentDir == "" {
		return nil
	}
	_, err := m.views.mergeLink(m.currentDir, id, false)
	return err
}

// UnlinkDir removes the link for the manager's current directory. Returns the
// previously linked project ID (empty if none was linked).
func (m *StateManager) UnlinkDir() (string, error) {
	if m.currentDir == "" {
		return "", nil
	}
	return m.views.mergeLink(m.currentDir, "", false)
}

// GetLastOpened re-reads state.json first: another TUI
// may have opened a project since this manager last loaded. The values are
// RFC 3339 timestamps, the format of a project's updated_at.
func (m *StateManager) GetLastOpened() map[string]string {
	m.mu.Lock()
	defer m.mu.Unlock()
	_ = m.load()
	return maps.Clone(m.state.LastOpened)
}

func (m *StateManager) SetLastOpened(projectID, at string) error {
	return m.update(func(s *State) bool {
		if s.LastOpened[projectID] == at {
			return false
		}
		if s.LastOpened == nil {
			s.LastOpened = make(map[string]string)
		}
		s.LastOpened[projectID] = at
		return true
	})
}

func (m *StateManager) DeleteLastOpened(projectID string) error {
	return m.update(func(s *State) bool {
		if _, ok := s.LastOpened[projectID]; !ok {
			return false
		}
		delete(s.LastOpened, projectID)
		return true
	})
}

// GetFoldedCategories returns the folded category and task IDs for a project.
func (m *StateManager) GetFoldedCategories(projectID string) []string {
	view, _ := m.views.View(projectID)
	return view.Folds
}

func (m *StateManager) SetFoldedCategories(projectID string, categoryIDs []string) error {
	return m.writeView(projectID, ViewChange{Folds: &categoryIDs})
}

// writeView sees past its own write, but not past an outside one that landed just
// before it: that one's fields are in the file and not yet on screen. Both checks
// run under the project's lock, so a writer holding it can't land between them.
// A project deleted while it was open gets no file back.
func (m *StateManager) writeView(projectID string, change ViewChange) error {
	lock, err := m.views.acquireProjectLock(projectID)
	if err != nil {
		return err
	}
	defer lock.Close()
	if !m.views.projectExists(projectID) {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	watched := m.viewWatch != nil && m.watchedView == projectID
	if watched && m.viewWatch.Changed() {
		m.viewOutside = true
	}
	err = m.views.mergeViewLocked(projectID, change)
	if watched {
		m.viewWatch.Changed()
	}
	return err
}

// ViewChanged reports, once per version, that another writer changed the project's
// view file. The first call for a project starts watching it from now.
func (m *StateManager) ViewChanged(projectID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.viewWatch == nil || m.watchedView != projectID {
		m.watchedView, m.viewOutside = projectID, false
		m.viewWatch = NewFileWatch(m.views.viewPath(projectID))
		return false
	}
	changed := m.viewWatch.Changed() || m.viewOutside
	m.viewOutside = false
	return changed
}

// GetCursor returns the remembered cursor for a project, or the zero Cursor when
// none is stored. It re-reads state.json first: the TUI
// reads this at project open, and a second TUI instance may have written its own
// cursor since this manager last loaded.
func (m *StateManager) GetCursor(projectID string) Cursor {
	m.mu.Lock()
	defer m.mu.Unlock()
	// A failed reload leaves the cache intact; a stale cursor beats no cursor,
	// and there is no error channel here to report on.
	_ = m.load()
	if m.state.Cursors == nil {
		return Cursor{}
	}
	return m.state.Cursors[projectID]
}

// SetCursor stores the focused row for a project. A zero Cursor clears the
// entry, so a project left with nothing selected does not pin the next session
// to a row that no longer means anything.
func (m *StateManager) SetCursor(projectID string, cursor Cursor) error {
	return m.update(func(s *State) bool {
		if cursor.IsZero() {
			if s.Cursors == nil {
				return false
			}
			if _, ok := s.Cursors[projectID]; !ok {
				return false
			}
			delete(s.Cursors, projectID)
			return true
		}
		if s.Cursors == nil {
			s.Cursors = make(map[string]Cursor)
		}
		if s.Cursors[projectID] == cursor {
			return false
		}
		s.Cursors[projectID] = cursor
		return true
	})
}

func (m *StateManager) DeleteCursor(projectID string) error {
	return m.update(func(s *State) bool {
		if s.Cursors == nil {
			return false
		}
		if _, ok := s.Cursors[projectID]; !ok {
			return false
		}
		delete(s.Cursors, projectID)
		return true
	})
}

func (m *StateManager) GetFilter(projectID string) Filter {
	view, _ := m.views.View(projectID)
	return view.Filter
}

func (m *StateManager) SetFilter(projectID string, filter Filter) error {
	return m.writeView(projectID, ViewChange{Filter: &filter})
}
