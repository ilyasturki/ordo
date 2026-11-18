package app

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/domain"
)

func emptyProject() domain.Project {
	return domain.Project{ID: "p1", Name: "Fresh", Categories: []domain.Category{}}
}

func TestEmptyProjectShowsTheHintCentred(t *testing.T) {
	m := newTestModel(t, emptyProject())
	m.rebuildPositions()
	rows := strings.Split(ansi.Strip(m.renderView()), "\n")
	require.Len(t, rows, m.ui.Screen.Height)
	at := -1
	for i, row := range rows {
		if strings.Contains(row, emptyProjectHint) {
			at = i
		}
	}
	require.NotEqual(t, -1, at, "the hint shows")
	assert.InDelta(t, m.ui.Screen.Height/2, at, 2, "it sits mid-screen")
	left := len(rows[at]) - len(strings.TrimLeft(rows[at], " "))
	assert.InDelta(t, (m.ui.Screen.Width-len(emptyProjectHint))/2, left, 1, "it is centred")
}

func TestEmptyProjectHintWrapsOnANarrowScreen(t *testing.T) {
	m := newTestModel(t, emptyProject())
	m.ui.Screen.Width = 24
	m.rebuildPositions()
	view := ansi.Strip(m.renderView())
	assert.Contains(t, view, "No categories yet: A")
	for _, row := range strings.Split(view, "\n") {
		assert.LessOrEqual(t, ansi.StringWidth(row), 24)
	}
}

func TestTheHintLeavesOnceACategoryExists(t *testing.T) {
	m := newTestModel(t, sampleProject())
	withStubStateManager(m)
	m.rebuildPositions()
	assert.NotContains(t, ansi.Strip(m.renderView()), emptyProjectHint)

	m.ui.Filter.statuses[domain.StatusCancelled] = true
	m.rebuildPositions()
	assert.NotContains(t, ansi.Strip(m.renderView()), emptyProjectHint, "a filter hiding every task is not an empty project")
}

func TestAddInAnEmptyProjectStartsItsFirstCategory(t *testing.T) {
	m := newTestModel(t, emptyProject())
	m.rebuildPositions()
	hit(m, "a")
	require.True(t, m.ui.Modes.IsEdit())
	require.Len(t, m.project.Categories, 1)

	m.ui.Edit.input.SetValue("Inbox")
	hit(m, "enter")
	assert.Equal(t, "Inbox", m.project.Categories[0].Name)
	assert.NotContains(t, ansi.Strip(m.renderView()), emptyProjectHint)
}
