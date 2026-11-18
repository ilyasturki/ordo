package app

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/app/selection"
	"github.com/ilyasturki/ordo/internal/domain"
)

func versionProject(names ...string) domain.Project {
	p := domain.Project{ID: "p1", Name: "ordo", UpdatedAt: "2026-01-01T00:00:00Z"}
	for _, n := range names {
		p.Categories = append(p.Categories, domain.Category{ID: n, Name: n, Tasks: []domain.Task{{ID: "t-" + n, Title: "task", Status: domain.StatusTodo}}})
	}
	return p
}

func TestGNOpensTheNextVersionInTheEditorOnTheNewestSide(t *testing.T) {
	for _, tc := range []struct {
		name string
		cats []string
		want []string
	}{
		{"newest first", []string{"v1.3.0", "v1.2.0"}, []string{"v1.4.0", "v1.3.0", "v1.2.0"}},
		{"oldest first", []string{"v1.2.0", "v1.3.0"}, []string{"v1.2.0", "v1.3.0", "v1.4.0"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := newTestModel(t, versionProject(tc.cats...))
			require.True(t, m.selectItem(selection.FocusCategory, "v1.3.0"))

			m = pressSweep(m, "g", "n")

			require.True(t, m.ui.Modes.IsEdit())
			assert.Equal(t, "v1.4.0", m.ui.Edit.input.Value())
			assert.Equal(t, tc.want, categoryNames(m))
			assert.Equal(t, "v1.4.0", m.project.Categories[focusIndex(t, m)].Name)

			typeText(m, "5")
			pressKey(m, tea.KeyEnter)
			require.False(t, m.ui.Modes.IsEdit())
			assert.Equal(t, "v1.4.05", m.project.Categories[focusIndex(t, m)].Name)
		})
	}
}

func TestGNFromATaskOrASubCategoryActsOnItsVersion(t *testing.T) {
	m := newBumpModel(t)
	require.True(t, m.selectTaskByID("u"))

	m = pressSweep(m, "g", "n")

	require.True(t, m.ui.Modes.IsEdit())
	assert.Equal(t, "v1.1.0", m.ui.Edit.input.Value())
	assert.Equal(t, "v1.1.0", m.project.Categories[focusIndex(t, m)].Name)
	assert.Empty(t, m.project.Categories[focusIndex(t, m)].ParentID)
}

func TestGNLandsBeforeWhatFollowsTheVersions(t *testing.T) {
	m := newBumpModel(t)
	require.True(t, m.selectItem(selection.FocusCategory, "v012"))

	m = pressSweep(m, "g", "n")

	assert.Equal(t, "v0.2.0", m.ui.Edit.input.Value())
	got := categoryNames(m)
	assert.Equal(t, []string{"v1.0.0", "ui", "v0.2.0", "FUTURE"}, got[len(got)-5:len(got)-1])
}

func TestGNIsOneUndoStep(t *testing.T) {
	m := newTestModel(t, versionProject("v1.3.0", "v1.2.0"))
	require.True(t, m.selectItem(selection.FocusCategory, "v1.3.0"))
	m = pressSweep(m, "g", "n")
	typeText(m, "1")
	pressKey(m, tea.KeyEnter)

	m = pressSweep(m, "u")

	assert.Equal(t, []string{"v1.3.0", "v1.2.0"}, categoryNames(m))
}

func TestGNOnAnEditThatCannotSaveKeepsTheClashChooser(t *testing.T) {
	m := newTestModel(t, versionProject("v1.3.0", "v1.2.0"))
	require.True(t, m.selectItem(selection.FocusCategory, "v1.3.0"))
	m = pressSweep(m, "g", "n")
	for range len("v1.4.0") {
		pressKey(m, tea.KeyBackspace)
	}
	typeText(m, "v1.2.0")

	pressKey(m, tea.KeyEnter)

	assert.True(t, m.ui.Modes.IsEdit())
	assert.NotEmpty(t, m.ui.Edit.clash)
}

func TestGNWithoutAVersionSaysSo(t *testing.T) {
	m := newBumpModel(t)
	require.True(t, m.selectItem(selection.FocusCategory, "mkt"))
	before := categoryNames(m)

	m = pressSweep(m, "g", "n")

	assert.False(t, m.ui.Modes.IsEdit())
	assert.Equal(t, "No version in this name", m.ui.Screen.StatusMsg)
	assert.Equal(t, before, categoryNames(m))
	assert.False(t, m.ui.History.CanUndo(), "a refused edit leaves no undo step")
}

func TestGNOnTheProjectRowAsksForACategory(t *testing.T) {
	m := newBumpModel(t)
	m.ui.Selection.MoveTo(0)

	m = pressSweep(m, "g", "n")

	assert.False(t, m.ui.Modes.IsEdit())
	assert.Equal(t, "Put the cursor on a category to add its next version", m.ui.Screen.StatusMsg)
}

func TestEscInTheGNEditorSavesTheName(t *testing.T) {
	m := newTestModel(t, versionProject("v1.3.0", "v1.2.0"))
	require.True(t, m.selectItem(selection.FocusCategory, "v1.3.0"))
	m = pressSweep(m, "g", "n")
	typeText(m, " - plan")

	pressKey(m, tea.KeyEscape)

	require.False(t, m.ui.Modes.IsEdit())
	assert.Equal(t, []string{"v1.4.0 - plan", "v1.3.0", "v1.2.0"}, categoryNames(m))
	m = pressSweep(m, "u")
	assert.Equal(t, []string{"v1.3.0", "v1.2.0"}, categoryNames(m))
}

func TestEscSavesAnOrdinaryNewCategoryToo(t *testing.T) {
	m := newTestModel(t, versionProject("v1.3.0"))
	require.True(t, m.selectItem(selection.FocusCategory, "v1.3.0"))
	m.startAddingCategory()
	typeText(m, "draft")

	pressKey(m, tea.KeyEscape)

	assert.Equal(t, []string{"v1.3.0", "draft"}, categoryNames(m))
}

func TestTheShortcutBarOffersGNOnAVersionOnly(t *testing.T) {
	m := newBumpModel(t)
	require.True(t, m.selectItem(selection.FocusCategory, "v012"))
	assert.Contains(t, ansi.Strip(m.renderShortcutBar()), "gn next version")
	require.True(t, m.selectTaskByID("u"))
	assert.Contains(t, ansi.Strip(m.renderShortcutBar()), "gn next version")
	require.True(t, m.selectItem(selection.FocusCategory, "mkt"))
	assert.NotContains(t, ansi.Strip(m.renderShortcutBar()), "gn next version")
}

func TestTheHelpListsGN(t *testing.T) {
	var found bool
	for _, s := range builtinHelpSections {
		for _, e := range s.entries {
			found = found || (e.keys == "gn" && e.desc == "add the next version's category")
		}
	}
	assert.True(t, found)
}

func categoryNames(m *model) []string {
	names := make([]string, len(m.project.Categories))
	for i, c := range m.project.Categories {
		names[i] = c.Name
	}
	return names
}
