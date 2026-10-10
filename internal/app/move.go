package app

import (
	"slices"

	"phasionary/internal/app/modes"
	"phasionary/internal/app/selection"
	"phasionary/internal/domain"
)

// moveSelectedRow is the normal-mode J/K entry: it moves the focused row (a
// category, task, separator — or, via its parent task, a description row) one
// visible step in direction dir. A parent moves with its children, and a
// child moves among its siblings.
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
		tasks := m.project.Categories[pos.CategoryIndex].Tasks
		reason = m.shiftTasks(pos.CategoryIndex, pos.TaskIndex, domain.BlockEnd(tasks, pos.TaskIndex), dir)
	default:
		return
	}
	if reason != "" {
		m.ui.Screen.StatusMsg = reason
	}
}

// edgeReason refuses a move off the end of the siblings in direction dir.
func edgeReason(dir int, nested bool) string {
	switch {
	case dir > 0 && nested:
		return "Already at the bottom of its parent"
	case dir > 0:
		return "Already at the bottom"
	case nested:
		return "Already at the top of its parent"
	}
	return "Already at the top"
}

// shiftCategories moves categories[lo:hi], whole sibling blocks, past the
// sibling block next to them in direction dir. A sub-category stays among its
// siblings. The cursor stays on its row. Returns a refusal reason, or "" when
// the move happened.
func (m *model) shiftCategories(lo, hi, dir int) string {
	cats := m.project.Categories
	nested := cats[lo].ParentID != ""
	head := lo
	if dir > 0 {
		head = domain.LastBlockStart(cats, lo, hi)
	}
	nlo, nhi, ok := domain.SiblingBlock(cats, head, dir)
	if !ok {
		return edgeReason(dir, nested)
	}
	m.recordHistory()
	if dir > 0 {
		domain.SwapRanges(cats, lo, hi, nhi)
	} else {
		domain.SwapRanges(cats, nlo, lo, hi)
	}
	m.project.UpdatedAt = domain.NowTimestamp()
	m.rebuildPositions()
	m.ensureVisible()
	m.storeTaskUpdate()
	return ""
}

// shiftTasks moves cat.Tasks[lo:hi], whole sibling blocks, one visible step in
// direction dir: past the next sibling block the list shows, crossing the
// blocks the filter hides on the way, so the visible order always changes.
// Top-level blocks with no such neighbor left cross into the adjacent category
// (top when moving down, bottom when moving up), refused when that category is
// folded away or the filter would hide them there; subtasks stay with their
// parent. Returns a refusal reason, or "" when the move happened.
func (m *model) shiftTasks(catIdx, lo, hi, dir int) string {
	cat := &m.project.Categories[catIdx]
	nested := cat.Tasks[lo].ParentID != ""
	a, b, ok := m.nextShownBlocks(*cat, lo, hi, dir)
	if !ok {
		if nested {
			return edgeReason(dir, true)
		}
		return m.shiftTasksAcross(catIdx, lo, hi, dir)
	}
	m.recordHistory()
	if dir > 0 {
		domain.SwapRanges(cat.Tasks, lo, hi, b)
	} else {
		domain.SwapRanges(cat.Tasks, a, lo, hi)
	}
	cat.UpdatedAt = domain.NowTimestamp()
	m.project.UpdatedAt = domain.NowTimestamp()
	m.rebuildPositions()
	m.ensureVisible()
	m.storeTaskUpdate()
	return ""
}

// nextShownBlocks returns the sibling blocks beside cat.Tasks[lo:hi] in
// direction dir, up to and including the first one the list shows, as [a, b).
func (m *model) nextShownBlocks(cat domain.Category, lo, hi, dir int) (int, int, bool) {
	shown := shownRows(cat, &m.ui.Filter, &m.ui.Fold)
	if dir > 0 {
		head := domain.LastBlockStart(cat.Tasks, lo, hi)
		for {
			nlo, nhi, ok := domain.SiblingBlock(cat.Tasks, head, 1)
			if !ok {
				return 0, 0, false
			}
			if slices.Contains(shown[nlo:nhi], true) {
				return hi, nhi, true
			}
			head = nlo
		}
	}
	head := lo
	for {
		nlo, _, ok := domain.SiblingBlock(cat.Tasks, head, -1)
		if !ok {
			return 0, 0, false
		}
		if slices.Contains(shown[nlo:head], true) {
			return nlo, lo, true
		}
		head = nlo
	}
}

// shiftTasksAcross moves top-level blocks into the adjacent category in
// direction dir.
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
	if m.ui.Fold.IsFolded(dst.ID) || (dst.ParentID != "" && m.ui.Fold.IsFolded(dst.ParentID)) {
		return "Can't move into a folded category"
	}
	// A block the filter hides in the destination would take the cursor with
	// it into the invisible — refuse instead of letting the selection vanish.
	if !m.ui.Filter.CategoryVisible(dst.ID, dst.ParentID) {
		return "Can't move: the filter would hide it there"
	}
	insertAt := 0
	if dir < 0 {
		insertAt = len(dst.Tasks)
	}
	m.recordHistory()
	block := slices.Clone(src.Tasks[lo:hi])
	src.Tasks = slices.Delete(src.Tasks, lo, hi)
	dst.Tasks = slices.Insert(dst.Tasks, insertAt, block...)
	now := domain.NowTimestamp()
	src.UpdatedAt, dst.UpdatedAt, m.project.UpdatedAt = now, now, now
	m.rebuildPositions()
	m.ensureVisible()
	m.storeTaskUpdate()
	return ""
}
