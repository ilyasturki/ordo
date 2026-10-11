package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/app/selection"
)

func focusedTaskID(t *testing.T, m *model) string {
	t.Helper()
	pos, ok := m.selectedPosition()
	require.True(t, ok)
	require.Equal(t, selection.FocusTask, pos.Kind)
	return m.project.Categories[pos.CategoryIndex].Tasks[pos.TaskIndex].ID
}

func TestFocusTask_UnfoldsItsCategory(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Fold.Toggle("c2")
	m.rebuildPositions()

	require.NoError(t, m.focusTask("t5"))
	assert.Equal(t, "t5", focusedTaskID(t, m))
	assert.False(t, m.ui.Fold.IsFolded("c2"))
}

func TestFocusTask_MatchesTitle(t *testing.T) {
	m := newTestModel(t, sampleProject())
	require.NoError(t, m.focusTask("delta"))
	assert.Equal(t, "t4", focusedTaskID(t, m))
}

func TestFocusTask_UnknownTask(t *testing.T) {
	m := newTestModel(t, sampleProject())
	assert.ErrorContains(t, m.focusTask("nope"), `task "nope" not found`)
}
