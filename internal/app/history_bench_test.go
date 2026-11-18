package app

import (
	"slices"
	"testing"

	"github.com/ilyasturki/ordo/internal/domain"
	"github.com/ilyasturki/ordo/internal/merge"
)

// fullHistory returns a 5k-task model whose undo stack is full, one priority change per step,
// and the project an outside retitle merges it into.
func fullHistory() (*model, domain.Project) {
	m := benchModel(benchProject(50, 100), false)
	prios := []string{domain.PriorityLow, domain.PriorityHigh}
	for i := range historyLimit {
		m.recordHistory()
		c := &m.project.Categories[i%len(m.project.Categories)]
		c.Tasks[i%len(c.Tasks)].Priority = prios[i%2]
	}
	merged := merge.Clone(m.project)
	merged.Categories[7].Tasks[3].Title = "Retitled elsewhere"
	return m, merged
}

func BenchmarkOutsideChangeThenUndo(b *testing.B) {
	m, merged := fullHistory()
	mine, history := m.project, m.ui.History
	b.ReportAllocs()
	for b.Loop() {
		m.project = mine
		m.ui.History = history
		m.ui.History.undo = slices.Clone(history.undo)
		m.ui.History.rebase(mine, merged)
		m.undo()
	}
}
