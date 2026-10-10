package operations

import (
	"errors"
	"testing"

	"phasionary/internal/domain"
)

// shape renders the first category as titles, a subtask prefixed with ">".
func shape(p *domain.Project) []string {
	out := make([]string, 0, len(p.Categories[0].Tasks))
	for _, task := range p.Categories[0].Tasks {
		switch {
		case task.IsSeparator():
			out = append(out, "--"+task.Title)
		case task.ParentID != "":
			out = append(out, ">"+task.Title)
		default:
			out = append(out, task.Title)
		}
	}
	return out
}

func TestCreateSubtaskLandsLastUnderItsParent(t *testing.T) {
	p, cid := projectWithTasks(t, "parent", "next")
	parent := p.Categories[0].Tasks[0].ID

	for _, title := range []string{"one", "two"} {
		if _, err := CreateTask(p, cid, TaskFields{Title: title, ParentID: parent}); err != nil {
			t.Fatalf("create %s: %v", title, err)
		}
	}
	if got := shape(p); !equal(got, []string{"parent", ">one", ">two", "next"}) {
		t.Fatalf("shape = %v", got)
	}

	child := p.Categories[0].Tasks[1].ID
	if _, err := CreateTask(p, cid, TaskFields{Title: "deep", ParentID: child}); !errors.Is(err, domain.ErrNestDepth) {
		t.Fatalf("nesting under a subtask: err = %v, want ErrNestDepth", err)
	}
	if _, err := CreateTask(p, cid, TaskFields{Title: "x", ParentID: "ghost"}); !errors.Is(err, domain.ErrTaskNotFound) {
		t.Fatalf("unknown parent: err = %v, want ErrTaskNotFound", err)
	}
	if _, err := CreateTask(p, cid, TaskFields{Kind: domain.KindSeparator, ParentID: parent}); !errors.Is(err, ErrSeparatorFieldNotAllowed) {
		t.Fatalf("nested separator: err = %v, want ErrSeparatorFieldNotAllowed", err)
	}
}

func TestCreateTaskAfterKeepsBlocksWhole(t *testing.T) {
	p, cid := projectWithTasks(t, "parent", "next")
	parent := p.Categories[0].Tasks[0].ID
	if _, err := CreateTask(p, cid, TaskFields{Title: "child", ParentID: parent}); err != nil {
		t.Fatal(err)
	}
	child := p.Categories[0].Tasks[1].ID

	if _, err := CreateTaskAfter(p, cid, TaskFields{Title: "sibling"}, parent); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateTaskAfter(p, cid, TaskFields{Title: "twin"}, child); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateTaskAfter(p, cid, TaskFields{Kind: domain.KindSeparator}, child); err != nil {
		t.Fatal(err)
	}
	want := []string{"parent", ">child", ">twin", "--", "sibling", "next"}
	if got := shape(p); !equal(got, want) {
		t.Fatalf("shape = %v, want %v", got, want)
	}
}

func TestDeleteTaskTakesSubtasks(t *testing.T) {
	p, cid := projectWithTasks(t, "parent", "next")
	parent := p.Categories[0].Tasks[0].ID
	if _, err := CreateTask(p, cid, TaskFields{Title: "child", ParentID: parent}); err != nil {
		t.Fatal(err)
	}
	if _, err := DeleteTask(p, cid, parent); err != nil {
		t.Fatal(err)
	}
	if got := shape(p); !equal(got, []string{"next"}) {
		t.Fatalf("shape = %v", got)
	}
}

func TestMoveTaskWithinCategoryMovesBlocks(t *testing.T) {
	p, cid := projectWithTasks(t, "a", "b")
	a := p.Categories[0].Tasks[0].ID
	if _, err := CreateTask(p, cid, TaskFields{Title: "a1", ParentID: a}); err != nil {
		t.Fatal(err)
	}
	if _, err := MoveTaskWithinCategory(p, cid, a, 1); err != nil {
		t.Fatal(err)
	}
	if got := shape(p); !equal(got, []string{"b", "a", ">a1"}) {
		t.Fatalf("shape = %v", got)
	}
	if _, err := MoveTaskWithinCategory(p, cid, p.Categories[0].Tasks[2].ID, -1); !errors.Is(err, ErrNoChange) {
		t.Fatalf("a lone subtask can't leave its parent: err = %v", err)
	}

	const stamp = "2000-01-01T00:00:00Z"
	p.Categories[0].UpdatedAt = stamp
	if _, err := MoveTaskWithinCategory(p, cid, p.Categories[0].Tasks[0].ID, 2); !errors.Is(err, ErrNoChange) {
		t.Fatalf("a move past the end: err = %v", err)
	}
	if got := shape(p); !equal(got, []string{"b", "a", ">a1"}) || p.Categories[0].UpdatedAt != stamp {
		t.Fatalf("a refused move left shape %v and updated_at %q", got, p.Categories[0].UpdatedAt)
	}
}

func TestDeleteCategoryTakesSubCategories(t *testing.T) {
	p, _ := projectWithTasks(t)
	parent := p.Categories[0].ID
	if _, err := p.AddCategoryUnder(parent, "sub"); err != nil {
		t.Fatal(err)
	}
	if _, err := p.AddCategoryUnder("", "other"); err != nil {
		t.Fatal(err)
	}
	if err := DeleteCategory(p, parent); err != nil {
		t.Fatal(err)
	}
	if len(p.Categories) != 1 || p.Categories[0].Name != "other" {
		t.Fatalf("categories = %+v", p.Categories)
	}
}
