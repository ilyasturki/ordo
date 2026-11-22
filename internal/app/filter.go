package app

import (
	"github.com/ilyasturki/ordo/internal/app/selection"
	"github.com/ilyasturki/ordo/internal/data"
	"github.com/ilyasturki/ordo/internal/domain"
)

func loadFilterState(sm data.StateRepository, project domain.Project) FilterState {
	f := filterStateFrom(sm.GetFilter(project.ID))
	f.pruneCategories(project.Categories)
	return f
}

func (m *model) saveFilter() {
	if m.deps.StateManager == nil || m.project.ID == "" {
		return
	}
	_ = m.deps.StateManager.SetFilter(m.project.ID, m.ui.Filter.saved())
}

func (m *model) toggleFilterOff() {
	if !m.ui.Filter.toggleOff() {
		m.ui.Screen.StatusMsg = "No filter to turn off: f sets one"
		return
	}
	m.saveFilter()
	m.rebuildPositions()
	m.ensureVisible()
	m.ui.Screen.StatusMsg = "Filter on"
	if m.ui.Filter.IsOff() {
		m.ui.Screen.StatusMsg = "Filter off, F turns it back on"
	}
}

func (m *model) onHiddenRow() bool {
	pos, ok := m.selectedPosition()
	return ok && hiddenRowKind(pos.Kind)
}

// hiddenRowKind is a line counting rows the filter hides, which no edit reaches.
func hiddenRowKind(kind selection.FocusKind) bool {
	return kind == selection.FocusHidden || kind == selection.FocusSection
}

// toggleHiddenTasks peeks at what the hidden row or a section's line hides or ends the peek, reporting whether the cursor was on one.
func (m *model) toggleHiddenTasks() bool {
	pos, ok := m.selectedPosition()
	if !ok || !hiddenRowKind(pos.Kind) {
		return false
	}
	cat := m.project.Categories[pos.CategoryIndex]
	if pos.Kind == selection.FocusSection {
		if m.ui.Filter.peeksSection(pos.ID) {
			m.hidePeek()
		} else if sectionHidden(cat, &m.ui.Filter, pos.TaskIndex) == 0 {
			m.ui.Screen.StatusMsg = "Nothing hidden there"
		} else {
			m.startPeek(pos, func() { m.ui.Filter.startSectionPeek(cat.ID, pos.ID) })
		}
		return true
	}
	if m.ui.Filter.peeksRows(cat.ID) {
		m.hidePeek()
		return true
	}
	m.startPeek(pos, func() { m.ui.Filter.startPeek(cat.ID) })
	return true
}

// startPeek lands on the first row the peek brings up.
func (m *model) startPeek(pos selection.Position, start func()) {
	shown := make(map[string]bool)
	for _, p := range m.positions() {
		if p.Kind == selection.FocusTask || p.Kind == selection.FocusSeparator {
			shown[p.ID] = true
		}
	}
	start()
	m.rebuildPositions()
	m.ui.Selection.SelectByPredicate(func(p selection.Position) bool {
		return (p.Kind == selection.FocusTask || p.Kind == selection.FocusSeparator) && p.CategoryIndex == pos.CategoryIndex && !shown[p.ID]
	})
	m.ensureVisible()
	m.ui.Screen.StatusMsg = "Hidden again by esc, or on leaving the category"
}

// hidePeek leaves the cursor where it is, or on the line the peek came from when its own row goes with it.
func (m *model) hidePeek() {
	from := itemRef{selection.FocusHidden, m.ui.Filter.peek}
	if m.ui.Filter.peekSection != "" {
		from = itemRef{selection.FocusSection, m.ui.Filter.peekSection}
	}
	was, _ := m.selectedPosition()
	m.ui.Filter.endPeek()
	m.rebuildPositions()
	if now, ok := m.selectedPosition(); !ok || now.Kind != was.Kind || now.ID != was.ID {
		m.selectItem(from.kind, from.id)
	}
	m.ensureVisible()
	m.ui.Screen.StatusMsg = "Hidden by the filter again"
}

// releasePeek waits for normal mode, like releaseFilteredOut.
func (m *model) releasePeek() {
	if !m.ui.Modes.IsNormal() || m.ui.Filter.peek == "" || m.ui.Filter.peeks(m.cursorCategoryID()) {
		return
	}
	m.ui.Filter.endPeek()
	m.rebuildPositions()
	m.ensureVisible()
}

func (m *model) cursorCategoryID() string {
	pos, ok := m.selectedPosition()
	if !ok || pos.CategoryIndex < 0 || pos.CategoryIndex >= len(m.project.Categories) {
		return ""
	}
	return m.project.Categories[pos.CategoryIndex].ID
}

// holdFilteredOut keeps each edited row the filter stops matching where it is, until the cursor leaves it.
func (m *model) holdFilteredOut(ids []string) {
	if !m.ui.Filter.HasActiveFilter() {
		return
	}
	shown := make(map[string]bool)
	for _, p := range m.positions() {
		if taggable(p.Kind) {
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
		m.ui.Filter.unhold(id)
		if m.filterShows(cat, ti) {
			continue
		}
		m.ui.Filter.hold(id)
		held = held || !m.ui.Filter.peeks(cat.ID)
	}
	m.rebuildPositions()
	m.ensureVisible()
	if held && m.ui.Screen.StatusMsg == "" {
		m.ui.Screen.StatusMsg = "Hidden by the filter once you move off it"
	}
}

// filterShows is the row's verdict with no peek: a line's follows its section.
func (m *model) filterShows(cat domain.Category, ti int) bool {
	if cat.Tasks[ti].IsTask() {
		return m.ui.Filter.TaskVisible(cat.Tasks[ti], cat.ID, cat.ParentID)
	}
	unpeeked := m.ui.Filter
	unpeeked.endPeek()
	return shownRows(cat, &unpeeked, nil)[ti]
}

// releaseFilteredOut waits for normal mode: a visual range may still cover the held rows.
func (m *model) releaseFilteredOut() {
	if !m.ui.Modes.IsNormal() || !m.ui.Filter.holding() {
		return
	}
	if pos, ok := m.selectedPosition(); ok && m.ui.Filter.holds(pos.ID) {
		return
	}
	m.ui.Filter.releaseHeld()
	m.rebuildPositions()
	m.ensureVisible()
}
