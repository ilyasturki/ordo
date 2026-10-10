package app

import (
	"fmt"

	"phasionary/internal/domain"
)

// nestIndent is how far each level of nesting shifts a row right.
const nestIndent = 2

// taskIndent is how far the row of cat.Tasks[ti] shifts right: a level for a
// sub-category's rows, another for a subtask.
func taskIndent(cat domain.Category, ti int) int {
	indent := 0
	if cat.ParentID != "" {
		indent += nestIndent
	}
	if cat.Tasks[ti].ParentID != "" {
		indent += nestIndent
	}
	return indent
}

// taskProgress is a parent task's badge, its finished subtasks out of all of
// them, marked when a fold hides them; "" for any other row.
func taskProgress(cat domain.Category, ti int, fold *FoldState) string {
	done, total := cat.ChildProgress(ti)
	if total == 0 {
		return ""
	}
	badge := fmt.Sprintf("%d/%d", done, total)
	if fold != nil && fold.IsFolded(cat.Tasks[ti].ID) {
		badge = foldedMark + " " + badge
	}
	return badge
}

const foldedMark = "▸"

// categoryHeader is what a category's header row shows.
type categoryHeader struct {
	name     string
	estimate int
	status   string
	// progress counts a parent's finished tasks, its sub-categories' included.
	progress string
	indent   int
	folded   bool
	cut      bool
}

// headerOf describes the header of the category at ci. A parent's status and
// progress cover its sub-categories too.
func headerOf(p *domain.Project, ci int, fold *FoldState, cut bool) categoryHeader {
	cat := p.Categories[ci]
	h := categoryHeader{
		name:     cat.Name,
		estimate: cat.EstimateMinutes,
		status:   cat.AggregateStatus(),
		folded:   fold != nil && fold.IsFolded(cat.ID),
		cut:      cut,
	}
	if cat.ParentID != "" {
		h.indent = nestIndent
		return h
	}
	if domain.HasChildren(p.Categories, ci) {
		h.status = p.TreeAggregateStatus(ci)
		if counts := p.TreeStatusCounts(ci); counts.Total() > 0 {
			h.progress = fmt.Sprintf("%d/%d", counts.Completed+counts.Cancelled, counts.Total())
		}
	}
	return h
}

// rowIndent is how far the row at the position shifts right.
func (m model) rowIndent(catIdx, taskIdx int) int {
	if catIdx < 0 || catIdx >= len(m.project.Categories) {
		return 0
	}
	cat := m.project.Categories[catIdx]
	if taskIdx < 0 || taskIdx >= len(cat.Tasks) {
		if cat.ParentID != "" {
			return nestIndent
		}
		return 0
	}
	return taskIndent(cat, taskIdx)
}
