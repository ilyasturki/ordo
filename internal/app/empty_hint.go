package app

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/ilyasturki/ordo/internal/ui"
)

const emptyProjectHint = "No categories yet: A adds one, ? shows the keys"

// withEmptyHint centres the hint in the rows left under content, above the bottom bar.
func (m model) withEmptyHint(content string) string {
	if len(m.project.Categories) > 0 {
		return content
	}
	hint := ui.MutedStyle.Width(m.ui.Screen.Width).Align(lipgloss.Center).Render(emptyProjectHint)
	used := strings.Count(content, "\n") + 1
	free := m.ui.Screen.Height - used - m.bottomBarHeight() - strings.Count(hint, "\n") - 1
	if free < 1 {
		return content
	}
	return content + strings.Repeat("\n", free/2+1) + hint
}
