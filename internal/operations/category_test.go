package operations_test

import (
	"errors"
	"testing"

	"phasionary/internal/domain"
	"phasionary/internal/operations"
)

func projectWithCategories(names ...string) *domain.Project {
	p := &domain.Project{ID: "p1", Name: "Test"}
	for _, n := range names {
		cat, _ := domain.NewCategory(n)
		p.Categories = append(p.Categories, cat)
	}
	return p
}

func categoryOrder(p *domain.Project) []string {
	names := make([]string, len(p.Categories))
	for i, c := range p.Categories {
		names[i] = c.Name
	}
	return names
}

func TestCreateCategory(t *testing.T) {
	p := projectWithCategories()

	got, err := operations.CreateCategory(p, "", "Feature")
	if err != nil {
		t.Fatalf("CreateCategory: %v", err)
	}
	if got.Name != "Feature" || got.ID == "" {
		t.Fatalf("returned category = %+v", got)
	}
	if len(p.Categories) != 1 {
		t.Fatalf("project holds %d categories, want 1", len(p.Categories))
	}
}

func TestCreateCategory_DuplicateCaseInsensitive(t *testing.T) {
	p := projectWithCategories("Feature")

	_, err := operations.CreateCategory(p, "", "feature")
	if !errors.Is(err, domain.ErrDuplicateCategoryName) {
		t.Fatalf("err = %v, want ErrDuplicateCategoryName", err)
	}
}

func TestCreateCategory_EmptyRejected(t *testing.T) {
	p := projectWithCategories()

	_, err := operations.CreateCategory(p, "", "   ")
	if !errors.Is(err, operations.ErrNameRequired) {
		t.Fatalf("err = %v, want ErrNameRequired", err)
	}
}

func TestRenameCategory(t *testing.T) {
	p := projectWithCategories("Feature")
	id := p.Categories[0].ID

	got, err := operations.RenameCategory(p, id, "Fix")
	if err != nil {
		t.Fatalf("RenameCategory: %v", err)
	}
	if got.Name != "Fix" {
		t.Fatalf("returned name = %q, want Fix", got.Name)
	}
	if p.Categories[0].Name != "Fix" {
		t.Fatalf("stored name = %q, want Fix", p.Categories[0].Name)
	}
}

func TestRenameCategory_DuplicateRejected(t *testing.T) {
	p := projectWithCategories("Feature", "Fix")
	id := p.Categories[1].ID

	_, err := operations.RenameCategory(p, id, "feature")
	if !errors.Is(err, domain.ErrDuplicateCategoryName) {
		t.Fatalf("err = %v, want ErrDuplicateCategoryName", err)
	}
}

func TestRenameCategory_UnknownCategory(t *testing.T) {
	p := projectWithCategories("Feature")

	_, err := operations.RenameCategory(p, "nope", "Fix")
	if !errors.Is(err, domain.ErrCategoryNotFound) {
		t.Fatalf("err = %v, want ErrCategoryNotFound", err)
	}
}

func TestMoveCategory_Down(t *testing.T) {
	p := projectWithCategories("A", "B", "C")
	id := p.Categories[0].ID

	_, err := operations.MoveCategory(p, id, 1)
	if err != nil {
		t.Fatalf("MoveCategory: %v", err)
	}
	if order := categoryOrder(p); order[0] != "B" || order[1] != "A" {
		t.Fatalf("order = %v, want B,A,C", order)
	}
}

func TestMoveCategory_UpAtTopIsNoChange(t *testing.T) {
	p := projectWithCategories("A", "B")
	id := p.Categories[0].ID

	_, err := operations.MoveCategory(p, id, -1)
	if !errors.Is(err, operations.ErrNoChange) {
		t.Fatalf("err = %v, want ErrNoChange", err)
	}
}

func TestDeleteCategory_CascadesTasks(t *testing.T) {
	p := projectWithCategories("A", "B")
	// give A a task so we prove the cascade removes it with the category
	task, _ := domain.NewTask("t")
	p.Categories[0].Tasks = append(p.Categories[0].Tasks, task)
	id := p.Categories[0].ID

	if err := operations.DeleteCategory(p, id); err != nil {
		t.Fatalf("DeleteCategory: %v", err)
	}
	if len(p.Categories) != 1 || p.Categories[0].Name != "B" {
		t.Fatalf("after delete categories = %v, want [B]", categoryOrder(p))
	}
}

func TestDeleteCategory_Unknown(t *testing.T) {
	p := projectWithCategories("A")

	if err := operations.DeleteCategory(p, "nope"); !errors.Is(err, domain.ErrCategoryNotFound) {
		t.Fatalf("err = %v, want ErrCategoryNotFound", err)
	}
}
