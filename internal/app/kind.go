package app

import (
	"github.com/ilyasturki/ordo/internal/app/modes"
	"github.com/ilyasturki/ordo/internal/app/selection"
	"github.com/ilyasturki/ordo/internal/domain"
	"github.com/ilyasturki/ordo/internal/operations"
)

func (m *model) convertSelected() {
	if !m.ui.Modes.CanPerformAction(modes.ActionEditItem) {
		return
	}
	pos, ok := m.selectedPosition()
	if !ok || !visualTaskLevel(pos.Kind) {
		m.ui.Screen.StatusMsg = "c turns a task into a note, or a note into a task"
		return
	}
	m.convertRows([]string{m.project.Categories[pos.CategoryIndex].Tasks[pos.TaskIndex].ID})
}

func (m *model) visualConvert() {
	if m.ui.Visual.Kind == selection.FocusCategory {
		m.ui.Screen.StatusMsg = "c turns tasks into notes, not categories"
		return
	}
	m.convertRows(positionIDs(m.project, visualTaskPositions(m.visualSelectedPositions())))
}

// convertRows turns ids into notes, or into tasks when the first row that isn't a separator is a note.
func (m *model) convertRows(ids []string) {
	kind, found := "", false
	for _, id := range ids {
		if t := m.taskByID(id); t != nil && !t.IsSeparator() {
			kind, found = domain.KindNote, true
			if t.IsNote() {
				kind = ""
			}
			break
		}
	}
	if !found {
		m.ui.Screen.StatusMsg = "A separator stays a separator"
		return
	}
	cursor, _ := m.selectedPosition()
	m.recordHistory()
	out, err := operations.Apply(&m.project, operations.Op{Action: "task.kind", IDs: ids, Value: kind})
	if !m.applied(err) {
		return
	}
	m.storeTaskUpdate()
	m.ui.Screen.StatusMsg = out.Msg
	m.holdFilteredOut(ids)
	m.rebuildPositions()
	if cursor.Kind == selection.FocusTask || cursor.Kind == selection.FocusSeparator {
		m.ui.Selection.SelectByPredicate(func(p selection.Position) bool {
			return (p.Kind == selection.FocusTask || p.Kind == selection.FocusSeparator) && p.ID == cursor.ID
		})
	}
	m.ensureVisible()
}
