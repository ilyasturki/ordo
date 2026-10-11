package app

import (
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/phasionary/internal/data"
)

func writeConflict(t *testing.T, store *data.Store, name string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(store.Dir, name), []byte("{}"), 0o644))
}

func TestConflictNotice(t *testing.T) {
	assert.Equal(t, "Conflict: a.json", conflictNotice([]string{"a.json"}))
	assert.Equal(t, "Conflict: a.json (+1 more)", conflictNotice([]string{"a.json", "b.json"}))
}

func TestAnnounceConflicts_NamesEachCopyOnce(t *testing.T) {
	m, _, alpha, _ := newPickerTestModel(t, "")
	store := m.deps.Store.(*data.Store)
	first := alpha.ID + ".sync-conflict-20260921-203208-ICE5FOT.json"
	writeConflict(t, store, first)

	m.announceConflicts("")
	assert.Equal(t, "Conflict: "+first, m.ui.Screen.StatusMsg, "named when the project opens")

	m.ui.Modes.ToProjectPicker()
	*m, _ = m.handleProjectPickerKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	assert.Empty(t, m.ui.Screen.StatusMsg, "leaving the picker does not repeat it")

	m.reloadProject()
	assert.Equal(t, "Reloaded from disk", m.ui.Screen.StatusMsg, "a reload does not repeat it")

	second := alpha.ID + ".sync-conflict-20261001-101010-ABCDEFG.json"
	writeConflict(t, store, second)
	m.reloadProject()
	assert.Equal(t, "Conflict: "+second, m.ui.Screen.StatusMsg, "a reload names only the new copy")
}

func TestSelectProject_NamesConflictFile(t *testing.T) {
	m, _, _, beta := newPickerTestModel(t, "")
	name := beta.ID + ".sync-conflict-20260921-203208-ICE5FOT.json"
	writeConflict(t, m.deps.Store.(*data.Store), name)

	m.selectProject()

	assert.Equal(t, beta.ID, m.project.ID)
	assert.Equal(t, "Conflict: "+name, m.ui.Screen.StatusMsg)
}

func TestPickerBackOntoOpenProject_NamesConflictFile(t *testing.T) {
	leaves := map[string]func(m *model){
		"enter": func(m *model) { m.selectProject() },
		"esc":   func(m *model) { *m, _ = m.handleProjectPickerKey(tea.KeyPressMsg{Code: tea.KeyEscape}) },
	}
	for name, leave := range leaves {
		t.Run(name, func(t *testing.T) {
			m, _, alpha, _ := newPickerTestModel(t, "")
			conflict := alpha.ID + ".sync-conflict-20260921-203208-ICE5FOT.json"
			writeConflict(t, m.deps.Store.(*data.Store), conflict)
			m.ui.Picker.selected = 0
			m.ui.Modes.ToProjectPicker()

			leave(m)

			assert.Equal(t, alpha.ID, m.project.ID)
			assert.Equal(t, "Conflict: "+conflict, m.ui.Screen.StatusMsg)
		})
	}
}

func TestReloadProject_NamesConflictFile(t *testing.T) {
	m, _, alpha, _ := newPickerTestModel(t, "")
	conflict := alpha.ID + ".sync-conflict-20260921-203208-ICE5FOT.json"
	writeConflict(t, m.deps.Store.(*data.Store), conflict)

	m.reloadProject()

	assert.Equal(t, "Conflict: "+conflict, m.ui.Screen.StatusMsg)
}
