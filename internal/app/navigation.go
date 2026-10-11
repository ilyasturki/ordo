package app

import (
	"github.com/ilyasturki/ordo/internal/app/modes"
	"github.com/ilyasturki/ordo/internal/app/selection"
)

func (m *model) moveSelection(delta int) {
	if !m.ui.Modes.CanPerformAction(modes.ActionNavigate) || m.ui.Selection.IsEmpty() {
		return
	}
	m.ui.Selection.MoveBy(delta)
	m.ensureVisible()
}

func (m *model) moveSelectionByPage(factor float64) {
	if !m.ui.Modes.CanPerformAction(modes.ActionNavigate) || m.ui.Selection.IsEmpty() {
		return
	}
	step := 1
	if factor < 0 {
		step = -1
	}
	layout := m.buildLayout()
	start, _, ok := layout.rowRange(m.selected())
	if !ok {
		m.moveSelection(step)
		return
	}
	// The last row the cursor may start on going down, the first going up.
	target := start + int(float64(m.availableHeight())*factor)
	next, row := m.selected(), 0
	for _, item := range layout.Items {
		pos := item.PositionIndex
		if pos >= 0 && (step > 0 && pos > m.selected() && row <= target || step < 0 && pos < next && row >= target) {
			next = pos
		}
		row += item.Height
	}
	if next == m.selected() {
		m.moveSelection(step)
		return
	}
	m.ui.Selection.MoveTo(next)
	m.ensureVisible()
}

func (m *model) jumpToFirst() {
	if !m.ui.Modes.CanPerformAction(modes.ActionNavigate) || m.ui.Selection.IsEmpty() {
		return
	}
	m.ui.Selection.JumpToFirst()
	m.ensureVisible()
}

func (m *model) jumpToLast() {
	if !m.ui.Modes.CanPerformAction(modes.ActionNavigate) || m.ui.Selection.IsEmpty() {
		return
	}
	m.ui.Selection.JumpToLast()
	m.ensureVisible()
}

func (m *model) jumpToNextCategory() {
	if !m.ui.Modes.CanPerformAction(modes.ActionNavigate) || m.ui.Selection.IsEmpty() {
		return
	}
	positions := m.ui.Selection.Positions()
	selected := m.ui.Selection.Selected()
	for i := selected + 1; i < len(positions); i++ {
		if positions[i].Kind == selection.FocusCategory {
			m.ui.Selection.MoveTo(i)
			m.ensureVisible()
			return
		}
	}
}

func (m *model) jumpToPrevCategory() {
	if !m.ui.Modes.CanPerformAction(modes.ActionNavigate) || m.ui.Selection.IsEmpty() {
		return
	}
	positions := m.ui.Selection.Positions()
	selected := m.ui.Selection.Selected()
	for i := selected - 1; i >= 0; i-- {
		if positions[i].Kind == selection.FocusCategory {
			m.ui.Selection.MoveTo(i)
			m.ensureVisible()
			return
		}
	}
}
