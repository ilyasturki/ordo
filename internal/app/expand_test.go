package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"phasionary/internal/app/selection"
	"phasionary/internal/domain"
	"phasionary/internal/ui"
)

func press(t *testing.T, m model, keys ...string) model {
	t.Helper()
	for _, k := range keys {
		r := []rune(k)[0]
		next, _ := m.Update(tea.KeyPressMsg{Code: r, Text: k})
		var ok bool
		m, ok = next.(model)
		require.True(t, ok)
	}
	return m
}

func expandProject(title, description string) domain.Project {
	return domain.Project{
		ID: "p1", Name: "P",
		Categories: []domain.Category{{
			ID: "c1", Name: "Cat",
			Tasks: []domain.Task{
				{ID: "t1", Title: title, Status: domain.StatusTodo, Description: description},
				{ID: "t2", Title: "after", Status: domain.StatusTodo},
			},
		}},
	}
}

func itemFor(t *testing.T, m *model, kind LayoutItemKind, taskIdx int) LayoutItem {
	t.Helper()
	for _, item := range m.buildLayout().Items {
		if item.Kind == kind && item.TaskIndex == taskIdx {
			return item
		}
	}
	t.Fatalf("no layout item of kind %d for task %d", kind, taskIdx)
	return LayoutItem{}
}

func assertDrawnAsReserved(t *testing.T, m *model, item LayoutItem) {
	t.Helper()
	drawn := strings.Count(m.renderLayoutItem(item), "\n") + 1
	assert.Equal(t, item.Height, drawn, "reserved %d rows, drew %d", item.Height, drawn)
}

func TestExpandRow_ShowsFullTitleUntilCursorLeaves(t *testing.T) {
	m := newTestModel(t, expandProject(longWords(600), ""))
	require.True(t, m.selectTaskIdx(0, 0))

	assert.Equal(t, ui.MaxLineRows, itemFor(t, m, LayoutTask, 0).Height, "a focused title stays clamped")

	am := press(t, *m, "z", "e")
	expanded := itemFor(t, &am, LayoutTask, 0)
	assert.Greater(t, expanded.Height, ui.MaxLineRows)
	assertDrawnAsReserved(t, &am, expanded)
	assert.Contains(t, am.renderLayoutItem(expanded), "ZEND")

	am = press(t, am, "j")
	assert.Empty(t, am.ui.Screen.ExpandedTaskID, "leaving the row collapses it")
	assert.Equal(t, ui.MaxLineRows, itemFor(t, &am, LayoutTask, 0).Height)

	am = press(t, am, "k")
	assert.Equal(t, ui.MaxLineRows, itemFor(t, &am, LayoutTask, 0).Height, "coming back finds it collapsed")
}

func TestExpandRow_ToggleCollapses(t *testing.T) {
	m := newTestModel(t, expandProject(longWords(600), ""))
	require.True(t, m.selectTaskIdx(0, 0))

	am := press(t, *m, "z", "e", "z", "e")

	assert.Empty(t, am.ui.Screen.ExpandedTaskID)
	assert.Equal(t, ui.MaxLineRows, itemFor(t, &am, LayoutTask, 0).Height)
}

func TestExpandRow_ShowsDescriptionWithDescriptionsHidden(t *testing.T) {
	m := newTestModel(t, expandProject("short", numberedLines(20)))
	require.True(t, m.selectTaskIdx(0, 0))
	require.False(t, m.ui.Screen.ExpandDescriptions)

	am := press(t, *m, "z", "e")
	desc := itemFor(t, &am, LayoutDescription, 0)
	assert.Equal(t, 20, desc.Height, "the whole body, past the focused cap")
	assertDrawnAsReserved(t, &am, desc)

	// The description belongs to the expanded row: moving onto it keeps it open.
	am = press(t, am, "j")
	pos, _ := am.selectedPosition()
	assert.Equal(t, selection.FocusDescription, pos.Kind)
	assert.Equal(t, "t1", am.ui.Screen.ExpandedTaskID)

	// Leaving drops the description row, and the cursor stays on the row it moved to.
	am = press(t, am, "j")
	assert.Empty(t, am.ui.Screen.ExpandedTaskID)
	pos, _ = am.selectedPosition()
	assert.Equal(t, selection.FocusTask, pos.Kind)
	assert.Equal(t, 1, pos.TaskIndex)
	for _, p := range am.positions() {
		assert.NotEqual(t, selection.FocusDescription, p.Kind)
	}
}

func TestExpandRow_JScrollsThroughATallRowBeforeMoving(t *testing.T) {
	m := newTestModel(t, expandProject("short", numberedLines(60)))
	m.ui.Screen.Height = 20
	require.True(t, m.selectTaskIdx(0, 0))

	am := press(t, *m, "z", "e", "j")
	pos, _ := am.selectedPosition()
	require.Equal(t, selection.FocusDescription, pos.Kind)
	start, end, ok := am.buildLayout().rowRange(am.selected())
	require.True(t, ok)
	require.Equal(t, start, am.ui.Screen.TopRow, "the tall row opens at its top")

	scrolls := 0
	for {
		before := am.ui.Screen.TopRow
		vp := NewViewport(am.buildLayout(), am.ui.Screen.Height, am.layoutConfig())
		endShown := end <= before+vp.windowAt(before)
		am = press(t, am, "j")
		pos, _ = am.selectedPosition()
		if pos.Kind != selection.FocusDescription {
			assert.True(t, endShown, "j moved on before the row's end showed")
			break
		}
		scrolls++
		require.Less(t, scrolls, 100)
		assert.Equal(t, before+1, am.ui.Screen.TopRow, "one row per press")
	}
	assert.Equal(t, 1, pos.TaskIndex, "j lands on the next task")
	assert.Greater(t, scrolls, 30, "the row scrolled through its body")
}

func TestExpandRow_KScrollsBackUpBeforeMoving(t *testing.T) {
	m := newTestModel(t, expandProject("short", numberedLines(60)))
	m.ui.Screen.Height = 20
	require.True(t, m.selectTaskIdx(0, 0))
	am := press(t, *m, "z", "e", "j")
	start, end, _ := am.buildLayout().rowRange(am.selected())
	vp := NewViewport(am.buildLayout(), am.ui.Screen.Height, am.layoutConfig())
	am.ui.Screen.TopRow = end - vp.windowAt(end) // bottom of the row on screen

	for am.ui.Screen.TopRow > start {
		before := am.ui.Screen.TopRow
		am = press(t, am, "k")
		require.Equal(t, before-1, am.ui.Screen.TopRow)
		pos, _ := am.selectedPosition()
		require.Equal(t, selection.FocusDescription, pos.Kind)
	}
	am = press(t, am, "k")
	pos, _ := am.selectedPosition()
	assert.Equal(t, selection.FocusTask, pos.Kind, "k moves on once the row's top has shown")
	assert.Equal(t, "t1", am.ui.Screen.ExpandedTaskID, "the title row is still the expanded task")
}

func TestExpandRow_OnlyTasks(t *testing.T) {
	m := newTestModel(t, expandProject("short", ""))
	require.True(t, m.ui.Selection.SelectByPredicate(func(p selection.Position) bool {
		return p.Kind == selection.FocusCategory
	}))

	am := press(t, *m, "z", "e")

	assert.Empty(t, am.ui.Screen.ExpandedTaskID)
	assert.Equal(t, "Only a task expands", am.ui.Screen.StatusMsg)
}
