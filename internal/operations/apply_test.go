package operations_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/ilyasturki/ordo/internal/domain"
	"github.com/ilyasturki/ordo/internal/operations"
)

// board is c1 [a, b (subtask b1), c] and c2 [d].
func board(t *testing.T) *domain.Project {
	t.Helper()
	p := projectWithCategory("c1")
	p.Categories = append(p.Categories, domain.Category{ID: "c2", Name: "Fix", Tasks: []domain.Task{}})
	for _, row := range []struct{ cat, id, parent string }{
		{"c1", "a", ""}, {"c1", "b", ""}, {"c1", "b1", "b"}, {"c1", "c", ""}, {"c2", "d", ""},
	} {
		if _, err := operations.CreateTask(p, row.cat, operations.TaskFields{Title: row.id, ID: row.id, ParentID: row.parent}); err != nil {
			t.Fatal(err)
		}
	}
	return p
}

func outline(p *domain.Project) map[string][]string {
	out := make(map[string][]string)
	for _, c := range p.Categories {
		for _, t := range c.Tasks {
			row := t.ID
			if t.ParentID != "" {
				row = t.ParentID + ">" + t.ID
			}
			out[c.ID] = append(out[c.ID], row)
		}
	}
	return out
}

func TestReplayingAnOpChangesNothingMore(t *testing.T) {
	ops := []operations.Op{
		{Action: "task.add", New: "n1", Cat: "c1", Title: "new"},
		{Action: "task.add", New: "n2", Parent: "b", Title: "sub"},
		{Action: "task.insert", New: "n3", Cat: "c1", IDs: []string{"a"}, Title: "below a"},
		{Action: "separator.add", New: "n4", Cat: "c2", Title: "later"},
		{Action: "task.title", IDs: []string{"a"}, Title: "renamed", Base: "a"},
		{Action: "task.description", IDs: []string{"a"}, Text: "details", Base: ""},
		{Action: "task.attach", IDs: []string{"a"}, Lines: []string{"![shot](attachments/0123456789abcdef0123456789abcdef.png)"}},
		{Action: "status.set", IDs: []string{"a", "c"}, Value: domain.StatusCompleted},
		{Action: "priority.set", IDs: []string{"a"}, Value: domain.PriorityHigh},
		{Action: "tag.set", IDs: []string{"a"}, Value: domain.TagBlue},
		{Action: "estimate.set", IDs: []string{"a"}, Value: "60"},
		{Action: "task.place", IDs: []string{"c"}, Cat: "c1", Parent: "b", Before: ""},
		{Action: "task.place", IDs: []string{"a"}, Cat: "c2", Before: "d"},
		{Action: "task.move", IDs: []string{"b"}, Cat: "c2"},
		{Action: "task.delete", IDs: []string{"d"}},
	}
	for _, op := range ops {
		t.Run(op.Action, func(t *testing.T) {
			if !operations.Replayable(op.Action) {
				t.Fatalf("%s is not replayable", op.Action)
			}
			p := board(t)
			if _, err := operations.Apply(p, op); err != nil {
				t.Fatalf("first apply: %v", err)
			}
			once := operations.Clone(*p)
			if _, err := operations.Apply(p, op); !errors.Is(err, operations.ErrNoChange) {
				t.Fatalf("second apply = %v, want ErrNoChange", err)
			}
			if !reflect.DeepEqual(once, *p) {
				t.Fatal("the second apply changed the project")
			}
		})
	}
}

func TestPerformSendsWhereAGestureLanded(t *testing.T) {
	for _, op := range []operations.Op{
		{Action: "status.cycle", IDs: []string{"a"}},
		{Action: "task.nest", IDs: []string{"c"}},
		{Action: "task.unnest", IDs: []string{"b1"}},
		{Action: "task.shift", IDs: []string{"a"}},
		{Action: "task.shift", IDs: []string{"b"}, Up: true},
		{Action: "task.shift", IDs: []string{"c"}},
	} {
		t.Run(op.Action, func(t *testing.T) {
			gestured, replayed := board(t), board(t)
			_, wire, err := operations.Perform(gestured, op)
			if err != nil {
				t.Fatal(err)
			}
			if !operations.Replayable(wire.Action) {
				t.Fatalf("Perform sent %s, which can't be queued", wire.Action)
			}
			if _, err := operations.Apply(replayed, wire); err != nil {
				t.Fatalf("replaying %+v: %v", wire, err)
			}
			if !reflect.DeepEqual(outline(gestured), outline(replayed)) {
				t.Fatalf("gesture gave %v, its replay %v", outline(gestured), outline(replayed))
			}
			if gestured.Categories[0].Tasks[0].Status != replayed.Categories[0].Tasks[0].Status {
				t.Fatal("the replay landed on another status")
			}
		})
	}
}

func TestPerformMintsTheIDOfAnAdd(t *testing.T) {
	p := board(t)
	out, wire, err := operations.Perform(p, operations.Op{Action: "task.add", Cat: "c1", Title: "new"})
	if err != nil {
		t.Fatal(err)
	}
	if wire.New == "" || out.Row != wire.New {
		t.Fatalf("row %q, wire new %q", out.Row, wire.New)
	}
	if _, err := operations.Apply(p, wire); !errors.Is(err, operations.ErrNoChange) {
		t.Fatalf("replaying the add on the phone's own copy = %v", err)
	}
}

func TestQueuedEditsOutliveTheRowTheyWereMadeBeside(t *testing.T) {
	p := board(t)
	if _, err := operations.Apply(p, operations.Op{Action: "task.delete", IDs: []string{"a", "b"}}); err != nil {
		t.Fatal(err)
	}
	for _, op := range []operations.Op{
		{Action: "task.insert", New: "n1", Cat: "c1", IDs: []string{"a"}, Title: "below a"},
		{Action: "task.add", New: "n2", Cat: "c1", Parent: "b", Title: "under b"},
		{Action: "task.place", IDs: []string{"d"}, Cat: "c1", Parent: "b", Before: "a"},
	} {
		if _, err := operations.Apply(p, op); err != nil {
			t.Fatalf("%s: %v", op.Action, err)
		}
	}
	want := map[string][]string{"c1": {"n1", "c", "n2", "d"}}
	if got := outline(p); !reflect.DeepEqual(got, want) {
		t.Fatalf("outline = %v, want %v", got, want)
	}
}

func TestAnEditToRowsDeletedElsewhereIsGone(t *testing.T) {
	for _, op := range []operations.Op{
		{Action: "status.set", IDs: []string{"x"}, Value: domain.StatusCompleted},
		{Action: "status.cycle", IDs: []string{"x"}},
		{Action: "priority.set", IDs: []string{"x"}, Value: domain.PriorityHigh},
		{Action: "tag.set", IDs: []string{"x"}, Value: domain.TagBlue},
		{Action: "estimate.set", IDs: []string{"x"}, Value: "30"},
		{Action: "task.title", IDs: []string{"x"}, Title: "y"},
		{Action: "task.place", IDs: []string{"x"}, Cat: "c1"},
		{Action: "task.move", IDs: []string{"x"}, Cat: "c2"},
	} {
		if _, err := operations.Apply(board(t), op); operations.VerdictOf(err) != operations.Gone {
			t.Errorf("%s on a deleted row: %v", op.Action, err)
		}
	}
	if _, err := operations.Apply(board(t), operations.Op{Action: "task.delete", IDs: []string{"x"}}); !errors.Is(err, operations.ErrNoChange) {
		t.Errorf("deleting a deleted row: %v", err)
	}
}

func TestAStaleTextEditCarriesTheirs(t *testing.T) {
	p := board(t)
	if _, err := operations.Apply(p, operations.Op{Action: "task.description", IDs: []string{"a"}, Text: "theirs"}); err != nil {
		t.Fatal(err)
	}
	before := operations.Clone(*p)
	_, err := operations.Apply(p, operations.Op{Action: "task.description", IDs: []string{"a"}, Text: "mine", Base: ""})
	var conflict *operations.Conflict
	if !errors.As(err, &conflict) || conflict.Theirs != "theirs" {
		t.Fatalf("err = %v", err)
	}
	if !reflect.DeepEqual(before, *p) {
		t.Fatal("a refused edit changed the project")
	}
}

func TestEveryWebActionApplies(t *testing.T) {
	for _, a := range operations.Actions {
		if a.Surface&operations.Web == 0 {
			continue
		}
		if _, err := operations.Apply(board(t), operations.Op{Action: a.ID}); errors.Is(err, operations.ErrUnknownAction) {
			t.Errorf("%s is offered on the web but Apply doesn't know it", a.ID)
		}
	}
	if _, err := operations.Apply(board(t), operations.Op{Action: "task.levitate"}); !errors.Is(err, operations.ErrUnknownAction) {
		t.Fatalf("unknown action: %v", err)
	}
}
