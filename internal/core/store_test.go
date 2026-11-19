//go:build !js

package core_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ilyasturki/ordo/internal/core"
	"github.com/ilyasturki/ordo/internal/data"
	"github.com/ilyasturki/ordo/internal/domain"
)

func pinClock(t *testing.T, at time.Time) {
	t.Helper()
	was := domain.Now
	domain.Now = func() time.Time { return at }
	t.Cleanup(func() { domain.Now = was })
}

func TestExportAndImportMatchTheStore(t *testing.T) {
	pinClock(t, time.Date(2026, 1, 2, 3, 4, 5, 999_000_000, time.UTC))
	dir := t.TempDir()
	store := data.NewStore(dir)
	created, err := store.CreateProject("Ferry")
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.ReadFile(filepath.Join(dir, created.ID+".json"))
	if err != nil {
		t.Fatal(err)
	}
	s := core.New()
	pid, err := s.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(s.Export(pid), file) {
		t.Fatalf("export =\n%s\nfile =\n%s", s.Export(pid), file)
	}

	web := core.New()
	id, err := web.Import(file)
	if err != nil {
		t.Fatal(err)
	}
	other := data.NewStore(t.TempDir())
	imported, err := other.ImportProject(created)
	if err != nil {
		t.Fatal(err)
	}
	stored, err := os.ReadFile(filepath.Join(other.Dir, imported.ID+".json"))
	if err != nil {
		t.Fatal(err)
	}
	if id != imported.ID || !bytes.Equal(web.Export(id), stored) {
		t.Fatalf("import =\n%s\nstore =\n%s", web.Export(id), stored)
	}
}
