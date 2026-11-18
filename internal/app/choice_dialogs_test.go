package app

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/app/selection"
	"github.com/ilyasturki/ordo/internal/domain"
	"github.com/ilyasturki/ordo/internal/ui"
)

func confirmingParentDelete(t *testing.T) *model {
	t.Helper()
	m := newNestedModel(t)
	selectRow(t, m, selection.FocusTask, "timer")
	m.deleteSelected()
	require.True(t, m.ui.Modes.IsConfirmDelete())
	return m
}

func TestDeleteConfirm_OffersDeleteKeepAndCancel(t *testing.T) {
	m := confirmingParentDelete(t)
	view := ansi.Strip(m.confirmDeleteView())
	assert.Contains(t, view, "Delete task")
	for _, label := range []string{"Delete", "Keep subtasks", "Cancel"} {
		assert.Contains(t, view, "  "+label)
	}
	assert.NotContains(t, view, "y/enter", "no hint footer")
	assert.NotContains(t, view, "esc")
}

func TestDeleteConfirm_KMovesAndEnterPicksTheRow(t *testing.T) {
	m := confirmingParentDelete(t)
	hit(m, "j", "k", "down")
	assert.True(t, m.ui.Modes.IsConfirmDelete(), "moving never picks")
	assert.Equal(t, 1, m.ui.ConfirmDelete.Pick)
	hit(m, "enter")
	assert.Equal(t, []string{"pick", "sound", "sets"}, shapeOf(catByID(t, m, "ui")), "Keep subtasks")
}

func TestDeleteConfirm_ArrowsReachCancel(t *testing.T) {
	m := confirmingParentDelete(t)
	before := shapeOf(catByID(t, m, "ui"))
	hit(m, "down", "down", "enter")
	assert.True(t, m.ui.Modes.IsNormal())
	assert.Equal(t, before, shapeOf(catByID(t, m, "ui")))
}

func TestDeleteConfirm_MnemonicsAndOldKeys(t *testing.T) {
	for _, tc := range []struct {
		key  string
		want []string
	}{
		{"d", []string{"sets"}},
		{"y", []string{"sets"}},
		{"enter", []string{"sets"}},
		{"s", []string{"pick", "sound", "sets"}},
		{"c", nil},
		{"n", nil},
		{"esc", nil},
		{"q", nil},
	} {
		m := confirmingParentDelete(t)
		before := shapeOf(catByID(t, m, "ui"))
		hit(m, tc.key)
		assert.True(t, m.ui.Modes.IsNormal(), tc.key)
		want := tc.want
		if want == nil {
			want = before
		}
		assert.Equal(t, want, shapeOf(catByID(t, m, "ui")), tc.key)
	}
}

func TestDeleteConfirm_ReopensOnDelete(t *testing.T) {
	m := confirmingParentDelete(t)
	hit(m, "j", "esc")
	selectRow(t, m, selection.FocusTask, "timer")
	m.deleteSelected()
	assert.Equal(t, 0, m.ui.ConfirmDelete.Pick, "a new confirm starts on Delete")
}

func TestProjectDeleteConfirm_DTakesNoProject(t *testing.T) {
	m, _, _, beta := newPickerTestModel(t, "")
	m.ui.Modes.ToProjectPicker()
	hit(m, "d")
	require.True(t, m.isConfirmingProjectDelete())
	hit(m, "d")
	assert.True(t, m.isConfirmingProjectDelete(), "a doubled d deletes no project")
	hit(m, "c")
	assert.True(t, m.ui.Modes.IsProjectPicker())

	hit(m, "d", "y")
	_, err := m.deps.Store.LoadProject(beta.ID)
	assert.Error(t, err, "y deletes")
}

func TestProjectDeleteConfirm_EnterDeletes(t *testing.T) {
	m, _, _, beta := newPickerTestModel(t, "")
	m.ui.Modes.ToProjectPicker()
	hit(m, "d", "enter")
	_, err := m.deps.Store.LoadProject(beta.ID)
	assert.Error(t, err)
}

func TestLeaveConfirm_AnywayCancelAndArrows(t *testing.T) {
	m, _, p := sharedModel(t)
	gone(t, m, p)
	m.selectItem(selection.FocusTask, tasksIn(p, 1)[0].ID)
	hit(m, "l")

	hit(m, "q")
	require.True(t, m.ui.Modes.IsConfirmLeave())
	view := ansi.Strip(m.confirmLeaveView())
	assert.Contains(t, view, "Quit anyway")
	assert.Contains(t, view, "Cancel")
	hit(m, "c")
	assert.True(t, m.ui.Modes.IsNormal())

	hit(m, "q", "down")
	assert.False(t, quits(hit(m, "enter")), "enter on Cancel stays")
	assert.True(t, quits(hit(m, "q", "a")))
}

func TestClashChooser_JKMoveAndTheMergeNoteFollows(t *testing.T) {
	m := newTestModel(t, sampleProject())
	addCategoryNamed(t, m, "cat b")
	assert.NotContains(t, ansi.Strip(m.clashChooserView()), "is dropped", "the note is about the merge row")
	hit(m, "j", "j")
	assert.Equal(t, 2, m.ui.Edit.clashPick)
	assert.Contains(t, ansi.Strip(m.clashChooserView()), "is dropped")
	hit(m, "k")
	assert.Equal(t, 1, m.ui.Edit.clashPick)
	c, _ := m.clashChoice()
	checkMnemonics(t, "clash", c)

	hit(m, "n")
	assert.True(t, m.ui.Modes.IsEdit(), "n goes back to the editor")
	assert.Empty(t, m.ui.Edit.clash)
}

func TestMovePicker_DigitsPickWhileDigitsFilterOnceTyping(t *testing.T) {
	m := newMoveModel(t)
	require.True(t, m.selectItem(selection.FocusTask, "n2"))
	m.startMove()
	assert.Contains(t, ansi.Strip(m.movePickerView()), "  2  ", "rows lead with their digit")
	want := m.ui.MovePicker.targets[1].categoryID
	m = movePickerKeys(m, "2")
	assert.False(t, m.ui.Modes.IsMovePicker())
	ci, _ := m.findTaskByID("n2")
	assert.Equal(t, want, m.project.Categories[ci].ID)

	require.True(t, m.selectItem(selection.FocusTask, "n2"))
	m.startMove()
	m = movePickerKeys(m, "/", "1")
	assert.True(t, m.ui.Modes.IsMovePicker(), "a digit is filter text")
	assert.Equal(t, "1", m.ui.MovePicker.query)
}

func TestTagPicker_DigitsPickAndSpaceStillTogglesTheBatch(t *testing.T) {
	p := sampleProject()
	p.Categories[0].Tasks[0].TagColor = domain.TagGreen
	p.Categories[0].Tasks[1].TagColor = domain.TagGreen
	m := newTestModel(t, p)
	m.ui.Selection.SetSelected(2)

	m.startTagEdit()
	view := ansi.Strip(m.tagEditView())
	assert.Contains(t, view, "None")
	assert.Contains(t, view, "space: all 2")
	m.handleTagEditKey(keyPress("space"))
	require.True(t, m.ui.TagEdit.batch)
	m.handleTagEditKey(keyPress("3"))
	assert.True(t, m.ui.Modes.IsNormal())
	assert.Equal(t, domain.TagBlue, m.project.Categories[0].Tasks[0].TagColor)
	assert.Equal(t, domain.TagBlue, m.project.Categories[0].Tasks[1].TagColor, "the batch took it")
}

func TestEveryDialogKeepsItsMnemonicsFree(t *testing.T) {
	m := confirmingParentDelete(t)
	c, _ := m.confirmDeleteChoice()
	checkMnemonics(t, "delete", c)
	m.ui.Save.Unsaved = 1
	m.ui.Save.Err = assert.AnError
	checkMnemonics(t, "leave", m.confirmLeaveChoice())

	p := newTestModel(t, sampleProject())
	require.True(t, p.selectTaskIdx(0, 0))
	p.pasteText(threeLines, false)
	require.True(t, p.ui.Modes.IsPasteChooser())
	checkMnemonics(t, "paste", p.pasteChoice())
	hit(p, "a")
	assert.Contains(t, p.project.Categories[0].Tasks[0].Description, "milk", "a appends at once")
}

func checkMnemonics(t *testing.T, name string, c ui.Choice) {
	t.Helper()
	seen := map[string]bool{}
	for _, o := range c.Options {
		for _, k := range append([]string{o.Key}, o.Aliases...) {
			if k == "" {
				continue
			}
			for _, r := range ui.ChoiceReservedKeys {
				assert.NotEqual(t, r, k, "%s: %s takes a reserved key", name, o.Label)
			}
			assert.False(t, seen[k], "%s: %s shares %q", name, o.Label, k)
			seen[k] = true
		}
		if o.Key != "" {
			assert.True(t, strings.Contains(strings.ToLower(o.Label), strings.ToLower(o.Key)), "%s: %q holds its key %q", name, o.Label, o.Key)
		}
	}
}
