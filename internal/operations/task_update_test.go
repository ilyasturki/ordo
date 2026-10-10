package operations

import (
	"errors"
	"testing"

	"phasionary/internal/domain"
)

// projectWithTask builds a one-category project holding a single task, and
// returns the project plus the category/task ids.
func projectWithTask(t *testing.T, title string) (*domain.Project, string, string) {
	t.Helper()
	p, err := domain.NewProject("Test")
	if err != nil {
		t.Fatalf("new project: %v", err)
	}
	cat, err := p.AddCategoryUnder("", "Work")
	if err != nil {
		t.Fatalf("add category: %v", err)
	}
	task, err := CreateTask(&p, cat.ID, TaskFields{Title: title})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	return &p, cat.ID, task.ID
}

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }

func TestUpdateTaskAppliesSetFields(t *testing.T) {
	p, cid, tid := projectWithTask(t, "Original")

	got, err := UpdateTask(p, cid, tid, TaskUpdate{
		Title:       strPtr("  Renamed  "),
		Status:      strPtr(domain.StatusInProgress),
		Priority:    strPtr(domain.PriorityHigh),
		Estimate:    intPtr(45),
		Description: strPtr("notes\n\n"),
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if got.Title != "Renamed" {
		t.Fatalf("title: want trimmed %q, got %q", "Renamed", got.Title)
	}
	if got.Description != "notes" {
		t.Fatalf("description: want trailing newlines trimmed, got %q", got.Description)
	}
	if got.Status != domain.StatusInProgress || got.Priority != domain.PriorityHigh ||
		got.EstimateMinutes != 45 {
		t.Fatalf("fields not applied: %+v", got)
	}
}

func TestUpdateTaskLeavesNilFieldsAlone(t *testing.T) {
	p, cid, tid := projectWithTask(t, "Keep me")
	if _, err := UpdateTask(p, cid, tid, TaskUpdate{Description: strPtr("keep this")}); err != nil {
		t.Fatalf("seed description: %v", err)
	}

	got, err := UpdateTask(p, cid, tid, TaskUpdate{Status: strPtr(domain.StatusCompleted)})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if got.Title != "Keep me" || got.Description != "keep this" {
		t.Fatalf("nil fields were modified: %+v", got)
	}
	if got.Status != domain.StatusCompleted {
		t.Fatalf("status: want completed, got %q", got.Status)
	}
	// SetStatus bookkeeping still runs through the domain setter.
	if got.CompletionDate == "" {
		t.Fatal("completing a task must stamp CompletionDate")
	}
}

func TestUpdateTaskEmptyUpdateReportsNoChange(t *testing.T) {
	p, cid, tid := projectWithTask(t, "Untouched")
	before := p.Categories[0].Tasks[0].UpdatedAt

	got, err := UpdateTask(p, cid, tid, TaskUpdate{})
	if !errors.Is(err, ErrNoChange) {
		t.Fatalf("want ErrNoChange, got %v", err)
	}
	if got.Title != "Untouched" {
		t.Fatalf("want the task returned unchanged, got %+v", got)
	}
	if p.Categories[0].Tasks[0].UpdatedAt != before {
		t.Fatal("an empty update must not bump UpdatedAt")
	}
}

func TestUpdateTaskRejectsInvalidFields(t *testing.T) {
	cases := []struct {
		name   string
		update TaskUpdate
		want   error
	}{
		{"blank title", TaskUpdate{Title: strPtr("   ")}, ErrTitleRequired},
		{"negative estimate", TaskUpdate{Estimate: intPtr(-1)}, ErrNegativeEstimate},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, cid, tid := projectWithTask(t, "Original")
			if _, err := UpdateTask(p, cid, tid, tc.update); !errors.Is(err, tc.want) {
				t.Fatalf("want %v, got %v", tc.want, err)
			}
		})
	}

	t.Run("invalid status", func(t *testing.T) {
		p, cid, tid := projectWithTask(t, "Original")
		if _, err := UpdateTask(p, cid, tid, TaskUpdate{Status: strPtr("nope")}); err == nil {
			t.Fatal("want an error for an invalid status")
		}
	})
	t.Run("invalid priority", func(t *testing.T) {
		p, cid, tid := projectWithTask(t, "Original")
		if _, err := UpdateTask(p, cid, tid, TaskUpdate{Priority: strPtr("urgentish")}); err == nil {
			t.Fatal("want an error for an invalid priority")
		}
	})
}

// A rejected field must not leave the task half-written: the caller saves
// whatever the project holds, so a partial mutation would persist.
func TestUpdateTaskRejectionLeavesTaskUntouched(t *testing.T) {
	p, cid, tid := projectWithTask(t, "Original")

	_, err := UpdateTask(p, cid, tid, TaskUpdate{
		Title:    strPtr("Renamed"),
		Estimate: intPtr(-5),
	})
	if !errors.Is(err, ErrNegativeEstimate) {
		t.Fatalf("want ErrNegativeEstimate, got %v", err)
	}
	if got := p.Categories[0].Tasks[0]; got.Title != "Original" {
		t.Fatalf("title was written before validation failed: %q", got.Title)
	}
}

func TestUpdateTaskNotFound(t *testing.T) {
	p, cid, tid := projectWithTask(t, "Original")

	if _, err := UpdateTask(p, "no-such-category", tid, TaskUpdate{Title: strPtr("x")}); !errors.Is(err, domain.ErrCategoryNotFound) {
		t.Fatalf("want ErrCategoryNotFound, got %v", err)
	}
	if _, err := UpdateTask(p, cid, "no-such-task", TaskUpdate{Title: strPtr("x")}); !errors.Is(err, domain.ErrTaskNotFound) {
		t.Fatalf("want ErrTaskNotFound, got %v", err)
	}
}

func TestDeleteTask(t *testing.T) {
	p, cid, tid := projectWithTask(t, "Doomed")

	got, err := DeleteTask(p, cid, tid)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if got.ID != tid || got.Title != "Doomed" {
		t.Fatalf("want the deleted task returned, got %+v", got)
	}
	if len(p.Categories[0].Tasks) != 0 {
		t.Fatalf("task not removed: %+v", p.Categories[0].Tasks)
	}
}

func TestDeleteTaskLeavesSiblingsInOrder(t *testing.T) {
	p, cid, _ := projectWithTask(t, "first")
	for _, title := range []string{"second", "third"} {
		if _, err := CreateTask(p, cid, TaskFields{Title: title}); err != nil {
			t.Fatalf("create %s: %v", title, err)
		}
	}
	middle := p.Categories[0].Tasks[1].ID

	if _, err := DeleteTask(p, cid, middle); err != nil {
		t.Fatalf("delete: %v", err)
	}
	tasks := p.Categories[0].Tasks
	if len(tasks) != 2 || tasks[0].Title != "first" || tasks[1].Title != "third" {
		t.Fatalf("surviving tasks wrong: %+v", tasks)
	}
}

func TestDeleteTaskNotFound(t *testing.T) {
	p, cid, tid := projectWithTask(t, "Task")

	if _, err := DeleteTask(p, "no-such-category", tid); !errors.Is(err, domain.ErrCategoryNotFound) {
		t.Fatalf("want ErrCategoryNotFound, got %v", err)
	}
	if _, err := DeleteTask(p, cid, "no-such-task"); !errors.Is(err, domain.ErrTaskNotFound) {
		t.Fatalf("want ErrTaskNotFound, got %v", err)
	}
}
