package app

import (
	"phasionary/internal/app/modes"
	"phasionary/internal/app/selection"
	"phasionary/internal/domain"
)

// moveSelectedRow is the normal-mode J/K entry: it moves the focused row (a
// category, task, separator — or, via its parent task, a description row) one
// visible step in direction dir.
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
		reason = m.shiftCategories(pos.CategoryIndex, pos.CategoryIndex, dir)
	case selection.FocusTask, selection.FocusSeparator, selection.FocusDescription:
		reason = m.shiftTasks(pos.CategoryIndex, pos.TaskIndex, pos.TaskIndex, dir)
	default:
		return
	}
	if reason != "" {
		m.ui.Screen.StatusMsg = reason
	}
}

// shiftCategories moves the block of categories [lo, hi] one step in direction
// dir by hopping the adjacent category over it. The cursor re-attaches to its
// category by ID. Returns a refusal reason, or "" when the move happened.
func (m *model) shiftCategories(lo, hi, dir int) string {
	var neighbor, dst int
	if dir > 0 {
		if hi >= len(m.project.Categories)-1 {
			return "Already at the bottom"
		}
		neighbor, dst = hi+1, lo
	} else {
		if lo <= 0 {
			return "Already at the top"
		}
		neighbor, dst = lo-1, hi
	}
	var cursorCatID string
	if pos, ok := m.selectedPosition(); ok && pos.CategoryIndex >= 0 && pos.CategoryIndex < len(m.project.Categories) {
		cursorCatID = m.project.Categories[pos.CategoryIndex].ID
	}
	m.recordHistory()
	moved := m.project.Categories[neighbor]
	_ = m.project.RemoveCategory(neighbor)
	m.project.InsertCategory(dst, moved)
	m.project.UpdatedAt = domain.NowTimestamp()
	m.rebuildPositions()
	if cursorCatID != "" {
		m.ui.Selection.SelectByPredicate(func(p selection.Position) bool {
			return p.Kind == selection.FocusCategory && m.project.Categories[p.CategoryIndex].ID == cursorCatID
		})
	}
	m.ensureVisible()
	m.storeTaskUpdate()
	return ""
}

// shiftTasks moves the contiguous block cat.Tasks[lo..hi] one visible step in
// direction dir. Within the category it hops the next *visible* row over the
// block — a run of filter-hidden rows in between is crossed in the same step,
// so the visible order always changes when the move succeeds. With no visible
// neighbor left in the category, the block crosses into the adjacent category
// (top when moving down, bottom when moving up), refused when that category is
// folded or the active filter would hide the moved rows there. The cursor
// re-attaches to its row by task ID. Returns a refusal reason, or "" when the
// move happened.
func (m *model) shiftTasks(catIdx, lo, hi, dir int) string {
	cat := &m.project.Categories[catIdx]
	neighbor := m.nextVisibleTaskRow(catIdx, lo, hi, dir)
	if neighbor < 0 {
		return m.shiftTasksAcross(catIdx, lo, hi, dir)
	}
	// Down: land right after the neighbor, whose index dropped by the block
	// size when the block above it was removed. Up: land right before it.
	insertAt := neighbor
	if dir > 0 {
		insertAt = neighbor - (hi - lo)
	}
	return m.spliceTaskBlock(cat, cat, lo, hi, insertAt)
}

// shiftTasksAcross moves the block into the adjacent category in direction dir.
func (m *model) shiftTasksAcross(catIdx, lo, hi, dir int) string {
	dstIdx := catIdx + dir
	if dstIdx < 0 {
		return "Already at the top"
	}
	if dstIdx >= len(m.project.Categories) {
		return "Already at the bottom"
	}
	src := &m.project.Categories[catIdx]
	dst := &m.project.Categories[dstIdx]
	if m.ui.Fold.IsFolded(dst.ID) {
		return "Can't move into a folded category"
	}
	// A row the filter hides in the destination would take the cursor with it
	// into the invisible — refuse instead of letting the selection vanish.
	for i := lo; i <= hi; i++ {
		if !taskRowVisible(&m.ui.Filter, src.Tasks[i], dst.ID, dst.ParentID) {
			return "Can't move: the filter would hide it there"
		}
	}
	insertAt := 0
	if dir < 0 {
		insertAt = len(dst.Tasks)
	}
	return m.spliceTaskBlock(src, dst, lo, hi, insertAt)
}

// spliceTaskBlock lifts src.Tasks[lo..hi] out and reinserts it in dst at
// insertAt (an index into dst after the block's removal), recording history,
// stamping the project, and persisting.
func (m *model) spliceTaskBlock(src, dst *domain.Category, lo, hi, insertAt int) string {
	cursorID, cursorOnDesc := m.visualCursorRowID()
	m.recordHistory()
	block := make([]domain.Task, hi-lo+1)
	copy(block, src.Tasks[lo:hi+1])
	// Remove from the bottom up so earlier indices stay valid.
	for i := hi; i >= lo; i-- {
		_ = src.RemoveTask(i)
	}
	for i, t := range block {
		dst.InsertTask(insertAt+i, t)
	}
	m.project.UpdatedAt = domain.NowTimestamp()
	m.rebuildPositions()
	// The hopped rows may span one line or two (task + expanded description),
	// so the cursor re-attaches by ID instead of shifting by a fixed count.
	m.selectTaskLevelRow(cursorID, cursorOnDesc)
	m.ensureVisible()
	m.storeTaskUpdate()
	return ""
}

// nextVisibleTaskRow returns the index of the closest row outside the block
// [lo, hi] in direction dir that the active filter shows, or -1 when the block
// has no visible neighbor left on that side of its category.
func (m *model) nextVisibleTaskRow(catIdx, lo, hi, dir int) int {
	cat := m.project.Categories[catIdx]
	i := hi + 1
	if dir < 0 {
		i = lo - 1
	}
	for ; i >= 0 && i < len(cat.Tasks); i += dir {
		if taskRowVisible(&m.ui.Filter, cat.Tasks[i], cat.ID, cat.ParentID) {
			return i
		}
	}
	return -1
}
