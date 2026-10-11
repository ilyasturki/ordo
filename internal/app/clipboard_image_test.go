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

// runAll runs cmd and feeds its clipboard and attach messages back through
// Update, up to the attach. Anything else, such as a cursor blink that ticks
// forever, ends it.
func runAll(t *testing.T, m *model, cmd tea.Cmd) {
	t.Helper()
	for cmd != nil {
		msg := cmd()
		switch msg.(type) {
		case clipboardLinesMsg, editorClipboardMsg:
			cmd = send(t, m, msg)
		case imagesStoredMsg:
			send(t, m, msg)
			return
		default:
			return
		}
	}
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
