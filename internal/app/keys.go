package app

import (
	tea "charm.land/bubbletea/v2"

	"phasionary/internal/app/selection"
)

type bindingAction func(*model) tea.Cmd

type keyBinding struct {
	keys    []string
	prefix  rune
	display string
	desc    string
	section string
	action  bindingAction
}

// The help reference groups the bindings by section and lists them in the
// order they are declared below, so keep each section's bindings together.
const (
	sectionNavigation = "Navigation"
	sectionTasks      = "Tasks"
	sectionOrganise   = "Organize"
	sectionClipboard  = "Copy & paste"
	sectionApp        = "App"
)

func void(fn func(*model)) bindingAction {
	return func(m *model) tea.Cmd {
		fn(m)
		return nil
	}
}

var normalBindings = []keyBinding{
	{keys: []string{"up", "k"}, display: "j / k", desc: "move up / down", section: sectionNavigation,
		action: func(m *model) tea.Cmd { m.stepCursor(-1); return nil }},
	{keys: []string{"down", "j"}, section: sectionNavigation,
		action: func(m *model) tea.Cmd { m.stepCursor(1); return nil }},
	{keys: []string{"ctrl+d"}, display: "ctrl+d / ctrl+u", desc: "half page down / up", section: sectionNavigation,
		action: func(m *model) tea.Cmd { m.moveSelectionByPage(0.5); return nil }},
	{keys: []string{"ctrl+u"}, section: sectionNavigation,
		action: func(m *model) tea.Cmd { m.moveSelectionByPage(-0.5); return nil }},
	{keys: []string{"ctrl+f"}, display: "ctrl+f / ctrl+b", desc: "full page down / up", section: sectionNavigation,
		action: func(m *model) tea.Cmd { m.moveSelectionByPage(1.0); return nil }},
	{keys: []string{"ctrl+b"}, section: sectionNavigation,
		action: func(m *model) tea.Cmd { m.moveSelectionByPage(-1.0); return nil }},
	{keys: []string{"g"}, prefix: 'g', display: "gg", desc: "first item", section: sectionNavigation,
		action: void((*model).jumpToFirst)},
	{keys: []string{"G"}, desc: "last item", section: sectionNavigation,
		action: void((*model).jumpToLast)},
	{keys: []string{"}"}, display: "} / {", desc: "next / previous category", section: sectionNavigation,
		action: void((*model).jumpToNextCategory)},
	{keys: []string{"{"}, section: sectionNavigation,
		action: void((*model).jumpToPrevCategory)},
	{keys: []string{"tab"}, display: "tab / za", desc: "fold / unfold category", section: sectionNavigation,
		action: void((*model).toggleFold)},
	{keys: []string{"a"}, prefix: 'z', section: sectionNavigation,
		action: void((*model).toggleFold)},
	{keys: []string{"c"}, prefix: 'z', display: "zc", desc: "fold every category", section: sectionNavigation,
		action: void((*model).foldAll)},
	{keys: []string{"o"}, prefix: 'z', display: "zo", desc: "unfold every category", section: sectionNavigation,
		action: void((*model).unfoldAll)},
	{keys: []string{"z"}, prefix: 'z', display: "zz", desc: "center on screen", section: sectionNavigation,
		action: void((*model).centerOnSelected)},
	{keys: []string{"t"}, prefix: 'z', display: "zt", desc: "scroll to top", section: sectionNavigation,
		action: void((*model).topOnSelected)},
	{keys: []string{"b"}, prefix: 'z', display: "zb", desc: "scroll to bottom", section: sectionNavigation,
		action: void((*model).bottomOnSelected)},
	{keys: []string{"d"}, prefix: 'z', display: "zd", desc: "show / hide descriptions", section: sectionNavigation,
		action: void((*model).toggleExpandDescriptions)},
	{keys: []string{"e"}, prefix: 'z', display: "ze", desc: "expand row in full", section: sectionNavigation,
		action: void((*model).toggleExpandRow)},
	{keys: []string{"/"}, desc: "search", section: sectionNavigation,
		action: func(m *model) tea.Cmd { return m.startSearch() }},
	{keys: []string{"n"}, display: "n / N", desc: "next / previous match", section: sectionNavigation,
		action: void((*model).searchNext)},
	{keys: []string{"N"}, section: sectionNavigation,
		action: void((*model).searchPrev)},

	{keys: []string{"a"}, desc: "add task", section: sectionTasks,
		action: void((*model).startAddingTask)},
	{keys: []string{"A"}, desc: "add category", section: sectionTasks,
		action: void((*model).startAddingCategory)},
	{keys: []string{"-"}, desc: "add separator", section: sectionTasks,
		action: void((*model).startAddingSeparator)},
	{keys: []string{"enter"}, desc: "edit", section: sectionTasks,
		action: func(m *model) tea.Cmd {
			if pos, ok := m.selectedPosition(); ok && pos.Kind == selection.FocusDescription {
				return m.startDescriptionInlineEdit(pos.CategoryIndex, pos.TaskIndex)
			}
			m.startEditing()
			return nil
		}},
	// Many terminals (kitty, ghostty, …) transmit Shift+Enter as the bytes
	// ESC+CR, which Bubble Tea decodes as "alt+enter"; others that speak the
	// Kitty protocol send a genuine "shift+enter". Bind both so the physical
	// Shift+Enter reaches this action regardless of terminal encoding.
	{keys: []string{"shift+enter", "alt+enter"}, display: "shift+enter", desc: "edit description", section: sectionTasks,
		action: func(m *model) tea.Cmd { return m.editOrFocusDescription() }},
	{keys: []string{"e"}, desc: "edit in your $EDITOR", section: sectionTasks,
		action: func(m *model) tea.Cmd { return m.startExternalEdit() }},
	{keys: []string{"space"}, display: "space / ⇧space", desc: "cycle status forward / back", section: sectionTasks,
		action: void((*model).toggleSelectedTask)},
	{keys: []string{"shift+space"}, section: sectionTasks,
		action: void((*model).toggleSelectedTaskReverse)},
	{keys: []string{"h"}, display: "h / l", desc: "priority down / up", section: sectionTasks,
		action: void((*model).decreasePriority)},
	{keys: []string{"l"}, section: sectionTasks,
		action: void((*model).increasePriority)},
	{keys: []string{"t"}, display: "t / T", desc: "cycle tag / edit tag", section: sectionTasks,
		action: void((*model).cycleTag)},
	{keys: []string{"T"}, section: sectionTasks,
		action: void((*model).startTagEdit)},
	{keys: []string{"ctrl+t"}, desc: "set time estimate", section: sectionTasks,
		action: void((*model).openEstimatePicker)},
	{keys: []string{"d"}, desc: "delete", section: sectionTasks,
		action: void((*model).deleteSelected)},

	{keys: []string{"J"}, display: "J / K", desc: "move item down / up", section: sectionOrganise,
		action: func(m *model) tea.Cmd { m.moveSelectedRow(+1); return nil }},
	{keys: []string{"K"}, section: sectionOrganise,
		action: func(m *model) tea.Cmd { m.moveSelectedRow(-1); return nil }},
	{keys: []string{"S"}, desc: "reverse category order", section: sectionOrganise,
		action: void((*model).reverseCategories)},
	{keys: []string{"v"}, desc: "select several rows", section: sectionOrganise,
		action: void((*model).enterVisualMode)},
	{keys: []string{"f"}, desc: "filter tasks", section: sectionOrganise,
		action: func(m *model) tea.Cmd { m.ui.Modes.ToFilter(); return nil }},

	{keys: []string{"y"}, display: "y / Y", desc: "copy / copy as markdown", section: sectionClipboard,
		action: func(m *model) tea.Cmd { return m.copySelected() }},
	{keys: []string{"Y"}, section: sectionClipboard,
		action: func(m *model) tea.Cmd { return m.copyCategoryContent() }},
	{keys: []string{"x"}, desc: "cut (p moves it, esc cancels)", section: sectionClipboard,
		action: func(m *model) tea.Cmd { return m.cutSelectedTask() }},
	{keys: []string{"p"}, desc: "paste (several lines ask how)", section: sectionClipboard,
		action: func(m *model) tea.Cmd { return m.paste() }},
	{keys: []string{"y"}, prefix: 'g', display: "gy", desc: "copy id, url, title…", section: sectionClipboard,
		action: func(m *model) tea.Cmd { return m.yankPartForSelected() }},
	{keys: []string{"t"}, prefix: 'g', display: "gt", desc: "copy tag (p paints it elsewhere)", section: sectionClipboard,
		action: void((*model).copyTagFromSelected)},

	{keys: []string{"ctrl+p"}, desc: "switch project", section: sectionApp,
		action: void((*model).openProjectPicker)},
	{keys: []string{"i"}, prefix: 'g', display: "gi", desc: "item details", section: sectionApp,
		action: func(m *model) tea.Cmd { m.ui.Info.ScrollOffset = 0; m.ui.Modes.ToInfo(); return nil }},
	{keys: []string{"x"}, prefix: 'g', display: "gx", desc: "open link in item", section: sectionApp,
		action: func(m *model) tea.Cmd { return m.openLinksForSelected() }},
	{keys: []string{"u"}, desc: "undo", section: sectionApp,
		action: void((*model).undo)},
	{keys: []string{"ctrl+r"}, desc: "redo", section: sectionApp,
		action: void((*model).redo)},
	{keys: []string{"r"}, desc: "reload from disk", section: sectionApp,
		action: void((*model).reloadProject)},
	{keys: []string{","}, desc: "options", section: sectionApp,
		action: func(m *model) tea.Cmd {
			m.ui.Modes.ToOptions()
			m.ui.Options = OptionsState{selectedOption: 0}
			return nil
		}},
	{keys: []string{"?"}, desc: "this help", section: sectionApp,
		action: func(m *model) tea.Cmd {
			m.ui.Modes.ToggleHelp()
			if m.ui.Modes.IsHelp() {
				m.ui.Help = HelpState{}
			}
			return nil
		}},
	{keys: []string{"q"}, desc: "quit", section: sectionApp,
		action: func(m *model) tea.Cmd { return tea.Quit }},
}

var chordPrefixes = func() map[rune]struct{} {
	set := make(map[rune]struct{})
	for _, b := range normalBindings {
		if b.prefix != 0 {
			set[b.prefix] = struct{}{}
		}
	}
	return set
}()

func matchBinding(prefix rune, key string) *keyBinding {
	for i := range normalBindings {
		b := &normalBindings[i]
		if b.prefix != prefix {
			continue
		}
		for _, k := range b.keys {
			if k == key {
				return b
			}
		}
	}
	return nil
}

func (m *model) dispatchNormalKey(key string) tea.Cmd {
	// Esc undoes one thing per press, innermost first: a pending chord, then an
	// active search highlight (the vim `:nohlsearch` gesture), then a pending
	// cut. The cut goes last so that cutting, searching for the destination and
	// pressing esc only drops the highlight — the cut survives, and a second esc
	// clears it. Only a cut is dropped, never a plain copy: nothing on screen
	// marks a copy, so clearing it would be invisible.
	if key == "esc" {
		switch {
		case m.ui.Screen.PendingKey != 0:
			m.ui.Screen.PendingKey = 0
		case m.ui.Search.query != "":
			m.clearSearch()
		case m.ui.Clipboard.IsCut:
			m.ui.Clipboard = ClipboardState{}
			m.ui.Screen.StatusMsg = "Cut cancelled"
		}
		return nil
	}
	if m.ui.Screen.PendingKey != 0 {
		if b := matchBinding(m.ui.Screen.PendingKey, key); b != nil {
			m.ui.Screen.PendingKey = 0
			return b.action(m)
		}
		if c := m.matchTaskCommand(m.ui.Screen.PendingKey, key); c != nil {
			m.ui.Screen.PendingKey = 0
			return m.startTaskCommand(*c)
		}
		m.ui.Screen.PendingKey = 0
	}

	runes := []rune(key)
	if len(runes) == 1 {
		if _, isChord := chordPrefixes[runes[0]]; isChord {
			if matchBinding(0, key) == nil {
				m.ui.Screen.PendingKey = runes[0]
				return nil
			}
		}
	}

	if b := matchBinding(0, key); b != nil {
		return b.action(m)
	}
	if c := m.matchTaskCommand(0, key); c != nil {
		return m.startTaskCommand(*c)
	}
	return nil
}
