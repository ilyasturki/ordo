package app

import "github.com/ilyasturki/ordo/internal/app/selection"

func (m *model) onHiddenRow() bool {
	pos, ok := m.selectedPosition()
	return ok && pos.Kind == selection.FocusHidden
}

// showHiddenTasks clears the whole filter from a hidden row, reporting whether the cursor was on one.
func (m *model) showHiddenTasks() bool {
	if !m.onHiddenRow() {
		return false
	}
	m.ui.Filter.ClearAll()
	m.rebuildPositions()
	m.ensureVisible()
	m.ui.Screen.StatusMsg = "Filter cleared"
	return true
}
