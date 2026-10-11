package app

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/phasionary/internal/app/modes"
	"github.com/ilyasturki/phasionary/internal/app/selection"
	"github.com/ilyasturki/phasionary/internal/domain"
	"github.com/ilyasturki/phasionary/internal/ui"
)

// Titles carry no length limit. These tests pin what bounds one instead: the
// list row's clamp and the inline editor's wrap.

// longWords builds a space-separated string of n runes ending in a sentinel, so
// a test can tell whether a render reached the end of the text.
func longWords(n int) string {
	var b strings.Builder
	for i := 0; b.Len() < n; i++ {
		b.WriteString("word ")
	}
	return b.String()[:n-4] + "ZEND"
}

func longTitleProject(title string) domain.Project {
	return domain.Project{
		ID:   "p1",
		Name: "Probe",
		Categories: []domain.Category{{
			ID:   "c1",
			Name: "Cat",
			Tasks: []domain.Task{
				{ID: "t1", Title: "before", Status: domain.StatusTodo},
				{ID: "t2", Title: title, Status: domain.StatusTodo},
				{ID: "t3", Title: "after", Status: domain.StatusTodo},
			},
		}},
	}
}

// longTitleItem returns the layout item for the long-titled task, and its
// position index.
func longTitleItem(t *testing.T, m *model) (LayoutItem, int) {
	t.Helper()
	for _, item := range m.buildLayout().Items {
		if item.Kind == LayoutTask && item.TaskIndex == 1 {
			return item, item.PositionIndex
		}
	}
	t.Fatal("no layout item for the long-titled task")
	return LayoutItem{}, -1
}

func TestLongTitle_RowStaysBounded(t *testing.T) {
	for _, size := range []int{200, 1024, 100_000} {
		for _, dim := range []struct{ w, h int }{{40, 24}, {80, 24}, {120, 40}} {
			m := newTestModel(t, longTitleProject(longWords(size)))
			m.ui.Screen.Width, m.ui.Screen.Height = dim.w, dim.h
			m.invalidateLayout()

			item, _ := longTitleItem(t, m)
			assert.LessOrEqual(t, item.Height, ui.MaxLineRows,
				"len=%d %dx%d: a row must never outgrow the clamp", size, dim.w, dim.h)

			// The reservation and the render have to agree, or the viewport's row
			// arithmetic drifts and content falls off the bottom.
			drawn := strings.Count(m.renderLayoutItem(item), "\n") + 1
			assert.Equal(t, item.Height, drawn,
				"len=%d %dx%d: reserved %d rows, drew %d", size, dim.w, dim.h, item.Height, drawn)

			rows := strings.Count(m.renderView(), "\n") + 1
			assert.Equal(t, dim.h, rows, "len=%d %dx%d: the view must fill the screen exactly", size, dim.w, dim.h)
		}
	}
}

func TestLongTitle_SearchWindowFollowsTheMatch(t *testing.T) {
	// A match past the clamp would otherwise be highlighted on a row nobody can
	// see: n/N would jump the cursor to a task showing no visible hit.
	m := newTestModel(t, longTitleProject(longWords(4000)))
	m.ui.Screen.Width, m.ui.Screen.Height = 80, 24
	m.ui.Search.query = "ZEND"
	m.invalidateLayout()

	item, _ := longTitleItem(t, m)
	assert.Contains(t, m.renderLayoutItem(item), "ZEND", "the window should open on the match")
	assert.Equal(t, item.Height, strings.Count(m.renderLayoutItem(item), "\n")+1,
		"shifting the window must not change the row count")
}

// startEdit puts the model in edit mode on pos, the way startEditing would.
func startEdit(m *model, pos int, buffer string, kind selection.FocusKind) {
	m.ui.Selection.SetSelected(pos)
	m.ui.Modes = modes.NewMachine(modes.ModeEdit)
	m.ui.Edit = newEditState(buffer, false, "", kind)
	m.ensureVisible()
}

// editingModel opens an editor holding buffer on the long-titled task.
func editingModel(t *testing.T, w, h int, buffer string) *model {
	t.Helper()
	m := newTestModel(t, longTitleProject("short"))
	m.ui.Screen.Width, m.ui.Screen.Height = w, h
	m.invalidateLayout()
	_, pos := longTitleItem(t, m)
	startEdit(m, pos, buffer, selection.FocusTask)
	return m
}

func editedItem(t *testing.T, m *model) LayoutItem {
	t.Helper()
	for _, item := range m.buildLayout().Items {
		if item.PositionIndex == m.selected() {
			return item
		}
	}
	t.Fatal("no layout item under the cursor")
	return LayoutItem{}
}

func TestLongTitle_EditRowWrapsToTheBuffer(t *testing.T) {
	for _, size := range []int{20, 1024, 100_000} {
		for _, dim := range []struct{ w, h int }{{40, 24}, {80, 24}, {120, 40}} {
			m := editingModel(t, dim.w, dim.h, longWords(size))

			item := editedItem(t, m)
			drawn := strings.Count(m.renderLayoutItem(item), "\n") + 1
			assert.Equal(t, item.Height, drawn, "len=%d %dx%d", size, dim.w, dim.h)

			// The shortcut bar is hidden while editing, so the view is only
			// bounded by the screen rather than padded out to it.
			rows := strings.Count(m.renderView(), "\n") + 1
			assert.LessOrEqual(t, rows, dim.h, "len=%d %dx%d: the view must not overflow the screen", size, dim.w, dim.h)
		}
	}
}

func TestLongTitle_EditRowGrowsPastTheDisplayClamp(t *testing.T) {
	m := newTestModel(t, longTitleProject(longWords(600)))
	m.ui.Screen.Width, m.ui.Screen.Height = 80, 40
	m.invalidateLayout()
	browsing, pos := longTitleItem(t, m)
	require.Equal(t, ui.MaxLineRows, browsing.Height, "the display clamp still applies while browsing")

	startEdit(m, pos, longWords(600), selection.FocusTask)
	assert.Greater(t, editedItem(t, m).Height, ui.MaxLineRows, "the editor shows the whole title")
}

// The caret's own cell can push a full row one column too wide, which a row
// count can't see — so this checks width.
func TestLongTitle_EditRowsFitTheWidth(t *testing.T) {
	for _, width := range []int{20, 40, 80} {
		buffer := longWords(300)
		for _, caret := range []int{0, 1, 19, 150, len([]rune(buffer)) - 1, len([]rune(buffer))} {
			m := editingModel(t, width, 40, buffer)
			m.ui.Edit.input.SetCursor(caret)

			item := editedItem(t, m)
			rendered := strings.Split(m.renderLayoutItem(item), "\n")
			assert.Len(t, rendered, item.Height, "w=%d caret=%d: reserved rows must match drawn", width, caret)
			for i, row := range rendered {
				assert.LessOrEqual(t, ansi.StringWidth(row), width,
					"w=%d caret=%d row=%d: %q overflows the field", width, caret, i, row)
			}
		}
	}
}

func TestLongTitle_EditCaretRowStaysOnScreen(t *testing.T) {
	m := editingModel(t, 80, 24, longWords(4000))
	require.Greater(t, editedItem(t, m).Height, 24, "the buffer must outgrow the screen for this to mean anything")

	// The caret starts at the end of the buffer, where the sentinel is.
	require.Greater(t, m.ui.Screen.TopRow, 0, "the caret at the end must have scrolled into the item")
	rendered := strings.Split(m.renderView(), "\n")
	assert.LessOrEqual(t, len(rendered), 24, "the view must not overflow the screen")
	assert.Contains(t, rendered[0], "more above", "rows scrolled off the top are marked")
	assert.NotContains(t, m.renderView(), "> [", "the item's first row is above the window")
	assert.Contains(t, m.renderView(), "ZEND", "the caret's row must be on screen")

	for range 4000 {
		if m.ui.Edit.input.Position() == 0 {
			break
		}
		m.moveEditCursorRow(false)
		m.ensureVisible()
	}
	start, _, ok := m.buildLayout().rowRange(m.selected())
	require.True(t, ok)
	assert.Equal(t, start, m.ui.Screen.TopRow, "the caret at the buffer's start scrolls the view back to its first row")
	assert.LessOrEqual(t, strings.Count(m.renderView(), "\n")+1, 24, "the view must not overflow the screen")
}

func TestLongTitle_EditCursorRowMovesByRow(t *testing.T) {
	// Multibyte runes: the wrap indexes bytes, the widget's cursor counts runes.
	buffer := "héllo wörld and thén söme more téxt that wraps over several rows of the field, " +
		"with enough wörds after it to be sure the caret has rows left to walk"
	m := editingModel(t, 40, 40, buffer)
	m.ui.Edit.input.SetCursor(0)

	available := safeWidth(40, m.editOverhead(mustPosition(t, m)))
	require.Greater(t, len(layoutEdit(buffer, 0, available).rows), 2, "the buffer must wrap for this to mean anything")

	m.moveEditCursorRow(true)
	assert.Equal(t, 1, layoutEdit(buffer, m.ui.Edit.input.Position(), available).cursorRow, "down moves one row")
	m.moveEditCursorRow(true)
	assert.Equal(t, 2, layoutEdit(buffer, m.ui.Edit.input.Position(), available).cursorRow)
	m.moveEditCursorRow(false)
	assert.Equal(t, 1, layoutEdit(buffer, m.ui.Edit.input.Position(), available).cursorRow, "up moves back")

	m.moveEditCursorRow(false)
	m.moveEditCursorRow(false)
	assert.Equal(t, 0, m.ui.Edit.input.Position(), "up past the first row lands on the buffer's start")

	for range 20 {
		m.moveEditCursorRow(true)
	}
	assert.Equal(t, len([]rune(buffer)), m.ui.Edit.input.Position(), "down past the last row lands on the buffer's end")
}

func mustPosition(t *testing.T, m *model) selection.Position {
	t.Helper()
	pos, ok := m.selectedPosition()
	require.True(t, ok)
	return pos
}

func TestLongTitle_CaretMovesWithoutDraggingTheView(t *testing.T) {
	// Correcting the scroll every keystroke would glue the caret to the bottom
	// row and slide the block under it — no visibility check sees that.
	m := editingModel(t, 80, 24, longWords(4000))
	caretRow := func() int {
		edit, ok := m.openEditRows()
		require.True(t, ok)
		return edit.cursorRow
	}
	up := func() {
		m.moveEditCursorRow(false)
		m.ensureVisible()
	}

	settled := m.ui.Screen.TopRow
	require.Greater(t, settled, 0, "the caret at the end must have scrolled into the item")
	start := caretRow()

	for i := 1; i <= 5; i++ {
		up()
		assert.Equal(t, settled, m.ui.Screen.TopRow, "step %d: the window must hold still", i)
		assert.Equal(t, start-i, caretRow(), "step %d: the caret must move one row", i)
	}

	for m.ui.Screen.TopRow == settled {
		up()
	}
	assert.Less(t, m.ui.Screen.TopRow, settled, "past the top of the window the view scrolls after the caret")
}

// Every inline editor wraps, not just a task's: each field's reserved rows and
// drawn rows come from one overhead, so a drift between them shows up here.
func TestLongTitle_EveryEditedRowWraps(t *testing.T) {
	for _, tc := range []struct {
		name string
		kind selection.FocusKind
	}{
		{"project", selection.FocusProject},
		{"category", selection.FocusCategory},
		{"separator", selection.FocusSeparator},
		{"task", selection.FocusTask},
	} {
		t.Run(tc.name, func(t *testing.T) {
			project := longTitleProject("short")
			project.Categories[0].Tasks = append(project.Categories[0].Tasks,
				domain.Task{ID: "s1", Kind: domain.KindSeparator, Title: "divider"})
			m := newTestModel(t, project)
			m.ui.Screen.Width, m.ui.Screen.Height = 80, 60
			m.invalidateLayout()

			pos := -1
			for i, p := range m.positions() {
				if p.Kind == tc.kind {
					pos = i
					break
				}
			}
			require.GreaterOrEqual(t, pos, 0, "no %s row to edit", tc.name)
			startEdit(m, pos, longWords(400), tc.kind)

			item := editedItem(t, m)
			assert.Greater(t, item.Height, ui.MaxLineRows, "the editor grows past the list's clamp")
			assert.Equal(t, item.Height, strings.Count(m.renderLayoutItem(item), "\n")+1)
		})
	}
}
