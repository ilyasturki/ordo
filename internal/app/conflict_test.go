package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"phasionary/internal/data"
)

func writeConflict(t *testing.T, store *data.Store, name string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(store.Dir, name), []byte("{}"), 0o644))
}

func TestConflictNotice(t *testing.T) {
	store := data.NewStore(t.TempDir())
	project, err := store.CreateProject("Live")
	require.NoError(t, err)

	assert.Empty(t, conflictNotice(store, project.ID))

	first := project.ID + ".sync-conflict-20260921-203208-ICE5FOT.json"
	writeConflict(t, store, first)
	assert.Equal(t, "Conflict: "+first, conflictNotice(store, project.ID))

	writeConflict(t, store, project.ID+".sync-conflict-20261001-101010-ABCDEFG.json")
	assert.Equal(t, "Conflict: "+first+" (+1 more)", conflictNotice(store, project.ID))
}

func TestSelectProject_NamesConflictFile(t *testing.T) {
	m, _, _, beta := newPickerTestModel(t, "")
	name := beta.ID + ".sync-conflict-20260921-203208-ICE5FOT.json"
	writeConflict(t, m.deps.Store.(*data.Store), name)

	m.selectProject()

	assert.Equal(t, beta.ID, m.project.ID)
	assert.Equal(t, "Conflict: "+name, m.ui.Screen.StatusMsg)
}
