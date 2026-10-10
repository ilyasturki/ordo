package cli

import (
	"path/filepath"
	"testing"

	"phasionary/internal/data"
	"phasionary/internal/domain"
)

func nestedStoreProject(t *testing.T, dataDir string) domain.Project {
	t.Helper()
	p := domain.Project{
		ID:   "nested-1",
		Name: "Nested",
		Categories: []domain.Category{
			{ID: "cat-v2", Name: "v0.2.0", Tasks: []domain.Task{{ID: "loose", Title: "Loose", Status: domain.StatusTodo}}},
			{ID: "cat-ui", Name: "ui", ParentID: "cat-v2", Tasks: []domain.Task{
				{ID: "timer", Title: "Rest timer", Status: domain.StatusInProgress},
				{ID: "pick", ParentID: "timer", Title: "Pick a default", Status: domain.StatusCompleted},
				{ID: "sets", Title: "Fix sets", Status: domain.StatusTodo},
			}},
			{ID: "cat-v1", Name: "v0.1.0"},
		},
	}
	saved, err := data.NewStore(filepath.Join(dataDir, "projects")).ImportProject(p)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	return saved
}

func structure(p domain.Project) []string {
	var out []string
	for ci, c := range p.Categories {
		line := c.Name
		if parent := domain.ParentIndex(p.Categories, ci); parent >= 0 {
			line = p.Categories[parent].Name + " > " + c.Name
		}
		out = append(out, line)
		for ti, task := range c.Tasks {
			line := "  " + task.Title + " " + task.Status
			if parent := domain.ParentIndex(c.Tasks, ti); parent >= 0 {
				line = "    " + c.Tasks[parent].Title + " > " + task.Title + " " + task.Status
			}
			out = append(out, line)
		}
	}
	return out
}

func TestExportImportRoundTripsNesting(t *testing.T) {
	for _, format := range []string{"json", "markdown"} {
		t.Run(format, func(t *testing.T) {
			dir := t.TempDir()
			want := nestedStoreProject(t, dir)
			file := filepath.Join(t.TempDir(), "out."+map[string]string{"json": "json", "markdown": "md"}[format])
			if out, err := runCLI(t, dir, "export", "--project", "Nested", "--format", format, "-o", file); err != nil {
				t.Fatalf("export: %v\n%s", err, out)
			}
			if out, err := runCLI(t, dir, "import", file, "--name", "Back"); err != nil {
				t.Fatalf("import: %v\n%s", err, out)
			}
			got, err := data.NewStore(filepath.Join(dir, "projects")).LoadProject("Back")
			if err != nil {
				t.Fatalf("load: %v", err)
			}
			a, b := structure(want), structure(got)
			if len(a) != len(b) {
				t.Fatalf("structure = %q, want %q", b, a)
			}
			for i := range a {
				if a[i] != b[i] {
					t.Fatalf("structure = %q, want %q", b, a)
				}
			}
		})
	}
}
