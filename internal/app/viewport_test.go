package app

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"phasionary/internal/app/selection"
	"phasionary/internal/domain"
)

func TestComputeVisibility_ScrollOffsetZero_ProjectVisible(t *testing.T) {
	// Create a simple project with categories and tasks
	project := domain.Project{
		Name: "Test Project",
		Categories: []domain.Category{
			{
				Name: "Category 1",
				Tasks: []domain.Task{
					{Title: "Task 1", Status: domain.StatusTodo},
					{Title: "Task 2", Status: domain.StatusTodo},
				},
			},
		},
	}

	// Build positions (mimicking what the app does)
	positions := []selection.Position{
		{Kind: selection.FocusProject, CategoryIndex: -1, TaskIndex: -1},
		{Kind: selection.FocusCategory, CategoryIndex: 0, TaskIndex: -1},
		{Kind: selection.FocusTask, CategoryIndex: 0, TaskIndex: 0},
		{Kind: selection.FocusTask, CategoryIndex: 0, TaskIndex: 1},
	}

	// Build layout
	builder := NewLayoutBuilder(DefaultLayoutConfig(), 80, "icons", nil)
	layout := builder.Build(project, positions)

	// Create viewport with reasonable screen height
	viewport := NewViewport(&layout, 20, DefaultLayoutConfig())

	// Compute visibility with scrollOffset = 0
	viewport.ComputeVisibility(0)

	// Verify the first item is a project
	require.True(t, len(layout.Items) > 0, "Layout should have items")
	assert.Equal(t, LayoutProject, layout.Items[0].Kind, "First item should be project")
	assert.Equal(t, 0, layout.Items[0].PositionIndex, "Project should have PositionIndex=0")

	// Key assertion: VisibleStart should be 0 when scrollOffset is 0
	assert.Equal(t, 0, viewport.VisibleStart, "VisibleStart should be 0 when scrollOffset is 0")
	assert.False(t, viewport.HasMoreAbove, "HasMoreAbove should be false when scrollOffset is 0")
}

func TestComputeVisibility_ZeroHeight_ProjectStillVisible(t *testing.T) {
	// This simulates the startup condition where Height is 0
	project := domain.Project{
		Name: "Test Project",
		Categories: []domain.Category{
			{
				Name: "Category 1",
				Tasks: []domain.Task{
					{Title: "Task 1", Status: domain.StatusTodo},
				},
			},
		},
	}

	positions := []selection.Position{
		{Kind: selection.FocusProject, CategoryIndex: -1, TaskIndex: -1},
		{Kind: selection.FocusCategory, CategoryIndex: 0, TaskIndex: -1},
		{Kind: selection.FocusTask, CategoryIndex: 0, TaskIndex: 0},
	}

	builder := NewLayoutBuilder(DefaultLayoutConfig(), 80, "icons", nil)
	layout := builder.Build(project, positions)

	// Create viewport with 0 height (startup condition)
	viewport := NewViewport(&layout, 0, DefaultLayoutConfig())
	viewport.ComputeVisibility(0)

	// Even with zero height, VisibleStart should be 0
	assert.Equal(t, 0, viewport.VisibleStart, "VisibleStart should be 0 even with zero height")

	// VisibleEnd should be at least 1 (at least the project should be visible)
	assert.GreaterOrEqual(t, viewport.VisibleEnd, 1, "VisibleEnd should be at least 1")
}

// View() must consume every row of the terminal — no blank rows below
// "↓ more below" — and a status message must appear next to the project title,
// never as a footer.
func TestView_FillsScreen_StatusOnProjectLine(t *testing.T) {
	project := domain.Project{
		ID:   "p1",
		Name: "Test",
		Categories: []domain.Category{
			{
				ID:   "c1",
				Name: "Cat",
				Tasks: []domain.Task{
					{ID: "t1", Title: "short1", Status: domain.StatusTodo},
					{ID: "t2", Title: "short2", Status: domain.StatusTodo},
					{ID: "t3", Title: "short3", Status: domain.StatusTodo},
					{ID: "t4", Title: "a very long task title that will definitely wrap over multiple lines to occupy more vertical space than a single row, with extra text appended so the row overflows regardless of whether status renders as text or as an icon", Status: domain.StatusTodo},
					// The long title above is clamped to ui.MaxLineRows, so overflow
					// has to come from the number of rows, not from one tall one.
					{ID: "t5", Title: "short5", Status: domain.StatusTodo},
					{ID: "t6", Title: "short6", Status: domain.StatusTodo},
					{ID: "t7", Title: "short7", Status: domain.StatusTodo},
					{ID: "t8", Title: "short8", Status: domain.StatusTodo},
					{ID: "t9", Title: "short9", Status: domain.StatusTodo},
				},
			},
		},
	}
	m := newTestModel(t, project)
	m.ui.Screen.Width = 30

	for _, h := range []int{10, 12, 14} {
		m.ui.Screen.Height = h
		m.ui.Screen.StatusMsg = ""
		view := m.View().Content
		got := strings.Count(view, "\n") + 1
		assert.Equal(t, h, got, "View should use exactly %d rows for screen height %d when content overflows", h, h)
		assert.Contains(t, view, "more below", "scroll indicator should still be shown when content overflows")
	}

	m.ui.Screen.Height = 14
	m.ui.Screen.StatusMsg = "Copied!"
	view := m.View().Content
	got := strings.Count(view, "\n") + 1
	assert.Equal(t, 14, got, "View should still fill all rows when a status message is present")
	assert.Contains(t, view, "Copied!", "status message should be rendered")
	// Status sits on the project line — the first rendered row should contain it.
	firstLine := strings.SplitN(view, "\n", 2)[0]
	assert.Contains(t, firstLine, "Copied!", "status message should be on the project (first) line")
}

func TestComputeVisibility_RemainingContentHeight(t *testing.T) {
	// Layout with a project + category + a single tall task that won't fit
	// after a sequence of small items. Verifies the unused rows are reported.
	project := domain.Project{
		Name: "Test",
		Categories: []domain.Category{
			{
				Name: "Cat",
				Tasks: []domain.Task{
					{Title: "short", Status: domain.StatusTodo},
					{Title: "short", Status: domain.StatusTodo},
				},
			},
		},
	}
	positions := []selection.Position{
		{Kind: selection.FocusProject, CategoryIndex: -1, TaskIndex: -1},
		{Kind: selection.FocusCategory, CategoryIndex: 0, TaskIndex: -1},
		{Kind: selection.FocusTask, CategoryIndex: 0, TaskIndex: 0},
		{Kind: selection.FocusTask, CategoryIndex: 0, TaskIndex: 1},
	}
	builder := NewLayoutBuilder(DefaultLayoutConfig(), 80, "icons", nil)
	layout := builder.Build(project, positions)

	// Inflate the last item's height so it cannot fit and forces wasted space.
	layout.Items[len(layout.Items)-1].Height = 20
	layout.TotalHeight += 19

	viewport := NewViewport(&layout, 12, DefaultLayoutConfig())
	viewport.ComputeVisibility(0)

	assert.True(t, viewport.HasMoreBelow, "tall trailing item should be flagged as more-below")
	assert.Greater(t, viewport.RemainingContentHeight(), 0, "unused rows below visible items should be reported")
}

func TestComputeVisibility_ScrollOffsetNonZero(t *testing.T) {
	project := domain.Project{
		Name: "Test Project",
		Categories: []domain.Category{
			{
				Name: "Category 1",
				Tasks: []domain.Task{
					{Title: "Task 1", Status: domain.StatusTodo},
					{Title: "Task 2", Status: domain.StatusTodo},
				},
			},
		},
	}

	positions := []selection.Position{
		{Kind: selection.FocusProject, CategoryIndex: -1, TaskIndex: -1},
		{Kind: selection.FocusCategory, CategoryIndex: 0, TaskIndex: -1},
		{Kind: selection.FocusTask, CategoryIndex: 0, TaskIndex: 0},
		{Kind: selection.FocusTask, CategoryIndex: 0, TaskIndex: 1},
	}

	builder := NewLayoutBuilder(DefaultLayoutConfig(), 80, "icons", nil)
	layout := builder.Build(project, positions)
	viewport := NewViewport(&layout, 20, DefaultLayoutConfig())

	// Compute visibility with scrollOffset = 1 (skip project)
	viewport.ComputeVisibility(1)

	// The visible start should NOT be 0 (the project should be scrolled past)
	assert.True(t, viewport.HasMoreAbove, "HasMoreAbove should be true when scrollOffset > 0")
	// VisibleStart should point to the first item that has PositionIndex >= scrollOffset
	assert.Greater(t, viewport.VisibleStart, 0, "VisibleStart should be > 0 when scrollOffset is 1")
}

// zz (CenterOnPosition) must place the selected item at a stable vertical
// position — near the center — regardless of which interior item is chosen.
// Regression test for the greedy-reset offset math that made items land near
// the top of the screen for most positions.
func TestCenterOnPosition_ConsistentlyCentered(t *testing.T) {
	var tasks []domain.Task
	for i := 0; i < 40; i++ {
		tasks = append(tasks, domain.Task{Title: "Task", Status: domain.StatusTodo})
	}
	project := domain.Project{
		Name:       "P",
		Categories: []domain.Category{{Name: "C", Tasks: tasks}},
	}
	positions := []selection.Position{
		{Kind: selection.FocusProject, CategoryIndex: -1, TaskIndex: -1},
		{Kind: selection.FocusCategory, CategoryIndex: 0, TaskIndex: -1},
	}
	for i := range tasks {
		positions = append(positions, selection.Position{Kind: selection.FocusTask, CategoryIndex: 0, TaskIndex: i})
	}

	config := DefaultLayoutConfig()
	layout := NewLayoutBuilder(config, 80, "icons", nil).Build(project, positions)
	const screenHeight = 24

	// rowsAbove reports the on-screen content row the target renders at, given a
	// scroll offset (i.e. how many rows sit above it inside the content area).
	rowsAbove := func(scrollOffset, posIndex int) int {
		vp := NewViewport(&layout, screenHeight, config)
		vp.ComputeVisibility(scrollOffset)
		rows := 0
		for i := vp.VisibleStart; i < vp.VisibleEnd; i++ {
			if layout.Items[i].PositionIndex == posIndex {
				return rows
			}
			rows += layout.Items[i].Height
		}
		return -1 // target not visible after centering — a bug
	}

	// Interior positions with ample content on both sides should all land at the
	// same central row. Positions near the ends can't fully center (clamped at
	// the content edges), so they're excluded here.
	first := rowsAbove(NewViewport(&layout, screenHeight, config).CenterOnPosition(15), 15)
	require.Greater(t, first, 1, "centered item should sit well below the top of the screen")

	for pos := 15; pos <= 35; pos++ {
		vp := NewViewport(&layout, screenHeight, config)
		off := vp.CenterOnPosition(pos)
		got := rowsAbove(off, pos)
		require.NotEqual(t, -1, got, "pos %d must remain visible after zz", pos)
		assert.InDelta(t, first, got, 1,
			"pos %d should center at the same row as the others (got rowsAbove=%d, want ~%d)", pos, got, first)
	}
}

func TestLayoutBuilder_ExpandedDescriptions_AddsSeparateRow(t *testing.T) {
	project := domain.Project{
		Name: "P",
		Categories: []domain.Category{{
			Name: "C",
			Tasks: []domain.Task{
				{Title: "Task A", Status: domain.StatusTodo, Description: "Line one\nLine two"},
				{Title: "Task B", Status: domain.StatusTodo}, // no description
			},
		}},
	}
	// Positions for the expanded case: project, cat, task A, desc A, task B.
	expandedPositions := []selection.Position{
		{Kind: selection.FocusProject, CategoryIndex: -1, TaskIndex: -1},
		{Kind: selection.FocusCategory, CategoryIndex: 0, TaskIndex: -1},
		{Kind: selection.FocusTask, CategoryIndex: 0, TaskIndex: 0},
		{Kind: selection.FocusDescription, CategoryIndex: 0, TaskIndex: 0},
		{Kind: selection.FocusTask, CategoryIndex: 0, TaskIndex: 1},
	}
	basePositions := []selection.Position{
		{Kind: selection.FocusProject, CategoryIndex: -1, TaskIndex: -1},
		{Kind: selection.FocusCategory, CategoryIndex: 0, TaskIndex: -1},
		{Kind: selection.FocusTask, CategoryIndex: 0, TaskIndex: 0},
		{Kind: selection.FocusTask, CategoryIndex: 0, TaskIndex: 1},
	}

	base := NewLayoutBuilder(DefaultLayoutConfig(), 80, "icons", nil).Build(project, basePositions)
	expanded := NewLayoutBuilder(DefaultLayoutConfig(), 80, "icons", nil).
		WithExpandedDescriptions(true).
		Build(project, expandedPositions)

	assert.Equal(t, base.TotalHeight+2, expanded.TotalHeight,
		"two description lines should add exactly 2 rows to total height")

	// LayoutDescription must be present, focusable, and 2 rows tall.
	var descItem *LayoutItem
	for i, it := range expanded.Items {
		if it.Kind == LayoutDescription {
			descItem = &expanded.Items[i]
			break
		}
	}
	assert.NotNil(t, descItem, "expected a LayoutDescription item")
	if descItem != nil {
		assert.Equal(t, 2, descItem.Height)
		assert.GreaterOrEqual(t, descItem.PositionIndex, 0, "description row must be focusable")
	}

	// Task A's own row height should be unchanged (description is now separate).
	var baseA, expA int
	for _, it := range base.Items {
		if it.Kind == LayoutTask && it.TaskIndex == 0 {
			baseA = it.Height
		}
	}
	for _, it := range expanded.Items {
		if it.Kind == LayoutTask && it.TaskIndex == 0 {
			expA = it.Height
		}
	}
	assert.Equal(t, baseA, expA, "task row height should not change when expanding descriptions")
}

// tallItemLayout is a 1-row project, an item of tallHeight rows, then two 1-row
// items — hand-built so these tests are about the viewport's row arithmetic
// rather than about what wraps to what.
func tallItemLayout(tallHeight int) *Layout {
	items := []LayoutItem{
		{Kind: LayoutProject, Height: 1, PositionIndex: 0, CategoryIndex: -1, TaskIndex: -1},
		{Kind: LayoutTask, Height: tallHeight, PositionIndex: 1, CategoryIndex: 0, TaskIndex: 0},
		{Kind: LayoutTask, Height: 1, PositionIndex: 2, CategoryIndex: 0, TaskIndex: 1},
		{Kind: LayoutTask, Height: 1, PositionIndex: 3, CategoryIndex: 0, TaskIndex: 2},
	}
	total := 0
	for _, item := range items {
		total += item.Height
	}
	return &Layout{Items: items, TotalHeight: total}
}

func TestComputeVisibility_EntersATallItem(t *testing.T) {
	layout := tallItemLayout(50)

	// 24 rows less the footer is 23, less a row for each scroll indicator is 21.
	// 40 of the item's rows remain, so its tail alone fills them.
	deep := NewViewport(layout, 24, LayoutConfig{FooterHeight: 1})
	deep.ComputeVisibility(11)
	require.Equal(t, 1, deep.VisibleStart, "the tall item is what the view starts on")
	assert.Equal(t, 10, deep.RowOffset, "the offset within the item is what is left of the top row")
	assert.True(t, deep.HasMoreAbove, "rows scrolled off the top are elided above")
	assert.True(t, deep.HasMoreBelow, "40 rows remain and only 21 fit")
	assert.Equal(t, 1, deep.VisibleEnd, "the item doesn't fit whole, so it is drawn as a partial")
	assert.Equal(t, 21, deep.RemainingContentHeight(), "every content row goes to it")

	// Far enough into the item and its tail leaves room for what follows.
	tail := NewViewport(layout, 24, LayoutConfig{FooterHeight: 1})
	tail.ComputeVisibility(31)
	assert.False(t, tail.HasMoreBelow, "20 remaining rows plus the two items below fit in 22")
	assert.Equal(t, len(layout.Items), tail.VisibleEnd, "the items below the tall one are drawn too")
}

func TestComputeVisibility_ClampsPastTheLastRow(t *testing.T) {
	layout := tallItemLayout(3)
	viewport := NewViewport(layout, 24, LayoutConfig{FooterHeight: 1})
	viewport.ComputeVisibility(99)

	assert.Equal(t, layout.TotalHeight-1, viewport.TopRow, "a top row past the end leaves the last row on screen")
	assert.Equal(t, len(layout.Items), viewport.VisibleEnd)
	assert.False(t, viewport.HasMoreBelow)
}

func TestScroll_StepsByRowThroughATallItem(t *testing.T) {
	m := editingModel(t, 80, 24, longWords(4000))

	// Park the view on the item's first row, then walk down into it.
	start, _, ok := m.buildLayout().rowRange(m.selected())
	require.True(t, ok)
	m.ui.Screen.TopRow = start
	m.scrollDown(5)
	assert.Equal(t, start+5, m.ui.Screen.TopRow, "each step reveals one more row of it")

	m.scrollUp(2)
	assert.Equal(t, start+3, m.ui.Screen.TopRow, "scrolling back walks the same rows")
	m.scrollUp(10)
	assert.Less(t, m.ui.Screen.TopRow, start, "past the item's first row the view moves on")
}
