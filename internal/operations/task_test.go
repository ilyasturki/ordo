package operations_test

import (
	"errors"
	"testing"

	"phasionary/internal/domain"
	"phasionary/internal/operations"
)

// projectWithCategory returns a *domain.Project holding one empty category
// with the given id, for driving the pure verbs against a literal.
func projectWithCategory(catID string) *domain.Project {
	return &domain.Project{
		ID:   "p1",
		Name: "Test",
		Categories: []domain.Category{
			{ID: catID, Name: "Feature", Tasks: []domain.Task{}},
		},
	}
}

func TestCreateTask_AppendsWithFields(t *testing.T) {
	p := projectWithCategory("c1")

	got, err := operations.CreateTask(p, "c1", operations.TaskFields{
		Title:       "Write docs",
		Status:      domain.StatusInProgress,
		Priority:    domain.PriorityHigh,
		Estimate:    90,
		Description: "the body",
	})
	if err != nil {
		t.Fatalf("CreateTask returned error: %v", err)
	}

	if got.Title != "Write docs" || got.Status != domain.StatusInProgress ||
		got.Priority != domain.PriorityHigh || got.EstimateMinutes != 90 ||
		got.Description != "the body" {
		t.Fatalf("returned task has wrong fields: %+v", got)
	}
	if got.ID == "" {
		t.Fatal("created task has no ID")
	}
	if n := len(p.Categories[0].Tasks); n != 1 {
		t.Fatalf("category holds %d tasks, want 1", n)
	}
	if stored := p.Categories[0].Tasks[0]; stored.ID != got.ID {
		t.Fatalf("stored task id %q != returned id %q", stored.ID, got.ID)
	}
}

func TestCreateTask_EmptyStatusDefaultsToTodo(t *testing.T) {
	p := projectWithCategory("c1")

	got, err := operations.CreateTask(p, "c1", operations.TaskFields{Title: "x"})
	if err != nil {
		t.Fatalf("CreateTask returned error: %v", err)
	}
	if got.Status != domain.StatusTodo {
		t.Fatalf("status = %q, want %q", got.Status, domain.StatusTodo)
	}
}

// Creating a completed task must set CompletionDate — proof the status went
// through domain.SetStatus rather than a raw field assignment (the CLI bug).
func TestCreateTask_CompletedSetsCompletionDate(t *testing.T) {
	p := projectWithCategory("c1")

	got, err := operations.CreateTask(p, "c1", operations.TaskFields{
		Title:  "done thing",
		Status: domain.StatusCompleted,
	})
	if err != nil {
		t.Fatalf("CreateTask returned error: %v", err)
	}
	if got.CompletionDate == "" {
		t.Fatal("completed task has empty CompletionDate; status was not set via domain.SetStatus")
	}
}

func TestCreateTask_EmptyTitleRejected(t *testing.T) {
	p := projectWithCategory("c1")

	_, err := operations.CreateTask(p, "c1", operations.TaskFields{Title: "   "})
	if !errors.Is(err, operations.ErrTitleRequired) {
		t.Fatalf("err = %v, want ErrTitleRequired", err)
	}
}

func TestCreateTask_UnknownCategory(t *testing.T) {
	p := projectWithCategory("c1")

	_, err := operations.CreateTask(p, "nope", operations.TaskFields{Title: "x"})
	if !errors.Is(err, domain.ErrCategoryNotFound) {
		t.Fatalf("err = %v, want ErrCategoryNotFound", err)
	}
}

func TestCreateTask_InvalidPriorityRejected(t *testing.T) {
	p := projectWithCategory("c1")

	_, err := operations.CreateTask(p, "c1", operations.TaskFields{Title: "x", Priority: "urgent"})
	if err == nil {
		t.Fatal("expected error for invalid priority, got nil")
	}
	if n := len(p.Categories[0].Tasks); n != 0 {
		t.Fatalf("invalid create left %d tasks behind, want 0", n)
	}
}

// The JSON API hands CreateTask a raw estimate int (no ParseEstimate), so the
// verb must reject a negative itself.
func TestCreateTask_NegativeEstimateRejected(t *testing.T) {
	p := projectWithCategory("c1")

	_, err := operations.CreateTask(p, "c1", operations.TaskFields{Title: "x", Estimate: -5})
	if !errors.Is(err, operations.ErrNegativeEstimate) {
		t.Fatalf("err = %v, want ErrNegativeEstimate", err)
	}
	if n := len(p.Categories[0].Tasks); n != 0 {
		t.Fatalf("negative-estimate create left %d tasks behind, want 0", n)
	}
}

func TestCreateTask_AppliesTag(t *testing.T) {
	p := projectWithCategory("c1")

	got, err := operations.CreateTask(p, "c1", operations.TaskFields{
		Title:    "Tagged",
		TagColor: domain.TagCyan,
	})
	if err != nil {
		t.Fatalf("CreateTask returned error: %v", err)
	}
	if got.TagColor != domain.TagCyan {
		t.Fatalf("tag not applied: color=%q", got.TagColor)
	}
}

func TestCreateTask_InvalidTagColorRejected(t *testing.T) {
	p := projectWithCategory("c1")

	_, err := operations.CreateTask(p, "c1", operations.TaskFields{Title: "x", TagColor: "puce"})
	if err == nil {
		t.Fatal("expected error for invalid tag color, got nil")
	}
	if n := len(p.Categories[0].Tasks); n != 0 {
		t.Fatalf("invalid create left %d tasks behind, want 0", n)
	}
}
