package operations_test

import (
	"errors"
	"testing"

	"github.com/ilyasturki/ordo/internal/domain"
	"github.com/ilyasturki/ordo/internal/operations"
)

// projectWithTask returns a project holding one todo task "t1" in category
// "c1", plus those ids.
func projectWithTask() *domain.Project {
	p := projectWithCategory("c1")
	task, _ := domain.NewTask("existing")
	task.ID = "t1"
	p.Categories[0].Tasks = append(p.Categories[0].Tasks, task)
	return p
}

func TestSetTaskStatus_CompletedThenReopen(t *testing.T) {
	p := projectWithTask()

	got, err := operations.SetTaskStatus(p, "c1", "t1", domain.StatusCompleted)
	if err != nil {
		t.Fatalf("SetTaskStatus completed: %v", err)
	}
	if got.CompletionDate == "" {
		t.Fatal("completed task has empty CompletionDate")
	}

	got, err = operations.SetTaskStatus(p, "c1", "t1", domain.StatusTodo)
	if err != nil {
		t.Fatalf("SetTaskStatus reopen: %v", err)
	}
	if got.CompletionDate != "" {
		t.Fatalf("reopened task still has CompletionDate %q", got.CompletionDate)
	}
}

func TestSetTaskStatus_Invalid(t *testing.T) {
	p := projectWithTask()

	_, err := operations.SetTaskStatus(p, "c1", "t1", "bogus")
	if err == nil {
		t.Fatal("expected error for invalid status")
	}
	if s := p.Categories[0].Tasks[0].Status; s != domain.StatusTodo {
		t.Fatalf("status mutated to %q despite invalid input", s)
	}
}

func TestSetTaskStatus_UnknownTask(t *testing.T) {
	p := projectWithTask()

	_, err := operations.SetTaskStatus(p, "c1", "nope", domain.StatusCompleted)
	if !errors.Is(err, domain.ErrTaskNotFound) {
		t.Fatalf("err = %v, want ErrTaskNotFound", err)
	}
}

func TestSetTaskStatus_UnknownCategory(t *testing.T) {
	p := projectWithTask()

	_, err := operations.SetTaskStatus(p, "nope", "t1", domain.StatusCompleted)
	if !errors.Is(err, domain.ErrCategoryNotFound) {
		t.Fatalf("err = %v, want ErrCategoryNotFound", err)
	}
}

func TestSetTaskPriority(t *testing.T) {
	p := projectWithTask()

	got, err := operations.SetTaskPriority(p, "c1", "t1", domain.PriorityHigh)
	if err != nil {
		t.Fatalf("SetTaskPriority: %v", err)
	}
	if got.Priority != domain.PriorityHigh {
		t.Fatalf("returned priority = %q, want high", got.Priority)
	}
	if stored := p.Categories[0].Tasks[0].Priority; stored != domain.PriorityHigh {
		t.Fatalf("stored priority = %q, want high", stored)
	}
}

func TestSetTaskPriority_Invalid(t *testing.T) {
	p := projectWithTask()

	_, err := operations.SetTaskPriority(p, "c1", "t1", "urgent")
	if err == nil {
		t.Fatal("expected error for invalid priority")
	}
}

func TestSetTaskEstimate(t *testing.T) {
	p := projectWithTask()

	got, err := operations.SetTaskEstimate(p, "c1", "t1", 120)
	if err != nil {
		t.Fatalf("SetTaskEstimate: %v", err)
	}
	if got.EstimateMinutes != 120 {
		t.Fatalf("returned estimate = %d, want 120", got.EstimateMinutes)
	}
}

func TestSetTaskEstimate_NegativeRejected(t *testing.T) {
	p := projectWithTask()

	_, err := operations.SetTaskEstimate(p, "c1", "t1", -5)
	if !errors.Is(err, operations.ErrNegativeEstimate) {
		t.Fatalf("err = %v, want ErrNegativeEstimate", err)
	}
}
