package operations_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/ilyasturki/ordo/internal/domain"
	"github.com/ilyasturki/ordo/internal/operations"
)

func group(value string) operations.Op {
	return operations.Op{Action: "project.group", Value: value}
}

func TestGroupSetsTrimsAndClears(t *testing.T) {
	p := board(t)
	p.UpdatedAt = "2026-01-02T03:04:05Z"
	out, err := operations.Apply(p, group("  Work "))
	if err != nil || p.Group != "Work" || out.Msg != "Group set to “Work”" {
		t.Fatalf("set: group %q, msg %q, err %v", p.Group, out.Msg, err)
	}
	if p.UpdatedAt != "2026-01-02T03:04:05Z" {
		t.Fatalf("a group restamped the project: %s", p.UpdatedAt)
	}
	out, err = operations.Apply(p, group(""))
	if err != nil || p.Group != "" || out.Msg != "Group cleared" {
		t.Fatalf("clear: group %q, msg %q, err %v", p.Group, out.Msg, err)
	}
}

func TestGroupRefusesTwoLines(t *testing.T) {
	p := board(t)
	if _, err := operations.Apply(p, group("Work\nPerso")); err == nil || p.Group != "" {
		t.Fatalf("group %q, err %v", p.Group, err)
	}
}

// A queued group can land twice, or on a file another device regrouped meanwhile.
func TestGroupReplays(t *testing.T) {
	if !operations.Replayable("project.group") {
		t.Fatal("project.group can't wait in a queue")
	}
	p := board(t)
	apply(t, p, group("Work"))
	if _, err := operations.Apply(p, group("Work")); !errors.Is(err, operations.ErrNoChange) {
		t.Fatalf("second landing: %v", err)
	}
	p.Group = "Perso"
	apply(t, p, group("Work"))
	if p.Group != "Work" {
		t.Fatalf("group %q", p.Group)
	}
}

func TestUndoOfGroup(t *testing.T) {
	before := domain.Project{ID: "p1", Name: "Ferry", Group: "Perso"}
	after := before
	after.Group = "Work"
	got := operations.Undo(before, after, group("Work"))
	if len(got) != 1 || !reflect.DeepEqual(got[0], group("Perso")) {
		t.Fatalf("undo: %+v", got)
	}
	p := board(t)
	p.Group = "Work"
	apply(t, p, got...)
	if p.Group != "Perso" {
		t.Fatalf("after undo: %q", p.Group)
	}
	if got := operations.Undo(domain.Project{ID: "p1"}, after, group("Work")); len(got) != 1 || !reflect.DeepEqual(got[0], group("")) {
		t.Fatalf("undo of a first group: %+v", got)
	}
}
