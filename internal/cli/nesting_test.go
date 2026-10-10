package cli

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"phasionary/internal/data"
	"phasionary/internal/domain"
)

// versionsDir seeds a project "Nested" with v0.2.0 › ui (a parent task with a
// subtask) and v0.1.0 › ui, so names repeat across parents.
func versionsDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	nestedStoreProject(t, dir)
	for _, args := range [][]string{
		{"category", "add", "ui", "--parent", "v0.1.0"},
		{"task", "add", "Old ui task", "-C", "v0.1.0/ui"},
	} {
		if out, err := runCLI(t, dir, append(args, "--project", "Nested")...); err != nil {
			t.Fatalf("%v: %v\n%s", args, err, out)
		}
	}
	return dir
}

func loadNested(t *testing.T, dir string) domain.Project {
	t.Helper()
	p, err := data.NewStore(filepath.Join(dir, "projects")).LoadProject("Nested")
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestCategoryAddParent(t *testing.T) {
	dir := versionsDir(t)
	p := loadNested(t, dir)
	if got := structure(p); !strings.Contains(strings.Join(got, "|"), "v0.1.0 > ui") {
		t.Fatalf("structure = %q, want ui under v0.1.0", got)
	}

	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"category", "add", "UI", "--parent", "v0.2.0"}, `"UI" already exists`},
		{[]string{"category", "add", "deep", "--parent", "v0.2.0/ui"}, "one level deep"},
		{[]string{"category", "add", "x", "--parent", "ui"}, "pass a path or an ID"},
	} {
		if out, err := runCLI(t, dir, append(tc.args, "--project", "Nested")...); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%v: err = %v, want %q\n%s", tc.args, err, tc.want, out)
		}
	}
}

func TestTaskAddParent(t *testing.T) {
	dir := versionsDir(t)
	if out, err := runCLI(t, dir, "task", "add", "Vibrate", "--parent", "Rest timer", "--project", "Nested"); err != nil {
		t.Fatalf("task add --parent: %v\n%s", err, out)
	}
	p := loadNested(t, dir)
	ui := p.Categories[1]
	var titles []string
	for _, task := range ui.Tasks {
		titles = append(titles, task.Title+"<"+task.ParentID)
	}
	timer := ui.Tasks[0].ID
	want := []string{"Rest timer<", "Pick a default<" + timer, "Vibrate<" + timer, "Fix sets<"}
	if strings.Join(titles, "|") != strings.Join(want, "|") {
		t.Fatalf("ui = %q, want %q", titles, want)
	}

	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"task", "add", "x", "--parent", "Rest timer", "-C", "v0.1.0/ui"}, "is in category"},
		{[]string{"task", "add", "x", "--parent", "Pick a default"}, "one level deep"},
		{[]string{"task", "add", "x"}, "--category is required"},
	} {
		if out, err := runCLI(t, dir, append(tc.args, "--project", "Nested")...); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%v: err = %v, want %q\n%s", tc.args, err, tc.want, out)
		}
	}
}

func TestListingsShowNesting(t *testing.T) {
	dir := versionsDir(t)

	out, err := runCLI(t, dir, "categories", "--project", "Nested")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "\n  ui ") || !strings.Contains(out, "v0.2.0") {
		t.Fatalf("categories should indent sub-categories:\n%s", out)
	}

	out, err = runCLI(t, dir, "categories", "--project", "Nested", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var cats CategoriesOutput
	if err := json.Unmarshal([]byte(out), &cats); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if cats.Categories[0].TaskCount != 4 || cats.Categories[1].Path != "v0.2.0/ui" || cats.Categories[1].ParentID != cats.Categories[0].ID {
		t.Fatalf("categories json = %+v", cats.Categories)
	}

	out, err = runCLI(t, dir, "tasks", "-C", "v0.2.0", "--project", "Nested", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var tasks TasksOutput
	if err := json.Unmarshal([]byte(out), &tasks); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if len(tasks.Tasks) != 4 {
		t.Fatalf("-C v0.2.0 should take its sub-category's tasks: %+v", tasks.Tasks)
	}
	if tasks.Tasks[2].Category != "v0.2.0/ui" || tasks.Tasks[2].ParentID != tasks.Tasks[1].ID {
		t.Fatalf("subtask item = %+v", tasks.Tasks[2])
	}

	out, err = runCLI(t, dir, "tasks", "-C", "v0.1.0/ui", "--project", "Nested")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Old ui task") || strings.Contains(out, "Rest timer") {
		t.Fatalf("-C v0.1.0/ui:\n%s", out)
	}

	out, err = runCLI(t, dir, "tasks", "-C", "v0.2.0/ui", "--project", "Nested")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "↳ Pick a default") {
		t.Fatalf("a subtask is marked in the listing:\n%s", out)
	}

	out, err = runCLI(t, dir, "task", "show", "Pick a default", "--project", "Nested")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Parent:   Rest timer") || !strings.Contains(out, "Category: v0.2.0/ui") {
		t.Fatalf("task show:\n%s", out)
	}
	out, err = runCLI(t, dir, "task", "show", "Rest timer", "--project", "Nested")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Subtasks: 1 of 1 done") {
		t.Fatalf("task show of a parent:\n%s", out)
	}
}

func TestDeletesTakeChildren(t *testing.T) {
	dir := versionsDir(t)
	if out, err := runCLI(t, dir, "task", "delete", "Rest timer", "-f", "--project", "Nested"); err != nil {
		t.Fatalf("task delete: %v\n%s", err, out)
	}
	p := loadNested(t, dir)
	if got := len(p.Categories[1].Tasks); got != 1 {
		t.Fatalf("ui keeps %d tasks, want Fix sets alone", got)
	}

	if out, err := runCLI(t, dir, "category", "delete", "v0.2.0", "-f", "--project", "Nested"); err != nil {
		t.Fatalf("category delete: %v\n%s", err, out)
	}
	p = loadNested(t, dir)
	if got := structure(p); len(got) != 3 || got[0] != "v0.1.0" {
		t.Fatalf("structure = %q, want v0.1.0 and its area only", got)
	}
}

func TestDeletePromptNamesChildren(t *testing.T) {
	dir := versionsDir(t)
	cmd := newRootCmd()
	var buf strings.Builder
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetIn(strings.NewReader("n\n"))
	cmd.SetArgs([]string{"--data", dir, "category", "delete", "v0.2.0", "--project", "Nested"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "1 sub-category and 4 tasks") {
		t.Fatalf("prompt = %q", buf.String())
	}
}

func TestResolveCategoryPaths(t *testing.T) {
	p := domain.Project{Categories: []domain.Category{
		{ID: "v2-id", Name: "v0.2.0"},
		{ID: "ux-id", Name: "ui/ux", ParentID: "v2-id"},
		{ID: "ui2-id", Name: "ui", ParentID: "v2-id"},
		{ID: "v1-id", Name: "v0.1.0"},
		{ID: "ui1-id", Name: "UI", ParentID: "v1-id"},
	}}
	for selector, want := range map[string]string{
		"ui/ux":          "ux-id",
		"v0.2.0/ui/ux":   "ux-id",
		"v0.2.0 / ui":    "ui2-id",
		"V0.1.0/ui":      "ui1-id",
		"v0.1.0":         "v1-id",
		"ui1-":           "ui1-id",
		"  v0.2.0/ui  ":  "ui2-id",
		"v0.2.0/ui/ux  ": "ux-id",
	} {
		cat, _, err := resolveCategory(p, selector)
		if err != nil || cat.ID != want {
			t.Errorf("resolveCategory(%q) = %v, %v; want %s", selector, cat, err, want)
		}
	}
	if _, _, err := resolveCategory(p, "ui"); err == nil || !strings.Contains(err.Error(), "v0.2.0/ui") || !strings.Contains(err.Error(), "v0.1.0/UI") {
		t.Fatalf("ambiguous name: err = %v, want both paths", err)
	}
	if _, _, err := resolveCategory(p, "v0.3.0/ui"); err == nil {
		t.Fatal("a path under a missing parent resolved")
	}
}
