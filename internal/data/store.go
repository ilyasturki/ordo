package data

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/ilyasturki/ordo/internal/domain"
	"github.com/ilyasturki/ordo/internal/fsutil"
)

var (
	ErrProjectNotFound      = errors.New("project not found")
	ErrDuplicateProjectName = errors.New("project name already exists")
)

type ProjectRepository interface {
	ListProjects() ([]domain.Project, error)
	LoadProject(selector string) (domain.Project, error)
	LoadProjectByID(id string) (domain.Project, error)
	SaveProjectLocked(project domain.Project) error
	WithProjectLocked(id string, fn func(*domain.Project) error) (domain.Project, error)
	CreateProject(name string) (domain.Project, error)
	CheckProjectName(name, excludeID string) error
	DeleteProject(id string) error
	ConflictFiles(id string) []string
	Unsaved(id string) (int, error)
	DropUnsaved(id string)
}

// Store manages JSON persistence in a directory.
type Store struct {
	Dir string

	mu     sync.Mutex
	copies map[string]*lineage
}

var _ ProjectRepository = (*Store)(nil)

func NewStore(dir string) *Store {
	return &Store{Dir: dir}
}

func (s *Store) Ensure() error {
	return os.MkdirAll(s.Dir, 0o755)
}

func (s *Store) ListProjects() ([]domain.Project, error) {
	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return []domain.Project{}, nil
		}
		return nil, err
	}
	projects := make([]domain.Project, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		id, ok := strings.CutSuffix(entry.Name(), ".json")
		if !ok || domain.ValidateID(id) != nil {
			continue
		}
		// Stamp deliberately dropped: listing must not refresh the open
		// project's baseline.
		project, _, err := s.loadProjectFile(s.projectPath(id))
		if err != nil {
			return nil, err
		}
		if project.ID != id {
			continue
		}
		projects = append(projects, project)
	}
	sort.Slice(projects, func(i, j int) bool {
		return strings.ToLower(projects[i].Name) < strings.ToLower(projects[j].Name)
	})
	return projects, nil
}

func (s *Store) LoadProjectByID(id string) (domain.Project, error) {
	// Every read reaches the filesystem through here, so this is where an ID
	// earns the right to become a path component. An ID that isn't well-formed
	// cannot name a project that exists, so it is reported as not-found rather
	// than as a distinct error — malformed and missing look identical to every
	// caller, confirming nothing about the shape of the path that was probed.
	if err := domain.ValidateID(id); err != nil {
		return domain.Project{}, ErrProjectNotFound
	}
	path := s.projectPath(id)
	if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
		return domain.Project{}, ErrProjectNotFound
	}
	project, info, err := s.loadProjectFile(path)
	if err != nil {
		return domain.Project{}, err
	}
	// Saves follow the inner ID, so a mismatch would split the project across two files.
	if project.ID != id {
		return domain.Project{}, ErrProjectNotFound
	}
	s.adopt(id, info)
	return project, nil
}

func (s *Store) LoadProject(selector string) (domain.Project, error) {
	// Fast path: a selector that is an exact project ID resolves with a single
	// file read instead of scanning and JSON-parsing every project on disk. Miss
	// falls through to a full scan for name selectors, case-insensitive IDs, or
	// the empty (first-project) selector.
	if strings.TrimSpace(selector) != "" {
		if project, err := s.LoadProjectByID(selector); err == nil {
			return project, nil
		} else if !errors.Is(err, ErrProjectNotFound) {
			return domain.Project{}, err
		}
	}
	projects, err := s.ListProjects()
	if err != nil {
		return domain.Project{}, err
	}
	if len(projects) == 0 {
		return domain.Project{}, ErrProjectNotFound
	}
	if strings.TrimSpace(selector) == "" {
		return s.LoadProjectByID(projects[0].ID)
	}
	needle := domain.NormalizeName(selector)
	for _, project := range projects {
		if strings.EqualFold(project.ID, selector) || domain.NormalizeName(project.Name) == needle {
			return s.LoadProjectByID(project.ID)
		}
	}
	return domain.Project{}, ErrProjectNotFound
}

// marshalProject snapshots a project to the exact bytes a save would persist,
// stamping UpdatedAt. It does no I/O and is cheap, so a latency-sensitive caller
// (e.g. the TUI event loop, via the async Saver) can run it to capture an
// immutable snapshot, then hand the bytes to WriteProjectLocked on a background
// goroutine where the fsync cost is hidden from the UI.
func (s *Store) marshalProject(project domain.Project) ([]byte, error) {
	project.UpdatedAt = domain.NowTimestamp()
	// Stamping on every save is also the migration: a pre-versioning file
	// gains the schema field the first time it is written back.
	project.Schema = domain.RequiredSchema(project)
	return json.MarshalIndent(project, "", "  ")
}

// writeProjectBytes atomically writes pre-marshaled bytes for id: an fsync'd
// temp file, renamed into place, then a directory fsync so the rename survives
// a crash. It performs no locking or existence check; callers layer those on.
// The version it returns is nil when the file can't be stat'ed afterwards.
func (s *Store) writeProjectBytes(id string, data []byte) (os.FileInfo, error) {
	if err := os.MkdirAll(s.Dir, 0o755); err != nil {
		return nil, err
	}
	if err := fsutil.WriteAtomic(s.projectPath(id), data, 0o644); err != nil {
		return nil, err
	}
	info, _ := os.Stat(s.projectPath(id))
	return info, nil
}

func (s *Store) saveProjectAtomic(project domain.Project) error {
	data, err := s.marshalProject(project)
	if err != nil {
		return err
	}
	info, err := s.writeProjectBytes(project.ID, data)
	if err != nil {
		return err
	}
	if info != nil {
		s.adopt(project.ID, info)
	}
	return nil
}

// createNewProjectLocked writes a brand-new project under the global lock,
// rejecting a name that collides with any existing project. prepare builds the
// project to save and runs while the lock is held, so it can safely consult
// on-disk state (e.g. to dodge an ID collision). This is the shared spine of
// CreateProject and ImportProject; keeping the lock/check/save discipline in
// one place stops the writers from drifting apart.
func (s *Store) createNewProjectLocked(name string, prepare func() (domain.Project, error)) (domain.Project, error) {
	// Hold the global lock across the duplicate-name check and the save so
	// two racing creates can't both observe a missing name and each write
	// their own project file.
	g, err := s.acquireGlobalLock()
	if err != nil {
		return domain.Project{}, err
	}
	defer g.Close()

	if err := s.checkProjectNameAvailableLocked(name, ""); err != nil {
		return domain.Project{}, err
	}
	project, err := prepare()
	if err != nil {
		return domain.Project{}, err
	}
	if err := s.saveNewProjectLocked(project); err != nil {
		return domain.Project{}, err
	}
	return project, nil
}

func (s *Store) CreateProject(name string) (domain.Project, error) {
	// Project names render in the TUI picker and the CLI list, so they get the
	// same single-line treatment task titles do. ImportProject validates its own
	// name via ValidateProjectText, which is why the check sits here rather than
	// in the shared createNewProjectLocked spine.
	if err := domain.ValidateLine(name); err != nil {
		return domain.Project{}, err
	}
	return s.createNewProjectLocked(name, func() (domain.Project, error) {
		project, err := domain.NewProject(name)
		if err != nil {
			return domain.Project{}, err
		}
		project.Categories, err = s.defaultCategories()
		if err != nil {
			return domain.Project{}, err
		}
		project.Categories = populateSampleTasks(project.Categories)
		return project, nil
	})
}

// ImportProject persists a fully-formed project parsed from an external file
// as a NEW project on disk. Unlike SaveProjectLocked it does not require the
// project to already exist, and unlike CreateProject it keeps the caller's
// categories and tasks instead of seeding defaults. The global lock is held
// across the duplicate-name check and the write so a concurrent create can't
// slip in with the same name. A blank ID or CreatedAt is filled in so imports
// of hand-written files still produce a valid record; an ID that collides with
// an existing project is replaced with a fresh one so an import can never
// silently overwrite (and destroy) another project — re-importing a project
// that still exists is rejected by the duplicate-name check instead.
func (s *Store) ImportProject(project domain.Project) (domain.Project, error) {
	// An imported file is the one genuinely untrusted input this tool takes, and
	// it never passes through the operations layer where writes are normally
	// validated — so it is checked here instead.
	if err := domain.ValidateProjectText(project); err != nil {
		return domain.Project{}, fmt.Errorf("importing project: %w", err)
	}
	domain.NormalizeNesting(&project)
	return s.createNewProjectLocked(project.Name, func() (domain.Project, error) {
		// ValidateID comes before projectExists so a malformed ID is never used
		// to stat a path; an ID like "../../x" would otherwise have become a
		// file written outside the data directory. A bad ID is replaced rather
		// than refused, matching how a blank or colliding one is already
		// handled — the ID is an internal detail the user never chose.
		if domain.ValidateID(project.ID) != nil || s.projectExists(project.ID) {
			id, err := domain.NewID()
			if err != nil {
				return domain.Project{}, err
			}
			project.ID = id
		}
		if project.CreatedAt == "" {
			project.CreatedAt = domain.NowTimestamp()
		}
		return project, nil
	})
}

// RenameProject updates a project's display name while holding the global
// lock, so the unique-name invariant survives concurrent renames/creates
// from other processes. excludeID lets the duplicate check skip the
// project being renamed.
func (s *Store) RenameProject(id, newName string) (domain.Project, error) {
	if err := domain.ValidateLine(newName); err != nil {
		return domain.Project{}, err
	}
	g, err := s.acquireGlobalLock()
	if err != nil {
		return domain.Project{}, err
	}
	defer g.Close()

	if err := s.checkProjectNameAvailableLocked(newName, id); err != nil {
		return domain.Project{}, err
	}
	// Per-project flock for the read-modify-write itself. Global lock is
	// already held, so the acquisition order is global → project.
	if _, err := os.Stat(s.projectPath(id)); errors.Is(err, fs.ErrNotExist) {
		return domain.Project{}, ErrProjectNotFound
	}
	pl, err := s.acquireProjectLock(id)
	if err != nil {
		return domain.Project{}, err
	}
	defer pl.Close()
	project, err := s.LoadProjectByID(id)
	if err != nil {
		return domain.Project{}, err
	}
	project.Name = newName
	if err := s.saveProjectAtomic(project); err != nil {
		return domain.Project{}, err
	}
	return project, nil
}

// CheckProjectName returns ErrDuplicateProjectName when a project other than
// excludeID already holds name, without writing anything. A caller that keeps
// the project in memory checks here, then saves the rename the usual way.
func (s *Store) CheckProjectName(name, excludeID string) error {
	if err := domain.ValidateLine(name); err != nil {
		return err
	}
	g, err := s.acquireGlobalLock()
	if err != nil {
		return err
	}
	defer g.Close()
	return s.checkProjectNameAvailableLocked(name, excludeID)
}

// checkProjectNameAvailableLocked returns ErrDuplicateProjectName if any
// existing project (other than excludeID) shares the normalized name.
// Caller must hold the global lock.
func (s *Store) checkProjectNameAvailableLocked(name, excludeID string) error {
	projects, err := s.ListProjects()
	if err != nil {
		return err
	}
	needle := domain.NormalizeName(name)
	for _, p := range projects {
		if p.ID == excludeID {
			continue
		}
		if domain.NormalizeName(p.Name) == needle {
			return fmt.Errorf("%w: %q", ErrDuplicateProjectName, name)
		}
	}
	return nil
}

func (s *Store) InitDefault() (domain.Project, error) {
	if err := s.Ensure(); err != nil {
		return domain.Project{}, err
	}
	projects, err := s.ListProjects()
	if err != nil {
		return domain.Project{}, err
	}
	if len(projects) > 0 {
		return s.LoadProjectByID(projects[0].ID)
	}
	return s.CreateProject("Default")
}

func (s *Store) defaultCategories() ([]domain.Category, error) {
	categories := make([]domain.Category, 0, len(domain.DefaultCategories))
	for _, name := range domain.DefaultCategories {
		category, err := domain.NewCategory(name)
		if err != nil {
			return nil, err
		}
		categories = append(categories, category)
	}
	return categories, nil
}

// Stat from the same fd as the bytes: os.Stat would pair new metadata with
// old content across a rename.
func (s *Store) loadProjectFile(path string) (domain.Project, os.FileInfo, error) {
	f, err := os.Open(path)
	if err != nil {
		return domain.Project{}, nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return domain.Project{}, nil, err
	}
	data, err := io.ReadAll(f)
	if err != nil {
		return domain.Project{}, nil, err
	}
	project, err := decodeProject(data, filepath.Base(path))
	return project, info, err
}

func decodeProject(data []byte, name string) (domain.Project, error) {
	var project domain.Project
	if err := json.Unmarshal(data, &project); err != nil {
		return domain.Project{}, err
	}
	// A file from a newer binary is refused, not repaired: in a fleet of
	// machines sharing a data directory, a half-upgraded member must fail
	// loudly rather than rewrite the file through an older format and drop
	// whatever the newer version added.
	if err := domain.CheckSchema(project); err != nil {
		return domain.Project{}, fmt.Errorf("%s: %w", name, err)
	}
	// Repair rather than reject: a project containing control characters —
	// written before these checks existed, or hand-edited — still opens, minus
	// the bytes a terminal would execute. Rejecting here would make a single
	// injected byte enough to lock the user out of their own project with no
	// in-app way to fix it. The stripped form becomes canonical on the next save.
	domain.StripProjectText(&project)
	domain.NormalizeNesting(&project)
	return project, nil
}

func (s *Store) projectPath(id string) string {
	return filepath.Join(s.Dir, fmt.Sprintf("%s.json", id))
}

// projectExists reports whether a project file with the given ID is already on
// disk. Callers that need a stable answer must hold the relevant lock.
func (s *Store) projectExists(id string) bool {
	_, err := os.Stat(s.projectPath(id))
	return err == nil
}

// Syncthing names the losing copy of a conflict "<id>.sync-conflict-<date>-<time>-<device>.json".
func (s *Store) ConflictFiles(id string) []string {
	if domain.ValidateID(id) != nil {
		return nil
	}
	entries, _ := os.ReadDir(s.Dir)
	prefix := id + ".sync-conflict-"
	var names []string
	for _, entry := range entries {
		if name := entry.Name(); strings.HasPrefix(name, prefix) && strings.HasSuffix(name, ".json") {
			names = append(names, name)
		}
	}
	return names
}

func (s *Store) lockPath(id string) string { return s.projectPath(id) + ".lock" }
func (s *Store) tmpPath(id string) string  { return s.projectPath(id) + ".tmp" }

// The .lock file stays: unlinking it breaks mutual exclusion for a process flocked to the old inode.
func (s *Store) DeleteProject(id string) error {
	f, err := s.acquireProjectLock(id)
	if err != nil {
		return err
	}
	defer f.Close()
	err = os.Remove(s.projectPath(id))
	if errors.Is(err, fs.ErrNotExist) {
		return ErrProjectNotFound
	}
	if err != nil {
		return err
	}
	_ = os.Remove(s.tmpPath(id))
	s.forget(id)
	return nil
}

type sampleTask struct {
	title    string
	status   string
	priority string
	estimate int
}

var sampleTasksByCategory = map[string][]sampleTask{
	"Feature": {
		{"Build the main dashboard", domain.StatusInProgress, domain.PriorityHigh, 480},
		{"Add user preferences panel", domain.StatusTodo, domain.PriorityMedium, 240},
	},
	"Fix": {
		{"Resolve login timeout issue", domain.StatusTodo, domain.PriorityHigh, 60},
		{"Fix date formatting in reports", domain.StatusCompleted, domain.PriorityLow, 30},
	},
	"Ergonomy": {
		{"Improve keyboard navigation", domain.StatusInProgress, domain.PriorityMedium, 120},
		{"Add dark mode support", domain.StatusTodo, domain.PriorityLow, 240},
	},
	"Documentation": {
		{"Write getting started guide", domain.StatusTodo, domain.PriorityMedium, 120},
		{"Document API endpoints", domain.StatusTodo, domain.PriorityLow, 480},
	},
	"Research": {
		{"Evaluate caching strategies", domain.StatusCompleted, domain.PriorityMedium, 240},
		{"Investigate performance bottlenecks", domain.StatusTodo, domain.PriorityHigh, 120},
	},
}

func populateSampleTasks(categories []domain.Category) []domain.Category {
	for i := range categories {
		samples, ok := sampleTasksByCategory[categories[i].Name]
		if !ok {
			continue
		}
		for _, s := range samples {
			task, err := domain.NewTask(s.title)
			if err != nil {
				continue
			}
			_ = task.SetStatus(s.status)
			_ = task.SetPriority(s.priority)
			task.SetEstimate(s.estimate)
			categories[i].AddTask(task)
		}
	}
	return categories
}
