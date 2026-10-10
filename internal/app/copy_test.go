package app

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"phasionary/internal/app/selection"
	"phasionary/internal/domain"
)

// copyProject has a task carrying a description so the FocusDescription row is
// meaningful to copy from.
func copyProject() domain.Project {
	return domain.Project{
		ID:   "p1",
		Name: "My Project",
		Categories: []domain.Category{{
			ID:   "c1",
			Name: "Cat A",
			Tasks: []domain.Task{
				{ID: "t1", Title: "Alpha", Description: "the body of alpha", Status: domain.StatusTodo},
				{ID: "sep", Title: "— divider —", Status: domain.StatusTodo, Kind: domain.KindSeparator},
				{ID: "t2", Title: "Bravo", Status: domain.StatusTodo},
			},
		}},
	}
}

func TestCopyTextForPosition(t *testing.T) {
	m := newTestModel(t, copyProject())

	cases := []struct {
		name string
		pos  selection.Position
		want string
	}{
		{"project", selection.Position{Kind: selection.FocusProject}, "My Project"},
		{"category", selection.Position{Kind: selection.FocusCategory, CategoryIndex: 0}, "Cat A"},
		{"task", selection.Position{Kind: selection.FocusTask, CategoryIndex: 0, TaskIndex: 0}, "Alpha\n\nthe body of alpha"},
		{"description", selection.Position{Kind: selection.FocusDescription, CategoryIndex: 0, TaskIndex: 0}, "the body of alpha"},
		{"separator", selection.Position{Kind: selection.FocusSeparator, CategoryIndex: 0, TaskIndex: 1}, "— divider —"},
		{"task without description", selection.Position{Kind: selection.FocusTask, CategoryIndex: 0, TaskIndex: 2}, "Bravo"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, m.copyTextForPosition(tc.pos))
		})
	}
}

// Regression: pressing `y` while focused on a description row must copy the
// description body, not an empty string.
func TestCopySelected_CopiesDescriptionBody(t *testing.T) {
	m := newTestModel(t, copyProject())
	// Description rows only exist as navigable positions when expanded.
	m.ui.Screen.ExpandDescriptions = true
	m.rebuildPositions()
	m.ui.Selection.SelectByPredicate(func(p selection.Position) bool {
		return p.Kind == selection.FocusDescription && p.CategoryIndex == 0 && p.TaskIndex == 0
	})

	pos, ok := m.selectedPosition()
	if !ok || pos.Kind != selection.FocusDescription {
		t.Fatalf("expected a focused description row, got %+v ok=%v", pos, ok)
	}
	assert.Equal(t, "the body of alpha", m.copyTextForPosition(pos))

	// A description copy is text-only: it must not stash a paste-able task.
	m.copySelected()
	assert.Empty(t, m.ui.Clipboard.Tasks)
}

// A copy that only reached the terminal via OSC 52 is unverifiable — the escape
// sequence gets no answer — so the status must say so rather than claim a
// confirmed write.
func TestClipboardResultMsg_MarksTheTerminalFallback(t *testing.T) {
	cases := []struct {
		name string
		msg  clipboardResultMsg
		want string
	}{
		{"utility", clipboardResultMsg{}, "Copied!"},
		{"terminal", clipboardResultMsg{viaTerminal: true}, "Copied! (via terminal)"},
		{"labelled utility", clipboardResultMsg{label: "URL: x"}, "Copied URL: x"},
		{"labelled terminal", clipboardResultMsg{label: "URL: x", viaTerminal: true}, "Copied URL: x (via terminal)"},
		{"failure", clipboardResultMsg{err: assert.AnError}, "Copy failed: " + assert.AnError.Error()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := newTestModel(t, copyProject())
			updated, _ := m.Update(tc.msg)
			assert.Equal(t, tc.want, updated.(model).ui.Screen.StatusMsg)
		})
	}
}
