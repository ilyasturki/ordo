package app

import (
	"phasionary/internal/app/components"
	"phasionary/internal/app/selection"
	"phasionary/internal/ui"
)

// maxFocusedDescriptionRows caps the description under the cursor, so one long
// body never takes the whole screen.
const maxFocusedDescriptionRows = 10

// rowLimits decides how many rows each list row may take. The layout reserves
// and the renderer draws from the same value, which is also the layout cache's
// key: any input that changes a row's height belongs here.
type rowLimits struct {
	// cursorCat/cursorTask address the description row under the cursor, -1/-1
	// when the cursor is elsewhere.
	cursorCat          int
	cursorTask         int
	focusedDescription int
	// expandedTask is shown uncapped, title and description.
	expandedTask string
}

func defaultRowLimits() rowLimits {
	return rowLimits{cursorCat: -1, cursorTask: -1, focusedDescription: components.DescriptionPreviewLines}
}

func (m *model) rowLimits() rowLimits {
	l := defaultRowLimits()
	if pos, ok := m.selectedPosition(); ok && pos.Kind == selection.FocusDescription {
		l.cursorCat, l.cursorTask = pos.CategoryIndex, pos.TaskIndex
	}
	l.focusedDescription = max(components.DescriptionPreviewLines, min(maxFocusedDescriptionRows, m.availableHeight()/2))
	if id := m.ui.Screen.ExpandedTaskID; id != "" && id == m.cursorTaskID() {
		l.expandedTask = id
	}
	return l
}

func (l rowLimits) expands(taskID string) bool {
	return taskID != "" && taskID == l.expandedTask
}

func (l rowLimits) titleRows(taskID string) int {
	if l.expands(taskID) {
		return components.Uncapped
	}
	return ui.MaxLineRows
}

func (l rowLimits) descriptionRows(catIdx, taskIdx int, taskID string) int {
	switch {
	case l.expands(taskID):
		return components.Uncapped
	case catIdx == l.cursorCat && taskIdx == l.cursorTask:
		return l.focusedDescription
	}
	return components.DescriptionPreviewLines
}

// cursorTaskID is the task under the cursor, on its title or its description
// row, or "" elsewhere.
func (m *model) cursorTaskID() string {
	pos, ok := m.selectedPosition()
	if !ok || (pos.Kind != selection.FocusTask && pos.Kind != selection.FocusDescription) {
		return ""
	}
	if pos.CategoryIndex < 0 || pos.CategoryIndex >= len(m.project.Categories) {
		return ""
	}
	tasks := m.project.Categories[pos.CategoryIndex].Tasks
	if pos.TaskIndex < 0 || pos.TaskIndex >= len(tasks) {
		return ""
	}
	return tasks[pos.TaskIndex].ID
}

// toggleExpandRow shows the focused task in full, its description included even
// while descriptions are hidden, or folds it back.
func (m *model) toggleExpandRow() {
	id := m.cursorTaskID()
	if id == "" {
		m.ui.Screen.StatusMsg = "Only a task expands"
		return
	}
	if m.ui.Screen.ExpandedTaskID == id {
		id = ""
	}
	m.setExpandedTask(id)
}

// collapseLeftRow folds the expanded task once the cursor is off it.
func (m *model) collapseLeftRow() {
	if id := m.ui.Screen.ExpandedTaskID; id != "" && id != m.cursorTaskID() {
		m.setExpandedTask("")
	}
}

// setExpandedTask rebuilds the rows around the cursor, since the expanded task
// may own a description row that exists only while it is expanded.
func (m *model) setExpandedTask(id string) {
	cursor, ok := m.currentCursor()
	m.ui.Screen.ExpandedTaskID = id
	m.rebuildPositions()
	if ok {
		m.restoreCursor(cursor)
	}
	m.ensureVisible()
}

// stepCursor is j/k: it moves one row, except on an expanded row reaching past
// the screen edge, which scrolls one row at a time until its end shows.
func (m *model) stepCursor(delta int) {
	if !m.scrollExpandedRow(delta) {
		m.moveSelection(delta)
	}
}

func (m *model) scrollExpandedRow(delta int) bool {
	if m.ui.Screen.ExpandedTaskID == "" || m.ui.Screen.ExpandedTaskID != m.cursorTaskID() || m.selected() < 0 {
		return false
	}
	layout := m.buildLayout()
	start, end, ok := layout.rowRange(m.selected())
	if !ok {
		return false
	}
	vp := NewViewport(layout, m.ui.Screen.Height, m.layoutConfig())
	vp.ComputeVisibility(m.ui.Screen.TopRow)
	top := vp.TopRow
	bottom := top + vp.windowAt(top)
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
