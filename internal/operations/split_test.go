package operations

import (
	"errors"
	"testing"

	"github.com/ilyasturki/ordo/internal/domain"
)

func TestSplitTitleMovesTheRestToTheTopOfTheDescription(t *testing.T) {
	p, cid, tid := projectWithTask(t, "Parse the timetable it fails on DST")
	if _, err := UpdateTask(p, cid, tid, TaskUpdate{Description: strPtr("older notes")}); err != nil {
		t.Fatal(err)
	}

	got, err := SplitTitle(p, tid, "Parse the timetable ", " it fails on DST")
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "Parse the timetable" {
		t.Errorf("title = %q", got.Title)
	}
	if want := "it fails on DST\n\nolder notes"; got.Description != want {
		t.Errorf("description = %q, want %q", got.Description, want)
	}
}

func TestSplitTitleWithNothingAfterKeepsTheDescription(t *testing.T) {
	p, cid, tid := projectWithTask(t, "Ship it")
	if _, err := UpdateTask(p, cid, tid, TaskUpdate{Description: strPtr("notes")}); err != nil {
		t.Fatal(err)
	}

	got, err := SplitTitle(p, tid, "Ship it now", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "Ship it now" || got.Description != "notes" {
		t.Errorf("got %q / %q", got.Title, got.Description)
	}
}

func TestSplitTitleRefusesAnEmptyTitle(t *testing.T) {
	p, _, tid := projectWithTask(t, "Keep")

	if _, err := SplitTitle(p, tid, "  ", "everything"); !errors.Is(err, ErrTitleRequired) {
		t.Fatalf("err = %v, want ErrTitleRequired", err)
	}
	if task := p.Categories[0].Tasks[0]; task.Title != "Keep" || task.Description != "" {
		t.Errorf("a refused split changed the task: %q / %q", task.Title, task.Description)
	}
}

func TestSplitTitleRefusesALine(t *testing.T) {
	p, _, _ := projectWithTask(t, "Task")
	note, err := domain.NewLine(domain.KindNote, "a note")
	if err != nil {
		t.Fatal(err)
	}
	p.Categories[0].InsertTask(1, note)

	if _, err := SplitTitle(p, note.ID, "a", "note"); !errors.Is(err, domain.ErrTaskNotFound) {
		t.Fatalf("err = %v, want ErrTaskNotFound", err)
	}
}
