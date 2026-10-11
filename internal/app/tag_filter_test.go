package app

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ilyasturki/phasionary/internal/domain"
)

// enableTagFilter toggles the given tag rows on via the filter's public
// navigation + toggle API, exercising the same path the UI drives.
func enableTagFilter(f *FilterState, colors ...string) {
	f.SetView(FilterViewTag)
	for _, want := range colors {
		// Move the tag cursor to the target row, then toggle it.
		for i := range filterTagColors {
			if filterTagColors[i] == want {
				for f.Selected() < i {
					f.MoveDown(0)
				}
				for f.Selected() > i {
					f.MoveUp()
				}
				f.ToggleSelected(nil)
				break
			}
		}
	}
}

func TestFilterState_TagVisibility(t *testing.T) {
	green := domain.Task{TagColor: domain.TagGreen}
	cyan := domain.Task{TagColor: domain.TagCyan}
	untagged := domain.Task{}

	t.Run("no tag filter shows everything", func(t *testing.T) {
		f := NewFilterState()
		assert.True(t, f.TaskVisible(green, "c", ""))
		assert.True(t, f.TaskVisible(untagged, "c", ""))
	})

	t.Run("color filter shows only that color", func(t *testing.T) {
		f := NewFilterState()
		enableTagFilter(&f, domain.TagGreen)
		assert.True(t, f.TaskVisible(green, "c", ""))
		assert.False(t, f.TaskVisible(cyan, "c", ""))
		assert.False(t, f.TaskVisible(untagged, "c", ""))
	})

	t.Run("untagged bucket matches tasks with no color", func(t *testing.T) {
		f := NewFilterState()
		enableTagFilter(&f, "")
		assert.True(t, f.TaskVisible(untagged, "c", ""))
		assert.False(t, f.TaskVisible(green, "c", ""))
	})

	t.Run("multiple rows union together", func(t *testing.T) {
		f := NewFilterState()
		enableTagFilter(&f, domain.TagGreen, "")
		assert.True(t, f.TaskVisible(green, "c", ""))
		assert.True(t, f.TaskVisible(untagged, "c", ""))
		assert.False(t, f.TaskVisible(cyan, "c", ""))
	})

	t.Run("clear all drops the tag filter", func(t *testing.T) {
		f := NewFilterState()
		enableTagFilter(&f, domain.TagGreen)
		f.ClearAll()
		assert.False(t, f.HasActiveFilter())
		assert.True(t, f.TaskVisible(cyan, "c", ""))
	})
}
