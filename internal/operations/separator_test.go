package operations

import (
	"errors"
	"testing"

	"phasionary/internal/domain"
)

// projectWithTasks builds a one-category project holding the named tasks and
// returns the project plus the category id.
func projectWithTasks(t *testing.T, titles ...string) (*domain.Project, string) {
	t.Helper()
	p, err := domain.NewProject("Test")
	if err != nil {
		t.Fatalf("new project: %v", err)
	}
	cat, err := p.AddCategoryUnder("", "Work")
	if err != nil {
		t.Fatalf("add category: %v", err)
	}
	for _, title := range titles {
		if _, err := CreateTask(&p, cat.ID, TaskFields{Title: title}); err != nil {
			t.Fatalf("create %s: %v", title, err)
		}
	}
	return &p, cat.ID
}

// titles renders a category as a slice of titles, with separators shown as
// "--" (or "--label") so ordering assertions read clearly.
func titles(p *domain.Project) []string {
	out := make([]string, 0, len(p.Categories[0].Tasks))
	for _, task := range p.Categories[0].Tasks {
		if task.IsSeparator() {
			out = append(out, "--"+task.Title)
			continue
		}
		out = append(out, task.Title)
	}
	return out
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestCreateTaskAfterInsertsBelowAnchor(t *testing.T) {
	p, cid := projectWithTasks(t, "first", "second", "third")
	anchor := p.Categories[0].Tasks[0].ID

	if _, err := CreateTaskAfter(p, cid, TaskFields{Title: "inserted"}, anchor); err != nil {
		t.Fatalf("insert: %v", err)
	}

	want := []string{"first", "inserted", "second", "third"}
	if got := titles(p); !equal(got, want) {
		t.Fatalf("order: want %v, got %v", want, got)
	}
}

func TestCreateTaskAfterEmptyAnchorAppends(t *testing.T) {
	p, cid := projectWithTasks(t, "first", "second")

	if _, err := CreateTaskAfter(p, cid, TaskFields{Title: "last"}, ""); err != nil {
		t.Fatalf("append: %v", err)
	}

	want := []string{"first", "second", "last"}
	if got := titles(p); !equal(got, want) {
		t.Fatalf("order: want %v, got %v", want, got)
	}
}

func TestCreateTaskAfterUnknownAnchorFails(t *testing.T) {
	p, cid := projectWithTasks(t, "first")

	_, err := CreateTaskAfter(p, cid, TaskFields{Title: "x"}, "ghost")
	if !errors.Is(err, domain.ErrTaskNotFound) {
		t.Fatalf("want ErrTaskNotFound, got %v", err)
	}
	// The failed insert must not have added anything.
	if got := titles(p); !equal(got, []string{"first"}) {
		t.Fatalf("category was modified: %v", got)
	}
}

func TestCreateSeparatorBelowAnchor(t *testing.T) {
	p, cid := projectWithTasks(t, "first", "second")
	anchor := p.Categories[0].Tasks[0].ID

	sep, err := CreateTaskAfter(p, cid,
		TaskFields{Kind: domain.KindSeparator, Title: "later"}, anchor)
	if err != nil {
		t.Fatalf("insert separator: %v", err)
	}
	if !sep.IsSeparator() {
		t.Fatalf("want a separator, got kind %q", sep.Kind)
	}
	// A separator must not pick up the task defaults.
	if sep.Status != "" {
		t.Fatalf("separator carries a status: %q", sep.Status)
	}

	want := []string{"first", "--later", "second"}
	if got := titles(p); !equal(got, want) {
		t.Fatalf("order: want %v, got %v", want, got)
	}
}

func TestCreateSeparatorAllowsEmptyLabel(t *testing.T) {
	p, cid := projectWithTasks(t, "first")

	sep, err := CreateTaskAfter(p, cid, TaskFields{Kind: domain.KindSeparator}, "")
	if err != nil {
		t.Fatalf("unlabeled separator must be allowed: %v", err)
	}
	if sep.Title != "" {
		t.Fatalf("want empty label, got %q", sep.Title)
	}
}

func TestCreateSeparatorRejectsTaskFields(t *testing.T) {
	p, cid := projectWithTasks(t, "first")

	_, err := CreateTaskAfter(p, cid, TaskFields{
		Kind:   domain.KindSeparator,
		Status: domain.StatusTodo,
	}, "")
	if !errors.Is(err, ErrSeparatorFieldNotAllowed) {
		t.Fatalf("want ErrSeparatorFieldNotAllowed, got %v", err)
	}
}

func TestCreateTaskRejectsUnknownKind(t *testing.T) {
	p, cid := projectWithTasks(t, "first")

	_, err := CreateTaskAfter(p, cid, TaskFields{Title: "x", Kind: "milestone"}, "")
	if !errors.Is(err, ErrInvalidKind) {
		t.Fatalf("want ErrInvalidKind, got %v", err)
	}
}

// The core of the guard: a separator must never absorb task fields, or the TUI
// would render a divider that secretly carries a status and an estimate.
func TestUpdateTaskRejectsTaskFieldsOnSeparator(t *testing.T) {
	status := domain.StatusCompleted
	priority := domain.PriorityHigh
	estimate := 30
	description := "notes"

	cases := []struct {
		name   string
		update TaskUpdate
	}{
		{"status", TaskUpdate{Status: &status}},
		{"priority", TaskUpdate{Priority: &priority}},
		{"estimate", TaskUpdate{Estimate: &estimate}},
		{"description", TaskUpdate{Description: &description}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, cid := projectWithTasks(t)
			sep, err := CreateTaskAfter(p, cid, TaskFields{Kind: domain.KindSeparator}, "")
			if err != nil {
				t.Fatalf("seed separator: %v", err)
			}

			if _, err := UpdateTask(p, cid, sep.ID, tc.update); !errors.Is(err, ErrSeparatorFieldNotAllowed) {
				t.Fatalf("want ErrSeparatorFieldNotAllowed, got %v", err)
			}
			stored := p.Categories[0].Tasks[0]
			if stored.Status != "" || stored.Priority != "" ||
				stored.EstimateMinutes != 0 || stored.Description != "" {
				t.Fatalf("separator was mutated: %+v", stored)
			}
		})
	}
}

func TestUpdateSeparatorSetsLabel(t *testing.T) {
	p, cid := projectWithTasks(t)
	sep, err := CreateTaskAfter(p, cid, TaskFields{Kind: domain.KindSeparator}, "")
	if err != nil {
		t.Fatalf("seed separator: %v", err)
	}

	label := "  Later  "
	got, err := UpdateTask(p, cid, sep.ID, TaskUpdate{Title: &label})
	if err != nil {
		t.Fatalf("relabel: %v", err)
	}
	if got.Title != "Later" {
		t.Fatalf("want trimmed label, got %q", got.Title)
	}
	if !got.IsSeparator() {
		t.Fatal("relabeling must not change the kind")
	}
}

// Unlike a task, a separator may have an empty label — that's how it goes back
// to being a plain rule.
func TestUpdateSeparatorAllowsClearingLabel(t *testing.T) {
	p, cid := projectWithTasks(t)
	sep, err := CreateTaskAfter(p, cid,
		TaskFields{Kind: domain.KindSeparator, Title: "Later"}, "")
	if err != nil {
		t.Fatalf("seed separator: %v", err)
	}

	blank := ""
	got, err := UpdateTask(p, cid, sep.ID, TaskUpdate{Title: &blank})
	if err != nil {
		t.Fatalf("clear label: %v", err)
	}
	if got.Title != "" {
		t.Fatalf("want empty label, got %q", got.Title)
	}
}

func TestUpdateSeparatorReportsNoChange(t *testing.T) {
	p, cid := projectWithTasks(t)
	sep, err := CreateTaskAfter(p, cid,
		TaskFields{Kind: domain.KindSeparator, Title: "Later"}, "")
	if err != nil {
		t.Fatalf("seed separator: %v", err)
	}

	if _, err := UpdateTask(p, cid, sep.ID, TaskUpdate{}); !errors.Is(err, ErrNoChange) {
		t.Fatalf("empty update: want ErrNoChange, got %v", err)
	}
	same := "Later"
	if _, err := UpdateTask(p, cid, sep.ID, TaskUpdate{Title: &same}); !errors.Is(err, ErrNoChange) {
		t.Fatalf("same label: want ErrNoChange, got %v", err)
	}
}

func TestDeleteSeparator(t *testing.T) {
	p, cid := projectWithTasks(t, "first", "second")
	anchor := p.Categories[0].Tasks[0].ID
	sep, err := CreateTaskAfter(p, cid, TaskFields{Kind: domain.KindSeparator}, anchor)
	if err != nil {
		t.Fatalf("seed separator: %v", err)
	}

	if _, err := DeleteTask(p, cid, sep.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	want := []string{"first", "second"}
	if got := titles(p); !equal(got, want) {
		t.Fatalf("order: want %v, got %v", want, got)
	}
}

// Separators are dividers, not work: they must stay out of every tally.
func TestSeparatorsAreExcludedFromCounts(t *testing.T) {
	p, cid := projectWithTasks(t, "first", "second")
	if _, err := CreateTaskAfter(p, cid, TaskFields{Kind: domain.KindSeparator}, ""); err != nil {
		t.Fatalf("seed separator: %v", err)
	}

	cat := &p.Categories[0]
	if got := cat.TaskCount(); got != 2 {
		t.Fatalf("TaskCount: want 2, got %d", got)
	}
	if got := cat.StatusCounts().Total(); got != 2 {
		t.Fatalf("StatusCounts total: want 2, got %d", got)
	}
}
