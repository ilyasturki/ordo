package app

import (
	"fmt"

	"github.com/ilyasturki/phasionary/internal/app/selection"
	"github.com/ilyasturki/phasionary/internal/domain"
)

func (m *model) toggleShippedSelected() {
	pos, ok := m.selectedPosition()
	if !ok || pos.Kind == selection.FocusProject {
		return
	}
	ci := pos.CategoryIndex
	// A shipped sub-category is a nested version: s reopens it rather than shipping its parent.
	if !m.project.Categories[ci].Shipped() {
		ci = domain.BlockStart(m.project.Categories, ci)
	}
	m.toggleShipped([]int{ci})
}

func (m *model) visualToggleShipped() {
	if m.ui.Visual.Kind != selection.FocusCategory {
		m.ui.Screen.StatusMsg = "Select categories to ship them"
		return
	}
	var idxs []int
	for _, pos := range m.visualSelectedPositions() {
		if m.project.Categories[pos.CategoryIndex].ParentID == "" {
			idxs = append(idxs, pos.CategoryIndex)
		}
	}
	m.toggleShipped(idxs)
	m.exitVisualMode()
}

// toggleShipped ships the open versions among idxs, or reopens them all when none is open.
func (m *model) toggleShipped(idxs []int) {
	cats := m.project.Categories
	ship := false
	for _, i := range idxs {
		ship = ship || (cats[i].IsVersion() && !cats[i].Shipped())
	}
	var targets []int
	for _, i := range idxs {
		if (ship && cats[i].IsVersion() && !cats[i].Shipped()) || (!ship && cats[i].Shipped()) {
			targets = append(targets, i)
		}
	}
	if len(targets) == 0 {
		m.ui.Screen.StatusMsg = "Not a version: name it vX.Y.Z to ship it"
		return
	}
	m.recordHistory()
	open := 0
	for _, i := range targets {
		c := &m.project.Categories[i]
		if ship {
			_ = c.Ship()
			counts := m.project.TreeStatusCounts(i)
			open += counts.Todo + counts.InProgress
		} else {
			_ = c.Unship()
		}
		m.project.UpdatedAt = c.UpdatedAt
		m.ui.Fold.ResetShipped(c.ID)
	}
	m.saveFoldState()
	m.rebuildPositions()
	m.ensureVisible()
	m.storeTaskUpdate()

	what := m.project.Categories[targets[0]].Name
	if len(targets) > 1 {
		what = fmt.Sprintf("%d versions", len(targets))
	}
	switch {
	case !ship:
		m.ui.Screen.StatusMsg = "Reopened " + what
	case open > 0:
		m.ui.Screen.StatusMsg = fmt.Sprintf("Shipped %s with %d open %s", what, open, plural(open, "task", "tasks"))
	default:
		m.ui.Screen.StatusMsg = "Shipped " + what
	}
}
