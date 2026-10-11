package app

import (
	"fmt"
	"time"

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
	warning  string
	indent   int
	folded   bool
	cut      bool
	shipped  bool
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
	hasChildren := domain.HasChildren(p.Categories, ci)
	if hasChildren {
		h.status = p.TreeAggregateStatus(ci)
	}
	isVersion := cat.IsVersion()
	if !hasChildren && !isVersion && !cat.Shipped() {
		return h
	}
	counts := p.TreeStatusCounts(ci)
	done := fmt.Sprintf("%d/%d", counts.Completed+counts.Cancelled, counts.Total())
	switch {
	case cat.Shipped():
		h.shipped = true
		h.progress = "shipped " + shipDate(cat.ReleasedAt, timeNow())
		if counts.Total() > 0 {
			h.progress += " · " + done
		}
		if open := counts.Todo + counts.InProgress; open > 0 {
			h.warning = fmt.Sprintf("! %d open", open)
		}
	case isVersion:
		if counts.Total() > 0 {
			h.progress = done
		}
		if counts.InProgress > 0 {
			h.progress += fmt.Sprintf(" · %d in progress", counts.InProgress)
		}
	case hasChildren && counts.Total() > 0:
		h.progress = done
	}
	return h
}

var timeNow = time.Now

// shipDate drops the year within the current one: "14 Sep", "14 Sep 2025".
func shipDate(ts string, now time.Time) string {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return ts
	}
	t = t.In(now.Location())
	if t.Year() == now.Year() {
		return t.Format("2 Jan")
	}
	return t.Format("2 Jan 2006")
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
