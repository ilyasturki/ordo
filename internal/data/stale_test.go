package data

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/phasionary/internal/domain"
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
