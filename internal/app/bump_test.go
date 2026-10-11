package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/app/selection"
	"github.com/ilyasturki/ordo/internal/domain"
)

func bumpProject() domain.Project {
	return domain.Project{
		ID:        "p1",
		Name:      "aplorio",
		UpdatedAt: "2026-01-01T00:00:00Z",
		Categories: []domain.Category{
			{ID: "v001", Name: "v0.0.1", Tasks: []domain.Task{{ID: "a", Title: "Old", Status: domain.StatusTodo}}},
			{ID: "v010", Name: "v0.1.0 - app polish", UpdatedAt: "2026-01-01T00:00:00Z"},
			{ID: "v011", Name: "v0.1.1"},
			{ID: "v012", Name: "v0.1.2"},
			{ID: "mkt", Name: "MARKETING + SEO", Tasks: []domain.Task{{ID: "nix", Title: "update nixpkgs to 26.05", Status: domain.StatusTodo}}},
			{ID: "v100", Name: "v1.0.0"},
			{ID: "ui", Name: "ui", ParentID: "v100", Tasks: []domain.Task{{ID: "u", Title: "Theme", Status: domain.StatusTodo}}},
			{ID: "fut", Name: "FUTURE"},
			{ID: "v020", Name: "v0.2.0", ParentID: "fut", Tasks: []domain.Task{{ID: "f", Title: "Later", Status: domain.StatusTodo}}},
		},
	}
}

func newBumpModel(t *testing.T) *model {
	t.Helper()
	m := newTestModel(t, bumpProject())
	m.rebuildPositions()
	return m
}

func pressSweep(m *model, keys ...string) *model {
	for _, k := range keys {
		next, _ := m.Update(sweepKey(k))
		updated := asModel(next)
		m = &updated
	}
	return m
}

func categoryName(m *model, id string) string {
	ci, err := m.project.FindCategoryByID(id)
	if err != nil {
		return ""
	}
	return m.project.Categories[ci].Name
}

func TestBumpKeysOnACategoryHeader(t *testing.T) {
	for _, tc := range []struct {
		keys []string
		want string
	}{
		{[]string{"ctrl+a"}, "v0.1.1 - app polish"},
		{[]string{"g", "m"}, "v0.2.0 - app polish"},
		{[]string{"g", "M"}, "v1.0.0 - app polish"},
	} {
		m := newBumpModel(t)
		selectRow(t, m, selection.FocusCategory, "v010")
		m = pressSweep(m, tc.keys...)
		assert.Equal(t, tc.want, categoryName(m, "v010"), tc.keys)
		assert.Equal(t, "Bumped to "+tc.want, m.ui.Screen.StatusMsg)
		assert.NotEqual(t, "2026-01-01T00:00:00Z", m.project.Categories[1].UpdatedAt)
		assert.NotEqual(t, "2026-01-01T00:00:00Z", m.project.UpdatedAt)
	}
}

func TestBumpDownStopsAtZero(t *testing.T) {
	m := newBumpModel(t)
	selectRow(t, m, selection.FocusCategory, "v010")
	m = pressSweep(m, "ctrl+x")
	assert.Equal(t, "v0.1.0 - app polish", categoryName(m, "v010"))
	assert.Equal(t, "v0.1.0 - app polish can't go below 0", m.ui.Screen.StatusMsg)
	assert.False(t, m.ui.History.CanUndo())
}

func TestBumpIsOneUndoStep(t *testing.T) {
	m := newBumpModel(t)
	selectRow(t, m, selection.FocusCategory, "v012")
	m = pressSweep(m, "ctrl+a", "ctrl+a")
	assert.Equal(t, "v0.1.4", categoryName(m, "v012"))

	m.undo()
	assert.Equal(t, "v0.1.3", categoryName(m, "v012"))
	m.undo()
	assert.Equal(t, "v0.1.2", categoryName(m, "v012"))
	assert.False(t, m.ui.History.CanUndo())
}

func TestBumpFromATaskActsOnItsCategory(t *testing.T) {
	m := newBumpModel(t)
	selectRow(t, m, selection.FocusTask, "a")
	m = pressSweep(m, "ctrl+x")
	assert.Equal(t, "v0.0.0", categoryName(m, "v001"))
	assert.Equal(t, "Old", m.project.Categories[0].Tasks[0].Title)
}

func TestBumpFromASubCategoryWithoutAVersionActsOnItsParent(t *testing.T) {
	m := newBumpModel(t)
	selectRow(t, m, selection.FocusTask, "u")
	m = pressSweep(m, "g", "m")
	assert.Equal(t, "v1.1.0", categoryName(m, "v100"))
	assert.Equal(t, "ui", categoryName(m, "ui"))

	selectRow(t, m, selection.FocusCategory, "ui")
	m = pressSweep(m, "ctrl+a")
	assert.Equal(t, "No version in this name", m.ui.Screen.StatusMsg, "a header is its own target")
	assert.Equal(t, "v1.1.0", categoryName(m, "v100"))
}

func TestBumpASubCategoryVersion(t *testing.T) {
	m := newBumpModel(t)
	selectRow(t, m, selection.FocusTask, "f")
	m = pressSweep(m, "ctrl+a")
	assert.Equal(t, "v0.2.1", categoryName(m, "v020"))
	assert.Equal(t, "FUTURE", categoryName(m, "fut"))
}

func TestBumpWithoutAVersionLeavesTaskTitlesAlone(t *testing.T) {
	m := newBumpModel(t)
	selectRow(t, m, selection.FocusTask, "nix")
	m = pressSweep(m, "ctrl+a")
	assert.Equal(t, "No version in this name", m.ui.Screen.StatusMsg)
	assert.Equal(t, "update nixpkgs to 26.05", m.project.Categories[4].Tasks[0].Title)
	assert.Equal(t, "MARKETING + SEO", categoryName(m, "mkt"))
	assert.False(t, m.ui.History.CanUndo())
}

func TestBumpRefusesASiblingsName(t *testing.T) {
	m := newBumpModel(t)
	selectRow(t, m, selection.FocusCategory, "v011")
	m = pressSweep(m, "ctrl+a")
	assert.Equal(t, "v0.1.1", categoryName(m, "v011"))
	assert.Equal(t, "v0.1.2 already exists", m.ui.Screen.StatusMsg)
	assert.False(t, m.ui.History.CanUndo())
	assert.True(t, m.ui.Modes.IsNormal(), "no merge offer")
}

func TestVisualBumpRenamesTheWholeRangeAtOnce(t *testing.T) {
	m := newBumpModel(t)
	selectRow(t, m, selection.FocusCategory, "v011")
	m = pressSweep(m, "v", "j", "j", "ctrl+a")

	assert.Equal(t, "v0.1.2", categoryName(m, "v011"), "v0.1.2 moves on in the same step")
	assert.Equal(t, "v0.1.3", categoryName(m, "v012"))
	assert.Equal(t, "MARKETING + SEO", categoryName(m, "mkt"))
	assert.Equal(t, "Bumped 2 versions, 1 had none", m.ui.Screen.StatusMsg)
	assert.True(t, m.ui.Modes.IsVisual(), "the range stays selected for another press")

	m = pressSweep(m, "esc")
	m.undo()
	assert.Equal(t, "v0.1.1", categoryName(m, "v011"))
	assert.Equal(t, "v0.1.2", categoryName(m, "v012"))
	assert.False(t, m.ui.History.CanUndo())
}

func TestVisualBumpRefusesAClashWithAnUnselectedSibling(t *testing.T) {
	m := newBumpModel(t)
	m.project.Categories[1].Name = "v0.1.0"
	m.rebuildPositions()
	selectRow(t, m, selection.FocusCategory, "v011")
	m = pressSweep(m, "v", "j", "ctrl+x")

	assert.Equal(t, "v0.1.0 already exists", m.ui.Screen.StatusMsg)
	assert.Equal(t, "v0.1.1", categoryName(m, "v011"))
	assert.Equal(t, "v0.1.2", categoryName(m, "v012"))
	assert.False(t, m.ui.History.CanUndo())
}

func TestVisualBumpSkipsVersionsAtZero(t *testing.T) {
	m := newBumpModel(t)
	selectRow(t, m, selection.FocusCategory, "v010")
	m = pressSweep(m, "v", "j", "j", "ctrl+x")
	assert.Equal(t, "v0.1.0 - app polish", categoryName(m, "v010"))
	assert.Equal(t, "v0.1.0", categoryName(m, "v011"))
	assert.Equal(t, "v0.1.1", categoryName(m, "v012"))
	assert.Equal(t, "Bumped 2 versions, 1 at 0", m.ui.Screen.StatusMsg)
}

func TestVisualBumpNeedsCategories(t *testing.T) {
	m := newBumpModel(t)
	selectRow(t, m, selection.FocusTask, "a")
	m = pressSweep(m, "v", "ctrl+a")
	assert.Equal(t, "Select categories to bump them", m.ui.Screen.StatusMsg)
	assert.Equal(t, "v0.0.1", categoryName(m, "v001"))
}

func TestBumpOnTheProjectRow(t *testing.T) {
	m := newBumpModel(t)
	m.ui.Selection.SetSelected(0)
	m = pressSweep(m, "ctrl+a")
	assert.Equal(t, "Put the cursor on a category to bump its version", m.ui.Screen.StatusMsg)
	require.False(t, m.ui.History.CanUndo())
}
