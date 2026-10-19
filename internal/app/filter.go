package app

import (
	"github.com/ilyasturki/ordo/internal/app/selection"
	"github.com/ilyasturki/ordo/internal/data"
	"github.com/ilyasturki/ordo/internal/domain"
)

func loadFilterState(sm data.StateRepository, project domain.Project) FilterState {
	f := filterStateFrom(sm.GetFilter(project.ID))
	if f.pruneCategories(project.Categories) {
		_ = sm.SetFilter(project.ID, f.saved())
	}
	return f
}

func (m *model) saveFilter() {
	if m.deps.StateManager == nil || m.project.ID == "" {
		return
	}
	_ = m.deps.StateManager.SetFilter(m.project.ID, m.ui.Filter.saved())
}

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
	m.saveFilter()
	m.rebuildPositions()
	m.ensureVisible()
	m.ui.Screen.StatusMsg = "Filter cleared"
	return true
}

// holdFilteredOut keeps each edited row the filter stops matching where it is, until the cursor leaves it.
func (m *model) holdFilteredOut(ids []string) {
	if !m.ui.Filter.HasActiveFilter() {
		return
	}
	shown := make(map[string]bool)
	for _, p := range m.positions() {
		if p.Kind == selection.FocusTask {
			shown[p.ID] = true
		}
	}
	held := false
	for _, id := range ids {
		ci, ti := m.findTaskByID(id)
		if ci < 0 || !shown[id] {
			continue
		}
		cat := m.project.Categories[ci]
		if m.ui.Filter.TaskVisible(cat.Tasks[ti], cat.ID, cat.ParentID) {
			m.ui.Filter.unhold(id)
			continue
		}
		m.ui.Filter.hold(id)
		held = true
	}
	m.rebuildPositions()
	m.ensureVisible()
	if held && m.ui.Screen.StatusMsg == "" {
		m.ui.Screen.StatusMsg = "Hidden by the filter once you move off it"
	}
}

// releaseFilteredOut waits for normal mode: a visual range may still cover the held rows.
func (m *model) releaseFilteredOut() {
	if !m.ui.Modes.IsNormal() || !m.ui.Filter.holding() {
		return
	}
	if id := m.cursorTaskID(); id != "" && m.ui.Filter.holds(id) {
		return
	}
	m.ui.Filter.releaseHeld()
	m.rebuildPositions()
	m.ensureVisible()
}
