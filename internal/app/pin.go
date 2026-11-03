package app

import (
	"strconv"

	"github.com/ilyasturki/ordo/internal/app/modes"
	"github.com/ilyasturki/ordo/internal/app/selection"
	"github.com/ilyasturki/ordo/internal/operations"
)

func (m *model) togglePin() {
	if !m.ui.Modes.CanPerformAction(modes.ActionMoveItem) {
		return
	}
	pos, ok := m.selectedPosition()
	if !ok {
		return
	}
	if pos.Kind != selection.FocusCategory {
		m.ui.Screen.StatusMsg = "Put the cursor on a category to pin it"
		return
	}
	cat := m.project.Categories[pos.CategoryIndex]
	m.recordHistory()
	out, err := operations.Apply(&m.project, operations.Op{Action: "category.pin", Cat: cat.ID, Value: strconv.FormatBool(!cat.Pinned)})
	if !m.applied(err) {
		return
	}
	m.rebuildPositions()
	m.selectItem(selection.FocusCategory, cat.ID)
	m.ensureVisible()
	m.storeTaskUpdate()
	m.ui.Screen.StatusMsg = out.Msg
}
