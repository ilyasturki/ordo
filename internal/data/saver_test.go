package data

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/ilyasturki/phasionary/internal/domain"
)

// The final edit must always reach disk even when many edits are coalesced.
func TestSaver_CloseFlushesLastWrite(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)
	if err := store.Ensure(); err != nil {
		t.Fatal(err)
	}
	p := domain.Project{ID: "p1", Name: "P"}
	if err := store.saveProjectAtomic(p); err != nil {
		t.Fatal(err)
	}

	saver := NewSaver(store)
	for i := 0; i < 50; i++ {
		p.Name = fmt.Sprintf("Name %d", i)
		saver.Enqueue(p)
	}
	saver.Close() // blocks until the last queued write lands

	got, err := store.LoadProjectByID("p1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Name 49" {
		t.Fatalf("final state not persisted: got %q want %q", got.Name, "Name 49")
	}
}

// A single Enqueue with no coalescing must persist.
func TestSaver_PersistsSingleEdit(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)
	if err := store.Ensure(); err != nil {
		t.Fatal(err)
	}
	p := domain.Project{ID: "p1", Name: "before"}
	if err := store.saveProjectAtomic(p); err != nil {
		t.Fatal(err)
	}
	saver := NewSaver(store)
	p.Name = "after"
	saver.Enqueue(p)
	saver.Close()

	got, err := store.LoadProjectByID("p1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "after" {
		t.Fatalf("got %q want %q", got.Name, "after")
	}
}

// A failed background write surfaces on Results rather than being swallowed.
func TestSaver_SurfacesWriteError(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)
	if err := store.Ensure(); err != nil {
		t.Fatal(err)
	}
	saver := NewSaver(store)
	defer saver.Close()

	// No file on disk for this ID → WriteProjectLocked returns ErrProjectNotFound.
	saver.Enqueue(domain.Project{ID: "ghost", Name: "X"})

	select {
	case err := <-saver.Results():
		if !errors.Is(err, ErrProjectNotFound) {
			t.Fatalf("want ErrProjectNotFound, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for save error")
	}
}
