package app

import (
	"github.com/ilyasturki/ordo/internal/app/modes"
	"github.com/ilyasturki/ordo/internal/app/selection"
	"github.com/ilyasturki/ordo/internal/domain"
	"github.com/ilyasturki/ordo/internal/operations"
)

func (m *model) moveSelectedRow(dir int) {
	if !m.ui.Modes.CanPerformAction(modes.ActionMoveItem) {
		return
	}
	pos, ok := m.selectedPosition()
	if !ok {
		return
	}
	var reason string
	switch pos.Kind {
	case selection.FocusCategory:
		reason = m.shiftCategories(pos.CategoryIndex, domain.BlockEnd(m.project.Categories, pos.CategoryIndex), dir)
	case selection.FocusTask, selection.FocusSeparator, selection.FocusDescription:
		m.shiftTasks([]string{m.project.Categories[pos.CategoryIndex].Tasks[pos.TaskIndex].ID}, dir)
	default:
		return
	}
	if reason != "" {
		m.ui.Screen.StatusMsg = reason
	}
}

// shiftCategories expects categories[lo:hi] to be whole sibling blocks.
func (m *model) shiftCategories(lo, hi, dir int) string {
	ids := make([]string, 0, hi-lo)
	for _, c := range m.project.Categories[lo:hi] {
		ids = append(ids, c.ID)
	}
	m.recordHistory()
	if err := operations.ShiftCategories(&m.project, ids, dir); err != nil {
		m.discardLastHistory()
		return err.Error()
	}
	if ci, err := m.project.FindCategoryByID(ids[0]); err == nil {
		m.revealChildrenOf(m.project.Categories[ci].ParentID)
	}
	m.rebuildPositions()
	m.ensureVisible()
	m.storeTaskUpdate()
	return ""
}

func (m *model) shiftTasks(ids []string, dir int) {
	m.recordHistory()
	if !m.applied(operations.ShiftTasks(&m.project, ids, dir, tuiView{m})) {
		return
	}
	m.rebuildPositions()
	m.ensureVisible()
	m.storeTaskUpdate()
}
