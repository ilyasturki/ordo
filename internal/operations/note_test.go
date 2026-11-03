package operations

import (
	"errors"
	"testing"

	"github.com/ilyasturki/ordo/internal/domain"
)

func TestNoteAddLandsBelowTheBlock(t *testing.T) {
	p, cid := projectWithTasks(t, "first", "second")
	parent := p.Categories[0].Tasks[0].ID
	sub, err := CreateTask(p, cid, TaskFields{Title: "sub", ParentID: parent})
	if err != nil {
		t.Fatal(err)
	}

	out, err := Apply(p, Op{Action: "note.add", New: "n1", Cat: cid, IDs: []string{sub.ID}, Title: "  ask Sam  "})
	if err != nil {
		t.Fatal(err)
	}
	if out.Msg != "Added a note" || out.Row != "n1" {
		t.Fatalf("outcome %+v", out)
	}
	note := p.Categories[0].Tasks[2]
	if !note.IsNote() || note.Title != "ask Sam" || note.ParentID != "" || note.Status != "" {
		t.Fatalf("a note below a subtask goes after its parent's block, top-level and statusless: %+v", note)
	}
}

func TestNoteNeedsText(t *testing.T) {
	p, cid := projectWithTasks(t, "first")
	if _, err := Apply(p, Op{Action: "note.add", New: "n1", Cat: cid, Title: "  "}); !errors.Is(err, ErrTitleRequired) {
		t.Fatalf("empty note.add = %v, want ErrTitleRequired", err)
	}
	note, err := CreateTaskAtTop(p, cid, TaskFields{Kind: domain.KindNote, Title: "text"})
	if err != nil {
		t.Fatal(err)
	}
	empty := " "
	if _, err := UpdateTask(p, cid, note.ID, TaskUpdate{Title: &empty}); !errors.Is(err, ErrTitleRequired) {
		t.Fatalf("emptying a note = %v, want ErrTitleRequired", err)
	}
	if got := p.Categories[0].Tasks[0].Title; got != "text" {
		t.Fatalf("a refused edit changed the note to %q", got)
	}
}

func TestNoteRefusesTaskFields(t *testing.T) {
	p, cid := projectWithTasks(t, "first")
	if _, err := CreateTask(p, cid, TaskFields{Kind: domain.KindNote, Title: "n", Status: domain.StatusTodo}); !errors.Is(err, ErrSeparatorFieldNotAllowed) {
		t.Fatalf("a note with a status = %v", err)
	}
	note, err := CreateTask(p, cid, TaskFields{Kind: domain.KindNote, Title: "n"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(p, Op{Action: "status.set", IDs: []string{note.ID}, Value: domain.StatusCompleted}); !errors.Is(err, ErrNoChange) {
		t.Fatalf("status.set on a note = %v, want ErrNoChange", err)
	}
	if _, err := Apply(p, Op{Action: "priority.set", IDs: []string{note.ID}, Value: domain.PriorityHigh}); !errors.Is(err, ErrNoChange) {
		t.Fatalf("priority.set on a note = %v, want ErrNoChange", err)
	}
}

func TestNoteDoesntNest(t *testing.T) {
	p, cid := projectWithTasks(t, "first")
	note, err := CreateTask(p, cid, TaskFields{Kind: domain.KindNote, Title: "n"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NestTasks(p, []string{note.ID}, true, AllShown); err == nil || err.Error() != "A note doesn't nest" {
		t.Fatalf("nesting a note = %v", err)
	}
	task, err := CreateTask(p, cid, TaskFields{Title: "below"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := MoveRows(p, []string{task.ID}, cid, 0, note.ID); err == nil || err.Error() != "A note doesn't nest" {
		t.Fatalf("placing under a note = %v", err)
	}
}

func TestDeleteNotesSaysNotes(t *testing.T) {
	for _, tc := range []struct {
		kinds []string
		want  string
	}{
		{[]string{domain.KindNote}, "Note deleted"},
		{[]string{domain.KindNote, domain.KindNote}, "2 notes deleted"},
		{[]string{domain.KindNote, domain.KindSeparator}, "2 rows deleted"},
		{[]string{domain.KindSeparator}, "Separator deleted"},
		{[]string{domain.KindNote, ""}, "1 task deleted"},
	} {
		p, cid := projectWithTasks(t)
		var ids []string
		for _, kind := range tc.kinds {
			row, err := CreateTask(p, cid, TaskFields{Kind: kind, Title: "t"})
			if err != nil {
				t.Fatal(err)
			}
			ids = append(ids, row.ID)
		}
		out, err := Apply(p, Op{Action: "task.delete", IDs: ids})
		if err != nil {
			t.Fatal(err)
		}
		if out.Msg != tc.want {
			t.Errorf("deleting %v says %q, want %q", tc.kinds, out.Msg, tc.want)
		}
	}
}

func TestRestoreChecksANote(t *testing.T) {
	if err := checkRow(domain.Task{ID: "n", Kind: domain.KindNote, Title: "back"}); err != nil {
		t.Fatalf("a note row: %v", err)
	}
	if err := checkRow(domain.Task{ID: "n", Kind: domain.KindNote}); !errors.Is(err, ErrTitleRequired) {
		t.Fatalf("an empty note row = %v", err)
	}
	if err := checkRow(domain.Task{ID: "n", Kind: "sticker", Title: "x"}); !errors.Is(err, ErrInvalidKind) {
		t.Fatalf("an unknown kind = %v", err)
	}
}
