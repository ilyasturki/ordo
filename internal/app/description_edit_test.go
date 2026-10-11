package app

import (
	"fmt"
	"os"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/phasionary/internal/app/modes"
	"github.com/ilyasturki/phasionary/internal/app/selection"
)

// descriptionMarkerRow renders the description textarea and returns the index of
// the single display row whose ">" gutter is drawn (i.e. the cursor's row).
func descriptionMarkerRow(t *testing.T, m *model) int {
	t.Helper()
	require.NotNil(t, m.ui.DescriptionEdit.cursorRow)
	*m.ui.DescriptionEdit.cursorRow = descriptionCursorDisplayRow(m.ui.DescriptionEdit.textarea)
	rows := strings.Split(ansi.Strip(m.ui.DescriptionEdit.textarea.View()), "\n")
	found := -1
	for i, r := range rows {
		if strings.HasPrefix(r, "> ") {
			require.Equal(t, -1, found, "expected one gutter marker, found rows %d and %d", found, i)
			found = i
		}
	}
	require.NotEqual(t, -1, found, "no > gutter marker in:\n%s", strings.Join(rows, "\n"))
	return found
}

func TestDescriptionEdit_GutterMarkerFollowsCursor(t *testing.T) {
	p := sampleProject()
	p.Categories[0].Tasks[0].Description = "alpha\nbeta\ngamma"
	m := newTestModel(t, p)
	m.ui.Screen.Width = 60
	m.ui.Screen.Height = 24
	_ = m.startDescriptionInlineEdit(0, 0)

	// Cursor starts at the end → last logical line.
	require.Equal(t, 2, descriptionMarkerRow(t, m))

	// Moving up walks the marker with the cursor, one row at a time.
	_, _ = m.handleDescriptionEditKey(tea.KeyPressMsg{Code: tea.KeyUp})
	require.Equal(t, 1, descriptionMarkerRow(t, m))
	_, _ = m.handleDescriptionEditKey(tea.KeyPressMsg{Code: tea.KeyUp})
	require.Equal(t, 0, descriptionMarkerRow(t, m))
}

func TestHandleMouseWheel_ScrollsDescriptionEditor(t *testing.T) {
	// A description taller than the editor viewport must scroll with the wheel.
	p := sampleProject()
	var b strings.Builder
	for i := 0; i < 30; i++ {
		if i > 0 {
			b.WriteByte('\n')
		}
		fmt.Fprintf(&b, "line%02d", i)
	}
	p.Categories[0].Tasks[0].Description = b.String()
	m := newTestModel(t, p)
	m.ui.Screen.Width = 60
	m.ui.Screen.Height = 24
	_ = m.startDescriptionInlineEdit(0, 0)
	// Render once so the textarea's viewport learns its content height, exactly
	// as the app's render loop does before any wheel event arrives.
	_ = m.ui.DescriptionEdit.textarea.View()

	// wheel sends `lines` worth of wheel events in one direction. It takes
	// wheelScrollDivisor raw events to move a single line.
	wheel := func(button tea.MouseButton, lines int) {
		for i := 0; i < lines*wheelScrollDivisor; i++ {
			_, _ = m.Update(tea.MouseWheelMsg{Button: button})
		}
	}

	// Cursor opens at the end (last line). One line of wheel-down is a no-op on
	// the cursor but forces the view to settle at the bottom, giving a baseline.
	require.Equal(t, 29, m.ui.DescriptionEdit.textarea.Line())
	wheel(tea.MouseWheelDown, 1)
	require.Equal(t, 29, m.ui.DescriptionEdit.textarea.Line())
	bottomOffset := m.ui.DescriptionEdit.textarea.ScrollYOffset()
	require.Positive(t, bottomOffset, "a description taller than the editor must scroll")

	// Twenty lines of wheel-up move the cursor up twenty lines and scroll the
	// viewport toward the top.
	wheel(tea.MouseWheelUp, 20)
	assert.Equal(t, 9, m.ui.DescriptionEdit.textarea.Line(),
		"wheel-up moves one line per wheelScrollDivisor events")
	assert.Less(t, m.ui.DescriptionEdit.textarea.ScrollYOffset(), bottomOffset, "view scrolled up")

	// Wheeling back down returns to the bottom.
	wheel(tea.MouseWheelDown, 20)
	assert.Equal(t, 29, m.ui.DescriptionEdit.textarea.Line())
	assert.Equal(t, bottomOffset, m.ui.DescriptionEdit.textarea.ScrollYOffset())
}

func TestWheelTick_AccumulatesAndReversesDirection(t *testing.T) {
	m := newTestModel(t, sampleProject())

	// It takes wheelScrollDivisor events to emit one tick.
	for i := 0; i < wheelScrollDivisor-1; i++ {
		assert.False(t, m.wheelTick(tea.MouseWheelDown), "event %d should still accumulate", i+1)
	}
	assert.True(t, m.wheelTick(tea.MouseWheelDown), "the wheelScrollDivisor-th event emits a tick")
	assert.Equal(t, 0, m.ui.Screen.WheelAccum, "accumulator resets after a tick")

	// Reversing direction discards the partial accumulation.
	assert.False(t, m.wheelTick(tea.MouseWheelDown))
	assert.False(t, m.wheelTick(tea.MouseWheelUp), "reversal restarts the count")
	assert.Equal(t, -1, m.ui.Screen.WheelAccum)
}

func TestStartDescriptionInlineEdit_EntersModeAndPrefillsTextarea(t *testing.T) {
	p := sampleProject()
	p.Categories[0].Tasks[0].Description = "Hello\nworld"
	m := newTestModel(t, p)

	_ = m.startDescriptionInlineEdit(0, 0)

	assert.True(t, m.ui.Modes.IsDescriptionEdit(), "should be in description-edit mode")
	assert.Equal(t, "Hello\nworld", m.ui.DescriptionEdit.textarea.Value())
	assert.Equal(t, 0, m.ui.DescriptionEdit.categoryIndex)
	assert.Equal(t, 0, m.ui.DescriptionEdit.taskIndex)
	assert.Equal(t, "Hello\nworld", m.ui.DescriptionEdit.original)
	assert.False(t, m.ui.DescriptionEdit.creating)
}

func TestStartDescriptionInlineEdit_CreatingWhenEmpty(t *testing.T) {
	m := newTestModel(t, sampleProject()) // t1 has no description
	_ = m.startDescriptionInlineEdit(0, 0)
	assert.True(t, m.ui.DescriptionEdit.creating, "creating flag set when starting from empty description")
}

func TestFinishDescriptionEdit_SavesAndLeavesMode(t *testing.T) {
	p := sampleProject()
	m := newTestModel(t, p)
	_ = m.startDescriptionInlineEdit(0, 0)
	m.ui.DescriptionEdit.textarea.SetValue("New body line 1\nline 2")
	m.finishDescriptionEdit()

	assert.True(t, m.ui.Modes.IsNormal())
	assert.Equal(t, "New body line 1\nline 2", m.project.Categories[0].Tasks[0].Description)
	assert.Equal(t, "Description added", m.ui.Screen.StatusMsg)
}

func TestFinishDescriptionEdit_EmptyValueClearsDescription(t *testing.T) {
	p := sampleProject()
	p.Categories[0].Tasks[0].Description = "to be removed"
	m := newTestModel(t, p)
	_ = m.startDescriptionInlineEdit(0, 0)
	m.ui.DescriptionEdit.textarea.SetValue("   ") // whitespace only
	m.finishDescriptionEdit()

	assert.Equal(t, "", m.project.Categories[0].Tasks[0].Description)
	assert.Equal(t, "Description cleared", m.ui.Screen.StatusMsg)
}

func TestCancelDescriptionEdit_DiscardsChanges(t *testing.T) {
	p := sampleProject()
	p.Categories[0].Tasks[0].Description = "original"
	m := newTestModel(t, p)
	_ = m.startDescriptionInlineEdit(0, 0)
	m.ui.DescriptionEdit.textarea.SetValue("would-be-new")
	m.cancelDescriptionEdit()

	assert.True(t, m.ui.Modes.IsNormal())
	assert.Equal(t, "original", m.project.Categories[0].Tasks[0].Description,
		"cancel must not persist the typed value")
}

func TestHandleDescriptionEditKey_EscCancels(t *testing.T) {
	m := newTestModel(t, sampleProject())
	_ = m.startDescriptionInlineEdit(0, 0)
	m.ui.DescriptionEdit.textarea.SetValue("typed but not saved")

	_, _ = m.handleDescriptionEditKey(tea.KeyPressMsg{Code: tea.KeyEscape})

	assert.True(t, m.ui.Modes.IsNormal())
	assert.Equal(t, "", m.project.Categories[0].Tasks[0].Description)
}

func TestHandleDescriptionEditKey_EnterSaves(t *testing.T) {
	p := sampleProject()
	p.Categories[0].Tasks[0].Description = "existing body"
	m := newTestModel(t, p)
	_ = m.startDescriptionInlineEdit(0, 0)

	_, _ = m.handleDescriptionEditKey(tea.KeyPressMsg{Code: tea.KeyEnter})

	assert.True(t, m.ui.Modes.IsNormal(), "plain enter saves and leaves edit mode")
	assert.Equal(t, "existing body", m.project.Categories[0].Tasks[0].Description)
}

func TestHandleDescriptionEditKey_ShiftEnterInsertsNewline(t *testing.T) {
	// Both wire encodings of Shift+Enter must insert a newline without saving:
	// a genuine "shift+enter" and the "alt+enter" (ESC+CR) that kitty/ghostty
	// deliver.
	for _, key := range []tea.KeyPressMsg{
		{Code: tea.KeyEnter, Mod: tea.ModShift},
		{Code: tea.KeyEnter, Mod: tea.ModAlt},
	} {
		p := sampleProject()
		p.Categories[0].Tasks[0].Description = "abc" // cursor starts at end
		m := newTestModel(t, p)
		_ = m.startDescriptionInlineEdit(0, 0)

		_, _ = m.handleDescriptionEditKey(key)

		assert.True(t, m.ui.Modes.IsDescriptionEdit(), "%s must not save", key.String())
		assert.Equal(t, "abc\n", m.ui.DescriptionEdit.textarea.Value(),
			"%s must insert a newline at the cursor", key.String())
	}
}

func TestHandleDescriptionEditKey_CtrlSSaves(t *testing.T) {
	m := newTestModel(t, sampleProject())
	_ = m.startDescriptionInlineEdit(0, 0)
	m.ui.DescriptionEdit.textarea.SetValue("saved via ctrl+s")

	_, _ = m.handleDescriptionEditKey(tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})

	assert.True(t, m.ui.Modes.IsNormal())
	assert.Equal(t, "saved via ctrl+s", m.project.Categories[0].Tasks[0].Description)
}

func TestForwardToInput_DescriptionEdit_InsertsBracketedPaste(t *testing.T) {
	// Bracketed paste from the terminal arrives as a tea.PasteMsg (not a key
	// press), routed through forwardToInput. Description-edit mode must forward
	// it to the textarea or paste silently does nothing (the reported bug).
	m := newTestModel(t, sampleProject())
	_ = m.startDescriptionInlineEdit(0, 0)
	m.ui.DescriptionEdit.textarea.SetValue("start ")
	m.ui.DescriptionEdit.textarea.CursorEnd()

	_, _ = m.forwardToInput(tea.PasteMsg{Content: "pasted\ntext"})

	assert.Equal(t, "start pasted\ntext", m.ui.DescriptionEdit.textarea.Value(),
		"bracketed paste must be inserted at the cursor")
}

func TestDescriptionEdit_CtrlArrowWordNavAndDelete(t *testing.T) {
	m := newTestModel(t, sampleProject())
	_ = m.startDescriptionInlineEdit(0, 0)
	m.ui.DescriptionEdit.textarea.SetValue("alpha beta gamma")
	m.ui.DescriptionEdit.textarea.CursorEnd()

	// ctrl+left steps back one word: cursor moves before "gamma".
	_, _ = m.handleDescriptionEditKey(tea.KeyPressMsg{Code: tea.KeyLeft, Mod: tea.ModCtrl})
	// ctrl+backspace deletes the word now to the left of the cursor ("beta ").
	_, _ = m.handleDescriptionEditKey(tea.KeyPressMsg{Code: tea.KeyBackspace, Mod: tea.ModCtrl})

	assert.Equal(t, "alpha gamma", m.ui.DescriptionEdit.textarea.Value(),
		"ctrl+left skips a word back, ctrl+backspace deletes the preceding word")
}

func TestEditOrFocusDescription_TaskWithoutDescription_EntersInlineEditMode(t *testing.T) {
	// Regression guard for the v2 of this feature: pressing shift+enter on a
	// task with no description used to open the external editor; it should
	// now drop the user straight into the in-app textarea.
	m := newTestModel(t, sampleProject())
	m.ui.Selection.SelectByPredicate(func(p selection.Position) bool {
		return p.Kind == selection.FocusTask && p.CategoryIndex == 0 && p.TaskIndex == 0
	})

	_ = m.editOrFocusDescription()

	require.Equal(t, modes.ModeDescriptionEdit, m.ui.Modes.Current())
	assert.Equal(t, "", m.ui.DescriptionEdit.textarea.Value())
	assert.True(t, m.ui.DescriptionEdit.creating)
}

func TestStartExternalEdit_FromDescriptionRow_EditsWholeTask(t *testing.T) {
	// `e` on a description row should open the external editor with the full
	// title + description body, not just the description.
	p := sampleProject()
	p.Categories[0].Tasks[0].Description = "Body text"
	m := newTestModel(t, p)
	m.ui.Screen.ExpandDescriptions = true
	m.rebuildPositions()
	m.ui.Selection.SelectByPredicate(func(pp selection.Position) bool {
		return pp.Kind == selection.FocusDescription && pp.CategoryIndex == 0 && pp.TaskIndex == 0
	})

	// We can't easily run the editor process inside a unit test, but we can
	// verify that the state transition uses ItemType = FocusTask.
	_ = m.startExternalEdit()
	assert.Equal(t, selection.FocusTask, m.ui.ExternalEdit.ItemType,
		"external edit from a description row must be scoped to the whole task")
	assert.Equal(t, 0, m.ui.ExternalEdit.CategoryIndex)
	assert.Equal(t, 0, m.ui.ExternalEdit.TaskIndex)
	// Clean up the temp file that startExternalEdit created.
	if m.ui.ExternalEdit.TempFilePath != "" {
		_ = os.Remove(m.ui.ExternalEdit.TempFilePath)
	}
}
