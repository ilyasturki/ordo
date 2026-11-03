package operations_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/ilyasturki/ordo/internal/domain"
	"github.com/ilyasturki/ordo/internal/operations"
)

// pinShape is each category's ID, "*" first when pinned, "<parent" after when nested.
func pinShape(p *domain.Project) []string {
	out := make([]string, len(p.Categories))
	for i, c := range p.Categories {
		out[i] = c.ID
		if c.Pinned {
			out[i] = "*" + out[i]
		}
		if c.ParentID != "" {
			out[i] += "<" + c.ParentID
		}
	}
	return out
}

func apply(t *testing.T, p *domain.Project, ops ...operations.Op) {
	t.Helper()
	for _, op := range ops {
		if _, err := operations.Apply(p, op); err != nil {
			t.Fatalf("%s %s: %v", op.Action, op.Cat, err)
		}
	}
}

func pin(cat string) operations.Op {
	return operations.Op{Action: "category.pin", Cat: cat, Value: "true"}
}

func unpin(cat string) operations.Op {
	return operations.Op{Action: "category.pin", Cat: cat, Value: "false"}
}

func TestPinCategory(t *testing.T) {
	for _, tc := range []struct {
		name string
		ops  []operations.Op
		want []string
	}{
		{"pinning takes the block first", []operations.Op{pin("b")}, []string{"*b", "b1<b", "a", "a1<a", "a2<a", "a3<a", "c"}},
		{"a second pin ends the pinned run", []operations.Op{pin("c"), pin("b")}, []string{"*c", "*b", "b1<b", "a", "a1<a", "a2<a", "a3<a"}},
		{"unpinning starts the unpinned run", []operations.Op{pin("c"), pin("b"), unpin("c")}, []string{"*b", "b1<b", "c", "a", "a1<a", "a2<a", "a3<a"}},
		{"a pinned sub-category goes first under its parent", []operations.Op{pin("a3")}, []string{"a", "*a3<a", "a1<a", "a2<a", "b", "b1<b", "c"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := placeBoard(t)
			apply(t, p, tc.ops...)
			if got := pinShape(p); !slices.Equal(got, tc.want) {
				t.Fatalf("order = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestPinCategoryAnswers(t *testing.T) {
	p := placeBoard(t)
	out, err := operations.Apply(p, pin("b"))
	if err != nil || out.Msg != "Pinned “b”" || out.Row != "b" {
		t.Fatalf("pin = %+v, %v", out, err)
	}
	if _, err := operations.Apply(p, pin("b")); !errors.Is(err, operations.ErrNoChange) {
		t.Fatalf("pinning twice = %v, want ErrNoChange", err)
	}
	if _, err := operations.Apply(p, operations.Op{Action: "category.pin", Cat: "b", Value: "yes"}); err == nil {
		t.Fatal("a value other than true or false applied")
	}
	if _, err := operations.Apply(p, pin("gone")); !errors.Is(err, domain.ErrCategoryNotFound) {
		t.Fatalf("pinning a gone category = %v", err)
	}
	if out, err := operations.Apply(p, unpin("b")); err != nil || out.Msg != "Unpinned “b”" {
		t.Fatalf("unpin = %+v, %v", out, err)
	}
}

func TestUndoPutsAPinnedCategoryBack(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup []operations.Op
		op    operations.Op
	}{
		{"a pin", nil, pin("b")},
		{"a pin of the last one", nil, pin("c")},
		{"an unpin", []operations.Op{pin("c"), pin("a")}, unpin("c")},
		{"an unpin of the last pinned one", []operations.Op{pin("c"), pin("a")}, unpin("a")},
		{"a sub-category's pin", nil, pin("a2")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := placeBoard(t)
			apply(t, p, tc.setup...)
			before := operations.Clone(*p)
			apply(t, p, tc.op)
			apply(t, p, operations.Undo(before, *p, tc.op)...)
			if plain(t, *p) != plain(t, before) {
				t.Fatalf("undo left %v, want %v", pinShape(p), pinShape(&before))
			}
		})
	}
}

func TestPlaceCategoryKeepsItsPinnedRun(t *testing.T) {
	for _, tc := range []struct {
		name, cat, dest string
		want            []string
	}{
		{"an unpinned one above a pinned one lands after the pinned run", "a", "c", []string{"*c", "*b", "b1<b", "a", "a1<a", "a2<a", "a3<a"}},
		{"a pinned one placed last lands at the end of the pinned run", "c", "", []string{"*b", "b1<b", "*c", "a", "a1<a", "a2<a", "a3<a"}},
		{"within the pinned run", "b", "c", []string{"*b", "b1<b", "*c", "a", "a1<a", "a2<a", "a3<a"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := placeBoard(t)
			apply(t, p, pin("c"), pin("b"))
			_, err := operations.Apply(p, operations.Op{Action: "category.place", Cat: tc.cat, Before: tc.dest})
			if err != nil && !errors.Is(err, operations.ErrNoChange) {
				t.Fatal(err)
			}
			if got := pinShape(p); !slices.Equal(got, tc.want) {
				t.Fatalf("order = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestRestoreKeepsPinsFirst(t *testing.T) {
	p := placeBoard(t)
	apply(t, p, pin("c"))
	before := operations.Clone(*p)
	del := operations.Op{Action: "category.delete", Cat: "c"}
	apply(t, p, del, operations.Op{Action: "category.place", Cat: "b", Before: "a"})
	apply(t, p, operations.Undo(before, before, del)...)
	if got, want := pinShape(p), []string{"*c", "b", "b1<b", "a", "a1<a", "a2<a", "a3<a"}; !slices.Equal(got, want) {
		t.Fatalf("order = %v, want %v", got, want)
	}
}

func TestShiftCategories(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup []operations.Op
		ids   []string
		dir   int
		want  []string
		err   string
	}{
		{name: "a block down", ids: []string{"a"}, dir: 1, want: []string{"b", "b1<b", "a", "a1<a", "a2<a", "a3<a", "c"}},
		{name: "the last top-level one refuses", ids: []string{"c"}, dir: 1, err: "Already at the bottom"},
		{name: "the first top-level one refuses", ids: []string{"a"}, dir: -1, err: "Already at the top"},
		{name: "a sub-category among its siblings", ids: []string{"a2"}, dir: 1, want: []string{"a", "a1<a", "a3<a", "a2<a", "b", "b1<b", "c"}},
		{name: "K on the first sub-category goes last under the parent above", ids: []string{"b1"}, dir: -1, want: []string{"a", "a1<a", "a2<a", "a3<a", "b1<a", "b", "c"}},
		{name: "J on the last sub-category goes first under the parent below", ids: []string{"a3"}, dir: 1, want: []string{"a", "a1<a", "a2<a", "b", "a3<b", "b1<b", "c"}},
		{name: "J into a parent with no sub-categories", ids: []string{"b1"}, dir: 1, want: []string{"a", "a1<a", "a2<a", "a3<a", "b", "c", "b1<c"}},
		{name: "a run of sub-categories passes together", ids: []string{"a2", "a3"}, dir: 1, want: []string{"a", "a1<a", "b", "a2<b", "a3<b", "b1<b", "c"}},
		{name: "K under the first top-level one refuses", ids: []string{"a1"}, dir: -1, err: "Already at the top of its parent"},
		{name: "J under the last top-level one refuses", setup: []operations.Op{{Action: "category.add", New: "c1", Parent: "c", Title: "c1"}}, ids: []string{"c1"}, dir: 1, err: "Already at the bottom of its parent"},
		{name: "a sibling of the same name there refuses", setup: []operations.Op{{Action: "category.rename", Cat: "b1", Title: "a1"}}, ids: []string{"b1"}, dir: -1, err: `"a" already has a "a1"`},
		{name: "J on the last pinned one refuses", setup: []operations.Op{pin("b")}, ids: []string{"b"}, dir: 1, err: "Already at the bottom of the pinned ones"},
		{name: "K on the first unpinned one refuses", setup: []operations.Op{pin("b")}, ids: []string{"a"}, dir: -1, err: "Already at the top of the unpinned ones"},
		{name: "pinned ones move among themselves", setup: []operations.Op{pin("c"), pin("b")}, ids: []string{"b"}, dir: -1, want: []string{"*b", "b1<b", "*c", "a", "a1<a", "a2<a", "a3<a"}},
		{name: "a pinned sub-category across parents ends the pinned run there", setup: []operations.Op{pin("a3"), pin("b1")}, ids: []string{"b1"}, dir: -1, want: []string{"a", "*a3<a", "*b1<a", "a1<a", "a2<a", "b", "c"}},
		{name: "an unpinned sub-category across parents lands after the pinned run there", setup: []operations.Op{pin("b1"), {Action: "category.add", New: "b2", Parent: "b", Title: "b2"}}, ids: []string{"a3"}, dir: 1, want: []string{"a", "a1<a", "a2<a", "b", "*b1<b", "a3<b", "b2<b", "c"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := placeBoard(t)
			apply(t, p, tc.setup...)
			before := operations.Clone(*p)
			err := operations.ShiftCategories(p, tc.ids, tc.dir)
			if tc.err != "" {
				if err == nil || err.Error() != tc.err {
					t.Fatalf("err = %v, want %q", err, tc.err)
				}
				if plain(t, *p) != plain(t, before) {
					t.Fatal("a refused shift changed the project")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if tc.want != nil && !slices.Equal(pinShape(p), tc.want) {
				t.Fatalf("order = %v, want %v", pinShape(p), tc.want)
			}
		})
	}
}
