package core_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/ilyasturki/ordo/internal/core"
	"github.com/ilyasturki/ordo/internal/domain"
	"github.com/ilyasturki/ordo/internal/operations"
)

func TestExportStampsTheSchemaAFileNeeds(t *testing.T) {
	s, pid := opened(t)
	if s.Export("gone") != nil {
		t.Fatal("exported a project it doesn't hold")
	}
	if raw := s.Export(pid); !bytes.Contains(raw, []byte("\n  \"schema\": 1,")) {
		t.Fatalf("export = %s", raw)
	}
	s.Perform(pid, op(operations.Op{Action: "note.add", Cat: "c1", New: "n1", Title: "n"}))
	var p domain.Project
	if err := json.Unmarshal(s.Export(pid), &p); err != nil || p.Schema != domain.RequiredSchema(p) || p.Schema != 4 {
		t.Fatalf("schema = %d, %v", p.Schema, err)
	}
}

func TestImportAddsAProjectLikeOrdoImport(t *testing.T) {
	s, pid := opened(t)
	if _, err := s.Import(s.Export(pid)); !errors.Is(err, operations.ErrProjectNameTaken) {
		t.Fatalf("a held name: %v", err)
	}
	if _, err := s.Import([]byte(`{"schema": 99, "id": "p9", "name": "New", "categories": []}`)); !errors.Is(err, domain.ErrSchemaTooNew) {
		t.Fatalf("a newer schema: %v", err)
	}
	if _, err := s.Import([]byte(`{"id": "p9", "name": "a\u0007b", "categories": []}`)); err == nil {
		t.Fatal("imported a control character")
	}
	id, err := s.Import([]byte(`{"id": "p1", "name": "Other", "categories": [
		{"id": "c1", "name": "Fix", "tasks": [{"id": "t1", "title": "a", "status": "todo", "tag_color": "cyan-diamond"}]},
		{"id": "c2", "name": "fix", "tasks": []}]}`))
	if err != nil || id == pid || domain.ValidateID(id) != nil {
		t.Fatalf("import = %q, %v", id, err)
	}
	var p domain.Project
	json.Unmarshal(s.Project(id), &p)
	if p.Name != "Other" || p.CreatedAt == "" || p.Categories[1].Name != "fix (2)" || p.Categories[0].Tasks[0].TagColor != domain.TagCyan {
		t.Fatalf("project = %+v", p)
	}
	if id, err := s.Import([]byte(`{"id": "../x", "name": "Third", "categories": []}`)); err != nil || domain.ValidateID(id) != nil {
		t.Fatalf("a bad ID = %q, %v", id, err)
	}
}

func TestAnImportedProjectTakesEdits(t *testing.T) {
	s := core.New()
	id, err := s.Import([]byte(`{"id": "p1", "name": "Ferry", "categories": [{"id": "c1", "name": "Fix", "tasks": []}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if a := s.Perform(id, op(operations.Op{Action: "task.add", Cat: "c1", New: "t1", Title: "Dock"})); a.Status != operations.Applied {
		t.Fatalf("add = %+v", a)
	}
	if !bytes.Contains(s.Export(id), []byte(`"title": "Dock"`)) {
		t.Fatalf("export = %s", s.Export(id))
	}
}
