package ordocore_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/ilyasturki/ordo/internal/domain"
	"github.com/ilyasturki/ordo/native/core/ordocore"
)

const webUI = "aaaaaaaa-0000-4000-8000-000000000001"

type answer struct {
	Status string            `json:"status"`
	Msg    string            `json:"msg"`
	Row    string            `json:"row"`
	Queue  map[string]any    `json:"queue"`
	Undo   []json.RawMessage `json:"undo"`
}

func seeded(t *testing.T) (*ordocore.Core, string) {
	t.Helper()
	raw, err := os.ReadFile("../../../testdata/web/seed.json")
	if err != nil {
		t.Fatal(err)
	}
	c := ordocore.New()
	pid, err := c.Open(string(raw))
	if err != nil || pid != "0rd0-web-fixture" {
		t.Fatalf("open: %q %v", pid, err)
	}
	return c, pid
}

func read(t *testing.T, raw string) answer {
	t.Helper()
	var a answer
	if err := json.Unmarshal([]byte(raw), &a); err != nil {
		t.Fatalf("answer %q: %v", raw, err)
	}
	return a
}

func status(t *testing.T, c *ordocore.Core, pid, id string) string {
	t.Helper()
	var p domain.Project
	if err := json.Unmarshal([]byte(c.Project(pid)), &p); err != nil {
		t.Fatal(err)
	}
	for _, cat := range p.Categories {
		for _, task := range cat.Tasks {
			if task.ID == id {
				return task.Status
			}
		}
	}
	t.Fatalf("no task %s", id)
	return ""
}

func TestPerformAnswersTheQueuedOpAndItsUndo(t *testing.T) {
	c, pid := seeded(t)
	a := read(t, c.Perform(pid, `{"action":"status.cycle","ids":["`+webUI+`"]}`))
	if a.Status != "applied" || a.Queue["action"] != "status.set" || a.Queue["value"] != "completed" {
		t.Fatalf("answer = %+v", a)
	}
	if got := status(t, c, pid, webUI); got != "completed" {
		t.Fatalf("status = %s", got)
	}
	if len(a.Undo) != 1 {
		t.Fatalf("undo = %s", a.Undo)
	}
	if back := read(t, c.Replay(pid, string(a.Undo[0]))); back.Status != "applied" {
		t.Fatalf("undo answer = %+v", back)
	}
	if got := status(t, c, pid, webUI); got != "in_progress" {
		t.Fatalf("status after undo = %s", got)
	}
}

func TestReplayRefusesAGesture(t *testing.T) {
	c, pid := seeded(t)
	if a := read(t, c.Replay(pid, `{"action":"status.cycle","ids":["`+webUI+`"]}`)); a.Status != "refused" {
		t.Fatalf("answer = %+v", a)
	}
}

func TestProjectIsEmptyOnceClosed(t *testing.T) {
	c, pid := seeded(t)
	c.Close(pid)
	if raw := c.Project(pid); raw != "" {
		t.Fatalf("project = %q", raw)
	}
	if a := read(t, c.Perform(pid, `{"action":"status.cycle","ids":["`+webUI+`"]}`)); a.Status != "gone" {
		t.Fatalf("answer = %+v", a)
	}
}

func TestOpenRefusesWhatOrdoWouldNotRead(t *testing.T) {
	c := ordocore.New()
	if _, err := c.Open(`{"id":"../escape","name":"x"}`); err == nil {
		t.Fatal("opened a project with an unsafe ID")
	}
	if _, err := c.Open(`{"schema":99,"id":"p1","name":"x"}`); err == nil {
		t.Fatal("opened a project from a newer schema")
	}
}

func TestProjectCreateMintsTheID(t *testing.T) {
	c := ordocore.New()
	a := read(t, c.Perform("", `{"action":"project.create","title":"Phone"}`))
	if a.Status != "applied" || domain.ValidateID(a.Row) != nil || c.Project(a.Row) == "" {
		t.Fatalf("answer = %+v", a)
	}
}
