package app

import (
	"slices"
	"strconv"

	"github.com/ilyasturki/ordo/internal/app/modes"
	"github.com/ilyasturki/ordo/internal/app/selection"
	"github.com/ilyasturki/ordo/internal/ui"
)

// Bottom "lazygit-style" hint footer. Only keeps shortcuts that aren't
// obvious from on-screen affordances or vim/arrow conventions (j/k, enter,
// arrows). Bindings already surfaced inside dialogs stay out.
var normalShortcuts = []ui.Hint{
	{Key: "?", Label: "help"},
	{Key: "a", Label: "add"},
	{Key: "A/gA", Label: "category/sub"},
	{Key: "m", Label: "move"},
	{Key: "f", Label: "filter"},
	{Key: "/", Label: "search"},
	{Key: "v", Label: "visual"},
	{Key: "y/x/p", Label: "copy/cut/paste"},
	{Key: "u/^r", Label: "undo/redo"},
	{Key: ",", Label: "options"},
	{Key: "q", Label: "quit"},
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
	{Key: "m", Label: "move to"},
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
	hints := m.normalHints()
	switch {
	case m.ui.Modes.IsVisual():
		hints = visualShortcuts
	case m.ui.Clipboard.IsCut:
		hints = cutShortcuts
		if hide, ok := m.peekEscHint(); ok {
			hints = slices.Clone(cutShortcuts)
			hints[slices.IndexFunc(hints, func(h ui.Hint) bool { return h.Key == "esc" })] = hide
		}
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

func (m model) normalHints() []ui.Hint {
	if !m.ui.Filter.HasValues() {
		return normalShortcuts
	}
	hints := slices.Clone(normalShortcuts)
	f := slices.IndexFunc(hints, func(h ui.Hint) bool { return h.Key == "f" })
	if m.ui.Filter.IsOff() {
		return slices.Insert(hints, f+1, ui.Hint{Key: "F", Label: "filter on"})
	}
	hints[f] = ui.Hint{Key: "f", Label: "filtered", Active: true}
	hints = slices.Insert(hints, f+1, ui.Hint{Key: "F", Label: "off"})
	hide, peeking := m.peekEscHint()
	pos, ok := m.selectedPosition()
	switch {
	case ok && pos.Kind == selection.FocusHidden:
		cat := m.project.Categories[pos.CategoryIndex]
		label := "show "
		if m.ui.Filter.peeks(cat.ID) {
			label = "hide "
		}
		hints = slices.Insert(hints, 0, ui.Hint{Key: "enter", Label: label + strconv.Itoa(hiddenByFilter(cat, &m.ui.Filter))})
	case peeking:
		hints = slices.Insert(hints, 0, hide)
	}
	return hints
}

func (m model) peekEscHint() (ui.Hint, bool) {
	if m.ui.Filter.peek == "" {
		return ui.Hint{}, false
	}
	ci, err := m.project.FindCategoryByID(m.ui.Filter.peek)
	if err != nil {
		return ui.Hint{}, false
	}
	n := hiddenByFilter(m.project.Categories[ci], &m.ui.Filter)
	return ui.Hint{Key: "esc", Label: "hide " + strconv.Itoa(n)}, n > 0
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
