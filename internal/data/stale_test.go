package data

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/domain"
	"github.com/ilyasturki/ordo/internal/fsutil"
)

func twoStores(t *testing.T) (*Store, *Store, domain.Project) {
	t.Helper()
	dir := t.TempDir()
	a, b := NewStore(dir), NewStore(dir)
	p, err := a.CreateProject("Shared")
	require.NoError(t, err)
	return a, b, p
}

func TestSnapshotSaveRefusedAfterExternalWrite(t *testing.T) {
	a, b, p := twoStores(t)
	mine, err := a.LoadProjectByID(p.ID)
	require.NoError(t, err)

	_, err = b.WithProjectLocked(p.ID, func(p *domain.Project) error {
		p.Name = "Renamed elsewhere"
		return nil
	})
	require.NoError(t, err)

	// The picker lists every project while the open one holds unsaved edits:
	// listing must not move the baseline or this save would go through.
	_, err = a.ListProjects()
	require.NoError(t, err)

	mine.Name = "Renamed here"
	err = a.SaveProjectLocked(mine)
	require.ErrorIs(t, err, ErrStaleProject)

	onDisk, err := b.LoadProjectByID(p.ID)
	require.NoError(t, err)
	assert.Equal(t, "Renamed elsewhere", onDisk.Name, "the refused save must not touch the file")

	fresh, err := a.LoadProjectByID(p.ID)
	require.NoError(t, err)
	fresh.Name = "Renamed here"
	require.NoError(t, a.SaveProjectLocked(fresh))
}

func TestFileWatchTellsEachVersionOnce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "any.view.json")
	w := NewFileWatch(path)
	assert.False(t, w.Changed(), "still missing")

	require.NoError(t, fsutil.WriteAtomic(path, []byte("one"), 0o644))
	assert.True(t, w.Changed(), "created")
	assert.False(t, w.Changed(), "told once")

	require.NoError(t, fsutil.WriteAtomic(path, []byte("two"), 0o644))
	assert.True(t, w.Changed(), "replaced")

	require.NoError(t, os.Remove(path))
	assert.True(t, w.Changed(), "removed")
	assert.False(t, w.Changed(), "told once")

	require.NoError(t, fsutil.WriteAtomic(path, []byte("three"), 0o644))
	assert.True(t, w.Changed(), "back")
	assert.False(t, NewFileWatch(path).Changed(), "a new watch starts from the file as it is")
}

func TestOutsideChangeSeesPastOwnSaves(t *testing.T) {
	a, b, p := twoStores(t)
	mine, err := a.LoadProjectByID(p.ID)
	require.NoError(t, err)
	assert.False(t, a.OutsideChange(p.ID), "just loaded")

	require.NoError(t, a.SaveProjectLocked(mine))
	assert.False(t, a.OutsideChange(p.ID), "its own save")

	renameTo(t, b, p.ID, "Elsewhere")
	assert.True(t, a.OutsideChange(p.ID))
	assert.False(t, a.OutsideChange(p.ID), "told once")

	_, err = a.LoadProjectByID(p.ID)
	require.NoError(t, err)
	assert.False(t, a.OutsideChange(p.ID), "the reload took it in")
	assert.False(t, a.OutsideChange("unknown"), "a project this process never loaded")
}

func TestOutsideChangeRefusesTheEditsAFailedSaveLeft(t *testing.T) {
	a, _, p := twoStores(t)
	mine, err := a.LoadProjectByID(p.ID)
	require.NoError(t, err)
	lock := a.lockPath(p.ID)
	require.NoError(t, os.Remove(lock))
	require.NoError(t, os.Mkdir(lock, 0o755))
	require.Error(t, a.SaveProjectLocked(mine))
	_, why := a.Unsaved(p.ID)
	require.NotErrorIs(t, why, ErrStaleProject)

	require.NoError(t, fsutil.WriteAtomic(a.projectPath(p.ID), []byte(`{}`), 0o644))
	assert.True(t, a.OutsideChange(p.ID))
	n, why := a.Unsaved(p.ID)
	assert.Equal(t, 1, n)
	assert.ErrorIs(t, why, ErrStaleProject)

	require.NoError(t, os.Remove(a.projectPath(p.ID)))
	assert.True(t, a.OutsideChange(p.ID))
	_, why = a.Unsaved(p.ID)
	assert.ErrorIs(t, why, ErrProjectNotFound)
}

func TestOutsideChangeWaitsForASnapshotInFlight(t *testing.T) {
	a, b, p := twoStores(t)
	mine, err := a.LoadProjectByID(p.ID)
	require.NoError(t, err)
	saver := NewSaver(a)

	_, err = b.WithProjectLocked(p.ID, func(other *domain.Project) error {
		other.Name = "Elsewhere"
		saver.Enqueue(mine)
		assert.False(t, a.OutsideChange(p.ID), "the worker waits on this lock")
		return nil
	})
	require.NoError(t, err)
	saver.Drain()

	n, why := a.Unsaved(p.ID)
	assert.Equal(t, 1, n)
	assert.ErrorIs(t, why, ErrStaleProject, "the save itself was refused")
	assert.True(t, a.OutsideChange(p.ID), "and the version is reported")
	assert.ErrorIs(t, saver.Close(), ErrStaleProject)
}

func TestOwnSavesStayFresh(t *testing.T) {
	a, _, p := twoStores(t)
	mine, err := a.LoadProjectByID(p.ID)
	require.NoError(t, err)
	require.NoError(t, a.SaveProjectLocked(mine))
	require.NoError(t, a.SaveProjectLocked(mine), "a write adopts the version it wrote")
}

func TestWithProjectLockedUnaffectedByExternalWrite(t *testing.T) {
	a, b, p := twoStores(t)
	_, err := a.LoadProjectByID(p.ID)
	require.NoError(t, err)
	_, err = b.WithProjectLocked(p.ID, func(p *domain.Project) error {
		p.Name = "B"
		return nil
	})
	require.NoError(t, err)
	got, err := a.WithProjectLocked(p.ID, func(p *domain.Project) error {
		require.Equal(t, "B", p.Name)
		p.Name = "A after B"
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, "A after B", got.Name)
}

func TestUnknownProjectSavesWithoutBaseline(t *testing.T) {
	_, b, p := twoStores(t)
	c := NewStore(b.Dir)
	p.Name = "blind"
	require.NoError(t, c.SaveProjectLocked(p))
}

func renameTo(t *testing.T, s *Store, id, name string) {
	t.Helper()
	_, err := s.WithProjectLocked(id, func(p *domain.Project) error {
		p.Name = name
		return nil
	})
	require.NoError(t, err)
}

func TestSnapshotQueuedBeforeReloadIsRefused(t *testing.T) {
	tui, web, p := twoStores(t)
	mine, err := tui.LoadProjectByID(p.ID)
	require.NoError(t, err)
	saver := NewSaver(tui)

	// Holding the lock parks the worker before its check, as a slow disk would.
	lock, err := tui.acquireProjectLock(p.ID)
	require.NoError(t, err)
	mine.Name = "TUI edit"
	saver.Enqueue(mine)
	edited := mine
	edited.Name = "Web edit"
	require.NoError(t, web.saveProjectAtomic(edited))
	_, err = tui.LoadProjectByID(p.ID)
	require.NoError(t, err)
	require.NoError(t, lock.Close())
	saver.Drain()

	onDisk, err := web.LoadProjectByID(p.ID)
	require.NoError(t, err)
	assert.Equal(t, "Web edit", onDisk.Name)
	n, err := tui.Unsaved(p.ID)
	assert.Zero(t, n, "the reloaded copy has no edits of its own")
	assert.NoError(t, err)
	assert.NoError(t, saver.Close())
}

func TestDrainWaitsForTheQueuedWrite(t *testing.T) {
	tui, web, p := twoStores(t)
	mine, err := tui.LoadProjectByID(p.ID)
	require.NoError(t, err)
	saver := NewSaver(tui)
	defer saver.Close()

	mine.Name = "Drained"
	saver.Enqueue(mine)
	saver.Drain()

	onDisk, err := web.LoadProjectByID(p.ID)
	require.NoError(t, err)
	assert.Equal(t, "Drained", onDisk.Name)
}

func TestUnsavedCountsRefusedSnapshotsUntilReload(t *testing.T) {
	tui, web, p := twoStores(t)
	mine, err := tui.LoadProjectByID(p.ID)
	require.NoError(t, err)
	renameTo(t, web, p.ID, "Elsewhere")

	mine.Name = "One"
	require.ErrorIs(t, tui.SaveProjectLocked(mine), ErrStaleProject)
	mine.Name = "Two"
	require.ErrorIs(t, tui.SaveProjectLocked(mine), ErrStaleProject)
	n, err := tui.Unsaved(p.ID)
	assert.Equal(t, 2, n)
	assert.ErrorIs(t, err, ErrStaleProject)

	_, err = tui.LoadProjectByID(p.ID)
	require.NoError(t, err)
	n, err = tui.Unsaved(p.ID)
	assert.Zero(t, n)
	assert.NoError(t, err)
}

func TestSaverCloseReportsEditsThatNeverLanded(t *testing.T) {
	tui, web, p := twoStores(t)
	mine, err := tui.LoadProjectByID(p.ID)
	require.NoError(t, err)
	renameTo(t, web, p.ID, "Elsewhere")
	saver := NewSaver(tui)

	mine.Name = "Lost"
	saver.Enqueue(mine)
	err = saver.Close()

	var unsaved *UnsavedError
	require.ErrorAs(t, err, &unsaved)
	assert.Equal(t, "Lost", unsaved.Name)
	assert.Equal(t, 1, unsaved.Edits)
	assert.ErrorIs(t, err, ErrStaleProject)
	assert.Equal(t, `1 edit to "Lost" not saved: project changed on disk since it was loaded`, err.Error())
}

func TestSaverCloseForgetsEditsAReloadDropped(t *testing.T) {
	tui, web, p := twoStores(t)
	mine, err := tui.LoadProjectByID(p.ID)
	require.NoError(t, err)
	renameTo(t, web, p.ID, "Elsewhere")
	saver := NewSaver(tui)

	mine.Name = "Dropped by r"
	saver.Enqueue(mine)
	saver.Drain()
	_, err = tui.LoadProjectByID(p.ID)
	require.NoError(t, err)

	assert.NoError(t, saver.Close())
}

func TestDropUnsavedKeepsCheckingTheCopy(t *testing.T) {
	tui, web, p := twoStores(t)
	mine, err := tui.LoadProjectByID(p.ID)
	require.NoError(t, err)
	renameTo(t, web, p.ID, "Elsewhere")
	require.ErrorIs(t, tui.SaveProjectLocked(mine), ErrStaleProject)

	tui.DropUnsaved(p.ID)
	n, err := tui.Unsaved(p.ID)
	assert.Zero(t, n)
	assert.NoError(t, err)
	assert.ErrorIs(t, tui.SaveProjectLocked(mine), ErrStaleProject, "dropping the edits doesn't make the copy current")
}
