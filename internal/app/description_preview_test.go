package app

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/app/components"
	"github.com/ilyasturki/ordo/internal/app/selection"
	"github.com/ilyasturki/ordo/internal/domain"
)

func longDescriptionProject() domain.Project {
	return domain.Project{
		ID:   "p1",
		Name: "P",
		Categories: []domain.Category{{
			ID:   "c1",
			Name: "Cat A",
			Tasks: []domain.Task{
				{ID: "t1", Title: "Alpha", Status: domain.StatusTodo,
					Description: "one\ntwo\nthree\nfour\nfive\nsix"},
			},
		}},
	}
}

func findDescriptionItem(t *testing.T, layout *Layout, catIdx, taskIdx int) LayoutItem {
	t.Helper()
	for _, item := range layout.Items {
		if item.Kind == LayoutDescription && item.CategoryIndex == catIdx && item.TaskIndex == taskIdx {
			return item
		}
	}
	t.Fatalf("no description layout item for (%d, %d)", catIdx, taskIdx)
	return LayoutItem{}
}

func TestDescriptionPreview_TruncatesUnfocusedRow(t *testing.T) {
	m := newTestModel(t, longDescriptionProject())
	m.ui.Screen.ExpandDescriptions = true
	m.rebuildPositions()
	require.True(t, m.selectTaskIdx(0, 0)) // cursor on the task row, not the description

	layout := m.buildLayout()
	item := findDescriptionItem(t, layout, 0, 0)
	// 6 lines cap to 3 shown + 1 tail row.
	assert.Equal(t, components.DescriptionPreviewLines+1, item.Height)

	rendered := m.renderLayoutItem(item)
	assert.Contains(t, rendered, "+3 more lines")
	assert.NotContains(t, rendered, "six")
}

func numberedLines(n int) string {
	lines := make([]string, n)
	for i := range lines {
		lines[i] = fmt.Sprintf("line%02d", i+1)
	}
	return strings.Join(lines, "\n")
}

func focusDescription(t *testing.T, m *model) {
	t.Helper()
	m.ui.Screen.ExpandDescriptions = true
	m.rebuildPositions()
	require.True(t, m.ui.Selection.SelectByPredicate(func(p selection.Position) bool {
		return p.Kind == selection.FocusDescription && p.TaskIndex == 0
	}))
}

func TestDescriptionPreview_FocusedCapsAtTenRows(t *testing.T) {
	p := longDescriptionProject()
	p.Categories[0].Tasks[0].Description = numberedLines(30)
	m := newTestModel(t, p)
	focusDescription(t, m)

	item := findDescriptionItem(t, m.buildLayout(), 0, 0)
	assert.Equal(t, maxFocusedDescriptionRows+1, item.Height, "10 lines plus the tail")

	rendered := m.renderLayoutItem(item)
	assert.Equal(t, item.Height, strings.Count(rendered, "\n")+1)
	assert.Contains(t, rendered, "line10")
	assert.NotContains(t, rendered, "line11")
	assert.Contains(t, rendered, "+20 more lines · gi full view")
}

func TestDescriptionPreview_FocusedCapIsHalfAShortScreen(t *testing.T) {
	p := longDescriptionProject()
	p.Categories[0].Tasks[0].Description = numberedLines(30)
	m := newTestModel(t, p)
	m.ui.Screen.Height = 14
	focusDescription(t, m)

	item := findDescriptionItem(t, m.buildLayout(), 0, 0)
	want := m.availableHeight() / 2
	require.Less(t, want, maxFocusedDescriptionRows)
	assert.Equal(t, want+1, item.Height)
}

func TestDescriptionPreview_WindowFollowsSearchMatch(t *testing.T) {
	p := longDescriptionProject()
	p.Categories[0].Tasks[0].Description = numberedLines(30)
	m := newTestModel(t, p)
	focusDescription(t, m)
	item := findDescriptionItem(t, m.buildLayout(), 0, 0)

	m.ui.Search.query = "line15"
	m.invalidateLayout()
	searched := findDescriptionItem(t, m.buildLayout(), 0, 0)
	rendered := ansi.Strip(m.renderLayoutItem(searched))

	assert.Equal(t, item.Height, searched.Height, "the match moves the window, never resizes it")
	assert.Equal(t, searched.Height, strings.Count(rendered, "\n")+1)
	assert.Contains(t, rendered, "line15")
	assert.Contains(t, rendered, "…line14", "the first shown line marks the lines above")
	assert.NotContains(t, rendered, "line01")
	assert.Contains(t, rendered, "+7 more lines", "the tail counts the lines below")
}

func TestDescriptionPreview_WindowAtTheEndCountsLinesAbove(t *testing.T) {
	p := longDescriptionProject()
	p.Categories[0].Tasks[0].Description = numberedLines(30)
	m := newTestModel(t, p)
	focusDescription(t, m)
	m.ui.Search.query = "line29"
	m.invalidateLayout()

	rendered := ansi.Strip(m.renderLayoutItem(findDescriptionItem(t, m.buildLayout(), 0, 0)))

	assert.Contains(t, rendered, "line30")
	assert.Contains(t, rendered, "… 20 lines above · gi full view")
}

func TestDescriptionPreview_ShortFocusedDescriptionShowsInFull(t *testing.T) {
	m := newTestModel(t, longDescriptionProject())
	m.ui.Screen.ExpandDescriptions = true
	m.rebuildPositions()
	require.True(t, m.ui.Selection.SelectByPredicate(func(p selection.Position) bool {
		return p.Kind == selection.FocusDescription && p.TaskIndex == 0
	}))

	layout := m.buildLayout()
	item := findDescriptionItem(t, layout, 0, 0)
	assert.Equal(t, 6, item.Height, "a body under the cap lays out in full")

	rendered := m.renderLayoutItem(item)
	assert.Contains(t, rendered, "six")
	assert.NotContains(t, rendered, "more line")
}

func TestDescriptionPreview_NoTruncationWhenTailSavesNothing(t *testing.T) {
	p := longDescriptionProject()
	p.Categories[0].Tasks[0].Description = "one\ntwo\nthree\nfour" // preview+1 lines
	m := newTestModel(t, p)
	m.ui.Screen.ExpandDescriptions = true
	m.rebuildPositions()
	require.True(t, m.selectTaskIdx(0, 0))

	layout := m.buildLayout()
	item := findDescriptionItem(t, layout, 0, 0)
	assert.Equal(t, 4, item.Height)
	assert.NotContains(t, m.renderLayoutItem(item), "more line")
}

// Regression: the layout's description height must match what actually
// renders, including for tagged tasks — the old layout indent ignored the tag
// segment, so a tagged task's wrapping description was counted too short and
// the viewport math drifted.
func TestDescriptionHeight_MatchesRenderedLines(t *testing.T) {
	long := strings.Repeat("wrap me around the narrow screen ", 6)
	cases := []struct {
		name string
		task domain.Task
	}{
		{"plain", domain.Task{ID: "t1", Title: "Alpha", Status: domain.StatusTodo, Description: long}},
		{"tagged", domain.Task{ID: "t1", Title: "Alpha", Status: domain.StatusTodo, Description: long,
			TagColor: domain.TagGreen}},
		{"tagged with priority", domain.Task{ID: "t1", Title: "Alpha", Status: domain.StatusTodo, Description: long,
			TagColor: domain.TagGreen, Priority: domain.PriorityHigh}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := domain.Project{ID: "p1", Categories: []domain.Category{{
				ID: "c1", Name: "Cat A", Tasks: []domain.Task{tc.task},
			}}}
			m := newTestModel(t, p)
			m.ui.Screen.Width = 44
			m.ui.Screen.ExpandDescriptions = true
			m.rebuildPositions()
			// Focus the description row so no preview truncation interferes
			// with the full-height comparison.
			require.True(t, m.ui.Selection.SelectByPredicate(func(pp selection.Position) bool {
				return pp.Kind == selection.FocusDescription
			}))

			layout := m.buildLayout()
			item := findDescriptionItem(t, layout, 0, 0)
			rendered := m.renderLayoutItem(item)
			assert.Equal(t, item.Height, strings.Count(rendered, "\n")+1,
				"layout height must equal rendered line count")
		})
	}
}
