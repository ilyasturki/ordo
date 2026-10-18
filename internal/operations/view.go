package operations

import "github.com/ilyasturki/ordo/internal/domain"

// View is what a front end displays, so a structural move skips past rows the user can't see.
type View interface {
	Shown(cat domain.Category) []bool
	Filtering() bool
	Folded(categoryID string) bool
	CategoryVisible(cat domain.Category) bool
}

// AllShown is the View of a front end with no filter and nothing folded.
var AllShown View = allShown{}

type allShown struct{}

func (allShown) Shown(cat domain.Category) []bool {
	shown := make([]bool, len(cat.Tasks))
	for i := range shown {
		shown[i] = true
	}
	return shown
}

func (allShown) Filtering() bool                      { return false }
func (allShown) Folded(string) bool                   { return false }
func (allShown) CategoryVisible(domain.Category) bool { return true }
