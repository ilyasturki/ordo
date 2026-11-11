package app

import (
	"fmt"

	"github.com/ilyasturki/ordo/internal/app/components"
	"github.com/ilyasturki/ordo/internal/app/selection"
	"github.com/ilyasturki/ordo/internal/ui"
)

// rowLimits is the layout cache's key: any input that changes a row's height belongs here.
type rowLimits struct {
	// lineCap bounds every title, name and description; components.Uncapped shows them whole.
	lineCap int
	// expandedTask is shown uncapped, title and description.
	expandedTask string
}

var lineCapSteps = []int{ui.MaxLineRows, 6, 10, components.Uncapped}

func defaultRowLimits() rowLimits {
	return rowLimits{lineCap: lineCapSteps[0]}
}

func (m *model) rowLimits() rowLimits {
	l := defaultRowLimits()
	l.lineCap = lineCapSteps[m.ui.Screen.LineCapStep]
	l.expandedTask = m.ui.Screen.ExpandedTaskID
	return l
}

func (m *model) stepLineCap(delta int) {
	m.ui.Screen.LineCapStep = min(max(m.ui.Screen.LineCapStep+delta, 0), len(lineCapSteps)-1)
	m.ui.Screen.StatusMsg = lineCapStatus(lineCapSteps[m.ui.Screen.LineCapStep])
	m.ensureVisible()
}

func lineCapStatus(limit int) string {
	if limit == components.Uncapped {
		return "Long text shows whole"
	}
	return fmt.Sprintf("Long text shows up to %d rows", limit)
}

func (l rowLimits) expands(taskID string) bool {
	return taskID != "" && taskID == l.expandedTask
}

func (l rowLimits) titleRows(taskID string) int {
	if l.expands(taskID) {
		return components.Uncapped
	}
	return l.lineCap
}

func (l rowLimits) descriptionRows(taskID string) int {
	if l.expands(taskID) {
		return components.Uncapped
	}
	return l.lineCap
}

func (l rowLimits) categoryRows(name string) int {
	return components.MaxRows(l.lineCap, name)
}

// cursorTaskID is "" unless the cursor is on a task's title or description row, or on a note.
func (m *model) cursorTaskID() string {
	pos, ok := m.selectedPosition()
	if !ok || (pos.Kind != selection.FocusTask && pos.Kind != selection.FocusDescription && pos.Kind != selection.FocusSeparator) {
		return ""
	}
	if pos.CategoryIndex < 0 || pos.CategoryIndex >= len(m.project.Categories) {
		return ""
	}
	tasks := m.project.Categories[pos.CategoryIndex].Tasks
	if pos.TaskIndex < 0 || pos.TaskIndex >= len(tasks) || tasks[pos.TaskIndex].IsSeparator() {
		return ""
	}
	return tasks[pos.TaskIndex].ID
}

// toggleExpandRow shows the description too, even while descriptions are hidden.
func (m *model) toggleExpandRow() {
	id := m.cursorTaskID()
	if id == "" {
		m.ui.Screen.StatusMsg = "Only a task or a note expands"
		return
	}
	if m.ui.Screen.ExpandedTaskID == id {
		id = ""
	}
	m.setExpandedTask(id)
}

// Search restores its origin by index and a visual anchor may sit on the row, so both keep it.
func (m *model) collapseLeftRow() {
	if m.ui.Modes.IsSearch() || m.ui.Modes.IsVisual() {
		return
	}
	if id := m.ui.Screen.ExpandedTaskID; id != "" && id != m.cursorTaskID() {
		m.setExpandedTask("")
	}
}

func (m *model) setExpandedTask(id string) {
	m.ui.Screen.ExpandedTaskID = id
	m.rebuildPositions()
	m.ensureVisible()
}

// stepCursor scrolls an expanded row, or one taller than the screen, reaching past its edge before moving off it.
func (m *model) stepCursor(delta int) {
	if !m.scrollExpandedRow(delta) {
		m.moveSelection(delta)
	}
}

func (m *model) scrollExpandedRow(delta int) bool {
	if m.selected() < 0 {
		return false
	}
	expanded := m.ui.Screen.ExpandedTaskID != "" && m.ui.Screen.ExpandedTaskID == m.cursorTaskID()
	layout := m.buildLayout()
	start, end, ok := layout.rowRange(m.selected())
	if !ok {
		return false
	}
	vp := NewViewport(layout, m.ui.Screen.Height, m.layoutConfig())
	vp.ComputeVisibility(m.ui.Screen.TopRow)
	top := vp.TopRow
	bottom := top + vp.windowAt(top)
	if !expanded && end-start <= bottom-top {
		return false
	}
	if start >= bottom || end <= top {
		return false // off screen: moving brings the cursor back into view
	}
	switch {
	case delta > 0 && end > bottom:
		m.ui.Screen.TopRow = top + 1
	case delta < 0 && start < top:
		m.ui.Screen.TopRow = top - 1
	default:
		return false
	}
	return true
}
