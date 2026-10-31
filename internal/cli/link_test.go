package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ilyasturki/ordo/internal/data"
)

func TestProjectLinkWritesTheSharedFile(t *testing.T) {
	dir := initDefault(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	work := filepath.Join(home, "work")
	if err := os.Mkdir(work, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(work)

	if _, err := runCLI(t, dir, "project", "link", "Default"); err != nil {
		t.Fatalf("link: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(dir, "projects", ".directories.json"))
	if err != nil {
		t.Fatal(err)
	}
	var links map[string]string
	if err := json.Unmarshal(raw, &links); err != nil {
		t.Fatal(err)
	}
	if links["~/work"] != loadDefault(t, dir).ID {
		t.Fatalf("links = %v", links)
	}

	if _, err := runCLI(t, dir, "project", "unlink"); err != nil {
		t.Fatalf("unlink: %v", err)
	}
	if got := data.NewStateManager(dir, work).GetProjectForDir(); got != "" {
		t.Fatalf("still linked to %q", got)
	}
}

func TestProjectDeleteRemovesTheViewFile(t *testing.T) {
	dir := initDefault(t)
	p := loadDefault(t, dir)
	store := data.NewStore(filepath.Join(dir, "projects"))
	folds := []string{p.Categories[0].ID}
	if err := store.MergeView(p.ID, data.ViewChange{Folds: &folds}); err != nil {
		t.Fatal(err)
	}

	if _, err := runCLI(t, dir, "project", "delete", "Default", "--force"); err != nil {
		t.Fatalf("delete: %v", err)
	}

	if _, ok := store.View(p.ID); ok {
		t.Fatal("the view file outlived its project")
	}
}
