package data

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sync"

	"phasionary/internal/fsutil"
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

type State struct {
	DirectoryProjects map[string]string   `json:"directory_projects,omitempty"`
	ProjectOrder      []string            `json:"project_order,omitempty"`
	FoldedCategories  map[string][]string `json:"folded_categories,omitempty"`
	// UnfoldedCategories holds the shipped versions unfolded by hand; they fold by default.
	UnfoldedCategories map[string][]string `json:"unfolded_categories,omitempty"`
	Cursors            map[string]Cursor   `json:"cursors,omitempty"`
}

// StateRepository abstracts persistence of UI state. Implemented by *StateManager.
type StateRepository interface {
	GetProjectForDir() string
	SetProjectForDir(id string) error
	GetProjectOrder() []string
	SetProjectOrder(order []string) error
	GetFoldedCategories(projectID string) []string
	SetFoldedCategories(projectID string, categoryIDs []string) error
	GetUnfoldedCategories(projectID string) []string
	SetUnfoldedCategories(projectID string, categoryIDs []string) error
	DeleteFoldedCategories(projectID string) error
	GetCursor(projectID string) Cursor
	SetCursor(projectID string, cursor Cursor) error
	DeleteCursor(projectID string) error
}

// StateManager reads and writes state.json. Multiple processes may share that
// file — the TUI, the CLI, a second TUI on the same data dir — and every save
// rewrites it whole, so writes go through update: reload, apply, save. The
// mutex covers concurrent goroutines within one process.
type StateManager struct {
	mu         sync.Mutex
	path       string
	currentDir string
	state      State
}

var _ StateRepository = (*StateManager)(nil)

// NewStateManager creates a state manager that persists to <stateDir>/state.json.
// currentDir is the working directory that owns the linked project; pass "" for an
// unscoped (global fallback) entry.
func NewStateManager(stateDir, currentDir string) *StateManager {
	return &StateManager{
		path:       filepath.Join(stateDir, "state.json"),
		currentDir: currentDir,
	}
}

func (m *StateManager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.load()
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

	// Use a temporary struct to handle migration from old format
	var raw struct {
		LastProjectID      string              `json:"last_project_id"`
		DirectoryProjects  map[string]string   `json:"directory_projects,omitempty"`
		ProjectOrder       []string            `json:"project_order,omitempty"`
		FoldedCategories   map[string][]string `json:"folded_categories,omitempty"`
		UnfoldedCategories map[string][]string `json:"unfolded_categories,omitempty"`
		Cursors            map[string]Cursor   `json:"cursors,omitempty"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	m.state.DirectoryProjects = raw.DirectoryProjects
	m.state.ProjectOrder = raw.ProjectOrder
	m.state.FoldedCategories = raw.FoldedCategories
	m.state.UnfoldedCategories = raw.UnfoldedCategories
	m.state.Cursors = raw.Cursors
	if m.state.DirectoryProjects == nil {
		m.state.DirectoryProjects = make(map[string]string)
	}

	// Migrate old last_project_id to directory_projects[""] if not already set
	if raw.LastProjectID != "" {
		if _, ok := m.state.DirectoryProjects[""]; !ok {
			m.state.DirectoryProjects[""] = raw.LastProjectID
		}
	}

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
	// every fold toggle and project switch, so a crash mid-write would truncate
	// it and lose every fold and the project ordering — cheap to prevent, and
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
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.state.DirectoryProjects == nil {
		return ""
	}
	return m.state.DirectoryProjects[m.currentDir]
}

// SetProjectForDir links the manager's current directory to the given project ID.
func (m *StateManager) SetProjectForDir(id string) error {
	return m.update(func(s *State) bool {
		if s.DirectoryProjects == nil {
			s.DirectoryProjects = make(map[string]string)
		}
		s.DirectoryProjects[m.currentDir] = id
		return true
	})
}

// UnlinkDir removes the link for the manager's current directory. Returns the
// previously linked project ID (empty if none was linked).
func (m *StateManager) UnlinkDir() (string, error) {
	var prev string
	err := m.update(func(s *State) bool {
		if s.DirectoryProjects == nil {
			return false
		}
		id, ok := s.DirectoryProjects[m.currentDir]
		if !ok {
			return false
		}
		prev = id
		delete(s.DirectoryProjects, m.currentDir)
		return true
	})
	if err != nil {
		return "", err
	}
	return prev, nil
}

func (m *StateManager) GetProjectOrder() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return slices.Clone(m.state.ProjectOrder)
}

func (m *StateManager) SetProjectOrder(order []string) error {
	return m.update(func(s *State) bool {
		// Clone: the cache outlives this call, and the caller is free to keep
		// mutating the slice it passed in.
		s.ProjectOrder = slices.Clone(order)
		return true
	})
}

// GetFoldedCategories returns the collapsed category IDs for a project. It
// re-reads state.json first: another process may fold categories while the TUI
// is running, and the TUI reads this at project open, so a cached read would
// show yesterday's folds until a restart.
func (m *StateManager) GetFoldedCategories(projectID string) []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	// A failed reload leaves the cache intact; stale folds beat no folds, and
	// there is no error channel here to report on.
	_ = m.load()
	if m.state.FoldedCategories == nil {
		return nil
	}
	return slices.Clone(m.state.FoldedCategories[projectID])
}

func (m *StateManager) SetFoldedCategories(projectID string, categoryIDs []string) error {
	return m.update(func(s *State) bool { return setProjectIDs(&s.FoldedCategories, projectID, categoryIDs) })
}

func (m *StateManager) GetUnfoldedCategories(projectID string) []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	_ = m.load()
	return slices.Clone(m.state.UnfoldedCategories[projectID])
}

func (m *StateManager) SetUnfoldedCategories(projectID string, categoryIDs []string) error {
	return m.update(func(s *State) bool { return setProjectIDs(&s.UnfoldedCategories, projectID, categoryIDs) })
}

// setProjectIDs reports no change for the same set in any order, so a fold toggle saves at most the list it touched.
func setProjectIDs(byProject *map[string][]string, projectID string, ids []string) bool {
	prev := (*byProject)[projectID]
	if len(prev) == len(ids) && containsAll(prev, ids) {
		return false
	}
	if len(ids) == 0 {
		delete(*byProject, projectID)
		return true
	}
	if *byProject == nil {
		*byProject = make(map[string][]string)
	}
	(*byProject)[projectID] = slices.Clone(ids)
	return true
}

func containsAll(set, ids []string) bool {
	for _, id := range ids {
		if !slices.Contains(set, id) {
			return false
		}
	}
	return true
}

func (m *StateManager) DeleteFoldedCategories(projectID string) error {
	return m.update(func(s *State) bool {
		_, folded := s.FoldedCategories[projectID]
		_, unfolded := s.UnfoldedCategories[projectID]
		if !folded && !unfolded {
			return false
		}
		delete(s.FoldedCategories, projectID)
		delete(s.UnfoldedCategories, projectID)
		return true
	})
}

// GetCursor returns the remembered cursor for a project, or the zero Cursor when
// none is stored. Like GetFoldedCategories it re-reads state.json first: the TUI
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
