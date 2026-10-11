package app

import (
	"slices"

	"github.com/ilyasturki/ordo/internal/app/modes"
	"github.com/ilyasturki/ordo/internal/app/selection"
	"github.com/ilyasturki/ordo/internal/domain"
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
		tasks := m.project.Categories[pos.CategoryIndex].Tasks
		reason = m.shiftTasks(pos.CategoryIndex, pos.TaskIndex, domain.BlockEnd(tasks, pos.TaskIndex), dir)
	default:
		return
	}
	if reason != "" {
		m.ui.Screen.StatusMsg = reason
	}
}

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

// shiftCategories expects categories[lo:hi] to be whole sibling blocks.
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

// shiftTasks expects cat.Tasks[lo:hi] to be whole sibling blocks.
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

// nextShownBlocks spans [a, b) from the run through the first shown sibling block in dir.
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

// shiftTasksAcross expects tasks [lo, hi) of category catIdx to be top-level blocks.
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
