package app

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ilyasturki/phasionary/internal/app/modes"
	"github.com/ilyasturki/phasionary/internal/app/selection"
	"github.com/ilyasturki/phasionary/internal/config"
	"github.com/ilyasturki/phasionary/internal/domain"
)

// benchProject builds a project with cats categories, each holding tasksPer
// tasks. Every 4th task carries a description so the wrapping paths get exercised.
func benchProject(cats, tasksPer int) domain.Project {
	statuses := []string{domain.StatusTodo, domain.StatusInProgress, domain.StatusCompleted, domain.StatusCancelled}
	prios := []string{domain.PriorityLow, domain.PriorityMedium, domain.PriorityHigh}
	p := domain.Project{ID: "bench", Name: "Bench Project"}
	for c := 0; c < cats; c++ {
		cat := domain.Category{ID: fmt.Sprintf("c%d", c), Name: fmt.Sprintf("Category number %d", c)}
		for t := 0; t < tasksPer; t++ {
			task := domain.Task{
				ID:              fmt.Sprintf("c%dt%d", c, t),
				Title:           fmt.Sprintf("Task %d in category %d with a reasonably long title that may wrap", t, c),
				Status:          statuses[t%len(statuses)],
				Priority:        prios[t%len(prios)],
				EstimateMinutes: (t % 5) * 30,
			}
			if t%4 == 0 {
				task.Description = "A multi-line description.\nWith a second paragraph long enough to wrap across the available width of the terminal."
			}
			cat.Tasks = append(cat.Tasks, task)
		}
		p.Categories = append(p.Categories, cat)
	}
	return p
}

func benchModel(p domain.Project, expandDesc bool) *model {
	positions := rebuildPositions(p.Categories, nil, nil, expandDesc, "")
	sel := selection.NewManager(positions, 0)
	mode := modes.NewMachine(modes.ModeNormal)
	ui := NewUIState(sel, mode)
	ui.Screen.Width = 100
	ui.Screen.Height = 50
	ui.Screen.WindowFocused = true
	ui.Screen.ExpandDescriptions = expandDesc
	return &model{
		project: p,
		ui:      ui,
		deps:    &Dependencies{CfgManager: &stubConfigReader{cfg: config.DefaultConfig()}},
	}
}

var sizes = []struct {
	name           string
	cats, tasksPer int
}{
	{"small_5x10", 5, 10},
	{"medium_10x30", 10, 30},
	{"large_15x100", 15, 100},
	{"huge_20x250", 20, 250},
}

// BenchmarkRenderView measures repeated renders with no intervening mutation —
// i.e. plain j/k navigation, the most common interaction. With the layout cache
// this should be near-independent of total task count.
func BenchmarkRenderView(b *testing.B) {
	for _, s := range sizes {
		m := benchModel(benchProject(s.cats, s.tasksPer), false)
		b.Run(s.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = m.renderView()
			}
		})
	}
}

// BenchmarkRenderAfterEdit measures the worst case: a render forced to rebuild
// the layout from scratch, as after a content mutation invalidates the cache.
func BenchmarkRenderAfterEdit(b *testing.B) {
	for _, s := range sizes {
		m := benchModel(benchProject(s.cats, s.tasksPer), false)
		b.Run(s.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				m.invalidateLayout()
				_ = m.renderView()
			}
		})
	}
}

// BenchmarkRenderLongTitle measures a title far past any length a cap would have
// allowed. Titles are unbounded, so the guarantee that replaces the old cap is
// this one: the clamp slices the text down to a screenful before it reaches
// ansi.Wrap, so cost stops tracking length. Every size here should come out
// flat, in time and in allocations both.
func BenchmarkRenderLongTitle(b *testing.B) {
	for _, n := range []int{64, 1024, 65536, 1_000_000} {
		p := benchProject(1, 3)
		p.Categories[0].Tasks[1].Title = strings.Repeat("word ", n/5)
		m := benchModel(p, false)
		b.Run(fmt.Sprintf("len_%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				m.invalidateLayout()
				_ = m.renderView()
			}
		})
	}
}
