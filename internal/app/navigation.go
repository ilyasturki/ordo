package app

import (
	"github.com/ilyasturki/phasionary/internal/app/modes"
	"github.com/ilyasturki/phasionary/internal/app/selection"
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
	pageSize := int(float64(m.availableHeight()) * factor)
	if pageSize == 0 {
		if factor > 0 {
			pageSize = 1
		} else {
			pageSize = -1
		}
	}
	m.moveSelection(pageSize)
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
