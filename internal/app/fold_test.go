package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/phasionary/internal/app/selection"
	"github.com/ilyasturki/phasionary/internal/data"
)

type stubStateManager struct {
	folded   map[string][]string
	unfolded map[string][]string
	cursors  map[string]data.Cursor
}

func newStubStateManager() *stubStateManager {
	return &stubStateManager{
		folded:   make(map[string][]string),
		unfolded: make(map[string][]string),
		cursors:  make(map[string]data.Cursor),
	}
}

func (s *stubStateManager) GetProjectForDir() string             { return "" }
func (s *stubStateManager) SetProjectForDir(id string) error     { return nil }
func (s *stubStateManager) GetProjectOrder() []string            { return nil }
func (s *stubStateManager) SetProjectOrder(order []string) error { return nil }
func (s *stubStateManager) GetFoldedCategories(projectID string) []string {
	return s.folded[projectID]
}
func (s *stubStateManager) SetFoldedCategories(projectID string, categoryIDs []string) error {
	s.folded[projectID] = append([]string(nil), categoryIDs...)
	return nil
}
func (s *stubStateManager) GetUnfoldedCategories(projectID string) []string {
	return s.unfolded[projectID]
}
func (s *stubStateManager) SetUnfoldedCategories(projectID string, categoryIDs []string) error {
	s.unfolded[projectID] = append([]string(nil), categoryIDs...)
	return nil
}
func (s *stubStateManager) DeleteFoldedCategories(projectID string) error {
	delete(s.folded, projectID)
	delete(s.unfolded, projectID)
	return nil
}
func (s *stubStateManager) GetCursor(projectID string) data.Cursor {
	return s.cursors[projectID]
}
func (s *stubStateManager) SetCursor(projectID string, cursor data.Cursor) error {
	if cursor.IsZero() {
		delete(s.cursors, projectID)
		return nil
	}
	s.cursors[projectID] = cursor
	return nil
}
func (s *stubStateManager) DeleteCursor(projectID string) error {
	delete(s.cursors, projectID)
	return nil
}

func withStubStateManager(m *model) *stubStateManager {
	stub := newStubStateManager()
	m.deps.StateManager = stub
	return stub
}

func TestToggleFold_OnCategory_FoldsAndHidesTasks(t *testing.T) {
	m := newTestModel(t, sampleProject())
	stub := withStubStateManager(m)
	m.ui.Selection.MoveTo(1) // Cat A header

	m.toggleFold()

	assert.True(t, m.ui.Fold.IsFolded("c1"))
	// Positions should now skip Cat A's tasks: project, Cat A, Cat B, t4, t5 = 5.
	assert.Equal(t, 5, m.ui.Selection.Count())
	assert.ElementsMatch(t, []string{"c1"}, stub.folded["p1"])
}

func TestToggleFold_OnTask_JumpsSelectionToCategoryHeader(t *testing.T) {
	m := newTestModel(t, sampleProject())
	withStubStateManager(m)
	m.ui.Selection.MoveTo(3) // t2 inside Cat A

	m.toggleFold()

	pos, ok := m.selectedPosition()
	require.True(t, ok)
	assert.Equal(t, selection.FocusCategory, pos.Kind)
	assert.Equal(t, 0, pos.CategoryIndex)
	assert.True(t, m.ui.Fold.IsFolded("c1"))
}

func TestToggleFold_NoopOnProjectRow(t *testing.T) {
	m := newTestModel(t, sampleProject())
	withStubStateManager(m)
	m.ui.Selection.MoveTo(0) // project row

	m.toggleFold()

	assert.False(t, m.ui.Fold.HasFolded())
}

func TestToggleFold_TwiceIsAnUnfold(t *testing.T) {
	m := newTestModel(t, sampleProject())
	withStubStateManager(m)
	m.ui.Selection.MoveTo(1) // Cat A

	m.toggleFold()
	require.True(t, m.ui.Fold.IsFolded("c1"))
	m.toggleFold()
	assert.False(t, m.ui.Fold.IsFolded("c1"))
}

func TestFoldAll_FoldsEveryCategory(t *testing.T) {
	m := newTestModel(t, sampleProject())
	stub := withStubStateManager(m)
	m.ui.Selection.MoveTo(3) // task inside Cat A

	m.foldAll()

	assert.True(t, m.ui.Fold.IsFolded("c1"))
	assert.True(t, m.ui.Fold.IsFolded("c2"))
	// Selection moved off the task onto Cat A header.
	pos, ok := m.selectedPosition()
	require.True(t, ok)
	assert.Equal(t, selection.FocusCategory, pos.Kind)
	// Only project + 2 category rows visible.
	assert.Equal(t, 3, m.ui.Selection.Count())
	assert.ElementsMatch(t, []string{"c1", "c2"}, stub.folded["p1"])
}

func TestUnfoldAll_ClearsAllFolds(t *testing.T) {
	m := newTestModel(t, sampleProject())
	stub := withStubStateManager(m)
	m.ui.Selection.MoveTo(1)
	m.toggleFold() // fold c1
	require.True(t, m.ui.Fold.IsFolded("c1"))

	m.unfoldAll()

	assert.False(t, m.ui.Fold.HasFolded())
	assert.Empty(t, stub.folded["p1"])
	// All rows visible again: project + 2 cats + 5 tasks = 8.
	assert.Equal(t, 8, m.ui.Selection.Count())
}
