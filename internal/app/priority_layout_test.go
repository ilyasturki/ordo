package app

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"phasionary/internal/domain"
)

// The critical/trivial icons must be single-cell like ▲/▼ so every priority
// row's title starts at the same column. Guards against reintroducing an
// ambiguous-width glyph (e.g. the ⏫/⏬ emoji, whose column advance the width
// table counts as two), which would push critical/trivial titles a cell right.
func TestTaskTitleColumn_ExtremesMatchNeighbors(t *testing.T) {
	col := func(priority string) int {
		return taskTitleColumn(domain.Task{Title: "x", Status: domain.StatusTodo, Priority: priority}, "", 0)
	}
	assert.Equal(t, col(domain.PriorityHigh), col(domain.PriorityCritical), "critical must align like high")
	assert.Equal(t, col(domain.PriorityLow), col(domain.PriorityTrivial), "trivial must align like low")
}
