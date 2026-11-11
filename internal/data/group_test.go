package data

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/domain"
)

func TestARegroupKeepsTheStampAndAnEditRestampsIt(t *testing.T) {
	s := NewStore(t.TempDir())
	p, err := s.CreateProject("Ferry")
	require.NoError(t, err)
	raw, err := s.LoadProjectByID(p.ID)
	require.NoError(t, err)
	raw.UpdatedAt = "2026-01-02T03:04:05Z"
	data, err := marshalStamped(raw)
	require.NoError(t, err)
	require.NoError(t, s.saveBytes(p.ID, data))

	_, err = s.WithProjectLocked(p.ID, func(p *domain.Project) error {
		p.Group = "Work"
		return nil
	})
	require.NoError(t, err)
	got, err := s.LoadProjectByID(p.ID)
	require.NoError(t, err)
	assert.Equal(t, "Work", got.Group)
	assert.Equal(t, "2026-01-02T03:04:05Z", got.UpdatedAt)

	_, err = s.WithProjectLocked(p.ID, func(p *domain.Project) error {
		p.Group = "Perso"
		p.Name = "Ferry boat"
		return nil
	})
	require.NoError(t, err)
	got, err = s.LoadProjectByID(p.ID)
	require.NoError(t, err)
	assert.Equal(t, "Perso", got.Group)
	assert.NotEqual(t, "2026-01-02T03:04:05Z", got.UpdatedAt)
}
