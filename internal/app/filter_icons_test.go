package app

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"

	"github.com/ilyasturki/ordo/internal/config"
)

func dialogRows(view string) []string {
	var rows []string
	for _, line := range strings.Split(ansi.Strip(view), "\n") {
		line = strings.Trim(line, "│ ")
		if strings.Contains(line, "[ ]") || strings.Contains(line, "[x]") {
			rows = append(rows, line)
		}
	}
	return rows
}

func TestStatusFilterShowsEachRowsMarkInAColumn(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Filter.SetView(FilterViewStatus)

	assert.Equal(t, []string{
		"> [ ]   Todo",
		"[ ] / In Progress",
		"[ ] x Completed",
		"[ ] - Cancelled",
	}, dialogRows(m.filterStatusView()))
}

func TestStatusFilterFollowsTheTextDisplay(t *testing.T) {
	m := newTestModel(t, sampleProject())
	_ = m.deps.CfgManager.Update(func(cfg *config.Config) { cfg.StatusDisplay = config.StatusDisplayText })
	m.ui.Filter.SetView(FilterViewStatus)

	assert.Equal(t, "[ ] In Progress", dialogRows(m.filterStatusView())[1])
}

func TestPriorityFilterShowsEachRowsGlyphInAColumn(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Filter.SetView(FilterViewPriority)
	m.ui.Filter.prioritySelected = 1

	assert.Equal(t, []string{
		"[ ] ⇈ Critical",
		"> [ ] ▲ High",
		"[ ]   Medium",
		"[ ] ▼ Low",
		"[ ] ⇊ Trivial",
	}, dialogRows(m.filterPriorityView()))
}
