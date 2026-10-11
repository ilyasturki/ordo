package app

import (
	"encoding/json"
	"testing"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/phasionary/internal/ui"
)

// helpFiltering reconciles the cursor and scroll the way the key handler would.
func helpFiltering(t *testing.T, m *model, query string) {
	t.Helper()
	m.ui.Modes.ToHelp()
	ti := textinput.New()
	ti.SetValue(query)
	m.ui.Help = HelpState{Filtering: true, Filter: ti}
	m.ensureHelpVisible()
}

func helpExpand(t *testing.T, m *model) {
	t.Helper()
	m.ui.Modes.ToHelp()
	m.ui.Help = HelpState{}
	m.ui.Screen.HelpExpanded = true
	m.ensureHelpVisible()
}

func helpEntries(b helpBody) []helpEntry {
	out := make([]helpEntry, 0, len(b.targets))
	for _, target := range b.targets {
		out = append(out, *b.entryAt(target))
	}
	return out
}

func TestHelpFilter_SlashEntersFiltering(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Modes.ToHelp()

	after, _ := m.handleHelpKey(tea.KeyPressMsg{Text: "/"})
	am := after.(model)
	assert.True(t, am.ui.Help.Filtering)
	assert.Equal(t, "", am.ui.Help.Filter.Value(), "filter starts empty")
	assert.Equal(t, 0, am.ui.Help.Focused)
}

func TestHelpFilter_EmptyQueryShowsTheCard(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Modes.ToHelp()

	assert.Equal(t, helpCardLines, m.helpBody().lines,
		"an unfiltered, unexpanded dialog shows the essentials card")
}

func TestHelpFilter_NarrowsByLabel(t *testing.T) {
	entries := helpEntries(filteredHelpBody(t, "options"))
	require.Len(t, entries, 1, "only the options binding matches its label")
	assert.Equal(t, "options", entries[0].desc)
	assert.True(t, entries[0].runnable)
}

func TestHelpFilter_MatchesByShortcutKey(t *testing.T) {
	// "?" appears only as the help binding's own key, so keying it surfaces
	// exactly that row — which is not runnable, since firing it from inside the
	// dialog would only reopen the dialog.
	entries := helpEntries(filteredHelpBody(t, "?"))
	require.Len(t, entries, 1)
	assert.Equal(t, "this help", entries[0].desc)
	assert.False(t, entries[0].runnable)
}

func TestHelpFilter_SearchesSectionsTheCardHides(t *testing.T) {
	// "markdown" exists as the y/Y clipboard binding and in the visual-mode
	// reference, which the card hides. The filter finds both from the card.
	entries := helpEntries(filteredHelpBody(t, "markdown"))
	require.Len(t, entries, 2)
	assert.True(t, entries[0].runnable)
	assert.False(t, entries[1].runnable, "visual-mode shortcuts are reference-only")
}

func TestHelpFilter_EnterRunsFocusedMatchAndCloses(t *testing.T) {
	m := newTestModel(t, sampleProject())
	helpFiltering(t, m, "options")

	after, _ := m.handleHelpKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	am := after.(model)
	assert.True(t, am.ui.Modes.IsOptions(), "Enter fires the matched binding")
}

func TestHelpFilter_EnterOnReferenceRowDoesNothing(t *testing.T) {
	m := newTestModel(t, sampleProject())
	helpFiltering(t, m, "swap which end")

	after, cmd := m.handleHelpKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	am := after.(model)
	assert.Nil(t, cmd)
	assert.True(t, am.ui.Modes.IsHelp(), "a reference-only row keeps the dialog open")
}

func TestHelpFilter_EscClearsFilterButKeepsHelpOpen(t *testing.T) {
	m := newTestModel(t, sampleProject())
	helpFiltering(t, m, "options")

	after, _ := m.handleHelpKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	am := after.(model)
	assert.False(t, am.ui.Help.Filtering, "Esc leaves the filter")
	assert.True(t, am.ui.Modes.IsHelp(), "Esc stays in the help dialog")
	assert.Equal(t, "", am.ui.Help.Filter.Value())
}

func TestHelpFilter_NoMatchShowsEmptyState(t *testing.T) {
	body := filteredHelpBody(t, "zzzznomatch")
	assert.Empty(t, body.targets, "nothing to run")
	require.Len(t, body.lines, 1)
	assert.Equal(t, "nothing matches that", body.lines[0][0].note)
}

func TestHelpFilter_MatchesAreHighlighted(t *testing.T) {
	m := newTestModel(t, sampleProject())
	entry := helpEntry{keys: "u", desc: "undo"}
	cell := m.renderHelpCell(helpCell{entry: &entry}, m.dialogWidth(), helpKeyCol, false, "undo")
	assert.Contains(t, cell, ui.SearchMatchStyle.Render("undo"),
		"an unfocused matching row carries the search-highlight style")
}

func filteredHelpBody(t *testing.T, query string) helpBody {
	t.Helper()
	m := newTestModel(t, sampleProject())
	helpFiltering(t, m, query)
	return m.helpBody()
}

func TestHelp_CardShortcutsAllResolveToBindings(t *testing.T) {
	for _, groups := range helpCardColumns {
		for _, g := range groups {
			for _, s := range g.shortcuts {
				e := s.entry()
				assert.Truef(t, e.runnable,
					"card shortcut %q (%s) resolves to no binding", s.key, s.desc)
			}
		}
	}
}

func TestHelp_EveryShortcutIsReachable(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Screen.Width, m.ui.Screen.Height = 120, 30
	helpExpand(t, m)

	body := m.helpBody()
	require.NotEmpty(t, body.targets)
	last := body.targets[len(body.targets)-1]
	assert.Equal(t, len(body.lines)-1, last.line,
		"the cursor reaches the final line of the reference")

	for range len(body.targets) {
		m.moveHelpFocus(1)
	}
	assert.Equal(t, len(body.lines)-m.helpBodyHeight(), m.ui.Help.ScrollOffset,
		"pressing j to the end scrolls that line into view")
}

func TestHelp_TitleRowDropsThePositionWhenEverythingFits(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Screen.Width, m.ui.Screen.Height = 120, 200
	helpExpand(t, m)

	body := m.helpBody()
	require.LessOrEqual(t, len(body.lines), m.helpBodyHeight())
	assert.NotContains(t, m.helpTitleRow(body, m.dialogWidth()), " of ",
		"no scroll position is shown when the whole reference is on screen")
}

func TestHelp_GeometryIsStableAcrossFiltering(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Screen.Width, m.ui.Screen.Height = 120, 40
	m.ui.Modes.ToHelp()

	before := m.helpView()
	helpFiltering(t, m, "tag")
	during := m.helpView()
	helpFiltering(t, m, "zzzznomatch")
	empty := m.helpView()

	assert.Equal(t, lipgloss.Width(before), lipgloss.Width(during), "width holds while filtering")
	assert.Equal(t, lipgloss.Width(before), lipgloss.Width(empty), "width holds with no matches")
	assert.Equal(t, lipgloss.Height(before), lipgloss.Height(during), "height holds while filtering")
	assert.Equal(t, lipgloss.Height(before), lipgloss.Height(empty), "height holds with no matches")
}

func TestHelp_ToggleLastsTheSession(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Screen.Width, m.ui.Screen.Height = 120, 40
	m.ui.Modes.ToHelp()
	require.False(t, m.helpExpanded(), "help opens on the essentials card")

	after, _ := m.handleHelpKey(tea.KeyPressMsg{Text: "a"})
	am := after.(model)
	assert.True(t, am.helpExpanded())
	assert.Equal(t, 0, am.ui.Help.Focused)

	reopened, _ := am.handleKeyMsg(tea.KeyPressMsg{Text: "?"})
	reopened, _ = reopened.(model).handleKeyMsg(tea.KeyPressMsg{Text: "?"})
	assert.True(t, reopened.(model).helpExpanded(), "reopening help keeps the choice")

	data, err := json.Marshal(am.deps.CfgManager.Get())
	require.NoError(t, err)
	assert.NotContains(t, string(data), "help", "the choice never reaches the config")

	back, _ := am.handleHelpKey(tea.KeyPressMsg{Text: "a"})
	assert.False(t, back.(model).helpExpanded())
}
