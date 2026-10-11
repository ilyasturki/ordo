package app

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/app/modes"
	"github.com/ilyasturki/ordo/internal/app/selection"
	"github.com/ilyasturki/ordo/internal/attachments"
)

var screenshotRef = regexp.MustCompile(`^!\[Screenshot \d{4}-\d\d-\d\d \d\d\.\d\d\.\d\d\]\(attachments/[0-9a-f]{32}\.png\)$`)

func newAttachModel(t *testing.T) (*model, *attachments.Store) {
	t.Helper()
	m := newTestModel(t, sampleProject())
	store := attachments.NewStore(t.TempDir())
	m.deps.Attachments = store
	return m, store
}

// imageFile writes a png named name into a fresh directory.
func imageFile(t *testing.T, name string, w int) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, pngBytes(t, w, w), 0o600))
	return path
}

func storedFiles(t *testing.T, store *attachments.Store) []string {
	t.Helper()
	entries, err := os.ReadDir(store.Dir())
	if os.IsNotExist(err) {
		return nil
	}
	require.NoError(t, err)
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Name()
	}
	return names
}

func TestPaste_AttachesTheClipboardImageToTheFocusedTask(t *testing.T) {
	m, store := newAttachModel(t)
	require.True(t, m.selectTaskIdx(0, 1))
	clipboardHolds(t, pngBytes(t, 64, 64))

	runAll(t, m, m.paste())

	beta := m.project.Categories[0].Tasks[1]
	assert.Regexp(t, screenshotRef, beta.Description)
	assert.Equal(t, []string{"Alpha", "Beta", "Gamma"}, taskTitles(m.project.Categories[0]), "no paste chooser for the image's bytes")
	assert.Regexp(t, `^Attached Screenshot .* to Beta$`, m.ui.Screen.StatusMsg)
	refs := attachments.Refs(beta.Description)
	require.Len(t, refs, 1)
	path, _ := store.Path(refs[0].Target)
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, pngBytes(t, 64, 64), got)

	m.undo()
	assert.Empty(t, m.project.Categories[0].Tasks[1].Description)
	assert.Len(t, storedFiles(t, store), 1, "undo never deletes the file")
}

func TestPaste_AppendsTheImageBelowTheDescription(t *testing.T) {
	m, _ := newAttachModel(t)
	m.project.Categories[0].Tasks[0].Description = "the overflow here:\n"
	m.ui.Screen.ExpandDescriptions = true
	m.rebuildPositions()
	selectRow(t, m, selection.FocusDescription, "t1")
	clipboardHolds(t, pngBytes(t, 8, 8))

	runAll(t, m, m.paste())

	lines := strings.Split(m.project.Categories[0].Tasks[0].Description, "\n")
	require.Len(t, lines, 2)
	assert.Equal(t, "the overflow here:", lines[0])
	assert.Regexp(t, screenshotRef, lines[1])
}

func TestPaste_AnImageNeedsATask(t *testing.T) {
	m, store := newAttachModel(t)
	selectRow(t, m, selection.FocusCategory, "c1")
	clipboardHolds(t, pngBytes(t, 8, 8))

	runAll(t, m, m.paste())

	assert.Equal(t, "Focus a task to attach an image", m.ui.Screen.StatusMsg)
	assert.Empty(t, storedFiles(t, store), "nothing stored for a refused image")
	for _, task := range m.project.Categories[0].Tasks {
		assert.Empty(t, task.Description)
	}
}

func TestPaste_AnImageCopiedAfterOursRetiresOurCopy(t *testing.T) {
	m := copiedWhileFocused(t)
	m.deps.Attachments = attachments.NewStore(t.TempDir())
	m = sendMsg(m, tea.BlurMsg{})
	m = sendMsg(m, tea.FocusMsg{})
	clipboardHolds(t, pngBytes(t, 4, 4))

	runAll(t, m, m.paste())

	assert.Equal(t, []string{"Delta", "Epsilon"}, taskTitles(m.project.Categories[1]), "the copied task must not be pasted")
	assert.Regexp(t, screenshotRef, m.project.Categories[1].Tasks[0].Description)
	assert.True(t, m.ui.Clipboard.empty())

	m.undo()
	assert.False(t, m.ui.Clipboard.empty(), "undo brings the copy back with the description")
	assert.Empty(t, m.project.Categories[1].Tasks[0].Description)
}

func TestDrop_AttachesEveryImageInOneUndoStep(t *testing.T) {
	m, store := newAttachModel(t)
	require.True(t, m.selectTaskIdx(0, 0))
	spaced := imageFile(t, "Screenshot 2026-10-11 10-10-10.png", 4)
	quoted := imageFile(t, "it's here.png", 5)
	uri := imageFile(t, "shot #3.png", 6)
	escaped := strings.ReplaceAll(spaced, " ", `\ `)
	drop := escaped + " '" + strings.ReplaceAll(quoted, "'", `'\''`) + "'\n" +
		"file://" + strings.ReplaceAll(strings.ReplaceAll(uri, " ", "%20"), "#", "%23") + "\n"

	runAll(t, m, send(t, m, tea.PasteMsg{Content: drop}))

	assert.Equal(t, "![Screenshot 2026-10-11 10-10-10.png](attachments/"+hashName(t, spaced)+")\n"+
		"![it's here.png](attachments/"+hashName(t, quoted)+")\n"+
		"![shot #3.png](attachments/"+hashName(t, uri)+")", m.project.Categories[0].Tasks[0].Description)
	assert.Equal(t, "Attached 3 images to Alpha", m.ui.Screen.StatusMsg)
	assert.Len(t, storedFiles(t, store), 3)

	m.undo()
	assert.Empty(t, m.project.Categories[0].Tasks[0].Description, "one undo step for the whole drop")
}

func TestDrop_AnythingButImageFilesIsPastedAsBefore(t *testing.T) {
	img := imageFile(t, "a.png", 4)
	notes := filepath.Join(t.TempDir(), "notes.png")
	require.NoError(t, os.WriteFile(notes, []byte("not an image"), 0o600))
	for name, text := range map[string]string{
		"a file that isn't an image": img + " " + notes,
		"a missing file":             img + " /nowhere/b.png",
		"a word that isn't a path":   img + " please",
		"an unclosed quote":          "'" + img,
	} {
		t.Run(name, func(t *testing.T) {
			m, store := newAttachModel(t)
			require.True(t, m.selectTaskIdx(0, 0))

			runAll(t, m, send(t, m, tea.PasteMsg{Content: text}))

			assert.Len(t, m.project.Categories[0].Tasks, 4, "pasted as a task, as today")
			assert.Empty(t, m.project.Categories[0].Tasks[0].Description)
			assert.Empty(t, storedFiles(t, store))
		})
	}
}

func TestPaste_AnImagePathOnTheClipboardAttachesTheFile(t *testing.T) {
	m, _ := newAttachModel(t)
	require.True(t, m.selectTaskIdx(0, 0))
	img := imageFile(t, "clip-1.png", 4)
	clipboardHolds(t, []byte(img))

	runAll(t, m, m.paste())

	assert.Equal(t, "![clip-1.png](attachments/"+hashName(t, img)+")", m.project.Categories[0].Tasks[0].Description)
}

func TestPaste_CopiedFilesAttachWhenAllAreImages(t *testing.T) {
	m, _ := newAttachModel(t)
	require.True(t, m.selectTaskIdx(0, 0))
	img := imageFile(t, "My Shot.png", 4)

	runAll(t, m, send(t, m, clipboardLinesMsg{text: img, paths: []string{img}}))
	assert.Equal(t, "![My Shot.png](attachments/"+hashName(t, img)+")", m.project.Categories[0].Tasks[0].Description)

	require.True(t, m.selectTaskIdx(0, 1))
	runAll(t, m, send(t, m, clipboardLinesMsg{text: "/nowhere/a.png", paths: []string{"/nowhere/a.png"}}))
	assert.Empty(t, m.project.Categories[0].Tasks[1].Description)
	assert.Len(t, m.project.Categories[0].Tasks, 4, "a missing file is pasted as text")
}

func TestDescriptionEditor_InsertsTheImageAtTheCursor(t *testing.T) {
	t.Run("ctrl+v", func(t *testing.T) {
		m, _ := newAttachModel(t)
		m.project.Categories[0].Tasks[0].Description = "before after"
		m.rebuildPositions()
		require.True(t, m.selectTaskIdx(0, 0))
		m.startDescriptionInlineEdit(0, 0)
		m.ui.DescriptionEdit.textarea.SetCursorColumn(len("before "))
		clipboardHolds(t, pngBytes(t, 4, 4))

		runAll(t, m, send(t, m, key("ctrl+v")))

		require.True(t, m.ui.Modes.IsDescriptionEdit())
		value := m.ui.DescriptionEdit.textarea.Value()
		assert.Regexp(t, `^before !\[Screenshot [^\]]+\]\(attachments/[0-9a-f]{32}\.png\)after$`, value)
		assert.Regexp(t, `^Attached Screenshot .* to Alpha$`, m.ui.Screen.StatusMsg)

		m.finishDescriptionEdit()
		assert.Equal(t, value, m.project.Categories[0].Tasks[0].Description)
		m.undo()
		assert.Equal(t, "before after", m.project.Categories[0].Tasks[0].Description)
	})
	t.Run("a drop", func(t *testing.T) {
		m, _ := newAttachModel(t)
		require.True(t, m.selectTaskIdx(0, 0))
		m.startDescriptionInlineEdit(0, 0)
		img := imageFile(t, "a b.png", 4)

		runAll(t, m, send(t, m, tea.PasteMsg{Content: strings.ReplaceAll(img, " ", `\ `)}))

		assert.Equal(t, "![a b.png](attachments/"+hashName(t, img)+")", m.ui.DescriptionEdit.textarea.Value())
		assert.Empty(t, m.project.Categories[0].Tasks[0].Description, "saved only when the editor is")
	})
	t.Run("a pasted text stays text", func(t *testing.T) {
		m, _ := newAttachModel(t)
		require.True(t, m.selectTaskIdx(0, 0))
		m.startDescriptionInlineEdit(0, 0)

		send(t, m, tea.PasteMsg{Content: "/nowhere/a.png"})

		assert.Equal(t, "/nowhere/a.png", m.ui.DescriptionEdit.textarea.Value())
	})
}

func TestTitleEditor_ADropStaysText(t *testing.T) {
	m, store := newAttachModel(t)
	require.True(t, m.selectTaskIdx(0, 0))
	m.startEditing()
	img := imageFile(t, "a.png", 4)

	runAll(t, m, send(t, m, tea.PasteMsg{Content: img}))

	assert.Equal(t, "Alpha"+img, m.ui.Edit.input.Value())
	assert.Empty(t, storedFiles(t, store))
}

func TestAttach_ReportsWhatItRefuses(t *testing.T) {
	dir := t.TempDir()
	big := filepath.Join(dir, "big.png")
	require.NoError(t, os.WriteFile(big, pngBytes(t, 2, 2), 0o600))
	require.NoError(t, os.Truncate(big, attachments.MaxSize+1))
	large := filepath.Join(dir, "large.png")
	require.NoError(t, os.WriteFile(large, pngBytes(t, 2, 2), 0o600))
	require.NoError(t, os.Truncate(large, attachments.WarnSize+(1<<20)))

	m, store := newAttachModel(t)
	require.True(t, m.selectTaskIdx(0, 0))
	runAll(t, m, send(t, m, tea.PasteMsg{Content: big}))
	assert.Equal(t, "big.png is over 25 MB: not attached", m.ui.Screen.StatusMsg)
	assert.Empty(t, m.project.Categories[0].Tasks[0].Description)
	assert.Empty(t, storedFiles(t, store))

	runAll(t, m, send(t, m, tea.PasteMsg{Content: large}))
	assert.Equal(t, "Attached large.png to Alpha (large: 6.0 MB)", m.ui.Screen.StatusMsg)
}

func TestAttach_TheTaskCanGoWhileTheImageIsStored(t *testing.T) {
	m, _ := newAttachModel(t)
	require.True(t, m.selectTaskIdx(0, 0))
	img := imageFile(t, "a.png", 4)
	cmd := send(t, m, tea.PasteMsg{Content: img})
	require.NotNil(t, cmd)
	m.project.Categories[0].Tasks = m.project.Categories[0].Tasks[1:]
	m.rebuildPositions()

	send(t, m, cmd())

	assert.Equal(t, "The task is gone: nothing attached", m.ui.Screen.StatusMsg)
}

func TestAttach_LandsInTheTasksOpenEditor(t *testing.T) {
	m, _ := newAttachModel(t)
	require.True(t, m.selectTaskIdx(0, 0))
	cmd := m.attachImages(imageFiles([]string{imageFile(t, "a.png", 4)}), false)
	m.startDescriptionInlineEdit(0, 0)

	runAll(t, m, cmd)

	assert.True(t, m.ui.Modes.Current() == modes.ModeDescriptionEdit)
	assert.Contains(t, m.ui.DescriptionEdit.textarea.Value(), "![a.png](attachments/")
	assert.Empty(t, m.project.Categories[0].Tasks[0].Description, "finishing the editor must not drop it")
}

func TestShellWords(t *testing.T) {
	for in, want := range map[string][]string{
		`/a/b.png`:                          {"/a/b.png"},
		`/a/My\ Shot.png /c/d.png`:          {"/a/My Shot.png", "/c/d.png"},
		`'/a/My Shot.png' "/b/x \"y\".png"`: {"/a/My Shot.png", `/b/x "y".png`},
		"/a.png\n/b.png\n":                  {"/a.png", "/b.png"},
		`'it'\''s.png'`:                     {"it's.png"},
		"  ":                                nil,
	} {
		got, ok := shellWords(in)
		assert.True(t, ok, in)
		assert.Equal(t, want, got, in)
	}
	_, ok := shellWords(`'/a/unclosed.png`)
	assert.False(t, ok)
}

func hashName(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	target, err := attachments.NewStore(t.TempDir()).Add(data)
	require.NoError(t, err)
	return strings.TrimPrefix(target, "attachments/")
}
