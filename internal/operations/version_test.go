package operations_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/ilyasturki/ordo/internal/domain"
	"github.com/ilyasturki/ordo/internal/operations"
)

func focusOn(p *domain.Project, name string) string {
	for _, c := range p.Categories {
		if c.Name == name {
			return c.ID
		}
	}
	return ""
}

func TestNextVersionLandsOnTheNewestSide(t *testing.T) {
	for _, tc := range []struct {
		name  string
		cats  []string
		focus string
		want  []string
	}{
		{"newest first", []string{"v1.3.0", "v1.2.0", "v1.1.0"}, "v1.3.0", []string{"v1.4.0", "v1.3.0", "v1.2.0", "v1.1.0"}},
		{"oldest first", []string{"v1.1.0", "v1.2.0", "v1.3.0"}, "v1.3.0", []string{"v1.1.0", "v1.2.0", "v1.3.0", "v1.4.0"}},
		{"from an older version it steps past the taken ones", []string{"v1.3.0", "v1.2.0", "v1.1.0"}, "v1.1.0", []string{"v1.4.0", "v1.3.0", "v1.2.0", "v1.1.0"}},
		{"one version reads as newest first", []string{"Inbox", "v1.3.0"}, "v1.3.0", []string{"Inbox", "v1.4.0", "v1.3.0"}},
		{"oldest first leaves what follows the versions", []string{"v1.1.0", "v1.2.0", "Later"}, "v1.1.0", []string{"v1.1.0", "v1.2.0", "v1.3.0", "Later"}},
		{"keeps the prefix and format, drops the label", []string{"Release v0.9 - polish"}, "Release v0.9 - polish", []string{"Release v0.10", "Release v0.9 - polish"}},
		{"a labelled version holds its number", []string{"v0.10.0 - plan", "v0.9.1 - polish"}, "v0.9.1 - polish", []string{"v0.11.0", "v0.10.0 - plan", "v0.9.1 - polish"}},
		{"a bump resets the patch", []string{"1.2.7"}, "1.2.7", []string{"1.3.0", "1.2.7"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := projectWithCategories(tc.cats...)
			out, err := operations.Apply(p, operations.Op{Action: "version.next", Cat: focusOn(p, tc.focus)})
			if err != nil {
				t.Fatal(err)
			}
			if got := categoryOrder(p); !slices.Equal(got, tc.want) {
				t.Fatalf("order = %v, want %v", got, tc.want)
			}
			if _, err := p.FindCategoryByID(out.Row); err != nil {
				t.Fatalf("Row %q is not the new category", out.Row)
			}
		})
	}
}

func TestNextVersionWithoutAVersionIsRefused(t *testing.T) {
	p := projectWithCategories("Inbox", "v1.0.0")
	before := categoryOrder(p)
	_, err := operations.Apply(p, operations.Op{Action: "version.next", Cat: focusOn(p, "Inbox")})
	var refusal *operations.Refusal
	if !errors.As(err, &refusal) || refusal.Msg != "No version in this name" {
		t.Fatalf("err = %v", err)
	}
	if !slices.Equal(categoryOrder(p), before) {
		t.Fatal("a refused edit changed the project")
	}
}

func TestNextVersionOnASubCategoryWorksFromItsParent(t *testing.T) {
	p := projectWithCategories("v1.0.0", "v0.9.0")
	sub, err := operations.CreateCategory(p, p.Categories[0].ID, "ui")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := operations.Apply(p, operations.Op{Action: "version.next", Cat: sub.ID}); err != nil {
		t.Fatal(err)
	}
	if got := categoryOrder(p); !slices.Equal(got, []string{"v1.1.0", "v1.0.0", "ui", "v0.9.0"}) {
		t.Fatalf("order = %v", got)
	}
}

func TestNextVersionAmongSubCategoriesStaysUnderTheirParent(t *testing.T) {
	p := projectWithCategories("Releases", "Other")
	for _, name := range []string{"v1.1.0", "v1.0.0"} {
		if _, err := operations.CreateCategory(p, p.Categories[0].ID, name); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := operations.Apply(p, operations.Op{Action: "version.next", Cat: focusOn(p, "v1.1.0")}); err != nil {
		t.Fatal(err)
	}
	if got := categoryOrder(p); !slices.Equal(got, []string{"Releases", "v1.2.0", "v1.1.0", "v1.0.0", "Other"}) {
		t.Fatalf("order = %v", got)
	}
	if p.Categories[1].ParentID != p.Categories[0].ID {
		t.Fatal("the new version left its parent")
	}
}

func TestPerformNextVersionQueuesAReplayableAddAndUndoDeletesIt(t *testing.T) {
	for _, cats := range [][]string{{"v1.2.0", "v1.1.0"}, {"v1.1.0", "v1.2.0"}} {
		p := projectWithCategories(cats...)
		file := operations.Clone(*p)
		before := operations.Clone(*p)
		out, wire, err := operations.Perform(p, operations.Op{Action: "version.next", Cat: p.Categories[0].ID})
		if err != nil {
			t.Fatal(err)
		}
		if wire.Action != "category.add" || wire.New != out.Row || wire.Title != "v1.3.0" || !operations.Replayable(wire.Action) {
			t.Fatalf("queued %+v", wire)
		}
		for range 2 {
			if _, err := operations.Apply(&file, wire); err != nil && !errors.Is(err, operations.ErrNoChange) {
				t.Fatal(err)
			}
			if !slices.Equal(categoryOrder(&file), categoryOrder(p)) {
				t.Fatalf("file %v, app %v", categoryOrder(&file), categoryOrder(p))
			}
		}
		undo := operations.Undo(before, *p, wire)
		if len(undo) != 1 {
			t.Fatalf("undo = %v", undo)
		}
		if _, err := operations.Apply(p, undo[0]); err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(categoryOrder(p), cats) {
			t.Fatalf("after undo %v", categoryOrder(p))
		}
	}
}

func TestCategoryAddBeforeLandsAboveThatSibling(t *testing.T) {
	p := projectWithCategories("a", "b", "c")
	if _, err := operations.Apply(p, operations.Op{Action: "category.add", Title: "x", Before: p.Categories[1].ID}); err != nil {
		t.Fatal(err)
	}
	if got := categoryOrder(p); !slices.Equal(got, []string{"a", "x", "b", "c"}) {
		t.Fatalf("order = %v", got)
	}
	if _, err := operations.Apply(p, operations.Op{Action: "category.add", Title: "y", Before: "gone"}); err != nil {
		t.Fatal(err)
	}
	if got := categoryOrder(p); !slices.Equal(got, []string{"a", "x", "b", "c", "y"}) {
		t.Fatalf("order = %v", got)
	}
}
