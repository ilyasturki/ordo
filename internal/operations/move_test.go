package operations_test

import (
	"errors"
	"testing"

	"github.com/ilyasturki/ordo/internal/domain"
	"github.com/ilyasturki/ordo/internal/operations"
)

func projectWithTasks(ids ...string) *domain.Project {
	p := projectWithCategory("c1")
	for _, id := range ids {
		task, _ := domain.NewTask(id)
		task.ID = id
		p.Categories[0].Tasks = append(p.Categories[0].Tasks, task)
	}
	return p
}

func taskOrder(c domain.Category) []string {
	ids := make([]string, len(c.Tasks))
	for i, t := range c.Tasks {
		ids[i] = t.ID
	}
	return ids
}

func equalOrder(a, b []string) bool {
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

func TestMoveTaskWithinCategory_Down(t *testing.T) {
	p := projectWithTasks("t1", "t2", "t3")

	got, err := operations.MoveTaskWithinCategory(p, "c1", "t1", 1)
	if err != nil {
		t.Fatalf("MoveTaskWithinCategory: %v", err)
	}
	if got.ID != "t1" {
		t.Fatalf("returned task id = %q, want t1", got.ID)
	}
	if order := taskOrder(p.Categories[0]); !equalOrder(order, []string{"t2", "t1", "t3"}) {
		t.Fatalf("order = %v, want [t2 t1 t3]", order)
	}
}

func TestMoveTaskWithinCategory_UpAtTopIsNoChange(t *testing.T) {
	p := projectWithTasks("t1", "t2", "t3")

	_, err := operations.MoveTaskWithinCategory(p, "c1", "t1", -1)
	if !errors.Is(err, operations.ErrNoChange) {
		t.Fatalf("err = %v, want ErrNoChange", err)
	}
	if order := taskOrder(p.Categories[0]); !equalOrder(order, []string{"t1", "t2", "t3"}) {
		t.Fatalf("order changed on boundary no-op: %v", order)
	}
}

func TestMoveTaskWithinCategory_DownAtBottomIsNoChange(t *testing.T) {
	p := projectWithTasks("t1", "t2", "t3")

	_, err := operations.MoveTaskWithinCategory(p, "c1", "t3", 1)
	if !errors.Is(err, operations.ErrNoChange) {
		t.Fatalf("err = %v, want ErrNoChange", err)
	}
}

func TestMoveTaskWithinCategory_UnknownTask(t *testing.T) {
	p := projectWithTasks("t1")

	_, err := operations.MoveTaskWithinCategory(p, "c1", "nope", 1)
	if !errors.Is(err, domain.ErrTaskNotFound) {
		t.Fatalf("err = %v, want ErrTaskNotFound", err)
	}
}

func TestMoveTaskToCategory(t *testing.T) {
	p := &domain.Project{
		ID: "p1",
		Categories: []domain.Category{
			{ID: "c1", Name: "A", Tasks: []domain.Task{mustTask("t1")}},
			{ID: "c2", Name: "B", Tasks: []domain.Task{}},
		},
	}

	got, err := operations.MoveTaskToCategory(p, "c1", "t1", "c2")
	if err != nil {
		t.Fatalf("MoveTaskToCategory: %v", err)
	}
	if got.ID != "t1" {
		t.Fatalf("returned task id = %q, want t1", got.ID)
	}
	if len(p.Categories[0].Tasks) != 0 {
		t.Fatalf("source still holds %d tasks, want 0", len(p.Categories[0].Tasks))
	}
	if order := taskOrder(p.Categories[1]); !equalOrder(order, []string{"t1"}) {
		t.Fatalf("destination order = %v, want [t1]", order)
	}
}

func TestMoveTaskToCategory_SameCategoryErrors(t *testing.T) {
	p := projectWithTasks("t1")

	_, err := operations.MoveTaskToCategory(p, "c1", "t1", "c1")
	if err == nil {
		t.Fatal("expected error moving task to its own category")
	}
}

func mustTask(id string) domain.Task {
	task, _ := domain.NewTask(id)
	task.ID = id
	return task
}
