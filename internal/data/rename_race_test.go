package data

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/domain"
	"github.com/ilyasturki/ordo/internal/merge"
)

// twoRows returns a store and its project holding rows a and b.
func twoRows(t *testing.T) (*Store, domain.Project) {
	t.Helper()
	s := NewStore(t.TempDir())
	p, err := withSamples(t, s, "Shared")
	require.NoError(t, err)
	p, err = s.WithProjectLocked(p.ID, func(p *domain.Project) error {
		p.Categories[0].Tasks = []domain.Task{
			{ID: "a", Title: "A", Status: domain.StatusTodo},
			{ID: "b", Title: "B", Status: domain.StatusTodo},
		}
		return nil
	})
	require.NoError(t, err)
	return s, p
}

// syncthingSwaps has the next write find, just before its rename, the file replaced the way
// Syncthing replaces it: a temp file of its own renamed over it, with no flock.
func syncthingSwaps(t *testing.T, s *Store, id string, edit func(p *domain.Project)) {
	t.Helper()
	swapped := false
	beforeRename = func(string) {
		if swapped {
			return
		}
		swapped = true
		p, _, _, err := s.readProject(id)
		require.NoError(t, err)
		edit(&p)
		data, err := domain.FileBytes(p)
		require.NoError(t, err)
		tmp := filepath.Join(s.Dir, ".syncthing."+id+".json.tmp")
		require.NoError(t, os.WriteFile(tmp, data, 0o644))
		require.NoError(t, os.Rename(tmp, s.projectPath(id)))
	}
	t.Cleanup(func() { beforeRename = func(string) {} })
}

func taskTitled(t *testing.T, p domain.Project, id string) domain.Task {
	t.Helper()
	for _, c := range p.Categories {
		for _, task := range c.Tasks {
			if task.ID == id {
				return task
			}
		}
	}
	t.Fatalf("no task %s", id)
	return domain.Task{}
}

func TestSnapshotRefusedWhenTheFileIsReplacedBeforeItsRename(t *testing.T) {
	s, p := twoRows(t)
	mine, err := s.LoadProjectByID(p.ID)
	require.NoError(t, err)
	mine.Categories[0].Tasks[0].Title = "A mine"
	syncthingSwaps(t, s, p.ID, func(p *domain.Project) { p.Categories[0].Tasks[1].Title = "B theirs" })

	require.ErrorIs(t, s.SaveProjectLocked(mine), ErrStaleProject)

	onDisk, err := NewStore(s.Dir).LoadProjectByID(p.ID)
	require.NoError(t, err)
	assert.Equal(t, "B theirs", taskTitled(t, onDisk, "b").Title, "the swapped-in file stays")
	_, err = os.Stat(s.tmpPath(p.ID))
	assert.ErrorIs(t, err, os.ErrNotExist, "the refused write leaves no temp file")

	out, err := s.LoadOutside(p.ID)
	require.NoError(t, err)
	merged := merge.Three(out.Parent, mine, out.Theirs)
	require.Empty(t, merged.Collisions)
	require.NoError(t, s.SaveProjectLocked(merged.Project))

	onDisk, err = NewStore(s.Dir).LoadProjectByID(p.ID)
	require.NoError(t, err)
	assert.Equal(t, "A mine", taskTitled(t, onDisk, "a").Title)
	assert.Equal(t, "B theirs", taskTitled(t, onDisk, "b").Title)
}

func TestLockedWriteRunsAgainOnAFileReplacedBeforeItsRename(t *testing.T) {
	s, p := twoRows(t)
	syncthingSwaps(t, s, p.ID, func(p *domain.Project) { p.Categories[0].Tasks[1].Title = "B theirs" })

	runs := 0
	got, err := s.WithProjectLocked(p.ID, func(p *domain.Project) error {
		runs++
		p.Categories[0].Tasks[0].Title = "A server"
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 2, runs, "the edit runs again on the file that replaced the one it read")
	assert.Equal(t, "B theirs", taskTitled(t, got, "b").Title)

	onDisk, err := NewStore(s.Dir).LoadProjectByID(p.ID)
	require.NoError(t, err)
	assert.Equal(t, "A server", taskTitled(t, onDisk, "a").Title)
	assert.Equal(t, "B theirs", taskTitled(t, onDisk, "b").Title)
}

func TestLockedWriteGivesUpOnAFileThatKeepsChanging(t *testing.T) {
	s, p := twoRows(t)
	beforeRename = func(id string) {
		data, err := os.ReadFile(s.projectPath(id))
		require.NoError(t, err)
		tmp := filepath.Join(s.Dir, ".syncthing.tmp")
		require.NoError(t, os.WriteFile(tmp, data, 0o644))
		require.NoError(t, os.Rename(tmp, s.projectPath(id)))
	}
	t.Cleanup(func() { beforeRename = func(string) {} })

	runs := 0
	_, err := s.WithProjectLocked(p.ID, func(p *domain.Project) error {
		runs++
		p.Name = "Never lands"
		return nil
	})
	require.ErrorIs(t, err, ErrStaleProject)
	assert.Equal(t, rewriteTries, runs)
	onDisk, err := NewStore(s.Dir).LoadProjectByID(p.ID)
	require.NoError(t, err)
	assert.Equal(t, "Shared", onDisk.Name)
}

func TestRenameRunsAgainOnAFileReplacedBeforeItsRename(t *testing.T) {
	s, p := twoRows(t)
	syncthingSwaps(t, s, p.ID, func(p *domain.Project) { p.Categories[0].Tasks[1].Title = "B theirs" })

	_, err := s.RenameProject(p.ID, "Renamed")
	require.NoError(t, err)

	onDisk, err := NewStore(s.Dir).LoadProjectByID(p.ID)
	require.NoError(t, err)
	assert.Equal(t, "Renamed", onDisk.Name)
	assert.Equal(t, "B theirs", taskTitled(t, onDisk, "b").Title)
}
