package operations

import (
	"slices"
	"strings"

	"github.com/ilyasturki/ordo/internal/domain"
)

// CreateCategory fails with ErrNameRequired, or the domain's ErrDuplicateCategoryName, ErrCategoryNotFound or ErrNestDepth.
func CreateCategory(p *domain.Project, parentID, name string) (domain.Category, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.Category{}, ErrNameRequired
	}
	if err := domain.ValidateLine(name); err != nil {
		return domain.Category{}, err
	}
	cat, err := p.AddCategoryUnder(parentID, name)
	if err != nil {
		return domain.Category{}, err
	}
	return *cat, nil
}

// RenameCategory renames the category identified by categoryID to name
// (trimmed). Returns ErrNameRequired, domain.ErrCategoryNotFound, or
// domain.ErrDuplicateCategoryName.
func RenameCategory(p *domain.Project, categoryID, name string) (domain.Category, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.Category{}, ErrNameRequired
	}
	if err := domain.ValidateLine(name); err != nil {
		return domain.Category{}, err
	}
	idx, err := p.FindCategoryByID(categoryID)
	if err != nil {
		return domain.Category{}, err
	}
	if err := p.RenameCategory(idx, name); err != nil {
		return domain.Category{}, err
	}
	return p.Categories[idx], nil
}

// MoveCategory moves by delta sibling blocks; running off the end returns ErrNoChange.
func MoveCategory(p *domain.Project, categoryID string, delta int) (domain.Category, error) {
	idx, err := p.FindCategoryByID(categoryID)
	if err != nil {
		return domain.Category{}, err
	}
	before, stamp := slices.Clone(p.Categories), p.UpdatedAt
	at := idx
	for range max(delta, -delta) {
		var ok bool
		if at, ok = p.MoveCategoryBlock(at, sign(delta)); !ok {
			p.Categories, p.UpdatedAt = before, stamp
			return p.Categories[idx], ErrNoChange
		}
	}
	return p.Categories[at], nil
}

// DeleteCategory removes the identified category and, with it, its
// sub-categories and every task they held.
func DeleteCategory(p *domain.Project, categoryID string) error {
	idx, err := p.FindCategoryByID(categoryID)
	if err != nil {
		return err
	}
	p.RemoveCategoryBlock(idx)
	return nil
}

// ShipVersion fails with domain.ErrNotVersion or domain.ErrAlreadyShipped.
func ShipVersion(p *domain.Project, categoryID string) (domain.Category, error) {
	return setShipped(p, categoryID, (*domain.Category).Ship)
}

// UnshipVersion fails with domain.ErrNotShipped.
func UnshipVersion(p *domain.Project, categoryID string) (domain.Category, error) {
	return setShipped(p, categoryID, (*domain.Category).Unship)
}

func setShipped(p *domain.Project, categoryID string, change func(*domain.Category) error) (domain.Category, error) {
	idx, err := p.FindCategoryByID(categoryID)
	if err != nil {
		return domain.Category{}, err
	}
	if err := change(&p.Categories[idx]); err != nil {
		return p.Categories[idx], err
	}
	p.UpdatedAt = p.Categories[idx].UpdatedAt
	return p.Categories[idx], nil
}
