package data

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckProjectName(t *testing.T) {
	store := NewStore(t.TempDir())
	require.NoError(t, store.Ensure())
	alpha, err := store.CreateProject("Alpha")
	require.NoError(t, err)
	_, err = store.CreateProject("Beta")
	require.NoError(t, err)

	assert.ErrorIs(t, store.CheckProjectName(" beta ", alpha.ID), ErrDuplicateProjectName)
	assert.NoError(t, store.CheckProjectName("ALPHA", alpha.ID), "a project may keep its own name")
	assert.NoError(t, store.CheckProjectName("Gamma", alpha.ID))

	loaded, err := store.LoadProjectByID(alpha.ID)
	require.NoError(t, err)
	assert.Equal(t, "Alpha", loaded.Name, "checking writes nothing")
}
