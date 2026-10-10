package operations

import (
	"slices"
	"strings"

	"phasionary/internal/domain"
)

// CreateCategory adds a category named name (trimmed) to p. Returns
// ErrNameRequired for an empty name or domain.ErrDuplicateCategoryName when the
// name collides case-insensitively with an existing category.
func CreateCategory(p *domain.Project, name string) (domain.Category, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.Category{}, ErrNameRequired
	}
	if err := domain.ValidateLine(name); err != nil {
		return domain.Category{}, err
	}
	cat, err := p.AddCategoryNamed(name)
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

// MoveCategory shifts the identified category, with its sub-categories, by
// delta sibling blocks. A move that would fall off either end of its siblings
// changes nothing and returns ErrNoChange.
func MoveCategory(p *domain.Project, categoryID string, delta int) (domain.Category, error) {
	idx, err := p.FindCategoryByID(categoryID)
	if err != nil {
		return domain.Category{}, err
	}
	before := slices.Clone(p.Categories)
	at := idx
	for range max(delta, -delta) {
		var ok bool
		if at, ok = p.MoveCategoryBlock(at, sign(delta)); !ok {
			p.Categories = before
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
