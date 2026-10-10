package app

import (
	"errors"
	"fmt"

	"phasionary/internal/app/modes"
	"phasionary/internal/app/selection"
	"phasionary/internal/domain"
)

// nestSelected moves the focused task or category a level in, under the one
// above it, or out, after its parent.
func (m *model) nestSelected(in bool) {
	if !m.ui.Modes.CanPerformAction(modes.ActionMoveItem) {
		return
	}
	pos, ok := m.selectedPosition()
	if !ok {
		return
	}
	switch pos.Kind {
	case selection.FocusCategory:
		m.nestCategories(pos.CategoryIndex, pos.CategoryIndex+1, in)
	case selection.FocusTask, selection.FocusDescription:
		m.nestTasks(pos.CategoryIndex, pos.TaskIndex, pos.TaskIndex+1, in)
	case selection.FocusSeparator:
		m.ui.Screen.StatusMsg = "A separator doesn't nest"
	}
}

// visualNest nests or un-nests the rows of the visual selection together.
func (m *model) visualNest(in bool) {
	lo, hi, catIdx, ok := m.visualRun()
	if !ok {
		return
	}
	if m.ui.Visual.Kind == selection.FocusCategory {
		m.nestCategories(lo, hi, in)
		return
	}
	m.nestTasks(catIdx, lo, hi, in)
}

// nestTasks nests the tasks lo to hi-1 of a category under the task above
// them, or un-nests them after their parent.
func (m *model) nestTasks(catIdx, lo, hi int, in bool) {
	cat := &m.project.Categories[catIdx]
	m.recordHistory()
	var err error
	if in {
		err = cat.IndentTasks(lo, hi)
	} else {
		_, err = cat.OutdentTasks(lo, hi)
	}
	if err != nil {
		m.discardLastHistory()
		m.ui.Screen.StatusMsg = nestRefusal(err, cat.Tasks[lo].ParentID != "", "task", "subtask", "subtasks")
		return
	}
	if in {
		m.revealChildrenOf(cat.Tasks[lo].ParentID)
	}
	m.project.UpdatedAt = domain.NowTimestamp()
	m.rebuildPositions()
	m.ensureVisible()
	m.storeTaskUpdate()
}

// nestCategories nests the categories lo to hi-1 under the top-level category
// above them, or un-nests them after their parent.
func (m *model) nestCategories(lo, hi int, in bool) {
	m.recordHistory()
	var err error
	if in {
		err = m.project.IndentCategories(lo, hi)
	} else {
		_, err = m.project.OutdentCategories(lo, hi)
	}
	if err != nil {
		m.discardLastHistory()
		m.ui.Screen.StatusMsg = m.categoryNestRefusal(err, lo, hi, in)
		return
	}
	if in {
		m.revealChildrenOf(m.project.Categories[lo].ParentID)
	}
	m.rebuildPositions()
	m.ensureVisible()
	m.storeTaskUpdate()
}

// revealChildrenOf unfolds a parent that just took children, which would
// otherwise vanish into its fold.
func (m *model) revealChildrenOf(parentID string) {
	if m.ui.Fold.IsFolded(parentID) {
		m.ui.Fold.Toggle(parentID)
		m.saveFoldState()
	}
}

func nestRefusal(err error, nested bool, kind, child, children string) string {
	switch {
	case errors.Is(err, domain.ErrNothingToNest):
		return "Nothing above to nest under"
	case errors.Is(err, domain.ErrSeparatorNest):
		return "A separator doesn't nest"
	case errors.Is(err, domain.ErrNestDepth) && nested:
		return "Already a " + child + ": nesting goes one level deep"
	case errors.Is(err, domain.ErrNestDepth):
		return fmt.Sprintf("A %s with %s can't nest: nesting goes one level deep", kind, children)
	case errors.Is(err, domain.ErrNotNested):
		return "Already at the top level"
	case errors.Is(err, domain.ErrMixedLevels):
		return "Can't: the selection mixes levels"
	}
	return err.Error()
}

func (m *model) categoryNestRefusal(err error, lo, hi int, in bool) string {
	if !errors.Is(err, domain.ErrDuplicateCategoryName) {
		return nestRefusal(err, m.project.Categories[lo].ParentID != "", "category", "sub-category", "sub-categories")
	}
	cats := m.project.Categories
	if !in {
		for i := lo; i < hi; i++ {
			if domain.CategoryNameConflict(cats, "", cats[i].Name, i) >= 0 {
				return fmt.Sprintf("A top-level category is already named %q", cats[i].Name)
			}
		}
	}
	parent := cats[domain.BlockStart(cats, lo-1)]
	for i := lo; i < hi; i++ {
		if domain.CategoryNameConflict(cats, parent.ID, cats[i].Name, i) >= 0 {
			return fmt.Sprintf("%q already has a %q", parent.Name, cats[i].Name)
		}
	}
	return err.Error()
}
