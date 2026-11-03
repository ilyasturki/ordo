package app

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/app/selection"
	"github.com/ilyasturki/ordo/internal/domain"
)

func TestGAAddsASubCategoryUnderTheCursorsTopLevelCategory(t *testing.T) {
	for _, tc := range []struct {
		name   string
		focus  func(m *model) bool
		parent string
		want   []string
	}{
		{"on a top-level category", func(m *model) bool { return m.selectItem(selection.FocusCategory, "v2") }, "v2", []string{"v2", "ui<v2", "api<v2", "new<v2", "v1"}},
		{"on a task of a top-level category", func(m *model) bool { return m.selectTaskByID("loose") }, "v2", []string{"v2", "ui<v2", "api<v2", "new<v2", "v1"}},
		{"on a sub-category", func(m *model) bool { return m.selectItem(selection.FocusCategory, "ui") }, "v2", []string{"v2", "ui<v2", "api<v2", "new<v2", "v1"}},
		{"on a subtask in a sub-category", func(m *model) bool { return m.selectTaskByID("pick") }, "v2", []string{"v2", "ui<v2", "api<v2", "new<v2", "v1"}},
		{"on a category with none yet", func(m *model) bool { return m.selectItem(selection.FocusCategory, "v1") }, "v1", []string{"v2", "ui<v2", "api<v2", "v1", "new<v1"}},
		{"on a separator", func(m *model) bool {
			m.project.Categories[3].Tasks = append(m.project.Categories[3].Tasks, domain.Task{ID: "later", Kind: domain.KindSeparator, Title: "later"})
			m.rebuildPositions()
			return m.selectItem(selection.FocusSeparator, "later")
		}, "v1", []string{"v2", "ui<v2", "api<v2", "v1", "new<v1"}},
		{"on a description in a sub-category", func(m *model) bool {
			m.project.Categories[2].Tasks[0].Description = "details"
			m.ui.Screen.ExpandDescriptions = true
			m.rebuildPositions()
			return m.ui.Selection.SelectByPredicate(func(p selection.Position) bool { return p.Kind == selection.FocusDescription })
		}, "v2", []string{"v2", "ui<v2", "api<v2", "new<v2", "v1"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := newNestedModel(t)
			require.True(t, tc.focus(m))

			m.startAddingSubCategory()
			require.True(t, m.ui.Modes.IsEdit())
			typeText(m, "new")
			pressKey(m, tea.KeyEnter)

			require.False(t, m.ui.Modes.IsEdit())
			got := categoryParents(m.project)
			for i, c := range m.project.Categories {
				if c.Name == "new" {
					got[i] = "new<" + c.ParentID
				}
			}
			assert.Equal(t, tc.want, got)
			assert.Equal(t, "new", m.project.Categories[focusIndex(t, m)].Name)
		})
	}
}

func focusIndex(t *testing.T, m *model) int {
	t.Helper()
	pos, ok := m.selectedPosition()
	require.True(t, ok)
	return pos.CategoryIndex
}

func TestGAUnfoldsTheParent(t *testing.T) {
	m := newNestedModel(t)
	m.ui.Fold.Toggle("v2")
	m.rebuildPositions()
	require.True(t, m.selectItem(selection.FocusCategory, "v2"))

	m.startAddingSubCategory()

	assert.False(t, m.ui.Fold.IsFolded("v2"))
	pos, ok := m.selectedPosition()
	require.True(t, ok)
	assert.Equal(t, selection.FocusCategory, pos.Kind)
	assert.Equal(t, "v2", m.project.Categories[pos.CategoryIndex].ParentID)
}

func TestGAEscDropsTheNewSubCategory(t *testing.T) {
	m := newNestedModel(t)
	require.True(t, m.selectItem(selection.FocusCategory, "v1"))
	m.startAddingSubCategory()
	pressKey(m, tea.KeyEscape)
	assert.Equal(t, []string{"v2", "ui<v2", "api<v2", "v1"}, categoryParents(m.project))
}

func TestGAOnTheProjectRowSaysWhere(t *testing.T) {
	m := newNestedModel(t)
	m.jumpToFirst()
	m.startAddingSubCategory()
	assert.False(t, m.ui.Modes.IsEdit())
	assert.Equal(t, "Put the cursor on a category to add a sub-category", m.ui.Screen.StatusMsg)
}
