package components

import (
	"strings"

	"github.com/ilyasturki/ordo/internal/ui"
)

func RenderListPicker(title string, items []string, selected int, hints []ui.Hint) string {
	lines := []string{ui.DialogTitleStyle.Render(title), ""}
	for i, item := range items {
		prefix := "  "
		if i == selected {
			prefix = "> "
		}
		line := prefix + item
		if i == selected {
			line = ui.SelectedStyle.Render(line)
		}
		lines = append(lines, line)
	}
	lines = append(lines, "", ui.RenderHints(hints))
	return ui.HelpDialogStyle.Render(strings.Join(lines, "\n"))
}
