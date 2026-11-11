package core_test

import (
	"bytes"
	"encoding/json"
	"slices"
	"testing"

	"github.com/ilyasturki/ordo/internal/core"
	"github.com/ilyasturki/ordo/internal/domain"
	"github.com/ilyasturki/ordo/internal/operations"
)

func opened(t *testing.T) (*core.Session, string) {
	t.Helper()
	p := domain.Project{ID: "p1", Name: "Phone", Categories: []domain.Category{{ID: "c1", Name: "Feature"}}}
	for _, id := range []string{"a", "b"} {
		if _, err := operations.CreateTask(&p, "c1", operations.TaskFields{Title: id, ID: id}); err != nil {
			t.Fatal(err)
		}
	}
	raw, _ := json.Marshal(p)
	s := core.New()
	pid, err := s.Open(raw)
	if err != nil || pid != "p1" {
		t.Fatalf("open: %q %v", pid, err)
	}
	return s, pid
}

func op(v operations.Op) []byte {
	raw, _ := json.Marshal(v)
	return raw
}

func held(t *testing.T, s *core.Session, pid string) domain.Project {
	t.Helper()
	var p domain.Project
	if err := json.Unmarshal(s.Project(pid), &p); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestACategoryDragQueuesItselfAndItsUndo(t *testing.T) {
	s, pid := opened(t)
	order := func() []string {
		var ids []string
		for _, c := range held(t, s, pid).Categories {
			ids = append(ids, c.ID)
		}
		return ids
	}
	if a := s.Perform(pid, op(operations.Op{Action: "category.add", New: "c2", Title: "Fix"})); a.Status != operations.Applied {
		t.Fatalf("add = %+v", a)
	}
	a := s.Perform(pid, op(operations.Op{Action: "category.place", Cat: "c2", Before: "c1"}))
	if a.Status != operations.Applied || a.Queue == nil || a.Queue.Action != "category.place" || len(a.Undo) != 1 {
		t.Fatalf("answer = %+v", a)
	}
	if got, want := order(), []string{"c2", "c1"}; !slices.Equal(got, want) {
		t.Fatalf("order = %v, want %v", got, want)
	}
	if r := s.Replay(pid, op(*a.Queue)); r.Status != operations.Unchanged {
		t.Fatalf("landing twice = %+v", r)
	}
	if u := s.Perform(pid, op(a.Undo[0])); u.Status != operations.Applied {
		t.Fatalf("undo = %+v", u)
	}
	if got, want := order(), []string{"c1", "c2"}; !slices.Equal(got, want) {
		t.Fatalf("after undo = %v, want %v", got, want)
	}
}

func TestPerformQueuesWhatTheServerReplays(t *testing.T) {
	s, pid := opened(t)
	a := s.Perform(pid, op(operations.Op{Action: "status.cycle", IDs: []string{"a"}}))
	if a.Status != operations.Applied || a.Msg != "Marked in progress" {
		t.Fatalf("answer = %+v", a)
	}
	if a.Queue == nil || a.Queue.Action != "status.set" || a.Queue.Value != domain.StatusInProgress {
		t.Fatalf("queue = %+v", a.Queue)
	}
	if got := held(t, s, pid).Categories[0].Tasks[0].Status; got != domain.StatusInProgress {
		t.Fatalf("held status = %s", got)
	}
	if again := s.Replay(pid, op(*a.Queue)); again.Status != operations.Unchanged || again.Queue != nil {
		t.Fatalf("replay on the phone's own copy = %+v", again)
	}
}

func TestAnswersCarryWhatTheAppShows(t *testing.T) {
	s, pid := opened(t)
	add := s.Perform(pid, op(operations.Op{Action: "task.add", Cat: "c1", Title: "new"}))
	if add.Status != operations.Applied || add.Row == "" || add.Queue.New != add.Row {
		t.Fatalf("add = %+v", add)
	}
	stale := s.Perform(pid, op(operations.Op{Action: "task.title", IDs: []string{"a"}, Title: "mine", Base: "old"}))
	if stale.Status != operations.Conflicts || stale.Theirs == nil || *stale.Theirs != "a" || stale.Queue != nil {
		t.Fatalf("stale = %+v", stale)
	}
	if r := s.Replay(pid, op(operations.Op{Action: "task.shift", IDs: []string{"a"}})); r.Status != operations.Refused {
		t.Fatalf("replaying a gesture = %+v", r)
	}
	if r := s.Perform("nope", op(operations.Op{Action: "status.cycle"})); r.Status != operations.Gone {
		t.Fatalf("closed project = %+v", r)
	}
	if r := s.Perform(pid, []byte("{")); r.Status != operations.Refused {
		t.Fatalf("bad op = %+v", r)
	}
}

func TestOpenReadsLikeTheStore(t *testing.T) {
	s := core.New()
	if _, err := s.Open([]byte(`{"schema": 99, "id": "p1", "name": "x", "categories": []}`)); err == nil {
		t.Fatal("opened a project from a newer ordo")
	}
	if _, err := s.Open([]byte(`{"id": "../x", "name": "x", "categories": []}`)); err == nil {
		t.Fatal("opened a project with a bad ID")
	}
	pid, err := s.Open([]byte(`{"id": "p1", "name": "x", "categories": [{"id": "c1", "name": "C", "tasks": [{"id": "t1", "title": "a\u0007b", "status": "todo"}]}]}`))
	if err != nil {
		t.Fatal(err)
	}
	var p domain.Project
	json.Unmarshal(s.Project(pid), &p)
	if p.Categories[0].Tasks[0].Title != "ab" {
		t.Fatalf("title = %q", p.Categories[0].Tasks[0].Title)
	}
	pid, err = s.Open([]byte(`{"id": "p2", "name": "y", "categories": [{"id": "c1", "name": "C", "tasks": [{"id": "t1", "title": "a", "status": "todo", "tag_color": "blue-diamond"}]}]}`))
	if err != nil {
		t.Fatal(err)
	}
	json.Unmarshal(s.Project(pid), &p)
	if p.Categories[0].Tasks[0].TagColor != domain.TagBlue {
		t.Fatalf("tag = %q", p.Categories[0].Tasks[0].TagColor)
	}
	pid, err = s.Open([]byte(`{"id": "p3", "name": "z", "categories": [{"id": "c1", "name": "C", "tasks": [{"id": "t1", "title": "a", "status": "todo", "priority": "medium"}]}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(s.Project(pid), []byte(`"medium"`)) {
		t.Fatalf("project = %s", s.Project(pid))
	}
}

func TestPerformAnswersHowToUndoIt(t *testing.T) {
	s, pid := opened(t)
	a := s.Perform(pid, op(operations.Op{Action: "task.delete", IDs: []string{"a"}}))
	if len(a.Undo) != 1 || a.Undo[0].Action != "task.restore" || len(a.Undo[0].Rows) != 1 || a.Undo[0].Before != "b" {
		t.Fatalf("undo = %+v", a.Undo)
	}
	back := s.Perform(pid, op(a.Undo[0]))
	if back.Status != operations.Applied || back.Queue == nil || back.Queue.Action != "task.restore" {
		t.Fatalf("undoing = %+v", back)
	}
	if tasks := held(t, s, pid).Categories[0].Tasks; len(tasks) != 2 || tasks[0].ID != "a" {
		t.Fatalf("tasks = %+v", tasks)
	}
	if r := s.Perform(pid, op(a.Undo[0])); r.Status != operations.Unchanged || r.Msg != "" || r.Undo != nil {
		t.Fatalf("undoing twice = %+v", r)
	}
}

func TestAGroupQueuesItselfAndItsUndo(t *testing.T) {
	s, pid := opened(t)
	a := s.Perform(pid, op(operations.Op{Action: "project.group", Value: " Work "}))
	if a.Status != operations.Applied || a.Queue == nil || a.Queue.Action != "project.group" || a.Queue.Value != " Work " {
		t.Fatalf("group = %+v", a)
	}
	if g := held(t, s, pid).Group; g != "Work" {
		t.Fatalf("group %q", g)
	}
	if len(a.Undo) != 1 || a.Undo[0].Action != "project.group" || a.Undo[0].Value != "" {
		t.Fatalf("undo = %+v", a.Undo)
	}
	if r := s.Replay(pid, op(*a.Queue)); r.Status != operations.Unchanged {
		t.Fatalf("replay = %+v", r)
	}
}

func TestSessionCreatesRenamesAndDeletesProjects(t *testing.T) {
	s, _ := opened(t)
	created := s.Perform("", op(operations.Op{Action: "project.create", Title: " Laptop "}))
	if created.Status != operations.Applied || created.Row == "" || created.Queue == nil || created.Queue.Action != "project.create" {
		t.Fatalf("create = %+v", created)
	}
	id := created.Row
	if p := held(t, s, id); p.ID != id || p.Name != "Laptop" || p.Categories == nil || len(p.Categories) != 0 {
		t.Fatalf("created %+v", p)
	}
	if len(created.Undo) != 1 || created.Undo[0].Action != "project.delete" {
		t.Fatalf("create's undo = %+v", created.Undo)
	}
	if r := s.Replay(id, op(*created.Queue)); r.Status != operations.Unchanged {
		t.Fatalf("replayed create = %+v", r)
	}
	if r := s.Perform("p2", op(operations.Op{Action: "project.create", Title: "phone"})); r.Status != operations.Refused || r.Msg != "A project with that name already exists" {
		t.Fatalf("duplicate create = %+v", r)
	}
	if r := s.Perform("p2", op(operations.Op{Action: "project.create", Title: " "})); r.Status != operations.Refused || r.Msg != "A project needs a name" {
		t.Fatalf("nameless create = %+v", r)
	}
	if r := s.Perform(id, op(operations.Op{Action: "category.add", Title: "Feature"})); r.Status != operations.Applied {
		t.Fatalf("an edit in the new project = %+v", r)
	}

	renamed := s.Perform(id, op(operations.Op{Action: "project.rename", Title: "Desk"}))
	if renamed.Status != operations.Applied || held(t, s, id).Name != "Desk" {
		t.Fatalf("rename = %+v", renamed)
	}
	if len(renamed.Undo) != 1 || renamed.Undo[0].Action != "project.rename" || renamed.Undo[0].Title != "Laptop" {
		t.Fatalf("rename's undo = %+v", renamed.Undo)
	}
	for _, tc := range []struct {
		pid   string
		title string
		want  operations.Verdict
	}{{id, "Desk", operations.Unchanged}, {id, "PHONE", operations.Refused}, {"nope", "x", operations.Gone}} {
		if r := s.Replay(tc.pid, op(operations.Op{Action: "project.rename", Title: tc.title})); r.Status != tc.want {
			t.Fatalf("rename %s to %q = %+v", tc.pid, tc.title, r)
		}
	}

	deleted := s.Perform(id, op(operations.Op{Action: "project.delete"}))
	if deleted.Status != operations.Applied || deleted.Undo != nil || s.Project(id) != nil {
		t.Fatalf("delete = %+v", deleted)
	}
	if r := s.Replay(id, op(operations.Op{Action: "project.delete"})); r.Status != operations.Unchanged {
		t.Fatalf("replayed delete = %+v", r)
	}
}
