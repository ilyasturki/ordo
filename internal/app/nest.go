package app

import (
	"errors"
	"fmt"

	"github.com/ilyasturki/ordo/internal/app/modes"
	"github.com/ilyasturki/ordo/internal/app/selection"
	"github.com/ilyasturki/ordo/internal/domain"
	"github.com/ilyasturki/ordo/internal/operations"
)

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
		m.nestTasks([]string{m.project.Categories[pos.CategoryIndex].Tasks[pos.TaskIndex].ID}, in)
	case selection.FocusSeparator:
		m.ui.Screen.StatusMsg = "A separator doesn't nest"
	}
}

func (m *model) visualNest(in bool) {
	if m.ui.Visual.Kind != selection.FocusCategory {
		if ids := m.visualTaskIDs(); len(ids) > 0 {
			m.nestTasks(ids, in)
		}
		return
	}
	if lo, hi, ok := m.visualCategoryRun(); ok {
		m.nestCategories(lo, hi, in)
	}
}

func (m *model) nestTasks(ids []string, in bool) {
	m.recordHistory()
	parentID, err := operations.NestTasks(&m.project, ids, in, tuiView{m})
	if !m.applied(err) {
		return
	}
	if in {
		m.revealChildrenOf(parentID)
	}
	m.rebuildPositions()
	m.ensureVisible()
	m.storeTaskUpdate()
}

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

func (m *model) revealChildrenOf(parentID string) {
	if m.ui.Fold.IsFolded(parentID) {
		m.ui.Fold.Toggle(parentID)
		m.saveFoldState()
	}
}

func (m *model) categoryNestRefusal(err error, lo, hi int, in bool) string {
	if !errors.Is(err, domain.ErrDuplicateCategoryName) {
		return operations.NestRefusal(err, m.project.Categories[lo].ParentID != "", "category", "sub-category", "sub-categories").Error()
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
