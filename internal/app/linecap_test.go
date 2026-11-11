package app

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/domain"
	"github.com/ilyasturki/ordo/internal/ui"
)

func lineCapProject() domain.Project {
	long := longWords(1200)
	return domain.Project{
		ID: "p1", Name: "P",
		Categories: []domain.Category{{
			ID: "c1", Name: long,
			Tasks: []domain.Task{
				{ID: "t1", Title: long, Status: domain.StatusTodo, Description: numberedLines(30)},
				{ID: "n1", Kind: domain.KindNote, Title: long},
				{ID: "t2", Title: "after", Status: domain.StatusTodo},
			},
		}},
	}
}

func lineCapHeights(t *testing.T, m *model) map[LayoutItemKind]int {
	t.Helper()
	heights := map[LayoutItemKind]int{}
	for _, item := range m.buildLayout().Items {
		if item.Kind == LayoutSpacing || item.Kind == LayoutProject || item.TaskIndex == 2 {
			continue
		}
		drawn := strings.Count(m.renderLayoutItem(item), "\n") + 1
		require.Equal(t, item.Height, drawn, "kind %d reserved %d rows, drew %d", item.Kind, item.Height, drawn)
		heights[item.Kind] = item.Height
	}
	return heights
}

func TestLineCap_StepsRaiseEveryTitleAndDescription(t *testing.T) {
	m := newTestModel(t, lineCapProject())
	m.ui.Screen.Height = 200
	m.ui.Screen.ExpandDescriptions = true
	m.rebuildPositions()
	require.True(t, m.selectTaskIdx(0, 2))

	kinds := []LayoutItemKind{LayoutCategory, LayoutTask, LayoutSeparator, LayoutDescription}
	start := lineCapHeights(t, m)
	for _, kind := range kinds {
		want := ui.MaxLineRows
		if kind == LayoutDescription {
			want++ // the "… +N more lines" tail
		}
		assert.Equal(t, want, start[kind], "kind %d starts at the default cap", kind)
	}

	am := *m
	for _, step := range []struct {
		cap    int
		status string
	}{{6, "Long text shows up to 6 rows"}, {10, "Long text shows up to 10 rows"}} {
		am = press(t, am, "z", "+")
		assert.Equal(t, step.status, am.ui.Screen.StatusMsg)
		heights := lineCapHeights(t, &am)
		for _, kind := range kinds {
			want := step.cap
			if kind == LayoutDescription {
				want++
			}
			assert.Equal(t, want, heights[kind], "kind %d at cap %d", kind, step.cap)
		}
	}

	am = press(t, am, "z", "+")
	assert.Equal(t, "Long text shows whole", am.ui.Screen.StatusMsg)
	full := lineCapHeights(t, &am)
	assert.Equal(t, 30, full[LayoutDescription])
	for _, kind := range kinds {
		assert.Greater(t, full[kind], 10, "kind %d wraps past every cap", kind)
	}
	am = press(t, am, "z", "+")
	assert.Equal(t, full, lineCapHeights(t, &am), "past whole stays whole")

	for range lineCapSteps {
		am = press(t, am, "z", "-")
	}
	assert.Equal(t, "Long text shows up to 3 rows", am.ui.Screen.StatusMsg)
	assert.Equal(t, start, lineCapHeights(t, &am), "z- steps back to the default and stops there")
}

func TestLineCap_StartsAtTheDefaultEachSession(t *testing.T) {
	m := newTestModel(t, lineCapProject())
	assert.Equal(t, ui.MaxLineRows, m.rowLimits().lineCap)
}

func TestLineCap_JScrollsThroughARowTallerThanTheScreen(t *testing.T) {
	p := lineCapProject()
	p.Categories[0].Name = "Cat"
	p.Categories[0].Tasks[0].Title = "short"
	p.Categories[0].Tasks[0].Description = numberedLines(60)
	m := newTestModel(t, p)
	m.ui.Screen.Height = 20
	m.ui.Screen.ExpandDescriptions = true
	m.rebuildPositions()
	require.True(t, m.selectTaskIdx(0, 0))

	am := press(t, *m, "z", "+", "z", "+", "z", "+", "j")
	desc, _ := am.selectedPosition()
	for i := 0; !strings.Contains(strings.Join(screen(&am), "\n"), "line60"); i++ {
		require.Less(t, i, 100, "j never reached the last line")
		am = press(t, am, "j")
		now, _ := am.selectedPosition()
		require.Equal(t, desc, now, "j scrolls the tall row before leaving it (press %d)", i)
	}
	am = press(t, am, "j")
	now, _ := am.selectedPosition()
	assert.NotEqual(t, desc, now, "past the end j moves on")
}
