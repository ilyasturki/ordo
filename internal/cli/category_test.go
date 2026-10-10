package cli

import (
	"strings"
	"testing"

	"phasionary/internal/domain"
)

func initDefault(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if _, err := runCLI(t, dir, "init"); err != nil {
		t.Fatalf("init: %v", err)
	}
	return dir
}

func categoryNames(p domain.Project) []string {
	names := make([]string, len(p.Categories))
	for i, c := range p.Categories {
		names[i] = c.Name
	}
	return names
}

func TestCategoryAdd_TrimsAndValidates(t *testing.T) {
	dir := initDefault(t)

	if _, err := runCLI(t, dir, "category", "add", "  v0.2.0  ", "--project", "Default"); err != nil {
		t.Fatalf("category add: %v", err)
	}
	names := categoryNames(loadDefault(t, dir))
	if names[len(names)-1] != "v0.2.0" {
		t.Fatalf("names = %q, want a trimmed v0.2.0 last", names)
	}

	for _, bad := range []string{"   ", "V0.2.0 ", "a\x1b[2Jb"} {
		if _, err := runCLI(t, dir, "category", "add", bad, "--project", "Default"); err == nil {
			t.Errorf("category add %q succeeded, want it refused", bad)
		}
	}
	if got := len(loadDefault(t, dir).Categories); got != len(names) {
		t.Fatalf("refused adds left %d categories, want %d", got, len(names))
	}
}

func TestCategoryEdit_RefusesDuplicate(t *testing.T) {
	dir := initDefault(t)

	out, err := runCLI(t, dir, "category", "edit", "Fix", "--name", " feature ", "--project", "Default")
	if err == nil || !strings.Contains(err.Error(), `"feature" already exists`) {
		t.Fatalf("category edit to a taken name: err=%v out=%s", err, out)
	}
}

func TestImportJSON_SuffixesDuplicateCategoryNames(t *testing.T) {
	in := `{"id":"p1","name":"P","categories":[
		{"id":"a","name":"Todo","tasks":[]},
		{"id":"b","name":" todo","tasks":[]},
		{"id":"c","name":"Later","tasks":[]}]}`
	p, err := importJSON(strings.NewReader(in), "")
	if err != nil {
		t.Fatalf("importJSON: %v", err)
	}
	got := categoryNames(p)
	want := []string{"Todo", " todo (2)", "Later"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("names = %q, want %q", got, want)
		}
	}
}

func TestResolveCategory_RefusesAmbiguousName(t *testing.T) {
	p := domain.Project{Categories: []domain.Category{
		{ID: "aaaa-1111", Name: "v0.2.0"},
		{ID: "bbbb-2222", Name: "V0.2.0 "},
		{ID: "cccc-3333", Name: "next"},
	}}

	_, _, err := resolveCategory(p, "v0.2.0")
	if err == nil || !strings.Contains(err.Error(), "aaaa-1111, bbbb-2222") {
		t.Fatalf("ambiguous name: err = %v, want it to list both IDs", err)
	}

	if _, idx, err := resolveCategory(p, "bbbb-2222"); err != nil || idx != 1 {
		t.Fatalf("by ID: idx=%d err=%v", idx, err)
	}
	if _, idx, err := resolveCategory(p, "NEXT"); err != nil || idx != 2 {
		t.Fatalf("unique name: idx=%d err=%v", idx, err)
	}
	if _, _, err := resolveCategory(p, "later"); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("unknown name: err = %v", err)
	}
}
