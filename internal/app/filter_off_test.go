package app

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/app/selection"
	"github.com/ilyasturki/ordo/internal/data"
	"github.com/ilyasturki/ordo/internal/domain"
)

func TestFTurnsTheFilterOffAndBackOnKeepingItsValues(t *testing.T) {
	m := newTodoFilteredModel(t)
	stub := m.deps.StateManager.(*stubStateManager)
	filtered := rowIDs(m)

	m, _ = pressKeys(m, "F")
	assert.False(t, m.ui.Filter.HasActiveFilter())
	assert.True(t, m.ui.Filter.IsStatusEnabled(domain.StatusTodo), "the values stay")
	assert.Equal(t, []string{"project:", "category:c1", "task:t1", "task:t2", "task:t3", "category:c2", "task:t4", "task:t5"}, rowIDs(m))
	assert.Equal(t, "Filter off, F turns it back on", m.ui.Screen.StatusMsg)
	assert.Equal(t, data.Filter{Off: true, Statuses: []string{"todo"}}, stub.filters["p1"])
	assert.NotContains(t, strings.Join(screen(m), "\n"), "[filtered]")
	assert.Contains(t, ansi.Strip(m.renderShortcutBar()), "f filter  ·  F filter on")

	m, _ = pressKeys(m, "F")
	assert.True(t, m.ui.Filter.HasActiveFilter())
	assert.Equal(t, filtered, rowIDs(m))
	assert.Equal(t, "Filter on", m.ui.Screen.StatusMsg)
	assert.Equal(t, data.Filter{Statuses: []string{"todo"}}, stub.filters["p1"])
	lineWith(t, screen(m), "[filtered]")
	assert.Contains(t, ansi.Strip(m.renderShortcutBar()), "f filtered  ·  F off")
}

func TestFWithoutAFilterSaysSo(t *testing.T) {
	m := newTestModel(t, sampleProject())
	stub := withStubStateManager(m)

	m, _ = pressKeys(m, "F")

	assert.Equal(t, "No filter to turn off: f sets one", m.ui.Screen.StatusMsg)
	assert.False(t, m.ui.Filter.IsOff())
	assert.NotContains(t, stub.filters, "p1")
}

func TestFReleasesThePeekAndTheHeldRows(t *testing.T) {
	m := newTodoFilteredModel(t)
	selectRow(t, m, selection.FocusTask, "t1")
	m, _ = pressKeys(m, "space")
	require.True(t, m.ui.Filter.holding())
	selectRow(t, m, selection.FocusHidden, "c1")
	m, _ = pressKeys(m, "enter")
	require.True(t, m.ui.Filter.peeks("c1"))

	m, _ = pressKeys(m, "F")
	assert.False(t, m.ui.Filter.holding())
	assert.False(t, m.ui.Filter.peeks("c1"))
	assert.NotContains(t, rowIDs(m), "hidden:c1")
	assert.NotContains(t, strings.Join(screen(m), "\n"), "(filtered out)")

	m, _ = pressKeys(m, "F")
	assert.Contains(t, rowIDs(m), "hidden:c1", "back on, the hidden row comes back")
	assert.NotContains(t, rowIDs(m), "task:t1", "and the task the edit took out stays out")
}

func TestAFilterOffHidesNothingAndRefusesNoRow(t *testing.T) {
	m := newNestedModel(t)
	m.ui.Filter.statuses[domain.StatusCompleted] = true
	m.ui.Filter.off = true
	m.rebuildPositions()
	selectRow(t, m, selection.FocusTask, "sets")

	m.startAddingTask()

	assert.True(t, m.ui.Modes.IsEdit(), "an add the filter would refuse goes through while it is off")
	assert.True(t, m.ui.Filter.CategoryVisible("other", ""))
	assert.False(t, tuiView{m}.Filtering())
}

func TestAChangeInTheDialogTurnsTheFilterOn(t *testing.T) {
	m := newTodoFilteredModel(t)
	m, _ = pressKeys(m, "F")
	m.ui.Modes.ToFilter()
	assert.Contains(t, ansi.Strip(m.filterHubView()), "off, a change turns it on")

	m.handleFilterKey(keyPress("enter"))
	m.handleFilterKey(keyPress("j"))
	m.handleFilterKey(keyPress("space"))

	assert.True(t, m.ui.Filter.HasActiveFilter())
	assert.True(t, m.ui.Filter.IsStatusEnabled(domain.StatusTodo))
	assert.True(t, m.ui.Filter.IsStatusEnabled(domain.StatusInProgress))
}

func TestClearAllForgetsAFilterTurnedOff(t *testing.T) {
	m := newTodoFilteredModel(t)
	stub := m.deps.StateManager.(*stubStateManager)
	m, _ = pressKeys(m, "F")
	m.ui.Modes.ToFilter()
	m.ui.Filter.hubSelected = FilterHubClearAll

	m.handleFilterKey(keyPress("enter"))

	assert.False(t, m.ui.Filter.HasValues())
	assert.False(t, m.ui.Filter.IsOff())
	assert.NotContains(t, stub.filters, "p1")
}

func TestAFilterTurnedOffElsewhereApplies(t *testing.T) {
	m := newTodoFilteredModel(t)
	off := filterStateFrom(data.Filter{Off: true, Statuses: []string{"todo"}})
	require.False(t, m.ui.Filter.sameValues(off), "off alone is a change")

	m.ui.Filter.takeValues(off)
	m.rebuildPositions()

	assert.True(t, m.ui.Filter.IsOff())
	assert.Contains(t, rowIDs(m), "task:t2")
	assert.Equal(t, data.Filter{Off: true, Statuses: []string{"todo"}}, m.ui.Filter.saved())
}
