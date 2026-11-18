package operations

import (
	"errors"

	"github.com/ilyasturki/ordo/internal/domain"
)

// nextVersion adds the category of the minor version after the one in categoryID's name (its parent's,
// for a sub-category without one), beside the siblings that hold a version on their newest side: above
// the first when they run newest first, below the last when oldest first, and above when one or none
// tells. A name a sibling holds is stepped past. The op's Title overrides the name.
func nextVersion(p *domain.Project, op Op) (Outcome, error) {
	ci, err := p.FindCategoryByID(op.Cat)
	if err != nil {
		return Outcome{}, err
	}
	cats := p.Categories
	if !domain.HasVersion(cats[ci].Name) {
		if parent := domain.ParentIndex(cats, ci); parent >= 0 {
			ci = parent
		}
	}
	name := cats[ci].Name
	parent := cats[ci].ParentID
	for range 1000 {
		next, err := domain.BumpVersion(name, domain.MinorPart, 1)
		if err != nil {
			if errors.Is(err, domain.ErrNoVersion) {
				return Outcome{}, refuse("No version in this name")
			}
			return Outcome{}, err
		}
		name = next
		if domain.CategoryNameConflict(cats, parent, name, -1) < 0 {
			break
		}
	}
	return addCategory(p, Op{New: op.New, Parent: parent, Title: name, Before: newestSide(cats, parent)})
}

// newestSide is the sibling a new version lands above, empty for last.
func newestSide(cats []domain.Category, parent string) string {
	var first, last = -1, -1
	for i, c := range cats {
		if c.ParentID == parent && domain.HasVersion(c.Name) {
			if first < 0 {
				first = i
			}
			last = i
		}
	}
	a, _ := domain.VersionNumbers(cats[first].Name)
	b, _ := domain.VersionNumbers(cats[last].Name)
	if a == b || compareVersions(a, b) > 0 {
		return cats[first].ID
	}
	return nextCategory(cats, last)
}

func compareVersions(a, b [3]int) int {
	for i := range a {
		if a[i] != b[i] {
			return a[i] - b[i]
		}
	}
	return 0
}
