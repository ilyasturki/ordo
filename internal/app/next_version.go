package app

import (
	"github.com/ilyasturki/ordo/internal/app/selection"
	"github.com/ilyasturki/ordo/internal/operations"
)

// startNextVersion adds the next version of the cursor's category and opens its name in the editor.
func (m *model) startNextVersion() {
	pos, ok := m.selectedPosition()
	if !ok || pos.Kind == selection.FocusProject {
		m.ui.Screen.StatusMsg = "Put the cursor on a category to add its next version"
		return
	}
	m.recordHistory()
	out, err := operations.Apply(&m.project, operations.Op{Action: "version.next", Cat: m.project.Categories[pos.CategoryIndex].ID})
	if !m.applied(err) {
		return
	}
	ci, err := m.project.FindCategoryByID(out.Row)
	if err != nil {
		return
	}
	created := m.project.Categories[ci]
	m.revealChildrenOf(created.ParentID)
	m.rebuildPositions()
	m.selectItem(selection.FocusCategory, created.ID)
	m.ui.Modes.ToEdit()
	m.ui.Edit = newEditState(created.Name, true, created.ID, selection.FocusCategory)
	m.ensureVisible()
}
