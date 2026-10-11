package app

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/phasionary/internal/domain"
)

// namedPickerModel builds a picker-mode model whose projects have the given
// names (IDs are the names), on a roomy fixed screen.
func namedPickerModel(t *testing.T, names ...string) *model {
	t.Helper()
	m := newTestModel(t, sampleProject())
	m.ui.Screen.Width = 160
	m.ui.Screen.Height = 24
	projects := make([]domain.Project, len(names))
	for i, n := range names {
		projects[i] = domain.Project{ID: n, Name: n}
	}
	m.ui.Picker = ProjectPickerState{projects: projects}
	m.ui.Modes.ToProjectPicker()
	return m
}

func TestPickerFilter_SlashEntersFiltering(t *testing.T) {
	m := namedPickerModel(t, "Phasionary", "Billing", "Metaphase")
	m.ui.Picker.selected = 2

	after, _ := m.handleProjectPickerKey(tea.KeyPressMsg{Text: "/"})
	assert.True(t, after.ui.Picker.filtering)
	assert.False(t, after.ui.Picker.onNew)
	assert.Equal(t, 0, after.ui.Picker.selected, "filtering starts on the first project")
	assert.Len(t, after.ui.Picker.allProjects, 3, "full list is snapshotted")
}

func TestPickerFilter_SlashNoopWithoutProjects(t *testing.T) {
	m := newPickerModel(t, 0, 160, 24)
	after, _ := m.handleProjectPickerKey(tea.KeyPressMsg{Text: "/"})
	assert.False(t, after.ui.Picker.filtering, "nothing to filter with zero projects")
}

func TestPickerFilter_NarrowsBySmartcaseName(t *testing.T) {
	m := namedPickerModel(t, "Phasionary", "Billing", "Metaphase", "phase-two")
	m.ui.Picker.startFiltering()
	visible := m.pickerVisibleCount()

	// "pha" is case-insensitive (no uppercase), matching three names.
	m.ui.Picker.query = "pha"
	m.ui.Picker.applyFilter(visible)
	names := pickerNames(m.ui.Picker.projects)
	assert.Equal(t, []string{"Phasionary", "Metaphase", "phase-two"}, names)
	assert.Equal(t, 0, m.ui.Picker.selected)
	assert.False(t, m.ui.Picker.onNew)

	// Uppercase makes it case-sensitive (smartcase), matching only "Phasionary".
	m.ui.Picker.query = "Pha"
	m.ui.Picker.applyFilter(visible)
	assert.Equal(t, []string{"Phasionary"}, pickerNames(m.ui.Picker.projects))

	// Empty query restores the full snapshot.
	m.ui.Picker.query = ""
	m.ui.Picker.applyFilter(visible)
	assert.Len(t, m.ui.Picker.projects, 4)
}

func TestPickerFilter_TypingThroughHandlerNarrowsLive(t *testing.T) {
	m := namedPickerModel(t, "Phasionary", "Billing", "Metaphase")
	m.ui.Picker.startFiltering()

	after := m
	for _, ch := range []string{"p", "h", "a"} {
		var next model
		next, _ = after.handleProjectPickerKey(tea.KeyPressMsg{Text: ch})
		after = &next
	}
	assert.Equal(t, "pha", after.ui.Picker.query)
	assert.Equal(t, []string{"Phasionary", "Metaphase"}, pickerNames(after.ui.Picker.projects))
}

func TestPickerFilter_EscClearsAndKeepsHighlightedProject(t *testing.T) {
	m := namedPickerModel(t, "Phasionary", "Billing", "Metaphase")
	m.ui.Picker.startFiltering()
	visible := m.pickerVisibleCount()

	m.ui.Picker.query = "eta" // matches only "Metaphase"
	m.ui.Picker.applyFilter(visible)
	require.Equal(t, []string{"Metaphase"}, pickerNames(m.ui.Picker.projects))

	after, _ := m.handleProjectPickerKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	assert.False(t, after.ui.Picker.filtering)
	assert.True(t, after.ui.Modes.IsProjectPicker(), "clearing the filter stays in the picker")
	assert.Len(t, after.ui.Picker.projects, 3, "full list restored")
	assert.Equal(t, "Metaphase", after.ui.Picker.projects[after.ui.Picker.selected].Name,
		"cursor stays on the project that was highlighted")
}

func TestPickerFilter_EnterOnNoMatchIsNoop(t *testing.T) {
	m := namedPickerModel(t, "Phasionary", "Billing")
	m.ui.Picker.startFiltering()
	m.ui.Picker.query = "zzz"
	m.ui.Picker.applyFilter(m.pickerVisibleCount())
	require.Empty(t, m.ui.Picker.projects)

	after, _ := m.handleProjectPickerKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	assert.True(t, after.ui.Picker.filtering, "Enter with no matches keeps filtering")
	assert.True(t, after.ui.Modes.IsProjectPicker())
}

func TestPickerFilter_ArrowsStayWithinProjects(t *testing.T) {
	m := namedPickerModel(t, "Alpha", "Alto", "Alien")
	m.ui.Picker.startFiltering()
	m.ui.Picker.query = "Al"
	m.ui.Picker.applyFilter(m.pickerVisibleCount())
	require.Len(t, m.ui.Picker.projects, 3)

	// Up from the first match never lands on the hidden New Project row.
	after, _ := m.handleProjectPickerKey(tea.KeyPressMsg{Code: tea.KeyUp})
	assert.False(t, after.ui.Picker.onNew)
	assert.Equal(t, 0, after.ui.Picker.selected)

	// Down steps through the matches and clamps at the last one.
	after, _ = after.handleProjectPickerKey(tea.KeyPressMsg{Code: tea.KeyDown})
	assert.Equal(t, 1, after.ui.Picker.selected)
	after, _ = after.handleProjectPickerKey(tea.KeyPressMsg{Code: tea.KeyDown})
	after, _ = after.handleProjectPickerKey(tea.KeyPressMsg{Code: tea.KeyDown})
	assert.Equal(t, 2, after.ui.Picker.selected)
	assert.False(t, after.ui.Picker.onNew)
}

func pickerNames(projects []domain.Project) []string {
	names := make([]string, len(projects))
	for i, p := range projects {
		names[i] = p.Name
	}
	return names
}
