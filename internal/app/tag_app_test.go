package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"phasionary/internal/domain"
)

func TestCycleTag_AdvancesSelectedTask(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.SetSelected(2) // t1

	m.cycleTag()
	assert.Equal(t, domain.TagGreen, m.project.Categories[0].Tasks[0].TagColor)
	m.cycleTag()
	assert.Equal(t, domain.TagBlue, m.project.Categories[0].Tasks[0].TagColor)
}

func TestCycleTag_IgnoresNonTaskSelection(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.SetSelected(0) // project row
	m.cycleTag()
	assert.Equal(t, "", m.project.Categories[0].Tasks[0].TagColor)
}

func TestTagEdit_PicksSpecificColor(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.SetSelected(2)

	m.startTagEdit()
	m.ui.TagEdit.colorIdx = tagColorIndex(domain.TagMagenta)
	m.finishTagEdit()

	assert.True(t, m.ui.Modes.IsNormal())
	assert.Equal(t, domain.TagMagenta, m.project.Categories[0].Tasks[0].TagColor)
}

func TestTagEdit_NoneRemovesTag(t *testing.T) {
	p := sampleProject()
	p.Categories[0].Tasks[0].TagColor = domain.TagCyan
	m := newTestModel(t, p)
	m.ui.Selection.SetSelected(2)

	m.startTagEdit()
	// The editor seeds from the current color; select "None" to remove.
	require.Equal(t, tagColorIndex(domain.TagCyan), m.ui.TagEdit.colorIdx)
	m.ui.TagEdit.colorIdx = 0 // None
	m.finishTagEdit()

	assert.Equal(t, "", m.project.Categories[0].Tasks[0].TagColor)
}

func TestTagEdit_CancelLeavesTaskUntouched(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.SetSelected(2)

	m.startTagEdit()
	m.ui.TagEdit.colorIdx = tagColorIndex(domain.TagBlue)
	m.cancelTagEdit()

	assert.True(t, m.ui.Modes.IsNormal())
	assert.Equal(t, "", m.project.Categories[0].Tasks[0].TagColor)
}

func TestVisualCycleTag_SetsSharedColorAcrossSelection(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.SetSelected(2) // t1
	m.enterVisualMode()
	m.visualMoveCursor(1) // extend to t2

	m.visualCycleTag()
	assert.Equal(t, domain.TagGreen, m.project.Categories[0].Tasks[0].TagColor)
	assert.Equal(t, domain.TagGreen, m.project.Categories[0].Tasks[1].TagColor)
	assert.True(t, m.ui.Modes.IsVisual(), "visual mode should persist so t can be pressed again")

	m.visualCycleTag()
	assert.Equal(t, domain.TagBlue, m.project.Categories[0].Tasks[0].TagColor)
	assert.Equal(t, domain.TagBlue, m.project.Categories[0].Tasks[1].TagColor)
}

func TestVisualCycleTag_MixedColorsConvergeToShared(t *testing.T) {
	p := sampleProject()
	p.Categories[0].Tasks[1].TagColor = domain.TagMagenta
	m := newTestModel(t, p)
	m.ui.Selection.SetSelected(2)
	m.enterVisualMode()
	m.visualMoveCursor(1)

	// First task is untagged, so the whole block advances to green.
	m.visualCycleTag()
	assert.Equal(t, domain.TagGreen, m.project.Categories[0].Tasks[0].TagColor)
	assert.Equal(t, domain.TagGreen, m.project.Categories[0].Tasks[1].TagColor)
}

func TestVisualTagEdit_AppliesSharedTagAndExitsVisual(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.SetSelected(2)
	m.enterVisualMode()
	m.visualMoveCursor(1) // t1..t2

	m.visualStartTagEdit()
	require.True(t, m.ui.Modes.IsTagEdit())
	require.Len(t, m.ui.TagEdit.taskIDs, 2)
	m.ui.TagEdit.colorIdx = tagColorIndex(domain.TagCyan)
	m.finishTagEdit()

	assert.True(t, m.ui.Modes.IsNormal())
	assert.Equal(t, domain.TagCyan, m.project.Categories[0].Tasks[0].TagColor)
	assert.Equal(t, domain.TagCyan, m.project.Categories[0].Tasks[1].TagColor)
}

func TestCopyPasteTag_NormalMode(t *testing.T) {
	p := sampleProject()
	p.Categories[0].Tasks[0].TagColor = domain.TagMagenta
	m := newTestModel(t, p)

	// Copy the tag from t1.
	m.ui.Selection.SetSelected(2) // t1
	m.copyTagFromSelected()
	require.True(t, m.ui.TagClip.Set)
	assert.Equal(t, domain.TagMagenta, m.ui.TagClip.Color)

	// Paste onto t2 (leaves title/status untouched).
	m.ui.Selection.SetSelected(3) // t2
	titleBefore := m.project.Categories[0].Tasks[1].Title
	statusBefore := m.project.Categories[0].Tasks[1].Status
	m.pasteTagOntoSelected()

	assert.Equal(t, domain.TagMagenta, m.project.Categories[0].Tasks[1].TagColor)
	assert.Equal(t, titleBefore, m.project.Categories[0].Tasks[1].Title)
	assert.Equal(t, statusBefore, m.project.Categories[0].Tasks[1].Status)
}

func TestPasteTag_WithoutCopyIsNoop(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.SetSelected(2)
	m.pasteTagOntoSelected()
	assert.Equal(t, "", m.project.Categories[0].Tasks[0].TagColor)
}

func TestPaste_PaintsTagWhenTagCopiedLast(t *testing.T) {
	p := sampleProject()
	p.Categories[0].Tasks[0].TagColor = domain.TagMagenta
	m := newTestModel(t, p)

	m.ui.Selection.SetSelected(2) // t1
	m.copyTagFromSelected()
	require.True(t, m.ui.TagCopiedLast)

	// With the tag as the freshest copy, p paints it onto the focused task.
	m.ui.Selection.SetSelected(3) // t2
	m.paste()

	assert.Equal(t, domain.TagMagenta, m.project.Categories[0].Tasks[1].TagColor)
}

func TestPaste_MovesTaskWhenCutIsFreshest(t *testing.T) {
	p := sampleProject()
	p.Categories[0].Tasks[0].TagColor = domain.TagMagenta
	m := newTestModel(t, p)

	// Copy a tag, then cut a task: the cut is now the freshest clipboard action.
	m.ui.Selection.SetSelected(2) // t1
	m.copyTagFromSelected()
	m.ui.Selection.SetSelected(3) // t2
	m.cutSelectedTask()
	landWrite(m)
	require.False(t, m.ui.TagCopiedLast, "a cut resets the tag-copied-last flag")

	m.ui.Selection.SetSelected(4) // t3
	require.NotNil(t, m.paste(), "p reads the clipboard rather than painting the tag")
	m.resolvePaste(clipboardLinesMsg{text: m.ui.Clipboard.Text})

	assert.False(t, m.ui.Clipboard.IsCut, "p should move the cut task, not paint the tag")
	assert.Equal(t, []string{"t1", "t3", "t2"}, taskIDs(m.project.Categories[0]))
	assert.Equal(t, "", m.project.Categories[0].Tasks[1].TagColor, "t3 tag untouched")
}

func TestCopyPasteTag_EmptyTagClearsTarget(t *testing.T) {
	p := sampleProject()
	p.Categories[0].Tasks[1].TagColor = domain.TagBlue
	m := newTestModel(t, p)

	// t1 is untagged; copying it yields an empty tag that pastes as a clear.
	m.ui.Selection.SetSelected(2) // t1 (untagged)
	m.copyTagFromSelected()
	m.ui.Selection.SetSelected(3) // t2 (blue/old)
	m.pasteTagOntoSelected()

	assert.Equal(t, "", m.project.Categories[0].Tasks[1].TagColor)
}

func TestVisualPasteTag_AppliesToSelection(t *testing.T) {
	p := sampleProject()
	p.Categories[0].Tasks[0].TagColor = domain.TagCyan
	m := newTestModel(t, p)

	m.ui.Selection.SetSelected(2) // t1
	m.copyTagFromSelected()

	// Select t2..t3 and paint the tag onto both.
	m.ui.Selection.SetSelected(3) // t2
	m.enterVisualMode()
	m.visualMoveCursor(1) // extend to t3
	m.visualPasteTag()

	assert.True(t, m.ui.Modes.IsNormal())
	assert.Equal(t, domain.TagCyan, m.project.Categories[0].Tasks[1].TagColor)
	assert.Equal(t, domain.TagCyan, m.project.Categories[0].Tasks[2].TagColor)
}
