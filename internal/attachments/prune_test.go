package attachments

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var pruneNow = time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)

// pruneFixture is a projects directory holding a store whose files are each
// referenced, or not, from a different kind of file.
type pruneFixture struct {
	t        *testing.T
	projects string
	store    *Store
	next     int
}

func newPruneFixture(t *testing.T) *pruneFixture {
	projects := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(projects, DirName), 0o755))
	return &pruneFixture{t: t, projects: projects, store: NewStore(projects)}
}

// image stores a distinct file aged age and returns its name.
func (f *pruneFixture) image(age time.Duration) string {
	f.t.Helper()
	f.next++
	name := fmt.Sprintf("%032x.png", f.next)
	f.storeFile(name, age)
	return name
}

func (f *pruneFixture) storeFile(name string, age time.Duration) {
	f.t.Helper()
	path := filepath.Join(f.store.Dir(), name)
	require.NoError(f.t, os.WriteFile(path, []byte("bytes of "+name), 0o644))
	when := pruneNow.Add(-age)
	require.NoError(f.t, os.Chtimes(path, when, when))
}

func (f *pruneFixture) projectFile(name, description string) {
	f.t.Helper()
	body := fmt.Sprintf(`{"schema":1,"id":"p","name":"P","categories":[{"id":"c","name":"C","tasks":[{"id":"t","title":"T","status":"todo","description":%q}]}]}`, description)
	require.NoError(f.t, os.WriteFile(filepath.Join(f.projects, name), []byte(body), 0o644))
}

func (f *pruneFixture) orphanNames() []string {
	f.t.Helper()
	orphans, err := f.store.Unreferenced(pruneNow)
	require.NoError(f.t, err)
	var names []string
	for _, o := range orphans {
		names = append(names, o.Name)
	}
	return names
}

func (f *pruneFixture) exists(name string) bool {
	_, err := os.Stat(filepath.Join(f.store.Dir(), name))
	return err == nil
}

const old = 40 * 24 * time.Hour

func TestUnreferenced_KeepsEveryFileAnyProjectFileNames(t *testing.T) {
	f := newPruneFixture(t)
	inProject := f.image(old)
	inConflict := f.image(old)
	inTemp := f.image(old)
	asAbsolutePath := f.image(old)
	escaped := f.image(old)
	inTitleOfRenamedCopy := f.image(old)
	orphan := f.image(old)
	f.projectFile("p.json", "shot:\n![a](attachments/"+inProject+")")
	f.projectFile("p.sync-conflict-20261011-101010-ABCDEFG.json", "![b](attachments/"+inConflict+")")
	f.projectFile("p.json.tmp", "![c](attachments/"+inTemp+")")
	f.projectFile("q.json", "copied out: ![d]("+filepath.Join(f.store.Dir(), asAbsolutePath)+")")
	require.NoError(t, os.WriteFile(filepath.Join(f.projects, "r.json"),
		[]byte(`{"description":"![e](attachments\/`+escaped+`)"}`), 0o644))
	f.projectFile("p (copy).json", inTitleOfRenamedCopy)

	assert.Equal(t, []string{orphan}, f.orphanNames())
}

func TestUnreferenced_SparesYoungFiles(t *testing.T) {
	f := newPruneFixture(t)
	young := f.image(29 * 24 * time.Hour)
	justOld := f.image(PruneAge)
	f.projectFile("p.json", "nothing")

	assert.Equal(t, []string{justOld}, f.orphanNames())
	assert.True(t, f.exists(young))
}

func TestUnreferenced_TouchesOnlyStoredNames(t *testing.T) {
	f := newPruneFixture(t)
	f.projectFile("p.json", "nothing")
	for _, name := range []string{
		".0123456789abcdef0123456789abcdef.png.tmp",
		"0123456789abcdef0123456789abcdef.png.123.tmp",
		"0123456789ABCDEF0123456789ABCDEF.png",
		"0123456789abcdef0123456789abcdef.svg",
		"notes.txt",
		"shot.png",
	} {
		f.storeFile(name, old)
	}
	require.NoError(t, os.Mkdir(filepath.Join(f.store.Dir(), "0123456789abcdef0123456789abcdef.gif"), 0o755))

	assert.Empty(t, f.orphanNames())
	assert.Error(t, f.store.Remove([]Orphan{{Name: "../p.json"}}), "never a name outside the store")
	_, err := os.Stat(filepath.Join(f.projects, "p.json"))
	assert.NoError(t, err)
}

func TestUnreferenced_ClearsTheTempFileOfAnInterruptedWrite(t *testing.T) {
	f := newPruneFixture(t)
	f.projectFile("p.json", "nothing")
	stale := ".0123456789abcdef0123456789abcdef.png.2718281828.tmp"
	fresh := ".fedcba9876543210fedcba9876543210.jpg.31415.tmp"
	f.storeFile(stale, old)
	f.storeFile(fresh, time.Hour)

	assert.Equal(t, []string{stale}, f.orphanNames(), "one still young may be a write in progress")

	orphans, err := f.store.Unreferenced(pruneNow)
	require.NoError(t, err)
	require.NoError(t, f.store.Remove(orphans))
	assert.False(t, f.exists(stale))
	assert.True(t, f.exists(fresh))
}

func TestUnreferenced_FollowsASymlinkedProjectFile(t *testing.T) {
	f := newPruneFixture(t)
	kept := f.image(old)
	elsewhere := filepath.Join(t.TempDir(), "real.json")
	require.NoError(t, os.WriteFile(elsewhere, []byte(`"![k](attachments/`+kept+`)"`), 0o644))
	require.NoError(t, os.Symlink(elsewhere, filepath.Join(f.projects, "p.json")))
	require.NoError(t, os.Symlink("/nowhere.json", filepath.Join(f.projects, "dangling.json")))

	assert.Empty(t, f.orphanNames())
}

func TestUnreferenced_WithoutAStore(t *testing.T) {
	store := NewStore(t.TempDir())

	orphans, err := store.Unreferenced(pruneNow)

	require.NoError(t, err)
	assert.Empty(t, orphans)
}

func TestUnreferenced_AnUnreadableProjectFileStopsIt(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads everything")
	}
	f := newPruneFixture(t)
	f.image(old)
	f.projectFile("p.json", "nothing")
	require.NoError(t, os.Chmod(filepath.Join(f.projects, "p.json"), 0))

	_, err := f.store.Unreferenced(pruneNow)

	assert.Error(t, err, "a reference it can't read could be the only one")
}

func TestRemove_DeletesTheOrphansOnly(t *testing.T) {
	f := newPruneFixture(t)
	kept := f.image(old)
	gone := f.image(old)
	young := f.image(time.Hour)
	f.projectFile("p.json", "![k](attachments/"+kept+")")
	orphans, err := f.store.Unreferenced(pruneNow)
	require.NoError(t, err)
	require.Len(t, orphans, 1)
	assert.Equal(t, int64(len("bytes of "+gone)), orphans[0].Size)
	assert.True(t, orphans[0].ModTime.Equal(pruneNow.Add(-old)))

	require.NoError(t, f.store.Remove(orphans))
	require.NoError(t, f.store.Remove(orphans), "a file already gone is no error")

	assert.True(t, f.exists(kept))
	assert.False(t, f.exists(gone))
	assert.True(t, f.exists(young))
}
