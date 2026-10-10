package app

import "phasionary/internal/app/selection"

func (m *model) toggleFold() {
	pos, ok := m.selectedPosition()
	if !ok {
		return
	}

	var categoryID string
	switch pos.Kind {
	case selection.FocusCategory, selection.FocusTask:
		categoryID = m.project.Categories[pos.CategoryIndex].ID
	default:
		return
	}

	m.ui.Fold.Toggle(categoryID)
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
}
