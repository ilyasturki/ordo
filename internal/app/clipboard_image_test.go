package app

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/clipboard"
)

func pngBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, w, h))))
	return buf.Bytes()
}

// clipboardHolds puts data in the file standing in for the clipboard.
func clipboardHolds(t *testing.T, data []byte) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "clipboard")
	require.NoError(t, os.WriteFile(path, data, 0o600))
	t.Setenv(clipboard.EnvFile, path)
}

// runAll runs cmd and feeds every message it yields back through Update.
func runAll(t *testing.T, m *model, cmd tea.Cmd) {
	t.Helper()
	for cmd != nil {
		cmd = send(t, m, cmd())
	}
}

func TestPaste_AnImageOnTheClipboardIsNeverReadAsText(t *testing.T) {
	m := newTestModel(t, sampleProject())
	require.True(t, m.selectTaskIdx(0, 0))
	clipboardHolds(t, pngBytes(t, 64, 64))
	before := taskTitles(m.project.Categories[0])

	runAll(t, m, m.paste())

	assert.True(t, m.ui.Modes.IsNormal(), "no paste chooser for the image's bytes")
	assert.Equal(t, before, taskTitles(m.project.Categories[0]))
	assert.Equal(t, "The clipboard holds an image", m.ui.Screen.StatusMsg)
}

func TestPaste_AnImageCopiedAfterOursIsNotOurCopy(t *testing.T) {
	m := copiedWhileFocused(t)
	m = sendMsg(m, tea.BlurMsg{})
	m = sendMsg(m, tea.FocusMsg{})
	clipboardHolds(t, pngBytes(t, 4, 4))
	before := taskTitles(m.project.Categories[1])

	runAll(t, m, m.paste())

	assert.Equal(t, before, taskTitles(m.project.Categories[1]), "the copied task must not be pasted")
	assert.Equal(t, "The clipboard holds an image", m.ui.Screen.StatusMsg)
}

func TestPaste_BinaryBytesAreNotText(t *testing.T) {
	m := newTestModel(t, sampleProject())
	require.True(t, m.selectTaskIdx(0, 0))
	clipboardHolds(t, []byte("%PDF-1.7\n\x00\xe2\xe3"))

	runAll(t, m, m.paste())

	assert.Len(t, m.project.Categories[0].Tasks, 3)
	assert.Equal(t, "The clipboard holds no text", m.ui.Screen.StatusMsg)
}

func TestEditors_CtrlVWithAnImageInsertsNothing(t *testing.T) {
	clipboardHolds(t, pngBytes(t, 4, 4))
	t.Run("title", func(t *testing.T) {
		m := newTestModel(t, sampleProject())
		require.True(t, m.selectTaskIdx(0, 0))
		m.startEditing()

		runAll(t, m, send(t, m, key("ctrl+v")))

		assert.Equal(t, "Alpha", m.ui.Edit.input.Value())
		assert.Equal(t, "The clipboard holds an image", m.ui.Screen.StatusMsg)
	})
	t.Run("search", func(t *testing.T) {
		m := newTestModel(t, sampleProject())
		send(t, m, key("/"))
		require.True(t, m.ui.Modes.IsSearch())

		runAll(t, m, send(t, m, key("ctrl+v")))

		assert.Empty(t, m.ui.Search.input.Value())
		assert.Equal(t, "The clipboard holds an image", m.ui.Screen.StatusMsg)
	})
}
