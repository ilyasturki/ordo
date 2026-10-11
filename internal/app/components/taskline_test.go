package components

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"

	"github.com/ilyasturki/phasionary/internal/domain"
)

func TestNewTaskLineRenderer(t *testing.T) {
	renderer := NewTaskLineRenderer(80, "text", "full", true)
	assert.NotNil(t, renderer)
	assert.Equal(t, 80, renderer.width)
	assert.Equal(t, "text", renderer.statusDisplay)
	assert.True(t, renderer.focused)
}

func TestTaskLineRenderer_Render(t *testing.T) {
	t.Run("renders unselected task", func(t *testing.T) {
		renderer := NewTaskLineRenderer(0, "text", "full", true)
		task := domain.Task{
			Title:    "Test task",
			Status:   domain.StatusTodo,
			Priority: domain.PriorityMedium,
		}
		result := renderer.Render(task, false)
		assert.Contains(t, result, "Test task")
		assert.Contains(t, result, "todo")
		assert.True(t, strings.HasPrefix(result, "  "))
	})

	t.Run("renders selected task with cursor prefix", func(t *testing.T) {
		renderer := NewTaskLineRenderer(0, "text", "full", true)
		task := domain.Task{
			Title:    "Test task",
			Status:   domain.StatusInProgress,
			Priority: domain.PriorityHigh,
		}
		result := renderer.Render(task, true)
		assert.Contains(t, result, "Test task")
		assert.Contains(t, result, "progress")
	})

	t.Run("wraps long task titles", func(t *testing.T) {
		renderer := NewTaskLineRenderer(40, "text", "full", true)
		task := domain.Task{
			Title:  "This is a very long task title that should wrap to multiple lines",
			Status: domain.StatusTodo,
		}
		result := renderer.Render(task, false)
		lines := strings.Split(result, "\n")
		assert.True(t, len(lines) > 1)
	})

	t.Run("priority icon flows inline; wrapped lines align at icon column", func(t *testing.T) {
		renderer := NewTaskLineRenderer(40, "text", "full", true)
		task := domain.Task{
			Title:    "This is a very long task title that should wrap to multiple lines",
			Status:   domain.StatusTodo,
			Priority: domain.PriorityHigh,
		}
		result := renderer.Render(task, false)
		lines := strings.Split(result, "\n")
		assert.True(t, len(lines) > 1, "expected wrapping")
		plain0 := ansi.Strip(lines[0])
		iconCol := strings.Index(plain0, "▲")
		assert.GreaterOrEqual(t, iconCol, 0, "expected priority icon on first line")
		for i, line := range lines[1:] {
			plain := ansi.Strip(line)
			// Continuation lines start at the icon's column — no extra indent for the icon itself.
			leading := len(plain) - len(strings.TrimLeft(plain, " "))
			assert.Equal(t, iconCol, leading,
				"continuation line %d should align at icon column %d, got %d (line=%q)", i+1, iconCol, leading, plain)
		}
	})

	t.Run("renders with icon status display", func(t *testing.T) {
		renderer := NewTaskLineRenderer(0, "icons", "full", true)
		task := domain.Task{
			Title:  "Test task",
			Status: domain.StatusCompleted,
		}
		result := renderer.Render(task, false)
		assert.Contains(t, result, "[")
		assert.Contains(t, result, "]")
	})
}

func TestTaskLineRenderer_RenderDescription(t *testing.T) {
	t.Run("renders description with blockquote bar at given indent", func(t *testing.T) {
		renderer := NewTaskLineRenderer(80, "text", "full", true)
		result := renderer.RenderDescription(domain.Task{Description: "Some details about the task"}, 9, false)
		assert.Contains(t, result, "Some details about the task")
		// Each line should start with 9 spaces of indent + the bar glyph (after stripping styles).
		for _, line := range strings.Split(result, "\n") {
			plain := ansi.Strip(line)
			assert.True(t, strings.HasPrefix(plain, "         ▎ "), "line %q lacks indented blockquote prefix", line)
		}
	})

	t.Run("empty description renders nothing", func(t *testing.T) {
		renderer := NewTaskLineRenderer(80, "text", "full", true)
		assert.Equal(t, "", renderer.RenderDescription(domain.Task{}, 9, false))
	})

	t.Run("preserves paragraph breaks", func(t *testing.T) {
		renderer := NewTaskLineRenderer(80, "text", "full", true)
		result := renderer.RenderDescription(domain.Task{Description: "First paragraph\n\nSecond paragraph"}, 9, false)
		assert.Contains(t, result, "First paragraph")
		assert.Contains(t, result, "Second paragraph")
		// first para + blank + second para = 3 lines
		assert.Equal(t, 3, len(strings.Split(result, "\n")))
	})

	t.Run("Render no longer appends description to task line", func(t *testing.T) {
		renderer := NewTaskLineRenderer(80, "text", "full", true)
		task := domain.Task{
			Title:       "Test task",
			Status:      domain.StatusTodo,
			Description: "Some details",
		}
		result := renderer.Render(task, false)
		assert.NotContains(t, result, "Some details",
			"task-line render should not embed description; that's a separate row now")
	})
}

func TestTaskLineRenderer_StatusLabel(t *testing.T) {
	t.Run("returns text labels when not icons mode", func(t *testing.T) {
		renderer := NewTaskLineRenderer(0, "text", "full", true)
		assert.Contains(t, renderer.statusLabel(domain.StatusTodo), "todo")
		assert.Contains(t, renderer.statusLabel(domain.StatusInProgress), "progress")
		assert.Contains(t, renderer.statusLabel(domain.StatusCompleted), "completed")
		assert.Contains(t, renderer.statusLabel(domain.StatusCancelled), "cancelled")
	})

	t.Run("returns icons when icons mode", func(t *testing.T) {
		renderer := NewTaskLineRenderer(0, "icons", "full", true)
		assert.Equal(t, " ", renderer.statusLabel(domain.StatusTodo))
		assert.Equal(t, "/", renderer.statusLabel(domain.StatusInProgress))
		assert.Equal(t, "x", renderer.statusLabel(domain.StatusCompleted))
		assert.Equal(t, "-", renderer.statusLabel(domain.StatusCancelled))
	})
}

func TestSafeWidth(t *testing.T) {
	t.Run("returns available space", func(t *testing.T) {
		assert.Equal(t, 70, safeWidth(80, 10))
	})

	t.Run("returns minimum of 1 for negative result", func(t *testing.T) {
		assert.Equal(t, 1, safeWidth(10, 20))
	})
}
