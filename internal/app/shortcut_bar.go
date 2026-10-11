package app

import (
	"slices"

	"github.com/ilyasturki/ordo/internal/app/modes"
	"github.com/ilyasturki/ordo/internal/domain"
	"github.com/ilyasturki/ordo/internal/ui"
)

// Bottom "lazygit-style" hint footer. Only keeps shortcuts that aren't
// obvious from on-screen affordances or vim/arrow conventions (j/k, enter,
// arrows). Bindings already surfaced inside dialogs stay out.
var normalShortcuts = []ui.Hint{
	{Key: "?", Label: "help"},
	{Key: "a", Label: "add"},
	{Key: "A", Label: "category"},
	{Key: "f", Label: "filter"},
	{Key: "/", Label: "search"},
	{Key: "v", Label: "visual"},
	{Key: "y/x/p", Label: "copy/cut/paste"},
	{Key: "u/^r", Label: "undo/redo"},
	{Key: ",", Label: "options"},
	{Key: "q", Label: "quit"},
}

// versionShortcuts follow "A category" in a project that has versions.
var versionShortcuts = []ui.Hint{
	{Key: "s", Label: "ship"},
	{Key: "M", Label: "move to"},
}

// The cut's status line clears on the next key, the first step to the destination.
var cutShortcuts = []ui.Hint{
	{Key: "p", Label: "move here"},
	{Key: "esc", Label: "cancel cut"},
	{Key: "/", Label: "search"},
	{Key: "?", Label: "help"},
	{Key: "q", Label: "quit"},
}

var visualShortcuts = []ui.Hint{
	{Key: "J/K", Label: "move"},
	{Key: "y", Label: "copy"},
	{Key: "Y", Label: "markdown"},
	{Key: "x", Label: "cut"},
	{Key: "d", Label: "delete"},
	{Key: "esc", Label: "exit"},
}

func (m model) shortcutBarVisible() bool {
	if !m.deps.CfgManager.Get().ShowShortcutBar {
		return false
	}
	switch m.ui.Modes.Current() {
	case modes.ModeNormal, modes.ModeVisual:
		return true
	default:
		return false
	}
}

func (m model) shortcutBarHeight() int {
	if m.shortcutBarVisible() {
		return 1
	}
	return 0
}

func (m model) renderShortcutBar() string {
	if !m.shortcutBarVisible() {
		return ""
	}
	hints := normalShortcuts
	hasVersions := slices.ContainsFunc(m.project.Categories, func(c domain.Category) bool { return c.IsVersion() })
	if hasVersions {
		hints = slices.Concat(normalShortcuts[:3], versionShortcuts, normalShortcuts[3:])
	}
	switch {
	case m.ui.Modes.IsVisual() && hasVersions:
		hints = slices.Concat(visualShortcuts[:1], []ui.Hint{{Key: "M", Label: "move to"}}, visualShortcuts[1:])
	case m.ui.Modes.IsVisual():
		hints = visualShortcuts
	case m.ui.Clipboard.IsCut:
		hints = cutShortcuts
	case len(m.taskCommands()) > 0:
		cmds := m.taskCommands()
		// Right after help: the bar drops whatever overflows its right end.
		commandHints := make([]ui.Hint, len(cmds))
		for i, c := range cmds {
			commandHints[i] = ui.Hint{Key: c.Key, Label: c.Label}
		}
		hints = slices.Concat(hints[:1], commandHints, hints[1:])
	}
	return ui.RenderHintsToWidth(hints, m.ui.Screen.Width)
}

// bottomBarHeight is the rows reserved at the bottom of the screen: the search
// prompt while searching, otherwise the shortcut bar (when enabled).
func (m model) bottomBarHeight() int {
	if m.ui.Modes.IsSearch() || m.ui.Modes.IsCommandPrompt() {
		return 1
	}
	return m.shortcutBarHeight()
}

// renderBottomBar draws the search prompt while searching, otherwise the
// shortcut bar. The two never show at once (search isn't normal/visual mode).
func (m model) renderBottomBar() string {
	switch {
	case m.ui.Modes.IsSearch():
		return m.renderSearchBar()
	case m.ui.Modes.IsCommandPrompt():
		return m.renderCommandPrompt()
	}
	return m.renderShortcutBar()
}
