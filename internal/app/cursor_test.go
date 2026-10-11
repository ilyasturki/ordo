package app

import (
	"fmt"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/app/selection"
	"github.com/ilyasturki/ordo/internal/data"
	"github.com/ilyasturki/ordo/internal/domain"
)

// selectTask moves the cursor onto the task with the given ID, failing the test
// when no such row is on screen.
func selectTask(t *testing.T, m *model, taskID string) {
	t.Helper()
	ok := m.ui.Selection.SelectByPredicate(func(p selection.Position) bool {
		if p.Kind != selection.FocusTask || p.CategoryIndex >= len(m.project.Categories) {
			return false
		}
		cat := m.project.Categories[p.CategoryIndex]
		return p.TaskIndex < len(cat.Tasks) && cat.Tasks[p.TaskIndex].ID == taskID
	})
	require.True(t, ok, "task %s has no row", taskID)
}

// selectedTaskID reports the task ID under the cursor, or "" when the cursor is
// not on a task row.
func selectedTaskID(t *testing.T, m *model) string {
	t.Helper()
	pos, ok := m.selectedPosition()
	require.True(t, ok, "nothing selected")
	if pos.Kind != selection.FocusTask {
		return ""
	}
	return m.project.Categories[pos.CategoryIndex].Tasks[pos.TaskIndex].ID
}

func TestSaveCursorState_StoresStableIDs(t *testing.T) {
	m := newTestModel(t, sampleProject())
	stub := withStubStateManager(m)
	selectTask(t, m, "t4")

	m.saveCursorState()

	assert.Equal(t, data.Cursor{Kind: "task", CategoryID: "c2", TaskID: "t4"}, stub.cursors["p1"],
		"the cursor must be stored by ID, not by row index")
}

func TestApplyStoredCursor_ReturnsToSavedTask(t *testing.T) {
	m := newTestModel(t, sampleProject())
	stub := withStubStateManager(m)
	stub.cursors["p1"] = data.Cursor{Kind: "task", CategoryID: "c2", TaskID: "t5"}

	m.applyStoredCursor()

	assert.Equal(t, "t5", selectedTaskID(t, m))
}

// A row index would silently point at a different task once rows shift. IDs
// must survive tasks being inserted above the remembered one.
func TestApplyStoredCursor_SurvivesRowsInsertedAbove(t *testing.T) {
	project := sampleProject()
	m := newTestModel(t, project)
	withStubStateManager(m)
	selectTask(t, m, "t4")
	m.saveCursorState()

	// Another process prepends two tasks to the first category.
	project.Categories[0].Tasks = append([]domain.Task{
		{ID: "new1", Title: "New one", Status: domain.StatusTodo},
		{ID: "new2", Title: "New two", Status: domain.StatusTodo},
	}, project.Categories[0].Tasks...)
	m.project = project
	m.rebuildPositions()

	m.applyStoredCursor()

	assert.Equal(t, "t4", selectedTaskID(t, m))
}

func TestApplyStoredCursor_DeletedTaskFallsBackToItsCategory(t *testing.T) {
	m := newTestModel(t, sampleProject())
	stub := withStubStateManager(m)
	stub.cursors["p1"] = data.Cursor{Kind: "task", CategoryID: "c2", TaskID: "gone"}

	m.applyStoredCursor()

	pos, ok := m.selectedPosition()
	require.True(t, ok)
	assert.Equal(t, selection.FocusCategory, pos.Kind)
	assert.Equal(t, "c2", m.project.Categories[pos.CategoryIndex].ID,
		"a deleted task should leave the cursor on its category, not at the top")
}

func TestApplyStoredCursor_DeletedCategoryFallsBackToFirstTask(t *testing.T) {
	m := newTestModel(t, sampleProject())
	stub := withStubStateManager(m)
	stub.cursors["p1"] = data.Cursor{Kind: "task", CategoryID: "gone", TaskID: "gone"}

	m.applyStoredCursor()

	assert.Equal(t, "t1", selectedTaskID(t, m))
}

func TestApplyStoredCursor_NoStoredCursorStartsOnFirstTask(t *testing.T) {
	m := newTestModel(t, sampleProject())
	withStubStateManager(m)

	m.applyStoredCursor()

	assert.Equal(t, "t1", selectedTaskID(t, m))
}

// A cursor inside a category that is folded on open has no row to land on. It
// must resolve to that category's header rather than to an unrendered row.
func TestApplyStoredCursor_InsideFoldedCategoryLandsOnHeader(t *testing.T) {
	m := newTestModel(t, sampleProject())
	stub := withStubStateManager(m)
	stub.cursors["p1"] = data.Cursor{Kind: "task", CategoryID: "c1", TaskID: "t2"}
	m.ui.Fold = NewFoldStateFrom([]string{"c1"}, nil)
	m.rebuildPositions()

	m.applyStoredCursor()

	pos, ok := m.selectedPosition()
	require.True(t, ok)
	assert.Equal(t, selection.FocusCategory, pos.Kind)
	assert.Equal(t, "c1", m.project.Categories[pos.CategoryIndex].ID)
}

func TestCursorRoundTrip_SeparatorRow(t *testing.T) {
	project := sampleProject()
	project.Categories[0].Tasks = append(project.Categories[0].Tasks,
		domain.Task{ID: "sep1", Kind: domain.KindSeparator})
	m := newTestModel(t, project)
	stub := withStubStateManager(m)
	m.rebuildPositions()
	require.True(t, m.ui.Selection.SelectByPredicate(func(p selection.Position) bool {
		return p.Kind == selection.FocusSeparator
	}))

	m.saveCursorState()
	assert.Equal(t, data.Cursor{Kind: "separator", CategoryID: "c1", TaskID: "sep1"}, stub.cursors["p1"])

	m.ui.Selection.SetSelected(0)
	m.applyStoredCursor()

	pos, ok := m.selectedPosition()
	require.True(t, ok)
	assert.Equal(t, selection.FocusSeparator, pos.Kind)
	assert.Equal(t, "sep1", m.project.Categories[pos.CategoryIndex].Tasks[pos.TaskIndex].ID)
}

// Descriptions are expanded per config, so a session that saved a description
// row may reopen with descriptions collapsed and that row gone.
func TestApplyStoredCursor_DescriptionRowCollapsedFallsBackToParentTask(t *testing.T) {
	project := sampleProject()
	project.Categories[0].Tasks[1].Description = "notes"
	m := newTestModel(t, project)
	stub := withStubStateManager(m)
	stub.cursors["p1"] = data.Cursor{Kind: "description", CategoryID: "c1", TaskID: "t2"}

	m.ui.Screen.ExpandDescriptions = false
	m.rebuildPositions()
	m.applyStoredCursor()
	assert.Equal(t, "t2", selectedTaskID(t, m), "collapsed descriptions should land on the parent task")

	m.ui.Screen.ExpandDescriptions = true
	m.rebuildPositions()
	m.applyStoredCursor()
	pos, ok := m.selectedPosition()
	require.True(t, ok)
	assert.Equal(t, selection.FocusDescription, pos.Kind, "expanded descriptions should land on the row itself")
}

// Switching projects hands the outgoing project its cursor and adopts the
// incoming one's, so each project reopens where it was left.
func TestSelectProject_RemembersCursorPerProject(t *testing.T) {
	store := data.NewStore(t.TempDir())
	require.NoError(t, store.Ensure())

	twoTasks := func(name, catID, t1, t2 string) domain.Project {
		return domain.Project{
			Name: name,
			Categories: []domain.Category{{
				ID:   catID,
				Name: "Cat",
				Tasks: []domain.Task{
					{ID: t1, Title: "First", Status: domain.StatusTodo},
					{ID: t2, Title: "Second", Status: domain.StatusTodo},
				},
			}},
		}
	}
	alpha, err := store.ImportProject(twoTasks("Alpha", "ca", "a1", "a2"))
	require.NoError(t, err)
	beta, err := store.ImportProject(twoTasks("Beta", "cb", "b1", "b2"))
	require.NoError(t, err)

	m := newTestModel(t, alpha)
	withStubStateManager(m)
	m.deps.Store = store

	openPicker := func(target domain.Project) {
		m.ui.Picker = ProjectPickerState{projects: []domain.Project{alpha, beta}}
		if target.ID == beta.ID {
			m.ui.Picker.selected = 1
		}
		m.selectProject()
	}

	// Park alpha on its second task, then leave for beta.
	selectTask(t, m, "a2")
	openPicker(beta)
	require.Equal(t, beta.ID, m.project.ID)
	assert.Equal(t, "b1", selectedTaskID(t, m), "a project with no saved cursor opens on its first task")

	// Park beta on its second task, then return to alpha.
	selectTask(t, m, "b2")
	openPicker(alpha)
	require.Equal(t, alpha.ID, m.project.ID)
	assert.Equal(t, "a2", selectedTaskID(t, m), "returning to a project must restore its own cursor")

	// Beta kept its cursor independently of alpha's.
	openPicker(beta)
	assert.Equal(t, "b2", selectedTaskID(t, m))
}

// longProject builds a project tall enough to overflow any test viewport, so
// scroll position is actually observable.
func longProject() domain.Project {
	tasks := make([]domain.Task, 40)
	for i := range tasks {
		tasks[i] = domain.Task{
			ID:     fmt.Sprintf("t%02d", i),
			Title:  fmt.Sprintf("Task %02d", i),
			Status: domain.StatusTodo,
		}
	}
	return domain.Project{
		ID:         "p1",
		Name:       "P",
		Categories: []domain.Category{{ID: "c1", Name: "Cat", Tasks: tasks}},
	}
}

// rowsAboveSelected reports which on-screen content row the cursor renders at,
// so a test can tell "centered" from "pinned to an edge".
func rowsAboveSelected(t *testing.T, m *model) int {
	t.Helper()
	layout := m.buildLayout()
	vp := NewViewport(layout, m.ui.Screen.Height, m.layoutConfig())
	vp.ComputeVisibility(m.ui.Screen.TopRow)
	rows := 0
	for i := vp.VisibleStart; i < vp.VisibleEnd; i++ {
		if layout.Items[i].PositionIndex == m.selected() {
			return rows
		}
		rows += layout.Items[i].Height
	}
	t.Fatalf("cursor is not visible at scroll offset %d", m.ui.Screen.TopRow)
	return -1
}

// Reopening carries no information about where the view sat, so a restored row
// deep in the list must land mid-screen rather than pinned to an edge. The
// terminal size only arrives after Run builds the model, so the centering waits
// for the first WindowSizeMsg.
func TestFirstWindowSize_CentersRestoredCursor(t *testing.T) {
	m := newTestModel(t, longProject())
	stub := withStubStateManager(m)
	stub.cursors["p1"] = data.Cursor{Kind: "task", CategoryID: "c1", TaskID: "t30"}
	m.applyStoredCursor()
	m.ui.Screen.PendingCenter = true
	m.ui.Screen.Height = 0 // not yet known, exactly as Run leaves it

	_, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	require.Equal(t, "t30", selectedTaskID(t, m), "the restored row must survive the resize")
	assert.False(t, m.ui.Screen.PendingCenter, "the one-shot flag must be consumed")

	// Scrolling down from offset 0 until the row merely becomes visible lands it
	// on the bottom edge (row 20 of 23); centering puts it near row 11.
	assert.InDelta(t, m.availableHeight()/2, rowsAboveSelected(t, m), 3,
		"a restored row should land near the vertical center, not pinned to an edge")
}

// Only the first size message centers. A later resize is a live view the user is
// looking at, so it must merely keep the cursor on screen.
func TestLaterWindowSize_DoesNotRecenter(t *testing.T) {
	m := newTestModel(t, longProject())
	withStubStateManager(m)
	m.ui.Screen.PendingCenter = true
	m.ui.Screen.Height = 0

	_, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	require.False(t, m.ui.Screen.PendingCenter)

	// Park the cursor deep in the list, then scroll so it sits visible but well
	// off-center — a position centering would move and ensureVisible would not.
	selectTask(t, m, "t30")
	m.centerOnSelected()
	offCenter := m.ui.Screen.TopRow + 3
	m.ui.Screen.TopRow = offCenter
	require.NotEqual(t, -1, rowsAboveSelected(t, m), "cursor must still be visible for this test to mean anything")

	_, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	assert.Equal(t, offCenter, m.ui.Screen.TopRow,
		"a resize must not re-center a view the user is already looking at")
}

func TestSelectProject_CentersRestoredCursor(t *testing.T) {
	store := data.NewStore(t.TempDir())
	require.NoError(t, store.Ensure())

	deep := longProject()
	deep.ID = ""
	deep.Name = "Deep"
	deep, err := store.ImportProject(deep)
	require.NoError(t, err)
	other, err := store.ImportProject(domain.Project{
		Name: "Other",
		Categories: []domain.Category{{
			ID:    "co",
			Name:  "Cat",
			Tasks: []domain.Task{{ID: "o1", Title: "Only", Status: domain.StatusTodo}},
		}},
	})
	require.NoError(t, err)

	m := newTestModel(t, other)
	stub := withStubStateManager(m)
	m.deps.Store = store
	m.ui.Screen.Height = 24
	stub.cursors[deep.ID] = data.Cursor{
		Kind:       "task",
		CategoryID: deep.Categories[0].ID,
		TaskID:     "t30",
	}

	m.ui.Picker = ProjectPickerState{projects: []domain.Project{other, deep}, selected: 1}
	m.selectProject()

	require.Equal(t, deep.ID, m.project.ID)
	require.Equal(t, "t30", selectedTaskID(t, m))
	assert.InDelta(t, m.availableHeight()/2, rowsAboveSelected(t, m), 3,
		"switching to a project should center its restored row, not pin it to an edge")
}

// A project with no rows must clear its entry rather than leave a stale one that
// would pin a later session to a row that no longer means anything.
func TestSaveCursorState_EmptyProjectClearsEntry(t *testing.T) {
	m := newTestModel(t, sampleProject())
	stub := withStubStateManager(m)
	stub.cursors["p1"] = data.Cursor{Kind: "task", CategoryID: "c1", TaskID: "t1"}

	m.project.Categories = nil
	m.rebuildPositions()
	m.ui.Selection.SetPositions(nil)
	m.saveCursorState()

	assert.NotContains(t, stub.cursors, "p1")
}
