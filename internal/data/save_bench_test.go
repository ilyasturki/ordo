package data

import (
	"fmt"
	"testing"

	"github.com/ilyasturki/ordo/internal/domain"
)

func benchSaveProject(cats, tasksPer int) domain.Project {
	p := domain.Project{ID: "benchsave", Name: "Bench"}
	for c := 0; c < cats; c++ {
		cat := domain.Category{ID: fmt.Sprintf("c%d", c), Name: fmt.Sprintf("Cat %d", c)}
		for t := 0; t < tasksPer; t++ {
			cat.Tasks = append(cat.Tasks, domain.Task{
				ID:     fmt.Sprintf("c%dt%d", c, t),
				Title:  fmt.Sprintf("Task %d in category %d with a fairly descriptive title", t, c),
				Status: domain.StatusTodo,
			})
		}
		p.Categories = append(p.Categories, cat)
	}
	return p
}

// BenchmarkSaveProjectLocked measures the synchronous on-disk save cost,
// including the two fsyncs. This is what USED to run on the UI thread after
// every mutation; it now runs only on the background saver goroutine.
func BenchmarkSaveProjectLocked(b *testing.B) {
	for _, s := range []struct {
		name           string
		cats, tasksPer int
	}{
		{"small_5x10", 5, 10},
		{"medium_10x30", 10, 30},
		{"large_15x100", 15, 100},
	} {
		p := benchSaveProject(s.cats, s.tasksPer)
		dir := b.TempDir()
		store := NewStore(dir)
		if err := store.Ensure(); err != nil {
			b.Fatal(err)
		}
		if err := store.saveProjectAtomic(p); err != nil {
			b.Fatal(err)
		}
		b.Run(s.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if err := store.SaveProjectLocked(p); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkSaverEnqueue measures the NEW per-mutation cost on the UI thread:
// just the marshal snapshot (no fsync, no write). The worker absorbs the disk
// cost off-thread and coalesces bursts, so this is all a keystroke now pays.
func BenchmarkSaverEnqueue(b *testing.B) {
	for _, s := range []struct {
		name           string
		cats, tasksPer int
	}{
		{"small_5x10", 5, 10},
		{"medium_10x30", 10, 30},
		{"large_15x100", 15, 100},
	} {
		p := benchSaveProject(s.cats, s.tasksPer)
		dir := b.TempDir()
		store := NewStore(dir)
		if err := store.Ensure(); err != nil {
			b.Fatal(err)
		}
		if err := store.saveProjectAtomic(p); err != nil {
			b.Fatal(err)
		}
		saver := NewSaver(store)
		b.Run(s.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				saver.Enqueue(p)
			}
		})
		saver.Close()
	}
}

// BenchmarkListProjects measures a full directory scan + JSON parse of every
// project. LoadProject(id) no longer calls this for an ID selector (it reads one
// file), so startup and R-reload no longer scale with total project count.
func BenchmarkListProjects(b *testing.B) {
	for _, n := range []int{1, 10, 50} {
		dir := b.TempDir()
		store := NewStore(dir)
		if err := store.Ensure(); err != nil {
			b.Fatal(err)
		}
		for i := 0; i < n; i++ {
			p := benchSaveProject(8, 40)
			p.ID = fmt.Sprintf("proj%d", i)
			p.Name = fmt.Sprintf("Project %d", i)
			if err := store.saveProjectAtomic(p); err != nil {
				b.Fatal(err)
			}
		}
		b.Run(fmt.Sprintf("projects_%d", n), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if _, err := store.ListProjects(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkLoadProjectByID measures the fast path LoadProject now takes for an
// ID selector: a single file read regardless of how many projects exist.
func BenchmarkLoadProjectByID(b *testing.B) {
	dir := b.TempDir()
	store := NewStore(dir)
	if err := store.Ensure(); err != nil {
		b.Fatal(err)
	}
	for i := 0; i < 50; i++ {
		p := benchSaveProject(8, 40)
		p.ID = fmt.Sprintf("proj%d", i)
		p.Name = fmt.Sprintf("Project %d", i)
		if err := store.saveProjectAtomic(p); err != nil {
			b.Fatal(err)
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := store.LoadProject("proj25"); err != nil {
			b.Fatal(err)
		}
	}
}
