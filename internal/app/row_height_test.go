package app

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"phasionary/internal/config"
	"phasionary/internal/domain"
)

// badgedProject has a task and a category carrying every badge that eats title
// width: a labeled tag, a priority icon, an estimate and a description marker.
func badgedProject(title string) domain.Project {
	return domain.Project{
		ID:   "p1",
		Name: "Probe",
		Categories: []domain.Category{{
			ID:   "c1",
			Name: title, EstimateMinutes: 120,
			Tasks: []domain.Task{{
				ID: "t1", Title: title, Status: domain.StatusInProgress,
				Priority: domain.PriorityHigh, TagColor: "red", TagLabel: "work",
				EstimateMinutes: 90, Description: "body",
			}},
		}},
	}
}

// The ✂ badge comes and goes without a project change, so the layout cache has
// to notice it on its own.
func TestRowHeight_CutRefreshesCachedLayout(t *testing.T) {
	cut := ClipboardState{IsCut: true, TaskIDs: []string{"t1"}}
	heights := func(n int) (int, int) {
		m := newTestModel(t, badgedProject(strings.Repeat("x", n)))
		m.ui.Screen.Width = 60
		plain := m.buildLayout().TotalHeight
		m.ui.Clipboard = cut
		return plain, m.buildLayout().TotalHeight
	}
	for n := 20; n < 60; n++ {
		m := newTestModel(t, badgedProject(strings.Repeat("x", n)))
		m.ui.Screen.Width = 60
		m.ui.Clipboard = cut
		fresh := m.buildLayout().TotalHeight

		plain, cached := heights(n)
		if fresh != plain {
			assert.Equal(t, fresh, cached, "len=%d: the cached layout kept the uncut height", n)
			return
		}
	}
	t.Fatal("no title length where the badge adds a row")
}

// Every title length from one row to past the clamp, so some land on each wrap
// boundary the badges move.
func TestRowHeight_LayoutMatchesRenderWithBadges(t *testing.T) {
	for _, display := range []string{config.StatusDisplayIcons, config.StatusDisplayText} {
		for _, cut := range []bool{false, true} {
			for n := 10; n <= 200; n++ {
				m := newTestModel(t, badgedProject(longWords(n)))
				m.ui.Screen.Width = 60
				_ = m.deps.CfgManager.Update(func(cfg *config.Config) { cfg.StatusDisplay = display })
				if cut {
					m.ui.Clipboard = ClipboardState{IsCut: true, TaskIDs: []string{"t1"}, CategoryIDs: []string{"c1"}}
				}
				m.invalidateLayout()

				for _, item := range m.buildLayout().Items {
					if item.Kind != LayoutTask && item.Kind != LayoutCategory {
						continue
					}
					drawn := strings.Count(m.renderLayoutItem(item), "\n") + 1
					assert.Equal(t, item.Height, drawn, "%s cut=%v len=%d kind=%d: reserved %d rows, drew %d",
						display, cut, n, item.Kind, item.Height, drawn)
				}
			}
		}
	}
}
