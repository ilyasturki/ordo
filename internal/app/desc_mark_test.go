package app

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ilyasturki/ordo/internal/app/selection"
)

func TestDescriptionMark_DropsOnARowExpandedWithZe(t *testing.T) {
	p := nestedProject()
	p.Categories[0].Tasks[0].Description = "loose body"
	p.Categories[1].Tasks[0].Description = "timer body"
	m := newTestModel(t, p)

	assert.Contains(t, lineWith(t, screen(m), "Loose"), "¶")
	assert.Contains(t, lineWith(t, screen(m), "Rest timer"), "¶")

	selectRow(t, m, selection.FocusTask, "timer")
	m.toggleExpandRow()
	assert.NotContains(t, lineWith(t, screen(m), "Rest timer"), "¶", "its description is shown")
	assert.Contains(t, lineWith(t, screen(m), "Loose"), "¶", "the other rows keep it")

	m.toggleExpandRow()
	assert.Contains(t, lineWith(t, screen(m), "Rest timer"), "¶")
}

func TestRowHeight_LayoutMatchesRenderOnARowExpandedWithZe(t *testing.T) {
	for n := 40; n <= 90; n++ {
		p := badgedProject(longWords(n))
		m := newTestModel(t, p)
		m.ui.Screen.Width = 60
		selectRow(t, m, selection.FocusTask, "t1")
		m.toggleExpandRow()
		m.invalidateLayout()

		for _, item := range m.buildLayout().Items {
			if item.Kind != LayoutTask {
				continue
			}
			drawn := strings.Count(m.renderLayoutItem(item), "\n") + 1
			assert.Equal(t, item.Height, drawn, "len=%d: reserved %d rows, drew %d", n, item.Height, drawn)
		}
	}
}
