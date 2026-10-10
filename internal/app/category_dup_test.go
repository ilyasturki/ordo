package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestYankPasteCategory_SuffixesTheName(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(1) // Cat A header
	m.enterVisualMode()
	_ = m.visualCopyBullets()
	require.Len(t, m.ui.Clipboard.Categories, 1)

	m.pasteFromClipboard()

	assert.ElementsMatch(t, []string{"Cat A", "Cat A (2)", "Cat B"}, categoryNamesOf(m))
}

func categoryNamesOf(m *model) []string {
	var names []string
	for _, c := range m.project.Categories {
		names = append(names, c.Name)
	}
	return names
}
