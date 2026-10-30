package operations_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/ilyasturki/ordo/internal/domain"
	"github.com/ilyasturki/ordo/internal/operations"
)

// undoBoard is c1 [a, b (b1, b2), c] and c2 [d, separator s, e], with fields set on a few rows.
func undoBoard(t *testing.T) *domain.Project {
	t.Helper()
	p := board(t)
	for _, row := range []struct{ cat, id, parent, kind string }{
		{"c1", "b2", "b", ""}, {"c2", "s", "", domain.KindSeparator}, {"c2", "e", "", ""},
	} {
		if _, err := operations.CreateTask(p, row.cat, operations.TaskFields{Title: row.id, ID: row.id, ParentID: row.parent, Kind: row.kind}); err != nil {
			t.Fatal(err)
		}
	}
	for _, op := range []operations.Op{
		{Action: "status.set", IDs: []string{"c"}, Value: domain.StatusCompleted},
		{Action: "priority.set", IDs: []string{"a"}, Value: domain.PriorityHigh},
		{Action: "tag.set", IDs: []string{"a"}, Value: domain.TagBlue},
		{Action: "estimate.set", IDs: []string{"b"}, Value: "60"},
		{Action: "task.description", IDs: []string{"a"}, Text: "notes"},
	} {
		if _, err := operations.Apply(p, op); err != nil {
			t.Fatal(err)
		}
	}
	return p
}

// plain is p as JSON without the stamps an undo can't put back.
func plain(t *testing.T, p domain.Project) string {
	t.Helper()
	p = operations.Clone(p)
	p.UpdatedAt, p.Schema = "", 0
	for i := range p.Categories {
		c := &p.Categories[i]
		c.UpdatedAt = ""
		for j := range c.Tasks {
			c.Tasks[j].UpdatedAt, c.Tasks[j].CompletionDate = "", ""
		}
	}
	raw, err := json.MarshalIndent(p, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestUndoPutsBackWhatAnOpChanged(t *testing.T) {
	ids := func(ids ...string) []string { return ids }
	for _, tc := range []struct {
		name string
		op   operations.Op
		ops  int
	}{
		{"add", operations.Op{Action: "task.add", New: "n1", Cat: "c1", Title: "new"}, 1},
		{"add a subtask", operations.Op{Action: "task.add", New: "n1", Parent: "b", Title: "sub"}, 1},
		{"insert", operations.Op{Action: "task.insert", New: "n1", Cat: "c1", IDs: ids("a"), Title: "below a"}, 1},
		{"separator", operations.Op{Action: "separator.add", New: "n1", Cat: "c2", Title: "later"}, 1},
		{"title", operations.Op{Action: "task.title", IDs: ids("a"), Title: " renamed ", Base: "a"}, 1},
		{"separator label", operations.Op{Action: "task.title", IDs: ids("s"), Title: "", Base: "s"}, 1},
		{"description", operations.Op{Action: "task.description", IDs: ids("a"), Text: "new notes\n", Base: "notes"}, 1},
		{"attach", operations.Op{Action: "task.attach", IDs: ids("a"), Lines: []string{"![shot](attachments/0123456789abcdef0123456789abcdef.png)"}}, 1},
		{"status of a mixed selection", operations.Op{Action: "status.set", IDs: ids("a", "c", "s", "b1", "x"), Value: domain.StatusCancelled}, 2},
		{"status cycle", operations.Op{Action: "status.cycle", IDs: ids("c")}, 1},
		{"status already held by some", operations.Op{Action: "status.set", IDs: ids("a", "c"), Value: domain.StatusCompleted}, 1},
		{"priority", operations.Op{Action: "priority.set", IDs: ids("b", "a", "c"), Value: domain.PriorityLow}, 2},
		{"tag", operations.Op{Action: "tag.set", IDs: ids("a", "d"), Value: domain.TagGreen}, 2},
		{"estimate", operations.Op{Action: "estimate.set", IDs: ids("b", "c"), Value: "30"}, 2},
		{"delete non-contiguous rows", operations.Op{Action: "task.delete", IDs: ids("a", "c")}, 2},
		{"delete adjacent rows", operations.Op{Action: "task.delete", IDs: ids("a", "b")}, 2},
		{"delete a parent", operations.Op{Action: "task.delete", IDs: ids("b")}, 1},
		{"delete a lone subtask", operations.Op{Action: "task.delete", IDs: ids("b1")}, 1},
		{"delete across categories", operations.Op{Action: "task.delete", IDs: ids("b2", "d", "s")}, 3},
		{"delete keeping the subtasks", operations.Op{Action: "task.delete", IDs: ids("b"), Keep: true}, 3},
		{"delete keeping one subtask", operations.Op{Action: "task.delete", IDs: ids("b", "b1"), Keep: true}, 3},
		{"delete keeping, a row above", operations.Op{Action: "task.delete", IDs: ids("a", "b"), Keep: true}, 4},
		{"delete keeping no subtask", operations.Op{Action: "task.delete", IDs: ids("b", "b1", "b2"), Keep: true}, 1},
		{"move across categories", operations.Op{Action: "task.move", IDs: ids("a", "c"), Cat: "c2"}, 2},
		{"move a parent", operations.Op{Action: "task.move", IDs: ids("b"), Cat: "c2"}, 1},
		{"move a lone subtask", operations.Op{Action: "task.move", IDs: ids("b2"), Cat: "c2"}, 1},
		{"place under a parent", operations.Op{Action: "task.place", IDs: ids("c"), Cat: "c1", Parent: "b"}, 1},
		{"place across categories", operations.Op{Action: "task.place", IDs: ids("a", "c"), Cat: "c2", Before: "s"}, 2},
		{"place subtasks at the top", operations.Op{Action: "task.place", IDs: ids("b1", "b2"), Cat: "c1", Before: "a"}, 2},
		{"place two blocks above a third", operations.Op{Action: "task.place", IDs: ids("a", "c"), Cat: "c1", Before: "b"}, 2},
		{"place where one block stays", operations.Op{Action: "task.place", IDs: ids("c", "d"), Cat: "c2", Before: "s"}, 1},
		{"nest", operations.Op{Action: "task.nest", IDs: ids("c")}, 1},
		{"shift", operations.Op{Action: "task.shift", IDs: ids("a")}, 1},
		{"add a category", operations.Op{Action: "category.add", New: "k1", Title: "Docs"}, 1},
		{"add a sub-category", operations.Op{Action: "category.add", New: "k1", Parent: "c1", Title: "Sub"}, 1},
		{"rename a category", operations.Op{Action: "category.rename", Cat: "c2", Title: "Bugs"}, 1},
		{"delete the first category", operations.Op{Action: "category.delete", Cat: "c1"}, 1},
		{"delete the last category", operations.Op{Action: "category.delete", Cat: "c2"}, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := undoBoard(t)
			original := operations.Clone(*p)
			_, wire, err := operations.Perform(p, tc.op)
			if err != nil {
				t.Fatal(err)
			}
			undo := operations.Undo(original, *p, wire)
			if len(undo) != tc.ops {
				t.Fatalf("%d undo ops, want %d: %+v", len(undo), tc.ops, undo)
			}
			for _, u := range undo {
				if !operations.Replayable(u.Action) {
					t.Fatalf("%s can't be queued", u.Action)
				}
				if _, err := operations.Apply(p, u); err != nil {
					t.Fatalf("undo %+v: %v", u, err)
				}
			}
			if got, want := plain(t, *p), plain(t, original); got != want {
				t.Fatalf("after undo:\n%s\nwant:\n%s", got, want)
			}
		})
	}
}

func TestUndoKeepsTheRowsTimestamps(t *testing.T) {
	p := undoBoard(t)
	original := operations.Clone(*p)
	if _, err := operations.Apply(p, operations.Op{Action: "task.delete", IDs: []string{"b", "c"}}); err != nil {
		t.Fatal(err)
	}
	for _, u := range operations.Undo(original, *p, operations.Op{Action: "task.delete", IDs: []string{"b", "c"}}) {
		if _, err := operations.Apply(p, u); err != nil {
			t.Fatal(err)
		}
	}
	for i, task := range p.Categories[0].Tasks {
		if want := original.Categories[0].Tasks[i]; task != want {
			t.Fatalf("row %d = %+v, want %+v", i, task, want)
		}
	}
}

// nestedBoard is c1 [s1, s2] and c2, the sub-categories holding a row each.
func nestedBoard(t *testing.T) *domain.Project {
	t.Helper()
	p := undoBoard(t)
	for _, c := range []struct{ id, name string }{{"s1", "One"}, {"s2", "Two"}} {
		if _, err := operations.Apply(p, operations.Op{Action: "category.add", New: c.id, Parent: "c1", Title: c.name}); err != nil {
			t.Fatal(err)
		}
		if _, err := operations.CreateTask(p, c.id, operations.TaskFields{Title: "in " + c.id, ID: "t" + c.id}); err != nil {
			t.Fatal(err)
		}
	}
	return p
}

func TestUndoPutsBackADeletedCategoryWithItsSubCategories(t *testing.T) {
	for _, cat := range []string{"c1", "s1", "s2"} {
		t.Run(cat, func(t *testing.T) {
			p := nestedBoard(t)
			original := operations.Clone(*p)
			op := operations.Op{Action: "category.delete", Cat: cat}
			if _, err := operations.Apply(p, op); err != nil {
				t.Fatal(err)
			}
			undo := operations.Undo(original, *p, op)
			if len(undo) != 1 || !operations.Replayable(undo[0].Action) {
				t.Fatalf("undo = %+v", undo)
			}
			if _, err := operations.Apply(p, undo[0]); err != nil {
				t.Fatal(err)
			}
			if got, want := plain(t, *p), plain(t, original); got != want {
				t.Fatalf("after undo:\n%s\nwant:\n%s", got, want)
			}
			if _, err := operations.Apply(p, undo[0]); !errors.Is(err, operations.ErrNoChange) {
				t.Fatalf("restoring twice: %v", err)
			}
		})
	}
}

func TestACategoryRestoreOutlivesWhatItWasMadeBeside(t *testing.T) {
	sub := domain.Category{ID: "s9", Name: "Nine", ParentID: "gone", Tasks: []domain.Task{}}
	p := nestedBoard(t)
	if _, err := operations.Apply(p, operations.Op{Action: "category.restore", Before: "gone", Cats: []domain.Category{sub}}); err != nil {
		t.Fatal(err)
	}
	if last := p.Categories[len(p.Categories)-1]; last.ID != "s9" || last.ParentID != "" {
		t.Fatalf("a sub-category whose parent is gone = %+v", last)
	}
	sub.ID, sub.Name, sub.ParentID = "s8", "Eight", "c1"
	if _, err := operations.Apply(p, operations.Op{Action: "category.restore", Before: "c2", Cats: []domain.Category{sub}}); err != nil {
		t.Fatal(err)
	}
	if got := p.Categories[domain.BlockEnd(p.Categories, 0)-1]; got.ID != "s8" {
		t.Fatalf("a sub-category lands among its siblings, not above another level: %+v", got)
	}
	held := domain.Category{ID: "c9", Name: "Held", Tasks: []domain.Task{{ID: "a", Title: "a", Status: domain.StatusTodo}, {ID: "r1", Title: "r1", Status: domain.StatusTodo}}}
	if _, err := operations.Apply(p, operations.Op{Action: "category.restore", Cats: []domain.Category{held}}); err != nil {
		t.Fatal(err)
	}
	if tasks := p.Categories[len(p.Categories)-1].Tasks; len(tasks) != 1 || tasks[0].ID != "r1" {
		t.Fatalf("a row the project holds again stays where it is: %+v", tasks)
	}
}

func TestRefusedCategoryRestores(t *testing.T) {
	task := domain.Task{ID: "r1", Title: "x", Status: domain.StatusTodo}
	for _, tc := range []struct {
		name string
		cats []domain.Category
		msg  string
	}{
		{"a sibling has the name", []domain.Category{{ID: "c9", Name: " FIX "}}, "A category with that name already exists"},
		{"no name", []domain.Category{{ID: "c9", Name: " "}}, "A category needs a name"},
		{"a bad id", []domain.Category{{ID: "c/9", Name: "Nine"}}, "Can't restore that category"},
		{"a second category not under the first", []domain.Category{{ID: "c9", Name: "Nine"}, {ID: "c8", Name: "Eight"}}, "Can't restore that category"},
		{"a sub-category holding one", []domain.Category{{ID: "c9", Name: "Nine", ParentID: "c1"}, {ID: "c8", Name: "Eight", ParentID: "c9"}}, "Can't restore that category"},
		{"a row twice", []domain.Category{{ID: "c9", Name: "Nine", Tasks: []domain.Task{task, task}}}, "Can't restore that category"},
		{"a row without a title", []domain.Category{{ID: "c9", Name: "Nine", Tasks: []domain.Task{{ID: "r1", Title: " ", Status: domain.StatusTodo}}}}, "A task needs a title"},
		{"a control character", []domain.Category{{ID: "c9", Name: "a\x1bb"}}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := board(t)
			before := plain(t, *p)
			_, err := operations.Apply(p, operations.Op{Action: "category.restore", Cats: tc.cats})
			if operations.VerdictOf(err) != operations.Refused {
				t.Fatalf("err = %v", err)
			}
			if msg, _ := operations.Explain(err); tc.msg != "" && msg != tc.msg {
				t.Fatalf("%q, want %q", msg, tc.msg)
			}
			if plain(t, *p) != before {
				t.Fatal("a refused restore changed the project")
			}
		})
	}
}

func TestUndoOfProjectOps(t *testing.T) {
	before := domain.Project{ID: "p1", Name: "Old"}
	after := domain.Project{ID: "p1", Name: "New"}
	if got := operations.Undo(domain.Project{}, after, operations.Op{Action: "project.create", Title: "New"}); len(got) != 1 || got[0].Action != "project.delete" {
		t.Fatalf("create: %+v", got)
	}
	if got := operations.Undo(before, after, operations.Op{Action: "project.rename", Title: "New"}); len(got) != 1 || got[0].Action != "project.rename" || got[0].Title != "Old" {
		t.Fatalf("rename: %+v", got)
	}
	for _, op := range []operations.Op{{Action: "project.delete"}, {Action: "category.delete", Cat: "c1"}} {
		if got := operations.Undo(before, domain.Project{}, op); got != nil {
			t.Fatalf("%s: %+v", op.Action, got)
		}
	}
}
