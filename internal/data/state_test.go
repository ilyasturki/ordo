package data

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
)

func newLoadedStateManager(t *testing.T, dir, currentDir string) *StateManager {
	t.Helper()
	m := NewStateManager(dir, currentDir)
	if err := m.Load(); err != nil {
		t.Fatalf("load: %v", err)
	}
	return m
}

// The TUI and another process each hold their own StateManager over the same
// file, and every save rewrites it whole. Without the reload-before-write, the
// second manager's save would drop the first manager's key.
func TestStateManagerConcurrentWritersDoNotClobber(t *testing.T) {
	dir := t.TempDir()
	tui := newLoadedStateManager(t, dir, "/work/project")
	other := newLoadedStateManager(t, dir, "/work/project")

	if err := tui.SetProjectOrder([]string{"p1", "p2"}); err != nil {
		t.Fatalf("set order: %v", err)
	}
	// other's cache predates the write above; its own save must not erase it.
	if err := other.SetFoldedCategories("p1", []string{"cat-a"}); err != nil {
		t.Fatalf("set folds: %v", err)
	}

	fresh := newLoadedStateManager(t, dir, "/work/project")
	if got := fresh.GetProjectOrder(); !slices.Equal(got, []string{"p1", "p2"}) {
		t.Fatalf("project order clobbered: %v", got)
	}
	if got := fresh.GetFoldedCategories("p1"); !slices.Equal(got, []string{"cat-a"}) {
		t.Fatalf("folds clobbered: %v", got)
	}
}

// Another process folds a category while the TUI has the project open; the TUI
// reads folds at project open, so its next read must see the new value.
func TestGetFoldedCategoriesSeesOtherProcessWrites(t *testing.T) {
	dir := t.TempDir()
	tui := newLoadedStateManager(t, dir, "")
	other := newLoadedStateManager(t, dir, "")

	if got := tui.GetFoldedCategories("p1"); len(got) != 0 {
		t.Fatalf("want no folds initially, got %v", got)
	}
	if err := other.SetFoldedCategories("p1", []string{"cat-a", "cat-b"}); err != nil {
		t.Fatalf("set folds: %v", err)
	}

	got := tui.GetFoldedCategories("p1")
	if !slices.Equal(got, []string{"cat-a", "cat-b"}) {
		t.Fatalf("stale folds: want [cat-a cat-b], got %v", got)
	}
}

func TestSetFoldedCategoriesEmptyClearsProject(t *testing.T) {
	dir := t.TempDir()
	m := newLoadedStateManager(t, dir, "")

	if err := m.SetFoldedCategories("p1", []string{"cat-a"}); err != nil {
		t.Fatalf("set folds: %v", err)
	}
	if err := m.SetFoldedCategories("p1", nil); err != nil {
		t.Fatalf("clear folds: %v", err)
	}

	fresh := newLoadedStateManager(t, dir, "")
	if got := fresh.GetFoldedCategories("p1"); len(got) != 0 {
		t.Fatalf("want folds cleared, got %v", got)
	}
}

// A returned fold slice must not alias the cache, or a caller sorting it in
// place would silently rewrite state.
func TestGetFoldedCategoriesReturnsCopy(t *testing.T) {
	dir := t.TempDir()
	m := newLoadedStateManager(t, dir, "")
	if err := m.SetFoldedCategories("p1", []string{"cat-a"}); err != nil {
		t.Fatalf("set folds: %v", err)
	}

	got := m.GetFoldedCategories("p1")
	got[0] = "mutated"

	if again := m.GetFoldedCategories("p1"); again[0] != "cat-a" {
		t.Fatalf("caller mutation leaked into state: %v", again)
	}
}

func TestUnlinkDirReturnsPreviousAndIsNoOpWhenUnlinked(t *testing.T) {
	dir := t.TempDir()
	m := newLoadedStateManager(t, dir, "/work/project")

	prev, err := m.UnlinkDir()
	if err != nil || prev != "" {
		t.Fatalf("unlink with no link: want (\"\", nil), got (%q, %v)", prev, err)
	}

	if err := m.SetProjectForDir("p1"); err != nil {
		t.Fatalf("link: %v", err)
	}
	prev, err = m.UnlinkDir()
	if err != nil {
		t.Fatalf("unlink: %v", err)
	}
	if prev != "p1" {
		t.Fatalf("want previous id p1, got %q", prev)
	}
	if got := newLoadedStateManager(t, dir, "/work/project").GetProjectForDir(); got != "" {
		t.Fatalf("link survived unlink: %q", got)
	}
}

// Run with -race: the cursor saver writes from its own goroutine.
func TestStateManagerConcurrentWritesAreSerialized(t *testing.T) {
	dir := t.TempDir()
	m := newLoadedStateManager(t, dir, "")

	var wg sync.WaitGroup
	for i := range 8 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if err := m.SetFoldedCategories("p1", []string{"cat"}); err != nil {
				t.Errorf("set folds: %v", err)
			}
			_ = m.GetFoldedCategories("p1")
		}(i)
	}
	wg.Wait()

	if got := newLoadedStateManager(t, dir, "").GetFoldedCategories("p1"); !slices.Equal(got, []string{"cat"}) {
		t.Fatalf("want [cat], got %v", got)
	}
}

func TestCursorRoundTripsThroughDisk(t *testing.T) {
	dir := t.TempDir()
	m := newLoadedStateManager(t, dir, "")

	want := Cursor{Kind: "task", CategoryID: "c1", TaskID: "t2"}
	if err := m.SetCursor("p1", want); err != nil {
		t.Fatalf("set cursor: %v", err)
	}

	if got := newLoadedStateManager(t, dir, "").GetCursor("p1"); got != want {
		t.Fatalf("cursor: want %+v, got %+v", want, got)
	}
	if got := newLoadedStateManager(t, dir, "").GetCursor("p2"); !got.IsZero() {
		t.Fatalf("unset project should have no cursor, got %+v", got)
	}
}

// The TUI stores the zero cursor for a project left with nothing selected; that
// must clear the entry rather than persist a meaningless row.
func TestSetCursorZeroClearsProject(t *testing.T) {
	dir := t.TempDir()
	m := newLoadedStateManager(t, dir, "")

	if err := m.SetCursor("p1", Cursor{Kind: "task", CategoryID: "c1", TaskID: "t1"}); err != nil {
		t.Fatalf("set cursor: %v", err)
	}
	if err := m.SetCursor("p1", Cursor{}); err != nil {
		t.Fatalf("clear cursor: %v", err)
	}

	if got := newLoadedStateManager(t, dir, "").GetCursor("p1"); !got.IsZero() {
		t.Fatalf("want cursor cleared, got %+v", got)
	}
}

func TestDeleteCursorRemovesEntry(t *testing.T) {
	dir := t.TempDir()
	m := newLoadedStateManager(t, dir, "")

	if err := m.SetCursor("p1", Cursor{Kind: "category", CategoryID: "c1"}); err != nil {
		t.Fatalf("set cursor: %v", err)
	}
	if err := m.DeleteCursor("p1"); err != nil {
		t.Fatalf("delete cursor: %v", err)
	}

	if got := newLoadedStateManager(t, dir, "").GetCursor("p1"); !got.IsZero() {
		t.Fatalf("cursor survived delete: %+v", got)
	}
}

// The TUI writes its cursor on exit while another process may be writing
// folds; neither whole-file rewrite may drop the other's key.
func TestCursorAndFoldWritersDoNotClobber(t *testing.T) {
	dir := t.TempDir()
	tui := newLoadedStateManager(t, dir, "")
	other := newLoadedStateManager(t, dir, "")

	cursor := Cursor{Kind: "task", CategoryID: "c1", TaskID: "t1"}
	if err := tui.SetCursor("p1", cursor); err != nil {
		t.Fatalf("set cursor: %v", err)
	}
	if err := other.SetFoldedCategories("p1", []string{"cat-a"}); err != nil {
		t.Fatalf("set folds: %v", err)
	}

	fresh := newLoadedStateManager(t, dir, "")
	if got := fresh.GetCursor("p1"); got != cursor {
		t.Fatalf("cursor clobbered by a fold write: %+v", got)
	}
	if got := fresh.GetFoldedCategories("p1"); !slices.Equal(got, []string{"cat-a"}) {
		t.Fatalf("folds clobbered by a cursor write: %v", got)
	}
}

// A state.json written before cursors existed must load cleanly and simply
// report no cursor.
func TestLoadStateWithoutCursorsIsNotAnError(t *testing.T) {
	dir := t.TempDir()
	legacy := `{"directory_projects":{"":"p1"},"folded_categories":{"p1":["c1"]}}`
	if err := os.WriteFile(filepath.Join(dir, "state.json"), []byte(legacy), 0o644); err != nil {
		t.Fatalf("write legacy state: %v", err)
	}

	m := newLoadedStateManager(t, dir, "")
	if got := m.GetCursor("p1"); !got.IsZero() {
		t.Fatalf("want no cursor, got %+v", got)
	}
	if got := m.GetProjectForDir(); got != "p1" {
		t.Fatalf("legacy dir link lost: %q", got)
	}
}

func TestFilterRoundTripsThroughDisk(t *testing.T) {
	dir := t.TempDir()
	m := newLoadedStateManager(t, dir, "")

	want := Filter{Statuses: []string{"todo"}, Priorities: []string{"none", "high"}, Categories: []string{"c1"}, Tags: []string{"green"}}
	if err := m.SetFilter("p1", want); err != nil {
		t.Fatalf("set filter: %v", err)
	}

	fresh := newLoadedStateManager(t, dir, "")
	if got := fresh.GetFilter("p1"); !got.sameAs(want) {
		t.Fatalf("filter: want %+v, got %+v", want, got)
	}
	if got := fresh.GetFilter("p2"); !got.IsZero() {
		t.Fatalf("unset project should have no filter, got %+v", got)
	}
}

func TestSetFilterEmptyClearsProject(t *testing.T) {
	dir := t.TempDir()
	m := newLoadedStateManager(t, dir, "")

	if err := m.SetFilter("p1", Filter{Statuses: []string{"todo"}}); err != nil {
		t.Fatalf("set filter: %v", err)
	}
	if err := m.SetFilter("p1", Filter{}); err != nil {
		t.Fatalf("clear filter: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(dir, "state.json"))
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	if strings.Contains(string(raw), "filters") {
		t.Fatalf("a cleared filter left an entry:\n%s", raw)
	}
}

func TestDeleteFilterRemovesEntry(t *testing.T) {
	dir := t.TempDir()
	m := newLoadedStateManager(t, dir, "")

	if err := m.SetFilter("p1", Filter{Tags: []string{"none"}}); err != nil {
		t.Fatalf("set filter: %v", err)
	}
	if err := m.DeleteFilter("p1"); err != nil {
		t.Fatalf("delete filter: %v", err)
	}

	if got := newLoadedStateManager(t, dir, "").GetFilter("p1"); !got.IsZero() {
		t.Fatalf("filter survived delete: %+v", got)
	}
}

func TestFilterAndCursorWritersDoNotClobber(t *testing.T) {
	dir := t.TempDir()
	tui := newLoadedStateManager(t, dir, "")
	other := newLoadedStateManager(t, dir, "")

	filter := Filter{Statuses: []string{"todo"}}
	if err := tui.SetFilter("p1", filter); err != nil {
		t.Fatalf("set filter: %v", err)
	}
	cursor := Cursor{Kind: "task", CategoryID: "c1", TaskID: "t1"}
	if err := other.SetCursor("p1", cursor); err != nil {
		t.Fatalf("set cursor: %v", err)
	}

	fresh := newLoadedStateManager(t, dir, "")
	if got := fresh.GetFilter("p1"); !got.sameAs(filter) {
		t.Fatalf("filter clobbered by a cursor write: %+v", got)
	}
	if got := fresh.GetCursor("p1"); got != cursor {
		t.Fatalf("cursor clobbered by a filter write: %+v", got)
	}
}

func TestStateManagerPathIsStateJSON(t *testing.T) {
	dir := t.TempDir()
	m := NewStateManager(dir, "")
	if want := filepath.Join(dir, "state.json"); m.path != want {
		t.Fatalf("path: want %q, got %q", want, m.path)
	}
}
