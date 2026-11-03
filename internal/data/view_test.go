package data

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/ilyasturki/ordo/internal/domain"
)

func viewStore(t *testing.T) (*Store, string) {
	t.Helper()
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, "projects"))
	if err := store.Ensure(); err != nil {
		t.Fatal(err)
	}
	return store, dir
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func onDisk(t *testing.T, store *Store, id string) {
	t.Helper()
	writeFile(t, filepath.Join(store.Dir, id+".json"), `{"id":"`+id+`","name":"`+id+`","categories":[]}`)
}

func TestAWriteKeepsWhatItDidNotChange(t *testing.T) {
	store, _ := viewStore(t)
	filter := Filter{Statuses: []string{"todo"}}
	if err := store.MergeView("p1", ViewChange{Filter: &filter}); err != nil {
		t.Fatal(err)
	}
	folds := []string{"c2", "c1"}
	if err := store.MergeView("p1", ViewChange{Folds: &folds}); err != nil {
		t.Fatal(err)
	}
	got, ok := store.View("p1")
	if !ok || !slices.Equal(got.Folds, []string{"c1", "c2"}) || !slices.Equal(got.Filter.Statuses, []string{"todo"}) {
		t.Fatalf("view = %+v, %v", got, ok)
	}
	cleared := []string{}
	if err := store.MergeView("p1", ViewChange{Folds: &cleared}); err != nil {
		t.Fatal(err)
	}
	got, _ = store.View("p1")
	if len(got.Folds) != 0 || !slices.Equal(got.Filter.Statuses, []string{"todo"}) {
		t.Fatalf("after clearing the folds: %+v", got)
	}
}

func TestAWriteKeepsAFieldItDoesNotKnow(t *testing.T) {
	store, _ := viewStore(t)
	writeFile(t, store.viewPath("p1"), `{"zoom":3,"filter":{"statuses":["todo"]}}`)
	folds := []string{"c1"}
	if err := store.MergeView("p1", ViewChange{Folds: &folds}); err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	raw, _ := os.ReadFile(store.viewPath("p1"))
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if doc["zoom"] != float64(3) || doc["filter"] == nil || doc["folds"] == nil {
		t.Fatalf("file = %s", raw)
	}
}

func TestAWriteThatChangesNothingLeavesTheFile(t *testing.T) {
	store, _ := viewStore(t)
	folds := []string{"c1", "c2"}
	if err := store.MergeView("p1", ViewChange{Folds: &folds}); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(store.viewPath("p1"))
	if err != nil {
		t.Fatal(err)
	}
	same := []string{"c2", "c1"}
	if err := store.MergeView("p1", ViewChange{Folds: &same}); err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(store.viewPath("p1"))
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(before, after) {
		t.Fatal("an unchanged write replaced the file")
	}
}

func TestClearingNothingMakesNoFile(t *testing.T) {
	store, _ := viewStore(t)
	folds := []string{}
	if err := store.MergeView("p1", ViewChange{Folds: &folds, Filter: &Filter{}}); err != nil {
		t.Fatal(err)
	}
	if _, ok := store.View("p1"); ok {
		t.Fatal("clearing a view that isn't there made a file")
	}
}

func TestAClearedViewKeepsItsFile(t *testing.T) {
	store, _ := viewStore(t)
	filter := Filter{Statuses: []string{"todo"}}
	if err := store.MergeView("p1", ViewChange{Filter: &filter}); err != nil {
		t.Fatal(err)
	}
	if err := store.MergeView("p1", ViewChange{Filter: &Filter{}}); err != nil {
		t.Fatal(err)
	}
	if got, ok := store.View("p1"); !ok || !got.IsZero() {
		t.Fatalf("view = %+v, %v: it must stay to win over an older state.json", got, ok)
	}
}

func TestAnUnreadableViewIsLeftAlone(t *testing.T) {
	store, _ := viewStore(t)
	writeFile(t, store.viewPath("p1"), `{not json`)
	if got, ok := store.View("p1"); !ok || !got.IsZero() {
		t.Fatalf("view = %+v, %v", got, ok)
	}
	folds := []string{"c1"}
	if err := store.MergeView("p1", ViewChange{Folds: &folds}); err == nil {
		t.Fatal("wrote over a file it can't read")
	}
	if raw, _ := os.ReadFile(store.viewPath("p1")); string(raw) != `{not json` {
		t.Fatalf("file = %s", raw)
	}
}

func TestAViewKeepsWhatAFilterRowOffers(t *testing.T) {
	store, _ := viewStore(t)
	writeFile(t, store.viewPath("p1"), `{"folds":["ok","../up"],"filter":{"statuses":["todo","levitating"],"priorities":["","high"],"tags":["green-diamond","none","teal"],"categories":["c1","../x"]}}`)
	got, _ := store.View("p1")
	want := View{
		Folds: []string{"ok"},
		Filter: Filter{
			Statuses:   []string{"todo"},
			Priorities: []string{"high", "none"},
			Categories: []string{"c1"},
			Tags:       []string{"green", "none"},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("view = %+v", got)
	}
}

func TestAViewFoldsTheMediumBucketIntoNone(t *testing.T) {
	store, _ := viewStore(t)
	writeFile(t, store.viewPath("p1"), `{"filter":{"priorities":["medium","none","high"]}}`)
	got, _ := store.View("p1")
	if want := []string{"high", "none"}; !reflect.DeepEqual(got.Filter.Priorities, want) {
		t.Fatalf("priorities = %v", got.Filter.Priorities)
	}
	writeFile(t, store.viewPath("p2"), `{"filter":{"priorities":["medium"]}}`)
	got, _ = store.View("p2")
	if want := []string{"none"}; !reflect.DeepEqual(got.Filter.Priorities, want) {
		t.Fatalf("a ticked medium alone = %v", got.Filter.Priorities)
	}
}

func TestAViewIDNeverLeavesTheDirectory(t *testing.T) {
	store, dir := viewStore(t)
	folds := []string{"c1"}
	if err := store.MergeView("../outside", ViewChange{Folds: &folds}); err == nil {
		t.Fatal("wrote a view under a malformed ID")
	}
	if _, err := os.Stat(filepath.Join(dir, "outside.view.json")); err == nil {
		t.Fatal("the write left the projects directory")
	}
	if _, ok := store.View("../outside"); ok {
		t.Fatal("read a view under a malformed ID")
	}
}

func TestViewFilesAreNeverProjects(t *testing.T) {
	store, _ := viewStore(t)
	onDisk(t, store, "p1")
	writeFile(t, store.viewPath("p1"), `{"folds":["c1"]}`)
	writeFile(t, store.directoriesPath(), `{"~/x":"p1"}`)
	writeFile(t, filepath.Join(store.Dir, "p1.view.sync-conflict-20261031-120000-ABCDEFG.json"), `{"folds":["copy"]}`)
	writeFile(t, filepath.Join(store.Dir, ".directories.sync-conflict-20261031-120000-ABCDEFG.json"), `{"~/x":"copy"}`)

	list, err := store.ListProjects()
	if err != nil || len(list) != 1 || list[0].ID != "p1" {
		t.Fatalf("list = %+v, %v", list, err)
	}
	snaps, err := store.Snapshots(nil)
	if err != nil || len(snaps) != 1 {
		t.Fatalf("snapshots = %+v, %v", snaps, err)
	}
	if got, _ := store.View("p1"); !slices.Equal(got.Folds, []string{"c1"}) {
		t.Fatalf("a conflict copy reached the view: %+v", got)
	}
}

func TestDeleteProjectRemovesItsView(t *testing.T) {
	store, _ := viewStore(t)
	p, err := store.CreateProject("Doomed")
	if err != nil {
		t.Fatal(err)
	}
	folds := []string{"c1"}
	if err := store.MergeView(p.ID, ViewChange{Folds: &folds}); err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteProject(p.ID); err != nil {
		t.Fatal(err)
	}
	if _, ok := store.View(p.ID); ok {
		t.Fatal("the view file outlived its project")
	}
}

func legacyState(t *testing.T, dir, content string) {
	t.Helper()
	writeFile(t, filepath.Join(dir, "state.json"), content)
}

func TestMigrationMovesFoldsFiltersAndLinks(t *testing.T) {
	store, dir := viewStore(t)
	home := filepath.Join(dir, "home")
	t.Setenv("HOME", home)
	onDisk(t, store, "p1")
	legacyState(t, dir, `{
		"directory_projects": {"`+home+`/work": "p1", "/srv/x": "p1"},
		"folded_categories": {"p1": ["c2", "c1"]},
		"filters": {"p1": {"statuses": ["todo"], "tags": ["none"]}},
		"cursors": {"p1": {"kind": "task", "task_id": "t1"}},
		"last_opened": {"p1": "2026-10-31T12:00:00Z"}
	}`)

	m := newLoadedStateManager(t, dir, home+"/work")

	if got := m.GetFoldedCategories("p1"); !slices.Equal(got, []string{"c1", "c2"}) {
		t.Fatalf("folds = %v", got)
	}
	if got := m.GetFilter("p1"); !slices.Equal(got.Statuses, []string{"todo"}) || !slices.Equal(got.Tags, []string{"none"}) {
		t.Fatalf("filter = %+v", got)
	}
	if got := m.GetProjectForDir(); got != "p1" {
		t.Fatalf("link = %q", got)
	}
	links, err := os.ReadFile(store.directoriesPath())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(links), `"~/work"`) || !strings.Contains(string(links), `"/srv/x"`) {
		t.Fatalf("links = %s", links)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, gone := range []string{"directory_projects", "folded_categories", "filters"} {
		if strings.Contains(string(raw), gone) {
			t.Fatalf("state.json still holds %s:\n%s", gone, raw)
		}
	}
	if got := m.GetCursor("p1"); got.TaskID != "t1" {
		t.Fatalf("cursor = %+v", got)
	}
	if got := m.GetLastOpened(); got["p1"] != "2026-10-31T12:00:00Z" {
		t.Fatalf("last opened = %v", got)
	}
}

func TestMigrationLeavesAViewAlreadyThere(t *testing.T) {
	store, dir := viewStore(t)
	home := filepath.Join(dir, "home")
	t.Setenv("HOME", home)
	onDisk(t, store, "p1")
	onDisk(t, store, "p2")
	writeFile(t, store.viewPath("p1"), `{"folds":["other-machine"],"filter":{"statuses":["completed"]}}`)
	writeFile(t, store.directoriesPath(), `{"~/work":"p2","~/elsewhere":"p2"}`)
	legacyState(t, dir, `{
		"directory_projects": {"`+home+`/work": "p1", "`+home+`/mine": "p1"},
		"folded_categories": {"p1": ["this-machine"], "p2": ["c9"]},
		"filters": {"p1": {"statuses": ["todo"]}}
	}`)

	m := newLoadedStateManager(t, dir, home+"/work")

	if got := m.GetFoldedCategories("p1"); !slices.Equal(got, []string{"other-machine"}) {
		t.Fatalf("the view file lost: folds = %v", got)
	}
	if got := m.GetFilter("p1"); !slices.Equal(got.Statuses, []string{"completed"}) {
		t.Fatalf("the view file lost: filter = %+v", got)
	}
	if got := m.GetFoldedCategories("p2"); !slices.Equal(got, []string{"c9"}) {
		t.Fatalf("this machine's other project didn't move: %v", got)
	}
	if got := m.GetProjectForDir(); got != "p2" {
		t.Fatalf("the links file lost: %q", got)
	}
	var links map[string]string
	raw, _ := os.ReadFile(store.directoriesPath())
	if err := json.Unmarshal(raw, &links); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"~/work": "p2", "~/elsewhere": "p2", "~/mine": "p1"}
	if !reflect.DeepEqual(links, want) {
		t.Fatalf("links = %v", links)
	}
}

func TestMigrationRunsOnce(t *testing.T) {
	store, dir := viewStore(t)
	onDisk(t, store, "p1")
	legacyState(t, dir, `{"folded_categories": {"p1": ["c1"]}}`)
	m := newLoadedStateManager(t, dir, "")
	if err := m.SetFoldedCategories("p1", nil); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(store.viewPath("p1")); err != nil {
		t.Fatal(err)
	}
	if got := newLoadedStateManager(t, dir, "").GetFoldedCategories("p1"); len(got) != 0 {
		t.Fatalf("the old keys came back: %v", got)
	}
}

func TestMigrationWaitsForAProjectNotOnDisk(t *testing.T) {
	store, dir := viewStore(t)
	legacyState(t, dir, `{"folded_categories": {"late": ["c1"]}, "filters": {"late": {"statuses": ["todo"]}}}`)

	m := newLoadedStateManager(t, dir, "")
	if err := m.SetCursor("late", Cursor{Kind: "project"}); err != nil {
		t.Fatal(err)
	}
	if _, ok := store.View("late"); ok {
		t.Fatal("a view file for a project that isn't there")
	}

	onDisk(t, store, "late")
	again := newLoadedStateManager(t, dir, "")
	if got := again.GetFoldedCategories("late"); !slices.Equal(got, []string{"c1"}) {
		t.Fatalf("folds = %v", got)
	}
	if got := again.GetFilter("late"); !slices.Equal(got.Statuses, []string{"todo"}) {
		t.Fatalf("filter = %+v", got)
	}
}

func TestMigrationWritesNothingFromAnEmptyEntry(t *testing.T) {
	store, dir := viewStore(t)
	onDisk(t, store, "p1")
	legacyState(t, dir, `{"folded_categories": {"p1": []}}`)
	newLoadedStateManager(t, dir, "")
	if _, ok := store.View("p1"); ok {
		t.Fatal("an empty entry made a view file")
	}
}

func TestAGetterNeverMigratesOrWrites(t *testing.T) {
	store, dir := viewStore(t)
	onDisk(t, store, "p1")
	m := newLoadedStateManager(t, dir, "")
	legacyState(t, dir, `{"folded_categories": {"p1": ["c1"]}}`)
	m.GetFoldedCategories("p1")
	m.GetFilter("p1")
	m.GetCursor("p1")
	if _, ok := store.View("p1"); ok {
		t.Fatal("only Load migrates")
	}
}

func TestLinksMatchAcrossHomes(t *testing.T) {
	store, dir := viewStore(t)
	t.Setenv("HOME", "/home/alice")
	alice := newLoadedStateManager(t, dir, "/home/alice/Projects/x")
	if err := alice.SetProjectForDir("p1"); err != nil {
		t.Fatal(err)
	}

	t.Setenv("HOME", "/Users/bob")
	bob := newLoadedStateManager(t, dir, "/Users/bob/Projects/x")
	if got := bob.GetProjectForDir(); got != "p1" {
		t.Fatalf("link = %q", got)
	}

	raw, _ := os.ReadFile(store.directoriesPath())
	if strings.Contains(string(raw), "alice") {
		t.Fatalf("a home directory reached the shared file: %s", raw)
	}
}

func TestDirectoryKey(t *testing.T) {
	t.Setenv("HOME", "/home/yasso/")
	for dir, want := range map[string]string{
		"/home/yasso":            "~",
		"/home/yasso/":           "~",
		"/home/yasso/Projects/x": "~/Projects/x",
		"/home/yassine/x":        "/home/yassine/x",
		"/srv/x":                 "/srv/x",
	} {
		if got := directoryKey(dir); got != want {
			t.Errorf("directoryKey(%q) = %q, want %q", dir, got, want)
		}
	}
}

func TestTwoMachinesLinkingKeepEachOthersLinks(t *testing.T) {
	_, dir := viewStore(t)
	t.Setenv("HOME", "/home/a")
	first := newLoadedStateManager(t, dir, "/home/a/one")
	second := newLoadedStateManager(t, dir, "/home/a/two")

	if err := first.SetProjectForDir("p1"); err != nil {
		t.Fatal(err)
	}
	if err := second.SetProjectForDir("p2"); err != nil {
		t.Fatal(err)
	}

	if got := first.GetProjectForDir(); got != "p1" {
		t.Fatalf("first = %q", got)
	}
	if got := second.GetProjectForDir(); got != "p2" {
		t.Fatalf("second = %q", got)
	}
	if prev, err := first.UnlinkDir(); err != nil || prev != "p1" {
		t.Fatalf("unlink = %q, %v", prev, err)
	}
	if got := second.GetProjectForDir(); got != "p2" {
		t.Fatalf("unlinking one dropped the other: %q", got)
	}
}

func TestLinkingAgainLeavesTheFile(t *testing.T) {
	store, dir := viewStore(t)
	m := newLoadedStateManager(t, dir, "/srv/x")
	if err := m.SetProjectForDir("p1"); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(store.directoriesPath())
	if err != nil {
		t.Fatal(err)
	}
	if err := m.SetProjectForDir("p1"); err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(store.directoriesPath())
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(before, after) {
		t.Fatal("every launch would rewrite the synced file")
	}
}

func TestViewChangedSeesOthersNotItself(t *testing.T) {
	store, dir := viewStore(t)
	onDisk(t, store, "p1")
	m := newLoadedStateManager(t, dir, "")
	if m.ViewChanged("p1") {
		t.Fatal("the first call starts the watch")
	}
	if err := m.SetFoldedCategories("p1", []string{"c1"}); err != nil {
		t.Fatal(err)
	}
	if m.ViewChanged("p1") {
		t.Fatal("its own write counted")
	}

	other := []string{"c2"}
	if err := store.MergeView("p1", ViewChange{Folds: &other}); err != nil {
		t.Fatal(err)
	}
	if !m.ViewChanged("p1") {
		t.Fatal("missed another writer")
	}
	if m.ViewChanged("p1") {
		t.Fatal("told twice")
	}
}

func TestAWriteOfADeletedProjectBringsNoFileBack(t *testing.T) {
	store, dir := viewStore(t)
	onDisk(t, store, "p1")
	m := newLoadedStateManager(t, dir, "")
	if err := store.DeleteProject("p1"); err != nil {
		t.Fatal(err)
	}
	if err := m.SetFoldedCategories("p1", []string{"c1"}); err != nil {
		t.Fatal(err)
	}
	if _, ok := store.View("p1"); ok {
		t.Fatal("the view file came back")
	}
}

func TestDeletingAProjectAlreadyGoneStillDeletesItsView(t *testing.T) {
	store, _ := viewStore(t)
	folds := []string{"c1"}
	if err := store.MergeView("p1", ViewChange{Folds: &folds}); err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteProject("p1"); err != ErrProjectNotFound {
		t.Fatalf("err = %v", err)
	}
	if _, ok := store.View("p1"); ok {
		t.Fatal("the view file stayed")
	}
}

func TestAFileInAnotherShapeIsLeftAloneWhenNothingChanges(t *testing.T) {
	store, _ := viewStore(t)
	path := store.viewPath("p1")
	writeFile(t, path, "{\"filter\":{\"statuses\":[\"todo\"]},\"folds\":[\"c1\"]}\n")
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	same := []string{"c1"}
	if err := store.MergeView("p1", ViewChange{Folds: &same}); err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(before, after) {
		t.Fatal("an unchanged write replaced the file")
	}
}

func TestAWriteOverANullFileWorks(t *testing.T) {
	store, _ := viewStore(t)
	writeFile(t, store.viewPath("p1"), "null")
	folds := []string{"c1"}
	if err := store.MergeView("p1", ViewChange{Folds: &folds}); err != nil {
		t.Fatal(err)
	}
	if got, _ := store.View("p1"); !slices.Equal(got.Folds, folds) {
		t.Fatalf("view = %+v", got)
	}
}

func TestFoldsOfNothingButBadIDsClearTheField(t *testing.T) {
	store, _ := viewStore(t)
	folds := []string{"../x"}
	if err := store.MergeView("p1", ViewChange{Folds: &folds, Filter: &Filter{Statuses: []string{"todo"}}}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(store.viewPath("p1"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "folds") {
		t.Fatalf("file = %s", raw)
	}
}

func TestMigrationLeavesStateAloneWhenNothingMoved(t *testing.T) {
	_, dir := viewStore(t)
	legacyState(t, dir, `{"folded_categories": {"later": ["c1"]}}`)
	path := filepath.Join(dir, "state.json")
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	newLoadedStateManager(t, dir, "")
	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(before, after) {
		t.Fatal("a start with nothing to move rewrote state.json")
	}
}

func TestViewChangedKeepsAnOutsideWriteJustBeforeItsOwn(t *testing.T) {
	store, dir := viewStore(t)
	onDisk(t, store, "p1")
	m := newLoadedStateManager(t, dir, "")
	m.ViewChanged("p1")

	filter := Filter{Statuses: []string{"todo"}}
	if err := store.MergeView("p1", ViewChange{Filter: &filter}); err != nil {
		t.Fatal(err)
	}
	if err := m.SetFoldedCategories("p1", []string{"c1"}); err != nil {
		t.Fatal(err)
	}
	if !m.ViewChanged("p1") {
		t.Fatal("its own write hid the outside change before it")
	}
}

func TestViewChangedFollowsTheOpenProject(t *testing.T) {
	store, dir := viewStore(t)
	m := newLoadedStateManager(t, dir, "")
	m.ViewChanged("p1")
	folds := []string{"c1"}
	if err := store.MergeView("p2", ViewChange{Folds: &folds}); err != nil {
		t.Fatal(err)
	}
	if m.ViewChanged("p1") {
		t.Fatal("another project's file counted")
	}
	if m.ViewChanged("p2") {
		t.Fatal("a switch reads the view itself: nothing to tell")
	}
	if err := store.MergeView("p2", ViewChange{Filter: &Filter{Tags: []string{"none"}}}); err != nil {
		t.Fatal(err)
	}
	if !m.ViewChanged("p2") {
		t.Fatal("missed a change after the switch")
	}
}

func TestHasProjectAndViewFileStems(t *testing.T) {
	store, _ := viewStore(t)
	if store.HasProject("../x") || store.HasProject("p1") {
		t.Fatal("HasProject")
	}
	onDisk(t, store, "p1")
	if !store.HasProject("p1") {
		t.Fatal("HasProject")
	}
	if domain.ValidateID("p1.view") == nil {
		t.Fatal("a view file's stem must not pass as an ID: every scanner relies on it")
	}
}
