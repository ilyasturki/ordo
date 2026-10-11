package app

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/ilyasturki/ordo/internal/app/selection"
	"github.com/ilyasturki/ordo/internal/domain"
)

// bumpSelected acts on the row's category, or on its parent when the row sits in a sub-category without a version.
func (m *model) bumpSelected(part domain.VersionPart, delta int) {
	pos, ok := m.selectedPosition()
	if !ok || pos.Kind == selection.FocusProject {
		m.ui.Screen.StatusMsg = "Put the cursor on a category to bump its version"
		return
	}
	ci := pos.CategoryIndex
	cats := m.project.Categories
	if pos.Kind != selection.FocusCategory && !domain.HasVersion(cats[ci].Name) {
		if parent := domain.ParentIndex(cats, ci); parent >= 0 {
			ci = parent
		}
	}
	m.bumpCategories([]int{ci}, part, delta)
}

func (m *model) visualBump(delta int) {
	if m.ui.Visual.Kind != selection.FocusCategory {
		m.ui.Screen.StatusMsg = "Select categories to bump them"
		return
	}
	var idxs []int
	for _, pos := range m.visualSelectedPositions() {
		idxs = append(idxs, pos.CategoryIndex)
	}
	m.bumpCategories(idxs, domain.PatchPart, delta)
}

// bumpCategories checks every new name against the others' final names before renaming any, so a range never clashes with itself halfway.
func (m *model) bumpCategories(idxs []int, part domain.VersionPart, delta int) {
	cats := m.project.Categories
	final := slices.Clone(cats)
	var bumped []int
	none, atZero := 0, 0
	for _, i := range idxs {
		name, err := domain.BumpVersion(cats[i].Name, part, delta)
		switch {
		case errors.Is(err, domain.ErrVersionZero):
			atZero++
		case err != nil:
			none++
		default:
			final[i].Name = name
			bumped = append(bumped, i)
		}
	}
	if len(bumped) == 0 {
		switch {
		case atZero > 0 && len(idxs) == 1:
			m.ui.Screen.StatusMsg = cats[idxs[0]].Name + " can't go below 0"
		case atZero > 0:
			m.ui.Screen.StatusMsg = "No version can go below 0"
		case len(idxs) == 1:
			m.ui.Screen.StatusMsg = "No version in this name"
		default:
			m.ui.Screen.StatusMsg = "No version in these names"
		}
		return
	}
	for _, i := range bumped {
		if domain.CategoryNameConflict(final, final[i].ParentID, final[i].Name, i) >= 0 {
			m.ui.Screen.StatusMsg = final[i].Name + " already exists"
			return
		}
	}

	m.recordHistory()
	now := domain.NowTimestamp()
	for _, i := range bumped {
		m.project.Categories[i].Name = final[i].Name
		m.project.Categories[i].UpdatedAt = now
	}
	m.project.UpdatedAt = now
	m.rebuildPositions()
	m.ensureVisible()
	m.storeTaskUpdate()

	if len(idxs) == 1 {
		m.ui.Screen.StatusMsg = "Bumped to " + final[bumped[0]].Name
		return
	}
	status := fmt.Sprintf("Bumped %d %s", len(bumped), plural(len(bumped), "version", "versions"))
	var skipped []string
	if none > 0 {
		skipped = append(skipped, fmt.Sprintf("%d had none", none))
	}
	if atZero > 0 {
		skipped = append(skipped, fmt.Sprintf("%d at 0", atZero))
	}
	if len(skipped) > 0 {
		status += ", " + strings.Join(skipped, ", ")
	}
	m.ui.Screen.StatusMsg = status
}
