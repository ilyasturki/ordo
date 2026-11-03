package operations

import (
	"slices"
	"strings"

	"github.com/ilyasturki/ordo/internal/domain"
)

// CreateCategory fails with ErrNameRequired, or the domain's ErrDuplicateCategoryName, ErrCategoryNotFound or ErrNestDepth.
func CreateCategory(p *domain.Project, parentID, name string) (domain.Category, error) {
	return createCategory(p, "", parentID, name)
}

func createCategory(p *domain.Project, id, parentID, name string) (domain.Category, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.Category{}, ErrNameRequired
	}
	if err := domain.ValidateLine(name); err != nil {
		return domain.Category{}, err
	}
	cat, err := p.AddCategoryWithID(id, parentID, name)
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

// Clash is a category.add or category.rename refused for a name a sibling holds, with the names
// that would free it: This for the edited category, Other for the sibling, Taken.
type Clash struct {
	Taken string `json:"taken"`
	This  string `json:"this"`
	Other string `json:"other"`
}

// CategoryClash is op's clash on p, nil when op names a free name or is no add or rename.
func CategoryClash(p *domain.Project, op Op) *Clash {
	name := strings.TrimSpace(op.Title)
	parent, except := "", -1
	switch op.Action {
	case "category.add":
		if pi, err := p.FindCategoryByID(op.Parent); err == nil && p.Categories[pi].ParentID == "" {
			parent = op.Parent
		}
	case "category.rename":
		ci, err := p.FindCategoryByID(op.Cat)
		if err != nil {
			return nil
		}
		parent, except = p.Categories[ci].ParentID, ci
	default:
		return nil
	}
	ti := domain.CategoryNameConflict(p.Categories, parent, name, except)
	if ti < 0 {
		return nil
	}
	others := p.Categories
	if except >= 0 {
		others = slices.Delete(slices.Clone(others), except, except+1)
	}
	return &Clash{
		Taken: p.Categories[ti].ID,
		This:  domain.UniqueCategoryName(others, parent, name),
		Other: domain.UniqueCategoryName(p.Categories, parent, p.Categories[ti].Name),
	}
}

// MoveCategory moves by delta steps of ShiftCategories; a step it refuses returns ErrNoChange.
func MoveCategory(p *domain.Project, categoryID string, delta int) (domain.Category, error) {
	idx, err := p.FindCategoryByID(categoryID)
	if err != nil {
		return domain.Category{}, err
	}
	before, stamp := Clone(*p).Categories, p.UpdatedAt
	for range max(delta, -delta) {
		if err := ShiftCategories(p, []string{categoryID}, sign(delta)); err != nil {
			p.Categories, p.UpdatedAt = before, stamp
			return p.Categories[idx], ErrNoChange
		}
	}
	at, _ := p.FindCategoryByID(categoryID)
	return p.Categories[at], nil
}

// ShiftCategories moves the run of sibling blocks ids cover one sibling block in dir. A run of
// sub-categories at its parent's edge passes to the neighbouring parent: last under the one
// above, first under the one below. It never takes an unpinned category into the pinned run.
func ShiftCategories(p *domain.Project, ids []string, dir int) error {
	cats := p.Categories
	idxs := make([]int, 0, len(ids))
	for _, id := range ids {
		i, err := p.FindCategoryByID(id)
		if err != nil {
			return err
		}
		idxs = append(idxs, i)
	}
	lo, hi, err := domain.CoverBlocks(cats, idxs)
	if err != nil {
		return NestRefusal(err, false, "category", "sub-category", "sub-categories")
	}
	parent := cats[lo].ParentID
	var pinned, unpinned bool
	for _, c := range cats[lo:hi] {
		if c.ParentID == parent {
			pinned, unpinned = pinned || c.Pinned, unpinned || !c.Pinned
		}
	}
	head := lo
	if dir > 0 {
		head = domain.LastBlockStart(cats, lo, hi)
	}
	if nlo, nhi, ok := domain.SiblingBlock(cats, head, dir); ok {
		switch {
		case dir > 0 && pinned && !cats[nlo].Pinned:
			return refuse("Already at the bottom of the pinned ones")
		case dir < 0 && unpinned && cats[nlo].Pinned:
			return refuse("Already at the top of the unpinned ones")
		case dir > 0:
			domain.SwapRanges(cats, lo, hi, nhi)
		default:
			domain.SwapRanges(cats, nlo, lo, hi)
		}
		p.UpdatedAt = domain.NowTimestamp()
		return nil
	}
	if parent == "" {
		return EdgeRefusal(dir, false)
	}
	return shiftAcrossParents(p, lo, hi, dir)
}

// shiftAcrossParents expects cats[lo:hi] to be the sub-categories at their parent's edge in dir.
func shiftAcrossParents(p *domain.Project, lo, hi, dir int) error {
	cats := p.Categories
	pi := domain.ParentIndex(cats, lo)
	nlo, _, ok := domain.SiblingBlock(cats, pi, dir)
	if !ok {
		return EdgeRefusal(dir, true)
	}
	next := cats[nlo]
	for i := lo; i < hi; i++ {
		if domain.CategoryNameConflict(cats, next.ID, cats[i].Name, i) >= 0 {
			return refuse("%q already has a %q", next.Name, cats[i].Name)
		}
	}
	now := domain.NowTimestamp()
	for i := lo; i < hi; i++ {
		cats[i].ParentID = next.ID
		cats[i].UpdatedAt = now
	}
	if dir > 0 {
		domain.SwapRanges(cats, lo, hi, hi+1)
	} else {
		domain.SwapRanges(cats, pi, lo, hi)
	}
	domain.SettlePinned(cats)
	p.UpdatedAt = now
	return nil
}

// PinCategory pins or unpins the category: pinned, it ends its siblings' pinned run; unpinned, it
// starts the rest.
func PinCategory(p *domain.Project, categoryID string, pinned bool) (domain.Category, error) {
	ci, err := p.FindCategoryByID(categoryID)
	if err != nil {
		return domain.Category{}, err
	}
	if p.Categories[ci].Pinned == pinned {
		return p.Categories[ci], ErrNoChange
	}
	now := domain.NowTimestamp()
	p.Categories[ci].Pinned = pinned
	p.Categories[ci].UpdatedAt = now
	p.UpdatedAt = now
	domain.SettlePinned(p.Categories)
	ci, _ = p.FindCategoryByID(categoryID)
	return p.Categories[ci], nil
}

// PlaceCategory moves the category with its sub-categories above its sibling before; a before that
// is empty, gone or at another level lands it last among its siblings. It stays in its pinned or
// unpinned run, at the run's edge nearest before.
func PlaceCategory(p *domain.Project, categoryID, before string) (domain.Category, error) {
	ci, err := p.FindCategoryByID(categoryID)
	if err != nil {
		return domain.Category{}, err
	}
	if before == categoryID {
		return p.Categories[ci], ErrNoChange
	}
	end := domain.BlockEnd(p.Categories, ci)
	block := slices.Clone(p.Categories[ci:end])
	cats := slices.Delete(slices.Clone(p.Categories), ci, end)
	parent := block[0].ParentID
	at := len(cats)
	if pi := slices.IndexFunc(cats, func(c domain.Category) bool { return c.ID == parent }); parent != "" && pi >= 0 {
		at = domain.BlockEnd(cats, pi)
	}
	if bi := slices.IndexFunc(cats, func(c domain.Category) bool { return c.ID == before && c.ParentID == parent }); before != "" && bi >= 0 {
		at = bi
	}
	p.Categories = slices.Insert(cats, at, block...)
	domain.SettlePinned(p.Categories)
	p.UpdatedAt = domain.NowTimestamp()
	return block[0], nil
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
