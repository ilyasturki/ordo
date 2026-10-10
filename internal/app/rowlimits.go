package app

import (
	"phasionary/internal/app/components"
	"phasionary/internal/app/selection"
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
	return l
}

func (l rowLimits) descriptionRows(catIdx, taskIdx int) int {
	if catIdx == l.cursorCat && taskIdx == l.cursorTask {
		return l.focusedDescription
	}
	return components.DescriptionPreviewLines
}
