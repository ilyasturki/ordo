package app

import (
	"fmt"

	"phasionary/internal/domain"
)

const nestIndent = 2

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

type categoryHeader struct {
	name     string
	estimate int
	status   string
	progress string
	indent   int
	folded   bool
	cut      bool
}

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
