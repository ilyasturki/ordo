package app

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"phasionary/internal/app/selection"
	"phasionary/internal/domain"
)

const shippedAt = "2026-08-28T15:23:00Z"

func versionsProject() domain.Project {
	return domain.Project{
		ID:   "p1",
		Name: "P",
		Categories: []domain.Category{
			{ID: "inbox", Name: "next", Tasks: []domain.Task{
				{ID: "n1", Title: "Idea", Status: domain.StatusTodo},
			}},
			{ID: "v3", Name: "v0.9.1", Tasks: []domain.Task{
				{ID: "a1", Title: "Menu", Status: domain.StatusCompleted},
				{ID: "a2", Title: "Theme", Status: domain.StatusInProgress},
				{ID: "a3", Title: "Docs", Status: domain.StatusTodo},
			}},
			{ID: "v2", Name: "v0.9.0", ReleasedAt: shippedAt, Tasks: []domain.Task{
				{ID: "b1", Title: "Done", Status: domain.StatusCompleted},
				{ID: "b2", Title: "Left", Status: domain.StatusInProgress},
			}},
			{ID: "v1", Name: "v0.8.6", ReleasedAt: shippedAt, Tasks: []domain.Task{
				{ID: "c1t", Title: "Old", Status: domain.StatusCompleted},
			}},
		},
	}
}

func newVersionsModel(t *testing.T) (*model, *stubStateManager) {
	t.Helper()
	m := newTestModel(t, versionsProject())
	stub := withStubStateManager(m)
	m.rebuildPositions()
	return m, stub
}

func visibleIDs(m *model) []string {
	var ids []string
	for _, p := range m.ui.Selection.Positions() {
		if p.Kind != selection.FocusProject {
			ids = append(ids, p.ID)
		}
	}
	return ids
}

func TestShippedVersionsFoldByDefault(t *testing.T) {
	m, _ := newVersionsModel(t)
	assert.Equal(t, []string{"inbox", "n1", "v3", "a1", "a2", "a3", "v2", "v1"}, visibleIDs(m))
}

func TestHandUnfoldingAShippedVersionPersists(t *testing.T) {
	m, stub := newVersionsModel(t)
	require.True(t, m.selectItem(selection.FocusCategory, "v2"))

	m.toggleFold()
	assert.False(t, m.ui.Fold.IsFolded("v2"))
	assert.Equal(t, []string{"v2"}, stub.unfolded["p1"])
	assert.Empty(t, stub.folded["p1"], "a shipped version never lands in folded")

	reopened := loadFoldState(stub, m.project)
	assert.False(t, reopened.IsFolded("v2"), "the hand-unfold survives a restart")
	assert.True(t, reopened.IsFolded("v1"))

	m.toggleFold()
	assert.True(t, m.ui.Fold.IsFolded("v2"))
	assert.Empty(t, stub.unfolded["p1"])
}

func TestShipFoldsAndUndoReopens(t *testing.T) {
	m, _ := newVersionsModel(t)
	require.True(t, m.selectItem(selection.FocusTask, "a2"))

	m.toggleShippedSelected()

	v := m.project.Categories[1]
	assert.True(t, v.Shipped())
	assert.True(t, m.ui.Fold.IsFolded("v3"))
	pos, _ := m.selectedPosition()
	assert.Equal(t, "v3", pos.ID, "the cursor falls back to the folded version's header")
	assert.Equal(t, "Shipped v0.9.1 with 2 open tasks", m.ui.Screen.StatusMsg)

	m.undo()
	assert.False(t, m.project.Categories[1].Shipped())
	assert.False(t, m.ui.Fold.IsFolded("v3"))
}

func TestShipReopensAVersionNestedUnderACategory(t *testing.T) {
	m := newTestModel(t, domain.Project{ID: "p1", Categories: []domain.Category{
		{ID: "area", Name: "FUTURE"}, {ID: "old", Name: "v0.1.0", ParentID: "area", ReleasedAt: shippedAt},
	}})
	withStubStateManager(m)
	m.rebuildPositions()
	require.True(t, m.selectItem(selection.FocusCategory, "old"))

	m.toggleShippedSelected()

	assert.False(t, m.project.Categories[1].Shipped())
	assert.Equal(t, "Reopened v0.1.0", m.ui.Screen.StatusMsg)
}

func TestShipRefusesNonVersionsAndReopens(t *testing.T) {
	m, _ := newVersionsModel(t)
	require.True(t, m.selectItem(selection.FocusTask, "n1"))
	m.toggleShippedSelected()
	assert.False(t, m.project.Categories[0].Shipped())
	assert.Contains(t, m.ui.Screen.StatusMsg, "Not a version")
	assert.False(t, m.ui.History.CanUndo())

	require.True(t, m.selectItem(selection.FocusCategory, "v2"))
	m.toggleShippedSelected()
	assert.False(t, m.project.Categories[2].Shipped())
	assert.Equal(t, "Reopened v0.9.0", m.ui.Screen.StatusMsg)
	assert.False(t, m.ui.Fold.IsFolded("v2"))
}

func TestShippingAHandUnfoldedVersionFoldsIt(t *testing.T) {
	m, _ := newVersionsModel(t)
	require.True(t, m.selectItem(selection.FocusCategory, "v2"))
	m.toggleFold()
	m.toggleShippedSelected() // reopen
	m.toggleShippedSelected() // ship again
	assert.True(t, m.ui.Fold.IsFolded("v2"))
}

func TestShippingAHandFoldedVersionKeepsItFolded(t *testing.T) {
	m, _ := newVersionsModel(t)
	require.True(t, m.selectItem(selection.FocusCategory, "v3"))
	m.toggleFold()
	m.toggleShippedSelected()
	assert.True(t, m.ui.Fold.IsFolded("v3"))
	m.toggleShippedSelected()
	assert.True(t, m.ui.Fold.IsFolded("v3"), "it was folded by hand before shipping")
}

func TestUnfoldAndFoldEverythingCoverShippedVersions(t *testing.T) {
	m, _ := newVersionsModel(t)
	m.unfoldAll()
	assert.False(t, m.ui.Fold.IsFolded("v1"))
	assert.False(t, m.ui.Fold.IsFolded("v2"))
	m.foldAll()
	for _, id := range []string{"inbox", "v3", "v2", "v1"} {
		assert.True(t, m.ui.Fold.IsFolded(id), id)
	}
}

func TestVisualShipShipsEveryOpenVersionInRange(t *testing.T) {
	m := newTestModel(t, domain.Project{ID: "p1", Categories: []domain.Category{
		{ID: "x", Name: "FUTURE"}, {ID: "a", Name: "v0.2.0"}, {ID: "b", Name: "v0.1.0", ReleasedAt: shippedAt},
	}})
	withStubStateManager(m)
	m.rebuildPositions()
	require.True(t, m.selectItem(selection.FocusCategory, "x"))
	m.enterVisualMode()
	m.visualMoveCursor(2)

	m.visualToggleShipped()

	assert.False(t, m.project.Categories[0].Shipped(), "FUTURE is not a version")
	assert.True(t, m.project.Categories[1].Shipped())
	assert.Equal(t, shippedAt, m.project.Categories[2].ReleasedAt, "an already shipped one keeps its date")
	assert.Equal(t, "Shipped v0.2.0", m.ui.Screen.StatusMsg)
	assert.False(t, m.ui.Modes.IsVisual())
}

func TestVersionHeaders(t *testing.T) {
	p := versionsProject()
	fold := NewFoldState()
	fold.SetShipped(p.Categories)
	now := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)

	open := headerOf(&p, 1, &fold, false)
	assert.Equal(t, "1/3 · 1 in progress", open.progress)
	assert.False(t, open.shipped)
	assert.Empty(t, open.warning)

	shipped := headerOf(&p, 2, &fold, false)
	assert.True(t, shipped.shipped)
	assert.True(t, shipped.folded)
	assert.Equal(t, "shipped "+shipDate(shippedAt, now)+" · 1/2", shipped.progress)
	assert.Equal(t, "! 1 open", shipped.warning)

	clean := headerOf(&p, 3, &fold, false)
	assert.Empty(t, clean.warning)

	inbox := headerOf(&p, 0, &fold, false)
	assert.Empty(t, inbox.progress, "a plain category keeps its plain header")

	assert.Equal(t, "28 Aug", shipDate(shippedAt, now))
	assert.Equal(t, "28 Aug 2026", shipDate(shippedAt, now.AddDate(1, 0, 0)))
}

func TestShippedVersionsStackWithoutGaps(t *testing.T) {
	m, _ := newVersionsModel(t)
	rendered := ansi.Strip(m.renderView())
	lines := strings.Split(rendered, "\n")
	var v2, v1 int
	for i, l := range lines {
		switch {
		case strings.Contains(l, "v0.9.0"):
			v2 = i
		case strings.Contains(l, "v0.8.6"):
			v1 = i
		}
	}
	require.NotZero(t, v2)
	assert.Equal(t, v2+1, v1, "folded shipped versions sit on consecutive rows")
	assert.Contains(t, lines[v2], "▶ v0.9.0 [/] shipped 28 Aug · 1/2 ! 1 open")
	assert.NotContains(t, rendered, "(folded)")
}

func TestShortcutBarOffersShipOnlyWithVersions(t *testing.T) {
	m, _ := newVersionsModel(t)
	m.ui.Screen.Width = 200
	assert.Contains(t, ansi.Strip(m.renderShortcutBar()), "s ship")

	plain := newTestModel(t, sampleProject())
	plain.ui.Screen.Width = 200
	assert.NotContains(t, ansi.Strip(plain.renderShortcutBar()), "ship")
}

func TestSearchRestoresShippedFolds(t *testing.T) {
	m, _ := newVersionsModel(t)
	m.startSearch()
	m.ui.Search.input.SetValue("Left")
	m.previewSearch()
	assert.False(t, m.ui.Fold.IsFolded("v2"), "the preview reveals the match")
	m.cancelSearch()
	assert.True(t, m.ui.Fold.IsFolded("v2"))
}

func TestAOnAVersionProposesTheNextPatch(t *testing.T) {
	m := newMoveModel(t)
	require.True(t, m.selectItem(selection.FocusTask, "a1"))

	m.startAddingCategory()

	require.True(t, m.ui.Modes.IsEdit())
	assert.Equal(t, "v0.10.1", m.ui.Edit.input.Value())
	assert.Equal(t, m.ui.Edit.newItemID, m.project.Categories[1].ID, "above the newest version, below the inbox")
	m.finishEditing()
	assert.Equal(t, "v0.10.1", m.project.Categories[1].Name)
	assert.Equal(t, "v0.10.0", m.project.Categories[2].Name)
}

func TestAOnAVersionAppendsInAnOldestFirstProject(t *testing.T) {
	m := newTestModel(t, domain.Project{ID: "p1", Categories: []domain.Category{
		{ID: "a", Name: "v0.0.1"}, {ID: "b", Name: "v0.1.0 - app polish"}, {ID: "c", Name: "MARKETING"},
	}})
	m.rebuildPositions()
	require.True(t, m.selectItem(selection.FocusCategory, "a"))
	m.startAddingCategory()
	assert.Equal(t, "v0.1.1", m.ui.Edit.input.Value())
	assert.Equal(t, m.ui.Edit.newItemID, m.project.Categories[3].ID)
}

func TestAOutsideAVersionStaysPlain(t *testing.T) {
	m := newMoveModel(t)
	for _, at := range []struct {
		kind selection.FocusKind
		id   string
	}{{selection.FocusTask, "n2"}, {selection.FocusCategory, "v91ui"}} {
		require.True(t, m.selectItem(at.kind, at.id))
		m.startAddingCategory()
		assert.Empty(t, m.ui.Edit.input.Value(), at.id)
		m.cancelEditing()
	}
	assert.Len(t, m.project.Categories, 6)
}
