package app

import tea "charm.land/bubbletea/v2"

// wheelScrollStep is how many lines one emitted wheel tick scrolls.
const wheelScrollStep = 1

// wheelScrollDivisor slows scrolling by requiring this many raw wheel events per
// emitted tick. Raise it to scroll slower, lower it (min 1) to scroll faster.
const wheelScrollDivisor = 4

func (m *model) scrollUp(amount int) {
	m.ui.Screen.TopRow = max(m.ui.Screen.TopRow-amount, 0)
}

func (m *model) scrollDown(amount int) {
	layout := m.buildLayout()
	viewport := NewViewport(layout, m.ui.Screen.Height, m.layoutConfig())
	viewport.ComputeVisibility(m.ui.Screen.TopRow)
	if !viewport.HasMoreBelow {
		return
	}
	m.ui.Screen.TopRow = min(m.ui.Screen.TopRow+amount, max(layout.TotalHeight-viewport.windowAt(m.ui.Screen.TopRow), 0))
}

func (m model) availableHeight() int {
	config := m.layoutConfig()
	if m.ui.Screen.Height <= config.FooterHeight {
		return 1
	}
	return m.ui.Screen.Height - config.FooterHeight
}

func (m *model) ensureVisible() {
	if m.ui.Selection.IsEmpty() || m.selected() < 0 {
		m.ui.Screen.TopRow = 0
		return
	}
	layout := m.buildLayout()
	viewport := NewViewport(layout, m.ui.Screen.Height, m.layoutConfig())
	viewport.ComputeVisibility(m.ui.Screen.TopRow)

	start, end, ok := layout.rowRange(m.selected())
	if !ok {
		return
	}
	// An open editor follows its caret instead: a buffer taller than the screen
	// is never whole on it, so pinning the whole row would drag the block around
	// under a caret that only moved a row.
	if el, editing := m.openEditRows(); editing {
		start += min(el.cursorRow, end-start-1)
		end = start + 1
	}
	m.ui.Screen.TopRow = viewport.ShowRows(start, end)
}

type cursorSpan struct{ position, start, end int }

// shownCursorSpan is where the cursor's row lies, ok only while ensureVisible would leave the view alone.
func (m model) shownCursorSpan() (cursorSpan, bool) {
	if m.ui.Selection.IsEmpty() || m.selected() < 0 {
		return cursorSpan{}, false
	}
	layout := m.buildLayout()
	start, end, ok := layout.rowRange(m.selected())
	if !ok {
		return cursorSpan{}, false
	}
	viewport := NewViewport(layout, m.ui.Screen.Height, m.layoutConfig())
	viewport.ComputeVisibility(m.ui.Screen.TopRow)
	return cursorSpan{m.selected(), start, end}, viewport.ShowRows(start, end) == viewport.TopRow
}

// followCursorSpan scrolls to a cursor row that a key reshaped without moving
// the cursor: a cut badge, a priority or tag mark can wrap it past the bottom.
func followCursorSpan(next tea.Model, before cursorSpan) tea.Model {
	switch m := next.(type) {
	case model:
		m.followCursorSpan(before)
		return m
	case *model:
		m.followCursorSpan(before)
	}
	return next
}

func (m *model) followCursorSpan(before cursorSpan) {
	if m.selected() != before.position {
		return
	}
	if start, end, ok := m.buildLayout().rowRange(before.position); ok && (start != before.start || end != before.end) {
		m.ensureVisible()
	}
}

func (m *model) centerOnSelected() {
	if m.ui.Selection.IsEmpty() || m.selected() < 0 {
		return
	}
	layout := m.buildLayout()
	viewport := NewViewport(layout, m.ui.Screen.Height, m.layoutConfig())
	m.ui.Screen.TopRow = viewport.CenterOnPosition(m.selected())
}

func (m *model) topOnSelected() {
	if m.ui.Selection.IsEmpty() || m.selected() < 0 {
		return
	}
	if start, _, ok := m.buildLayout().rowRange(m.selected()); ok {
		m.ui.Screen.TopRow = start
	}
}

func (m *model) bottomOnSelected() {
	if m.ui.Selection.IsEmpty() || m.selected() < 0 {
		return
	}
	layout := m.buildLayout()
	viewport := NewViewport(layout, m.ui.Screen.Height, m.layoutConfig())
	m.ui.Screen.TopRow = viewport.BottomOnPosition(m.selected())
}
