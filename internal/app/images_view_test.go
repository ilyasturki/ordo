package app

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/app/components"
	"github.com/ilyasturki/ordo/internal/app/selection"
)

const goneTarget = "attachments/fedcba9876543210fedcba9876543210.png"

// imageTaskModel puts a stored image and a missing one on Alpha's description.
func imageTaskModel(t *testing.T) (*model, string) {
	t.Helper()
	m, store := newAttachModel(t)
	target, err := store.Add(pngBytes(t, 64, 32))
	require.NoError(t, err)
	m.project.Categories[0].Tasks[0].Title = "Alpha https://example.com/a"
	m.project.Categories[0].Tasks[0].Description = "see https://example.com/b\n" +
		"![Screenshot 10.10](" + target + ")\n![old shot](" + goneTarget + ")"
	m.rebuildPositions()
	require.True(t, m.selectTaskIdx(0, 0))
	path, _ := store.Path(target)
	return m, path
}

func TestGx_ListsTheTitlesAndTheDescriptionsLinksAndImages(t *testing.T) {
	m, path := imageTaskModel(t)

	assert.Nil(t, m.openLinksForSelected())

	require.True(t, m.ui.Modes.IsURLPicker())
	gone := filepath.Join(m.deps.Attachments.Dir(), filepath.Base(goneTarget))
	assert.Equal(t, []components.Link{
		{Label: "https://example.com/a", Target: "https://example.com/a"},
		{Label: "https://example.com/b", Target: "https://example.com/b"},
		{Label: "▣ Screenshot 10.10", Target: path},
		{Label: "▣ old shot", Target: gone},
	}, m.ui.URLPicker.Links)
}

func TestGx_AMissingImageIsNotSyncedYet(t *testing.T) {
	m, _ := imageTaskModel(t)

	gone := filepath.Join(m.deps.Attachments.Dir(), filepath.Base(goneTarget))
	assert.Nil(t, m.openLink(components.Link{Label: "▣ old shot", Target: gone}))
	assert.Equal(t, "old shot is not synced yet", m.ui.Screen.StatusMsg)

	assert.Nil(t, m.openLink(components.Link{Label: "▣ ext", Target: "/nowhere/ext.png"}))
	assert.Equal(t, "/nowhere/ext.png is missing", m.ui.Screen.StatusMsg)
}

func TestGx_OpensFromTheDescriptionRowToo(t *testing.T) {
	m, _ := imageTaskModel(t)
	m.ui.Screen.ExpandDescriptions = true
	m.rebuildPositions()
	selectRow(t, m, selection.FocusDescription, "t1")

	m.openLinksForSelected()

	assert.True(t, m.ui.Modes.IsURLPicker())
}

func TestGi_ListsEachImage(t *testing.T) {
	m, path := imageTaskModel(t)
	m.ui.Screen.Width = 200

	info := ansi.Strip(strings.Join(m.infoLines(), "\n"))

	assert.Contains(t, info, "  ▣ Screenshot 10.10\n    ")
	assert.Contains(t, strings.ReplaceAll(info, "\n    ", ""), path, "the path, wrapped at its slashes")
	assert.Regexp(t, `\n    \d+ B · 64×32\n`, info)
	assert.Contains(t, info, "  ▣ old shot\n    missing: not synced yet")
	assert.NotContains(t, info, "0123456789abcdef")
	assert.NotContains(t, info, "fedcba98", "the description reads with names, not hashes")
}

func TestGy_OffersEachImagesPath(t *testing.T) {
	m, path := imageTaskModel(t)
	gone := filepath.Join(m.deps.Attachments.Dir(), filepath.Base(goneTarget))

	values := entityValues(m.buildYankItems(selection.Position{Kind: selection.FocusTask, CategoryIndex: 0, TaskIndex: 0}))

	assert.Contains(t, values, path)
	assert.Contains(t, values, gone)
	for _, v := range values {
		assert.NotRegexp(t, `^[0-9]{4,}$`, v, "no digit run from a hash")
	}
}

func TestSearch_MatchesAnImagesNameNotItsHash(t *testing.T) {
	m, _ := imageTaskModel(t)
	cat := m.project.Categories[0]
	row := selection.Position{Kind: selection.FocusTask, CategoryIndex: 0, TaskIndex: 0, ID: cat.Tasks[0].ID}

	assert.True(t, m.rowMatches(row, "old shot"))
	assert.False(t, m.rowMatches(row, "fedcba"))
}
