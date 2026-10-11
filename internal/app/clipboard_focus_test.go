package app

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sendMsg(m *model, msg tea.Msg) *model {
	next, _ := m.Update(msg)
	updated := next.(model)
	return &updated
}

// copiedWhileFocused copies Alpha, lands the write, and puts the cursor on Delta.
func copiedWhileFocused(t *testing.T) *model {
	t.Helper()
	m := newTestModel(t, sampleProject())
	m = sendMsg(m, tea.FocusMsg{})
	require.True(t, m.selectTaskIdx(0, 0))
	m.copySelected()
	landWrite(m)
	require.True(t, m.selectTaskIdx(1, 0))
	return m
}

func TestPaste_NoBlurSinceOurWriteSkipsTheRead(t *testing.T) {
	m := copiedWhileFocused(t)

	assert.Nil(t, m.paste(), "nothing else can have copied: no read")
	assert.Equal(t, []string{"Delta", "Alpha", "Epsilon"}, taskTitles(m.project.Categories[1]))
}

func TestPaste_ABlurSinceOurWriteReadsAgain(t *testing.T) {
	m := copiedWhileFocused(t)
	m = sendMsg(m, tea.BlurMsg{})
	m = sendMsg(m, tea.FocusMsg{})

	assert.NotNil(t, m.paste(), "another app may have copied while the terminal was away")
}

func TestPaste_AWriteAnsweredAfterABlurIsNotFresh(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m = sendMsg(m, tea.FocusMsg{})
	require.True(t, m.selectTaskIdx(0, 0))
	m.copySelected()
	m = sendMsg(m, tea.BlurMsg{})
	m = sendMsg(m, tea.FocusMsg{})
	m.handleClipboardResult(clipboardResultMsg{text: m.ui.Clipboard.Text, ofCopy: true, blurs: 0})

	require.True(t, m.selectTaskIdx(1, 0))
	assert.NotNil(t, m.paste())
}

func TestPaste_ATerminalThatNeverReportsFocusAlwaysReads(t *testing.T) {
	m := newTestModel(t, sampleProject())
	require.True(t, m.selectTaskIdx(0, 0))
	m.copySelected()
	landWrite(m)

	require.True(t, m.selectTaskIdx(1, 0))
	assert.NotNil(t, m.paste())
}

func TestPaste_OurLastWriteOfOtherTextPastesThatText(t *testing.T) {
	m := copiedWhileFocused(t)
	m.handleClipboardResult(clipboardResultMsg{text: "9652ac64-1187"})

	assert.Nil(t, m.paste(), "ordo itself put the ID there")
	assert.Equal(t, []string{"Delta", "9652ac64-1187", "Epsilon"}, taskTitles(m.project.Categories[1]))
	assert.True(t, m.ui.Clipboard.empty(), "the clipboard no longer holds the copy, so the copy retires")
}

func TestPaste_AFailedWriteLeavesTheEarlierOneFresh(t *testing.T) {
	m := copiedWhileFocused(t)
	m.handleClipboardResult(clipboardResultMsg{text: "never arrived", err: assert.AnError})

	assert.Nil(t, m.paste())
	assert.Equal(t, []string{"Delta", "Alpha", "Epsilon"}, taskTitles(m.project.Categories[1]))
}

func TestPaste_ANewWriteAfterABlurIsFreshAgain(t *testing.T) {
	m := copiedWhileFocused(t)
	m = sendMsg(m, tea.BlurMsg{})
	m = sendMsg(m, tea.FocusMsg{})
	m.copySelected()
	landWrite(m)

	require.True(t, m.selectTaskIdx(0, 0))
	assert.Nil(t, m.paste())
	assert.Equal(t, []string{"Alpha", "Delta", "Beta", "Gamma"}, taskTitles(m.project.Categories[0]))
}

func TestPaste_ATaskCommandMayHaveCopiedSoPReadsAgain(t *testing.T) {
	m := copiedWhileFocused(t)
	m = sendMsg(m, taskCommandResultMsg{label: "copy branch"})

	assert.NotNil(t, m.paste(), "the command may have written the clipboard")
}

func TestPaste_TheExternalEditorMayHaveCopiedSoPReadsAgain(t *testing.T) {
	m := copiedWhileFocused(t)
	m.handleEditorFinished(editorFinishedMsg{err: assert.AnError})

	require.True(t, m.selectTaskIdx(1, 0))
	assert.NotNil(t, m.paste(), "no BlurMsg reaches us while the editor holds the terminal")
}
