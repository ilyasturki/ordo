package app

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo"
	"github.com/ilyasturki/ordo/internal/app/modes"
	"github.com/ilyasturki/ordo/internal/domain"
)

const sampleChangelog = `# Changelog

Preamble that never shows.

## [Unreleased]

## [1.1.0] - 2026-10-11

Read the [guide](https://example.com) first.

### Added

- ` + "`gc`" + ` shows the changelog in a dialog that scrolls

### Fixed

- One
- Two
`

func plainLines(lines []string) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = ansi.Strip(l)
	}
	return out
}

func TestRenderChangelog(t *testing.T) {
	got := plainLines(renderChangelog(sampleChangelog, 30))
	assert.Equal(t, []string{
		"1.1.0  2026-10-11",
		"",
		"Read the guide first.",
		"",
		"Added",
		"• gc shows the changelog in",
		"  a dialog that scrolls",
		"",
		"Fixed",
		"• One",
		"• Two",
	}, got)
}

func TestRenderChangelogFitsWidth(t *testing.T) {
	for _, l := range renderChangelog(ordo.Changelog, 40) {
		assert.LessOrEqual(t, ansi.StringWidth(l), 40, l)
	}
}

func TestChangelogDialogOpensAndScrolls(t *testing.T) {
	m := *newTestModel(t, domain.Project{ID: "p1", Name: "P"})
	m = press(t, m, "g", "c")
	require.Equal(t, modes.ModeChangelog, m.ui.Modes.Current())
	assert.Contains(t, ansi.Strip(m.changelogView()), "Changelog")

	m = press(t, m, "G")
	assert.Equal(t, m.changelogMaxScroll(), m.ui.Changelog.ScrollOffset)
	assert.Positive(t, m.ui.Changelog.ScrollOffset)
	assert.True(t, strings.Contains(ansi.Strip(m.changelogView()), "0.1.1"))

	m = press(t, m, "q")
	assert.Equal(t, modes.ModeNormal, m.ui.Modes.Current())
}
