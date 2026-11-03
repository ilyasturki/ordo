package operations_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/ilyasturki/ordo/internal/domain"
	"github.com/ilyasturki/ordo/internal/operations"
)

// placeBoardRows is c1 [a, b (b1, b2), separator s, c], c2 [d, e] and an empty c3.
func placeBoardRows(t *testing.T) *domain.Project {
	t.Helper()
	p := projectWithCategory("c1")
	p.Categories = append(p.Categories,
		domain.Category{ID: "c2", Name: "Fix", Tasks: []domain.Task{}},
		domain.Category{ID: "c3", Name: "Later", Tasks: []domain.Task{}},
	)
	for _, row := range []struct{ cat, id, parent, kind string }{
		{"c1", "a", "", ""}, {"c1", "b", "", ""}, {"c1", "b1", "b", ""}, {"c1", "b2", "b", ""},
		{"c1", "s", "", domain.KindSeparator}, {"c1", "c", "", ""}, {"c2", "d", "", ""}, {"c2", "e", "", ""},
	} {
		if _, err := operations.CreateTask(p, row.cat, operations.TaskFields{Title: row.id, ID: row.id, ParentID: row.parent, Kind: row.kind}); err != nil {
			t.Fatal(err)
		}
	}
	return p
}

// The op a drop on the phone sends: the row, its category, the row it now sits above.
func place(id, cat, before string) operations.Op {
	return operations.Op{Action: "task.place", IDs: []string{id}, Cat: cat, Before: before}
}

func TestPlaceRows(t *testing.T) {
	for _, tc := range []struct {
		name string
		op   operations.Op
		want map[string][]string
	}{
		{"down by one, above a separator", place("a", "c1", "s"), map[string][]string{
			"c1": {"b", "b>b1", "b>b2", "a", "s", "c"}, "c2": {"d", "e"}}},
		{"up by one, past a separator", place("c", "c1", "s"), map[string][]string{
			"c1": {"a", "b", "b>b1", "b>b2", "c", "s"}, "c2": {"d", "e"}}},
		{"last", place("a", "c1", ""), map[string][]string{
			"c1": {"b", "b>b1", "b>b2", "s", "c", "a"}, "c2": {"d", "e"}}},
		{"a dragged separator", place("s", "c1", "a"), map[string][]string{
			"c1": {"s", "a", "b", "b>b1", "b>b2", "c"}, "c2": {"d", "e"}}},
		{"a parent takes its subtasks", place("b", "c2", "e"), map[string][]string{
			"c1": {"a", "s", "c"}, "c2": {"d", "b", "b>b1", "b>b2", "e"}}},
		{"first of the next category", place("c", "c2", "d"), map[string][]string{
			"c1": {"a", "b", "b>b1", "b>b2", "s"}, "c2": {"c", "d", "e"}}},
		{"last of the previous category", place("d", "c1", ""), map[string][]string{
			"c1": {"a", "b", "b>b1", "b>b2", "s", "c", "d"}, "c2": {"e"}}},
		{"into an empty category", place("a", "c3", ""), map[string][]string{
			"c1": {"b", "b>b1", "b>b2", "s", "c"}, "c2": {"d", "e"}, "c3": {"a"}}},
		{"above the next row shown, behind the rows a filter hides", place("d", "c1", "c"), map[string][]string{
			"c1": {"a", "b", "b>b1", "b>b2", "s", "d", "c"}, "c2": {"e"}}},
		{"a subtask as before lands above its parent", place("d", "c1", "b2"), map[string][]string{
			"c1": {"a", "d", "b", "b>b1", "b>b2", "s", "c"}, "c2": {"e"}}},
		{"a before that is gone lands it last", place("a", "c1", "gone"), map[string][]string{
			"c1": {"b", "b>b1", "b>b2", "s", "c", "a"}, "c2": {"d", "e"}}},
		{"a before in another category lands it last", place("a", "c1", "d"), map[string][]string{
			"c1": {"b", "b>b1", "b>b2", "s", "c", "a"}, "c2": {"d", "e"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := placeBoardRows(t)
			out, err := operations.Apply(p, tc.op)
			if err != nil {
				t.Fatal(err)
			}
			if out.Msg != "Moved" || out.Row != tc.op.IDs[0] {
				t.Fatalf("outcome = %+v", out)
			}
			if got := outline(p); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("outline = %v, want %v", got, tc.want)
			}
			once := operations.Clone(*p)
			if _, err := operations.Apply(p, tc.op); !errors.Is(err, operations.ErrNoChange) {
				t.Fatalf("sent again = %v, want ErrNoChange", err)
			}
			if !reflect.DeepEqual(once, *p) {
				t.Fatal("sent again, it changed the project")
			}
		})
	}
}

func TestPlaceRowsWhereTheyAlreadyAreChangesNothing(t *testing.T) {
	for _, op := range []operations.Op{
		place("a", "c1", "b"), place("c", "c1", ""), place("b", "c1", "s"), place("d", "c2", "e"), place("e", "c2", "gone"),
	} {
		p := placeBoardRows(t)
		before := operations.Clone(*p)
		if _, err := operations.Apply(p, op); !errors.Is(err, operations.ErrNoChange) {
			t.Fatalf("%v: %v", op, err)
		}
		if !reflect.DeepEqual(before, *p) {
			t.Fatalf("%v changed the project", op)
		}
	}
}

func TestPlaceRowsOnAFileChangedSince(t *testing.T) {
	for _, tc := range []struct {
		name  string
		since []operations.Op
		op    operations.Op
		want  map[string][]string
	}{
		{"its before deleted: last", []operations.Op{{Action: "task.delete", IDs: []string{"c"}}},
			place("a", "c1", "c"), map[string][]string{"c1": {"b", "b>b1", "b>b2", "s", "a"}, "c2": {"d", "e"}}},
		{"its before moved away: last", []operations.Op{place("c", "c2", "")},
			place("a", "c1", "c"), map[string][]string{"c1": {"b", "b>b1", "b>b2", "s", "a"}, "c2": {"d", "e", "c"}}},
		{"its before nested since: above the parent", []operations.Op{{Action: "task.place", IDs: []string{"c"}, Cat: "c1", Parent: "b"}},
			place("d", "c1", "c"), map[string][]string{"c1": {"a", "d", "b", "b>b1", "b>b2", "b>c", "s"}, "c2": {"e"}}},
		{"rows added above its before: still right above it", []operations.Op{
			{Action: "task.add", New: "n1", Cat: "c2", Title: "n1"}, {Action: "task.insert", New: "n2", Cat: "c1", IDs: []string{"s"}, Title: "n2"},
		}, place("a", "c1", "c"), map[string][]string{"c1": {"b", "b>b1", "b>b2", "s", "n2", "a", "c"}, "c2": {"d", "e", "n1"}}},
		{"the row got a subtask since: it goes along", []operations.Op{{Action: "task.add", New: "a1", Parent: "a", Title: "a1"}},
			place("a", "c2", "e"), map[string][]string{"c1": {"b", "b>b1", "b>b2", "s", "c"}, "c2": {"d", "a", "a>a1", "e"}}},
		{"the row moved elsewhere since: the drop wins", []operations.Op{place("a", "c3", "")},
			place("a", "c2", "e"), map[string][]string{"c1": {"b", "b>b1", "b>b2", "s", "c"}, "c2": {"d", "a", "e"}}},
		{"the row nested since: the drop takes it out", []operations.Op{{Action: "task.place", IDs: []string{"a"}, Cat: "c1", Parent: "b"}},
			place("a", "c2", ""), map[string][]string{"c1": {"b", "b>b1", "b>b2", "s", "c"}, "c2": {"d", "e", "a"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := placeBoardRows(t)
			for _, op := range tc.since {
				if _, err := operations.Apply(p, op); err != nil {
					t.Fatalf("since, %s: %v", op.Action, err)
				}
			}
			if _, err := operations.Apply(p, tc.op); err != nil {
				t.Fatal(err)
			}
			got := outline(p)
			delete(got, "c3")
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("outline = %v, want %v", got, tc.want)
			}
			if _, err := operations.Apply(p, tc.op); !errors.Is(err, operations.ErrNoChange) {
				t.Fatalf("sent again = %v, want ErrNoChange", err)
			}
		})
	}
}

func TestPlaceRowsDeletedOrIntoACategoryDeletedElsewhere(t *testing.T) {
	p := placeBoardRows(t)
	if _, err := operations.Apply(p, operations.Op{Action: "task.delete", IDs: []string{"a"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := operations.Apply(p, place("a", "c2", "e")); operations.VerdictOf(err) != operations.Gone {
		t.Fatalf("a row deleted elsewhere: %v", err)
	}
	if _, err := operations.Apply(p, operations.Op{Action: "category.delete", Cat: "c2"}); err != nil {
		t.Fatal(err)
	}
	before := operations.Clone(*p)
	_, err := operations.Apply(p, place("c", "c2", "e"))
	if msg, _ := operations.Explain(err); operations.VerdictOf(err) != operations.Refused || msg != "That category is gone" {
		t.Fatalf("into a category deleted elsewhere: %v", err)
	}
	if !reflect.DeepEqual(before, *p) {
		t.Fatal("a refused place changed the project")
	}
}
