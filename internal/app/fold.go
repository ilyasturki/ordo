package app

import (
	"github.com/ilyasturki/phasionary/internal/app/selection"
	"github.com/ilyasturki/phasionary/internal/data"
	"github.com/ilyasturki/phasionary/internal/domain"
)

func (m *model) toggleFold() {
	pos, ok := m.selectedPosition()
	if !ok {
		return
	}

	var id string
	switch pos.Kind {
	case selection.FocusCategory:
		id = pos.ID
	case selection.FocusTask, selection.FocusDescription:
		cat := m.project.Categories[pos.CategoryIndex]
		id = cat.ID
		if head := domain.BlockStart(cat.Tasks, pos.TaskIndex); domain.HasChildren(cat.Tasks, head) {
			id = cat.Tasks[head].ID
		}
	default:
		return
	}

	m.ui.Fold.Toggle(id)
	m.saveFoldState()
	m.rebuildPositions()
	m.ensureVisible()
}

func (m *model) foldAll() {
	categoryIDs := make([]string, len(m.project.Categories))
	for i, cat := range m.project.Categories {
		categoryIDs[i] = cat.ID
	}
	m.ui.Fold.FoldAll(categoryIDs)
	m.saveFoldState()
	m.rebuildPositions()
	m.ensureVisible()
}

func (m *model) unfoldAll() {
	m.ui.Fold.UnfoldAll()
	m.saveFoldState()
	m.rebuildPositions()
	m.ensureVisible()
}

func (m *model) toggleExpandDescriptions() {
	m.ui.Screen.ExpandDescriptions = !m.ui.Screen.ExpandDescriptions
	m.rebuildPositions()
	m.ensureVisible()
}

func (m *model) saveFoldState() {
	_ = m.deps.StateManager.SetFoldedCategories(m.project.ID, m.ui.Fold.FoldedIDs())
	_ = m.deps.StateManager.SetUnfoldedCategories(m.project.ID, m.ui.Fold.UnfoldedIDs())
}

func loadFoldState(sm data.StateRepository, project domain.Project) FoldState {
	f := NewFoldStateFrom(sm.GetFoldedCategories(project.ID), sm.GetUnfoldedCategories(project.ID))
	f.SetShipped(project.Categories)
	return f
}
