package operations_test

import (
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/ilyasturki/ordo/internal/domain"
	"github.com/ilyasturki/ordo/internal/operations"
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

func categoryIDs(p *domain.Project) []string {
	ids := make([]string, len(p.Categories))
	for i, c := range p.Categories {
		ids[i] = c.ID
	}
	return ids
}

// placeBoard is a [a1, a2, a3], b [b1], c, with a sub-category under each of a and b.
func placeBoard(t *testing.T) *domain.Project {
	t.Helper()
	p := &domain.Project{ID: "p1", Name: "Test"}
	for _, c := range []struct{ id, parent string }{
		{"a", ""}, {"a1", "a"}, {"a2", "a"}, {"a3", "a"}, {"b", ""}, {"b1", "b"}, {"c", ""},
	} {
		if _, err := operations.Apply(p, operations.Op{Action: "category.add", New: c.id, Parent: c.parent, Title: c.id}); err != nil {
			t.Fatal(err)
		}
	}
	return p
}

func TestPlaceCategory(t *testing.T) {
	for _, tc := range []struct {
		name      string
		cat, dest string
		want      []string
	}{
		{"a block up, its sub-categories along", "b", "a", []string{"b", "b1", "a", "a1", "a2", "a3", "c"}},
		{"a block down", "a", "c", []string{"b", "b1", "a", "a1", "a2", "a3", "c"}},
		{"a block last", "a", "", []string{"b", "b1", "c", "a", "a1", "a2", "a3"}},
		{"a sub-category up", "a3", "a1", []string{"a", "a3", "a1", "a2", "b", "b1", "c"}},
		{"a sub-category down", "a1", "a3", []string{"a", "a2", "a1", "a3", "b", "b1", "c"}},
		{"a sub-category last among its siblings", "a1", "", []string{"a", "a2", "a3", "a1", "b", "b1", "c"}},
		{"a before that is gone lands it last", "a", "gone", []string{"b", "b1", "c", "a", "a1", "a2", "a3"}},
		{"a before one level down lands it last", "b", "a2", []string{"a", "a1", "a2", "a3", "c", "b", "b1"}},
		{"a sub-category's before under another parent", "a1", "b1", []string{"a", "a2", "a3", "a1", "b", "b1", "c"}},
		{"a sub-category's before one level up", "a1", "c", []string{"a", "a2", "a3", "a1", "b", "b1", "c"}},
		{"a before among its own sub-categories", "a", "a1", []string{"b", "b1", "c", "a", "a1", "a2", "a3"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := placeBoard(t)
			out, err := operations.Apply(p, operations.Op{Action: "category.place", Cat: tc.cat, Before: tc.dest})
			if err != nil {
				t.Fatal(err)
			}
			if out.Msg != "Moved “"+tc.cat+"”" || out.Row != tc.cat {
				t.Fatalf("outcome = %+v", out)
			}
			if got := categoryIDs(p); !slices.Equal(got, tc.want) {
				t.Fatalf("order = %v, want %v", got, tc.want)
			}
			for _, c := range p.Categories {
				if want := map[string]string{"a1": "a", "a2": "a", "a3": "a", "b1": "b"}[c.ID]; c.ParentID != want {
					t.Fatalf("%s now sits under %q, want %q", c.ID, c.ParentID, want)
				}
			}
		})
	}
}

func TestPlaceCategoryWhereItAlreadyIsChangesNothing(t *testing.T) {
	for _, tc := range []struct{ cat, dest string }{
		{"a", "b"}, {"a1", "a2"}, {"c", ""}, {"a3", ""}, {"b1", "gone"}, {"b", "b"},
	} {
		p := placeBoard(t)
		before := operations.Clone(*p)
		if _, err := operations.Apply(p, operations.Op{Action: "category.place", Cat: tc.cat, Before: tc.dest}); !errors.Is(err, operations.ErrNoChange) {
			t.Fatalf("%s above %q: %v", tc.cat, tc.dest, err)
		}
		if !reflect.DeepEqual(before, *p) {
			t.Fatalf("%s above %q changed the project", tc.cat, tc.dest)
		}
	}
}

func TestPlacingACategoryOnAChangedFile(t *testing.T) {
	place := operations.Op{Action: "category.place", Cat: "c", Before: "b"}
	p := placeBoard(t)
	if _, err := operations.Apply(p, operations.Op{Action: "category.delete", Cat: "b"}); err != nil {
		t.Fatal(err)
	}
	if _, err := operations.Apply(p, place); !errors.Is(err, operations.ErrNoChange) {
		t.Fatalf("already last with its before gone: %v", err)
	}
	place.Cat = "a"
	if _, err := operations.Apply(p, place); err != nil {
		t.Fatal(err)
	}
	if got, want := categoryIDs(p), []string{"c", "a", "a1", "a2", "a3"}; !slices.Equal(got, want) {
		t.Fatalf("order = %v, want %v", got, want)
	}
	if _, err := operations.Apply(p, operations.Op{Action: "category.add", New: "n", Title: "n"}); err != nil {
		t.Fatal(err)
	}
	if _, err := operations.Apply(p, operations.Op{Action: "category.place", Cat: "c", Before: "n"}); err != nil {
		t.Fatal(err)
	}
	if got, want := categoryIDs(p), []string{"a", "a1", "a2", "a3", "c", "n"}; !slices.Equal(got, want) {
		t.Fatalf("beside a category added since: order = %v, want %v", got, want)
	}
	_, err := operations.Apply(p, operations.Op{Action: "category.place", Cat: "b", Before: "c"})
	if !errors.Is(err, domain.ErrCategoryNotFound) || operations.VerdictOf(err) != operations.Gone {
		t.Fatalf("a category deleted elsewhere: %v", err)
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
