package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/attachments"
)

// seedImages stores an old referenced image, an old orphan and a young orphan.
func seedImages(t *testing.T) (dir, kept, orphan, young string) {
	t.Helper()
	dir = initDefault(t)
	dir2 := filepath.Join(dir, "projects", attachments.DirName)
	require.NoError(t, os.MkdirAll(dir2, 0o755))
	write := func(name string, age time.Duration) string {
		path := filepath.Join(dir2, name)
		require.NoError(t, os.WriteFile(path, []byte(name), 0o644))
		when := time.Now().Add(-age)
		require.NoError(t, os.Chtimes(path, when, when))
		return name
	}
	kept = write("00000000000000000000000000000001.png", 40*24*time.Hour)
	orphan = write("00000000000000000000000000000002.jpg", 40*24*time.Hour)
	young = write("00000000000000000000000000000003.png", time.Hour)
	_, err := runCLI(t, dir, "task", "add", "With a shot", "--description", "![s](attachments/"+kept+")", "--category", "Feature", "--project", "Default")
	require.NoError(t, err)
	return dir, kept, orphan, young
}

func TestAttachmentsPrune_ListsWithoutRemoving(t *testing.T) {
	dir, kept, orphan, young := seedImages(t)

	out, err := runCLI(t, dir, "attachments", "prune")

	require.NoError(t, err)
	assert.Contains(t, out, orphan)
	assert.NotContains(t, out, kept)
	assert.NotContains(t, out, young)
	assert.Contains(t, out, "Would remove 1 image, 36 B: run again with --yes to remove them")
	for _, name := range []string{kept, orphan, young} {
		assert.FileExists(t, filepath.Join(dir, "projects", attachments.DirName, name))
	}
}

func TestAttachmentsPrune_YesRemovesTheOrphansOnly(t *testing.T) {
	dir, kept, orphan, young := seedImages(t)

	out, err := runCLI(t, dir, "attachments", "prune", "--yes", "--json")

	require.NoError(t, err)
	var got PruneOutput
	require.NoError(t, json.Unmarshal([]byte(out), &got))
	assert.True(t, got.Removed)
	require.Len(t, got.Files, 1)
	assert.Equal(t, orphan, got.Files[0].File)
	store := filepath.Join(dir, "projects", attachments.DirName)
	assert.NoFileExists(t, filepath.Join(store, orphan))
	assert.FileExists(t, filepath.Join(store, kept))
	assert.FileExists(t, filepath.Join(store, young))

	out, err = runCLI(t, dir, "attachments", "prune")
	require.NoError(t, err)
	assert.Contains(t, out, "No unreferenced image older than 30 days")
}

func TestAttachmentsPrune_WithNoImagesYet(t *testing.T) {
	dir := initDefault(t)

	out, err := runCLI(t, dir, "attachments", "prune", "--yes")

	require.NoError(t, err)
	assert.Contains(t, out, "No unreferenced image")
}
