package operations_test

import (
	"encoding/json"
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
