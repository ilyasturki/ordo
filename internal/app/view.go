package app

import (
	"errors"

	"github.com/ilyasturki/ordo/internal/domain"
	"github.com/ilyasturki/ordo/internal/operations"
)

type tuiView struct{ m *model }

func (v tuiView) Shown(cat domain.Category) []bool {
	return shownRows(cat, &v.m.ui.Filter, &v.m.ui.Fold)
}

func (v tuiView) Filtering() bool       { return v.m.ui.Filter.HasActiveFilter() }
func (v tuiView) Folded(id string) bool { return v.m.ui.Fold.IsFolded(id) }

func (v tuiView) CategoryVisible(cat domain.Category) bool {
	return v.m.ui.Filter.CategoryVisible(cat.ID, cat.ParentID)
}

// applied keeps the history entry recorded for an operation that changed the project, and
// otherwise drops it, putting a refusal or failure on the status line.
func (m *model) applied(err error) bool {
	if err == nil {
		return true
	}
	m.discardLastHistory()
	if !errors.Is(err, operations.ErrNoChange) {
		m.ui.Screen.StatusMsg = err.Error()
	}
	return false
}
