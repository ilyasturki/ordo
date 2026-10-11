package data

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/phasionary/internal/domain"
)

func TestDeleteProject(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	require.NoError(t, store.Ensure())

	project, err := store.CreateProject("Test Project")
	require.NoError(t, err)

	projectPath := filepath.Join(tmpDir, project.ID+".json")
	_, err = os.Stat(projectPath)
	require.NoError(t, err, "project file should exist")

	err = store.DeleteProject(project.ID)
	require.NoError(t, err)

	_, err = os.Stat(projectPath)
	assert.True(t, os.IsNotExist(err), "project file should be deleted")

	projects, err := store.ListProjects()
	require.NoError(t, err)
	assert.Empty(t, projects)
}

func TestDeleteProject_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	require.NoError(t, store.Ensure())

	err := store.DeleteProject("nonexistent-id")
	assert.ErrorIs(t, err, ErrProjectNotFound)
}

func TestRenameProject_RejectsDuplicateName(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	require.NoError(t, store.Ensure())

	a, err := store.CreateProject("Alpha")
	require.NoError(t, err)
	_, err = store.CreateProject("Beta")
	require.NoError(t, err)

	_, err = store.RenameProject(a.ID, "Beta")
	assert.ErrorIs(t, err, ErrDuplicateProjectName)

	got, err := store.LoadProjectByID(a.ID)
	require.NoError(t, err)
	assert.Equal(t, "Alpha", got.Name, "rename must not partially apply when rejected")
}

func TestRenameProject_AllowsSameName(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	require.NoError(t, store.Ensure())

	p, err := store.CreateProject("Alpha")
	require.NoError(t, err)

	updated, err := store.RenameProject(p.ID, "Alpha")
	require.NoError(t, err, "renaming a project to its current name must not trip the duplicate check")
	assert.Equal(t, "Alpha", updated.Name)
}

func TestRenameProject_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	require.NoError(t, store.Ensure())

	_, err := store.RenameProject("nonexistent-id", "Anything")
	assert.ErrorIs(t, err, ErrProjectNotFound)
}

func TestSaveProjectLocked_RejectsMissing(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	require.NoError(t, store.Ensure())

	p, err := store.CreateProject("Ghost")
	require.NoError(t, err)
	require.NoError(t, store.DeleteProject(p.ID))

	// Simulates a TUI/CLI process holding a stale in-memory snapshot and
	// trying to save after another process deleted the project. The save
	// must refuse rather than resurrect the deleted file.
	err = store.SaveProjectLocked(p)
	assert.ErrorIs(t, err, ErrProjectNotFound)

	_, statErr := os.Stat(filepath.Join(tmpDir, p.ID+".json"))
	assert.True(t, os.IsNotExist(statErr), "deleted project must not be resurrected by SaveProjectLocked")
}

func TestImportProject_RejectsDuplicateName(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	require.NoError(t, store.Ensure())

	original, err := store.CreateProject("Roadmap")
	require.NoError(t, err)

	// Re-importing a project that still exists must not silently overwrite it;
	// the duplicate-name check rejects it (a different ID doesn't help).
	_, err = store.ImportProject(domain.Project{ID: "some-other-id", Name: "Roadmap"})
	assert.ErrorIs(t, err, ErrDuplicateProjectName)

	reloaded, err := store.LoadProjectByID(original.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, reloaded.Categories, "the original project must be left intact")
}

func TestImportProject_CollidingIDGetsFreshID(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	require.NoError(t, store.Ensure())

	original, err := store.CreateProject("Alpha")
	require.NoError(t, err)

	// An import whose ID collides with a live project (but whose name is
	// unique) must be saved under a fresh ID rather than clobbering the
	// existing file.
	imported, err := store.ImportProject(domain.Project{ID: original.ID, Name: "Beta"})
	require.NoError(t, err)
	assert.NotEqual(t, original.ID, imported.ID, "colliding import ID must be replaced")

	reloaded, err := store.LoadProjectByID(original.ID)
	require.NoError(t, err)
	assert.Equal(t, "Alpha", reloaded.Name, "the original project must survive the import")
}

func TestImportProject_FillsBlankIDAndTimestamp(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	require.NoError(t, store.Ensure())

	imported, err := store.ImportProject(domain.Project{Name: "Hand-written"})
	require.NoError(t, err)
	assert.NotEmpty(t, imported.ID, "a blank ID must be filled in")
	assert.NotEmpty(t, imported.CreatedAt, "a blank CreatedAt must be filled in")

	_, err = store.LoadProjectByID(imported.ID)
	require.NoError(t, err)
}

func TestLoadProjectByID_RejectsBlankIDFile(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	require.NoError(t, store.Ensure())

	// A file with empty "id" would otherwise load as a zero-ID project and
	// later be saved back to "<dir>/.json", splitting state.
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "broken.json"), []byte(`{"id":"","name":"x"}`), 0o644))

	_, err := store.LoadProjectByID("broken")
	assert.ErrorIs(t, err, ErrProjectNotFound)
}

func TestSchemaStampedOnSave(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	require.NoError(t, store.Ensure())

	project, err := store.CreateProject("Versioned")
	require.NoError(t, err)

	loaded, err := store.LoadProjectByID(project.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, loaded.Schema, "a project without nesting stays readable by older builds")
}

func TestSchemaFollowsNesting(t *testing.T) {
	store := NewStore(t.TempDir())
	project, err := store.CreateProject("Nested")
	require.NoError(t, err)
	schemaOnDisk := func() int {
		t.Helper()
		loaded, err := store.LoadProjectByID(project.ID)
		require.NoError(t, err)
		return loaded.Schema
	}
	require.Equal(t, 1, schemaOnDisk())

	cat := &project.Categories[0]
	require.GreaterOrEqual(t, len(cat.Tasks), 2)
	require.NoError(t, cat.IndentTasks(1, 2))
	require.NoError(t, store.SaveProjectLocked(project))
	assert.Equal(t, 2, schemaOnDisk(), "a subtask needs schema 2")

	_, err = cat.OutdentTasks(1, 2)
	require.NoError(t, err)
	require.NoError(t, store.SaveProjectLocked(project))
	assert.Equal(t, 1, schemaOnDisk(), "without nesting the file drops back to 1")

	require.NoError(t, project.IndentCategories(1, 2))
	require.NoError(t, store.SaveProjectLocked(project))
	assert.Equal(t, 2, schemaOnDisk(), "a sub-category needs schema 2")
}

func TestLoadRepairsNesting(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	require.NoError(t, store.Ensure())

	raw := `{"schema":2,"id":"nest-1","name":"N","created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z","categories":[
	  {"id":"c1","name":"C","created_at":"2024-01-01T00:00:00Z","tasks":[
	    {"id":"a","title":"A","status":"todo","created_at":"x","updated_at":"x"},
	    {"id":"b","title":"B","status":"todo","created_at":"x","updated_at":"x"},
	    {"id":"a1","parent_id":"a","title":"A1","status":"todo","created_at":"x","updated_at":"x"},
	    {"id":"o","parent_id":"gone","title":"O","status":"todo","created_at":"x","updated_at":"x"}
	  ]}]}`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "nest-1.json"), []byte(raw), 0o644))

	project, err := store.LoadProjectByID("nest-1")
	require.NoError(t, err)
	tasks := project.Categories[0].Tasks
	var got []string
	for _, task := range tasks {
		got = append(got, task.ID+":"+task.ParentID)
	}
	assert.Equal(t, []string{"a:", "a1:a", "b:", "o:"}, got)
}

// A file written before versioning existed has no schema field; it must load
// as version 1 and gain the field on its next save.
func TestSchemaLegacyFileLoadsAndMigrates(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	require.NoError(t, store.Ensure())

	legacy := `{"id":"legacy-1","name":"Old","created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z","categories":[]}`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "legacy-1.json"), []byte(legacy), 0o644))

	project, err := store.LoadProjectByID("legacy-1")
	require.NoError(t, err)
	assert.Zero(t, project.Schema, "legacy file carries no schema before a save")

	require.NoError(t, store.SaveProjectLocked(project))
	migrated, err := store.LoadProjectByID("legacy-1")
	require.NoError(t, err)
	assert.Equal(t, 1, migrated.Schema)
}

// A half-upgraded machine sharing a data directory must refuse a file from a
// newer binary instead of rewriting it through an older format.
func TestSchemaTooNewRefused(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	require.NoError(t, store.Ensure())

	future := `{"schema":99,"id":"future-1","name":"Future","created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z","categories":[]}`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "future-1.json"), []byte(future), 0o644))

	_, err := store.LoadProjectByID("future-1")
	require.ErrorIs(t, err, domain.ErrSchemaTooNew)
	assert.Contains(t, err.Error(), "future-1.json", "error must name the file")

	_, err = store.ListProjects()
	require.ErrorIs(t, err, domain.ErrSchemaTooNew)
}

func TestListProjects_SkipsStrays(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	project, err := store.CreateProject("Live")
	require.NoError(t, err)

	conflict := filepath.Join(tmpDir, project.ID+".sync-conflict-20260921-203208-ICE5FOT.json")
	require.NoError(t, os.WriteFile(conflict, []byte("not json"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "renamed.json"), []byte(`{"id":"original","name":"Copy"}`), 0o644))

	projects, err := store.ListProjects()
	require.NoError(t, err)
	require.Len(t, projects, 1)
	assert.Equal(t, project.ID, projects[0].ID)

	got, err := os.ReadFile(conflict)
	require.NoError(t, err)
	assert.Equal(t, "not json", string(got), "the conflict file is left as it was")
}

func TestLoadProjectByID_RejectsMismatchedID(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	require.NoError(t, store.Ensure())
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "renamed.json"), []byte(`{"id":"original","name":"Copy"}`), 0o644))

	_, err := store.LoadProjectByID("renamed")
	assert.ErrorIs(t, err, ErrProjectNotFound)
}

func TestConflictFiles(t *testing.T) {
	tmpDir := filepath.Join(t.TempDir(), "[sync]")
	store := NewStore(tmpDir)
	project, err := store.CreateProject("Live")
	require.NoError(t, err)
	other, err := store.CreateProject("Other")
	require.NoError(t, err)

	assert.Empty(t, store.ConflictFiles(project.ID))

	names := []string{
		project.ID + ".sync-conflict-20260921-203208-ICE5FOT.json",
		project.ID + ".sync-conflict-20261001-101010-ABCDEFG.json",
	}
	for _, name := range names {
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, name), []byte("{}"), 0o644))
	}

	assert.Equal(t, names, store.ConflictFiles(project.ID))
	assert.Empty(t, store.ConflictFiles(other.ID))
	assert.Empty(t, store.ConflictFiles("../"+project.ID))
}
