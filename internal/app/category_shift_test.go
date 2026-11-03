package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/app/selection"
	"github.com/ilyasturki/ordo/internal/domain"
)

func categoryParents(p domain.Project) []string {
	out := make([]string, len(p.Categories))
	for i, c := range p.Categories {
		out[i] = c.ID
		if c.ParentID != "" {
			out[i] += "<" + c.ParentID
		}
	}
	return out
}

func focusedCategory(t *testing.T, m *model) string {
	t.Helper()
	pos, ok := m.selectedPosition()
	require.True(t, ok)
	require.Equal(t, selection.FocusCategory, pos.Kind)
	return m.project.Categories[pos.CategoryIndex].ID
}

func TestJKPassesASubCategoryToTheNeighbouringParent(t *testing.T) {
	m := newNestedModel(t)
	require.True(t, m.selectItem(selection.FocusCategory, "api"))

	m.moveSelectedRow(+1)
	assert.Equal(t, []string{"v2", "ui<v2", "v1", "api<v1"}, categoryParents(m.project))
	assert.Equal(t, "api", focusedCategory(t, m))

	m.moveSelectedRow(-1)
	assert.Equal(t, []string{"v2", "ui<v2", "api<v2", "v1"}, categoryParents(m.project))
	assert.Equal(t, "api", focusedCategory(t, m))

	m.undo()
	assert.Equal(t, []string{"v2", "ui<v2", "v1", "api<v1"}, categoryParents(m.project))
}

func TestJKAcrossParentsUnfoldsAFoldedNeighbour(t *testing.T) {
	m := newNestedModel(t)
	m.ui.Fold.Toggle("v1")
	m.rebuildPositions()
	require.True(t, m.selectItem(selection.FocusCategory, "api"))

	m.moveSelectedRow(+1)

	assert.Equal(t, []string{"v2", "ui<v2", "v1", "api<v1"}, categoryParents(m.project))
	assert.False(t, m.ui.Fold.IsFolded("v1"))
	assert.Equal(t, "api", focusedCategory(t, m))
}

func TestJKAtTheOuterEdgeRefuses(t *testing.T) {
	m := newNestedModel(t)
	require.True(t, m.selectItem(selection.FocusCategory, "ui"))
	m.moveSelectedRow(-1)
	assert.Equal(t, "Already at the top of its parent", m.ui.Screen.StatusMsg)
	assert.Equal(t, []string{"v2", "ui<v2", "api<v2", "v1"}, categoryParents(m.project))
}

func TestJKAcrossParentsRefusesATakenName(t *testing.T) {
	p := nestedProject()
	p.Categories = append(p.Categories, domain.Category{ID: "api1", Name: "API", ParentID: "v1"})
	m := newTestModel(t, p)
	withStubStateManager(m)
	require.True(t, m.selectItem(selection.FocusCategory, "api"))
	m.moveSelectedRow(+1)
	assert.Equal(t, `"v0.1.0" already has a "api"`, m.ui.Screen.StatusMsg)
	assert.Equal(t, []string{"v2", "ui<v2", "api<v2", "v1", "api1<v1"}, categoryParents(m.project))
}

func TestJKSubtasksKeepTheirEdge(t *testing.T) {
	m := newNestedModel(t)
	require.True(t, m.selectTaskByID("sound"))
	m.moveSelectedRow(+1)
	assert.Equal(t, "Already at the bottom of its parent", m.ui.Screen.StatusMsg)
}
