package cli

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"phasionary/internal/data"
	"phasionary/internal/domain"
)

// versionsDir seeds v0.2.0 › ui and v0.1.0 › ui so a sub-category name repeats across parents.
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

func TestResolveCategoryTakesThePathItPrints(t *testing.T) {
	p := domain.Project{Categories: []domain.Category{
		{ID: "fix-top", Name: "Fix"},
		{ID: "v2-id", Name: "v0.2.0"},
		{ID: "fix-sub", Name: "Fix", ParentID: "v2-id"},
		{ID: "be-id", Name: "Backend"},
		{ID: "api-sub", Name: "API", ParentID: "be-id"},
		{ID: "api-top", Name: "Backend/API"},
	}}
	for i := range p.Categories[:3] {
		cat, _, err := resolveCategory(p, cliPath(p, i))
		if err != nil || cat.ID != p.Categories[i].ID {
			t.Errorf("resolveCategory(%q) = %v, %v; want %s", cliPath(p, i), cat, err, p.Categories[i].ID)
		}
	}
	if _, _, err := resolveCategory(p, "Backend/API"); err == nil || !strings.Contains(err.Error(), "api-sub") || !strings.Contains(err.Error(), "api-top") {
		t.Fatalf("a path two categories print: err = %v, want both refused", err)
	}
}

func TestTaskAddParentLooksInTheNamedCategory(t *testing.T) {
	dir := versionsDir(t)
	for _, args := range [][]string{
		{"task", "add", "Rest timer", "-C", "v0.1.0/ui"},
		{"task", "add", "Old vibrate", "--parent", "Rest timer", "-C", "v0.1.0/ui"},
	} {
		if out, err := runCLI(t, dir, append(args, "--project", "Nested")...); err != nil {
			t.Fatalf("%v: %v\n%s", args, err, out)
		}
	}
	p := loadNested(t, dir)
	for i, c := range p.Categories {
		if cliPath(p, i) != "v0.1.0/ui" {
			continue
		}
		var parent, child domain.Task
		for _, task := range c.Tasks {
			switch task.Title {
			case "Rest timer":
				parent = task
			case "Old vibrate":
				child = task
			}
		}
		if parent.ID == "" || child.ParentID != parent.ID {
			t.Fatalf("v0.1.0/ui tasks = %+v, want Old vibrate under its own Rest timer", c.Tasks)
		}
		return
	}
	t.Fatal("no v0.1.0/ui category")
}

func TestImportJSONMakesNamesUniqueAfterTheRepair(t *testing.T) {
	in := `{"id":"p1","name":"P","categories":[
		{"id":"a","name":"Work","tasks":[]},
		{"id":"b","name":"Work","parent_id":"gone","tasks":[]}]}`
	p, err := importJSON(strings.NewReader(in), "")
	if err != nil {
		t.Fatalf("importJSON: %v", err)
	}
	if got := categoryNames(p); got[1] != "Work (2)" || p.Categories[1].ParentID != "" {
		t.Fatalf("names = %q, parent = %q; want the orphan top-level as Work (2)", got, p.Categories[1].ParentID)
	}
}

func TestParentCompletionOffersTopLevelRowsOnly(t *testing.T) {
	dir := versionsDir(t)
	complete := func(args ...string) []string {
		t.Helper()
		out, err := runCLI(t, dir, append([]string{"__complete"}, args...)...)
		if err != nil {
			t.Fatalf("%v: %v\n%s", args, err, out)
		}
		var got []string
		for _, line := range strings.Split(out, "\n") {
			if strings.HasPrefix(line, ":") {
				break
			}
			got = append(got, line)
		}
		return got
	}

	if got := complete("category", "add", "x", "--project", "Nested", "--parent", ""); strings.Join(got, "|") != "v0.2.0|v0.1.0" {
		t.Errorf("category add --parent offers %q, want the top-level categories", got)
	}
	if got := complete("tasks", "--project", "Nested", "-C", ""); len(got) != 4 {
		t.Errorf("tasks -C offers %q, want every category", got)
	}
	got := strings.Join(complete("task", "add", "x", "--project", "Nested", "--parent", ""), "|")
	if !strings.Contains(got, "timer") || strings.Contains(got, "pick") {
		t.Errorf("task add --parent offers %q, want top-level tasks only", got)
	}
}
