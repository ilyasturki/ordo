package app

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/app/selection"
	"github.com/ilyasturki/ordo/internal/domain"
)

func TestMoveSelectedRow_DescriptionRowMovesParentTask(t *testing.T) {
	p := sampleProject()
	p.Categories[0].Tasks[0].Description = "alpha details"
	m := newTestModel(t, p)
	m.ui.Screen.ExpandDescriptions = true
	m.rebuildPositions()
	require.True(t, m.ui.Selection.SelectByPredicate(func(pp selection.Position) bool {
		return pp.Kind == selection.FocusDescription && pp.CategoryIndex == 0 && pp.TaskIndex == 0
	}), "focus t1's description row")

	m.moveSelectedRow(1)

	assert.Equal(t, []string{"t2", "t1", "t3"}, taskIDs(m.project.Categories[0]))
	pos, ok := m.selectedPosition()
	require.True(t, ok)
	assert.Equal(t, selection.FocusDescription, pos.Kind)
	assert.Equal(t, 1, pos.TaskIndex)
}

func TestMoveSelectedRow_HopsFilteredRunDownInOneStep(t *testing.T) {
	p := domain.Project{Categories: []domain.Category{{
		ID: "c1",
		Tasks: []domain.Task{
			{ID: "a", Title: "a", Status: domain.StatusTodo},
			{ID: "b", Title: "b", Status: domain.StatusCompleted},
			{ID: "c", Title: "c", Status: domain.StatusCompleted},
			{ID: "d", Title: "d", Status: domain.StatusTodo},
		},
	}}}
	m := newTestModel(t, p)
	m.ui.Filter = NewFilterState()
	m.ui.Filter.statuses = map[string]bool{domain.StatusTodo: true}
	m.rebuildPositions()
	require.True(t, m.selectTaskIdx(0, 0)) // a

	m.moveSelectedRow(1)

	// a hops past the hidden b/c run AND the visible neighbor d in one press,
	// so the visible order actually changes (a below d).
	assert.Equal(t, []string{"b", "c", "d", "a"}, taskIDs(m.project.Categories[0]))
	pos, ok := m.selectedPosition()
	require.True(t, ok)
	assert.Equal(t, "a", m.project.Categories[0].Tasks[pos.TaskIndex].ID)
}

func TestMoveSelectedRow_HopsFilteredRunUpInOneStep(t *testing.T) {
	p := domain.Project{Categories: []domain.Category{{
		ID: "c1",
		Tasks: []domain.Task{
			{ID: "a", Title: "a", Status: domain.StatusTodo},
			{ID: "b", Title: "b", Status: domain.StatusCompleted},
			{ID: "c", Title: "c", Status: domain.StatusTodo},
		},
	}}}
	m := newTestModel(t, p)
	m.ui.Filter = NewFilterState()
	m.ui.Filter.statuses = map[string]bool{domain.StatusTodo: true}
	m.rebuildPositions()
	require.True(t, m.selectTaskIdx(0, 2)) // c

	m.moveSelectedRow(-1)

	assert.Equal(t, []string{"c", "a", "b"}, taskIDs(m.project.Categories[0]))
}

func TestMoveSelectedRow_EdgesReportStatus(t *testing.T) {
	p := domain.Project{Categories: []domain.Category{{
		ID:    "c1",
		Tasks: []domain.Task{{ID: "a", Title: "a", Status: domain.StatusTodo}},
	}}}
	m := newTestModel(t, p)
	require.True(t, m.selectTaskIdx(0, 0))

	m.moveSelectedRow(-1)
	assert.Equal(t, "Already at the top", m.ui.Screen.StatusMsg)

	m.moveSelectedRow(1)
	assert.Equal(t, "Already at the bottom", m.ui.Screen.StatusMsg)
	assert.Equal(t, 0, m.ui.History.UndoDepth(), "refused moves record no history")
}

func TestMoveSelectedRow_RefusesFoldedDestination(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Fold.Toggle("c2")
	m.rebuildPositions()
	require.True(t, m.selectTaskIdx(0, 2)) // t3, last task of Cat A

	m.moveSelectedRow(1)

	assert.Contains(t, m.ui.Screen.StatusMsg, "folded")
	assert.Equal(t, []string{"t1", "t2", "t3"}, taskIDs(m.project.Categories[0]))
}

func TestMoveSelectedRow_RefusesWhenFilterHidesDestination(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Filter = NewFilterState()
	m.ui.Filter.categories = map[string]bool{"c1": true}
	m.rebuildPositions()
	require.True(t, m.selectTaskIdx(0, 2)) // t3, last task of Cat A

	m.moveSelectedRow(1)

	assert.Contains(t, m.ui.Screen.StatusMsg, "filter")
	assert.Equal(t, []string{"t1", "t2", "t3"}, taskIDs(m.project.Categories[0]))
	assert.Equal(t, []string{"t4", "t5"}, taskIDs(m.project.Categories[1]))
}

func TestVisualShift_ReportsNonContiguousSelection(t *testing.T) {
	p := domain.Project{Categories: []domain.Category{{
		ID: "c1",
		Tasks: []domain.Task{
			{ID: "a", Title: "a", Status: domain.StatusTodo},
			{ID: "b", Title: "b", Status: domain.StatusCompleted},
			{ID: "c", Title: "c", Status: domain.StatusTodo},
			{ID: "d", Title: "d", Status: domain.StatusTodo},
		},
	}}}
	m := newTestModel(t, p)
	m.ui.Filter = NewFilterState()
	m.ui.Filter.statuses = map[string]bool{domain.StatusTodo: true}
	m.rebuildPositions()
	require.True(t, m.selectTaskIdx(0, 0)) // a
	m.enterVisualMode()
	m.visualMoveCursor(1) // extends to c — b in between is hidden

	m.visualShift(1)

	assert.Contains(t, strings.ToLower(m.ui.Screen.StatusMsg), "contiguous")
	assert.Equal(t, []string{"a", "b", "c", "d"}, taskIDs(m.project.Categories[0]))
}
