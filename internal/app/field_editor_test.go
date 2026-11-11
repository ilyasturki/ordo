package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/app/modes"
	"github.com/ilyasturki/ordo/internal/app/selection"
)

type fakeEditor struct {
	seen string
}

// useFakeEditor makes $VISUAL a script that records the file it is handed, writes out and exits with code.
func useFakeEditor(t *testing.T, out string, code string) *fakeEditor {
	t.Helper()
	dir := t.TempDir()
	script := filepath.Join(dir, "editor")
	seen := filepath.Join(dir, "seen")
	require.NoError(t, os.WriteFile(script, []byte("#!/bin/sh\ncp \"$1\" \"$FAKE_SEEN\"\nprintf '%s' \"$FAKE_OUT\" > \"$1\"\nexit \"$FAKE_CODE\"\n"), 0o755))
	t.Setenv("VISUAL", script)
	t.Setenv("FAKE_SEEN", seen)
	t.Setenv("FAKE_OUT", out)
	t.Setenv("FAKE_CODE", code)
	return &fakeEditor{seen: seen}
}

func (f *fakeEditor) received(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(f.seen)
	require.NoError(t, err)
	return string(b)
}

// pressCtrlG runs ctrl+g through Update, then the editor the way tea.ExecProcess would, and hands its exit back.
func pressCtrlG(t *testing.T, m *model) (*model, string) {
	t.Helper()
	next, cmd := m.Update(tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl})
	am := asModel(next)
	require.NotNil(t, cmd, "ctrl+g starts the editor")
	path := am.ui.ExternalEdit.TempFilePath
	require.FileExists(t, path)
	err := exec.Command(getEditorCmd(), path).Run()
	next, _ = am.Update(editorFinishedMsg{err: err, tempFile: path})
	am = asModel(next)
	return &am, path
}

func titleEditModel(t *testing.T, buffer string) *model {
	t.Helper()
	m := newTestModel(t, sampleProject())
	require.True(t, m.selectTaskIdx(0, 0))
	startEdit(m, m.selected(), buffer, selection.FocusTask)
	return m
}

func TestCtrlG_TitleTakesTheEditorTextFolded(t *testing.T) {
	ed := useFakeEditor(t, "first line\n  second line\n\n", "0")
	m := titleEditModel(t, "draft title")

	m, path := pressCtrlG(t, m)

	assert.Equal(t, "draft title", ed.received(t), "the editor gets the unsaved text")
	assert.True(t, m.ui.Modes.IsEdit(), "the field stays open")
	assert.Equal(t, "first line second line", m.ui.Edit.input.Value())
	assert.Equal(t, utf8.RuneCountInString("first line second line"), m.ui.Edit.input.Position(), "cursor at the end")
	assert.NoFileExists(t, path)

	saved, _ := pressKeys(m, "enter")
	assert.True(t, saved.ui.Modes.IsNormal())
	assert.Equal(t, "first line second line", saved.project.Categories[0].Tasks[0].Title)
}

func TestCtrlG_UndoTakesTheEditorTextBack(t *testing.T) {
	useFakeEditor(t, "from the editor", "0")
	m := titleEditModel(t, "draft title")

	m, _ = pressCtrlG(t, m)
	next, _ := m.Update(tea.KeyPressMsg{Code: '_', Mod: tea.ModCtrl})

	assert.Equal(t, "draft title", asModel(next).ui.Edit.input.Value())
}

func TestCtrlG_EditorFailureLeavesTheFieldAsItWas(t *testing.T) {
	useFakeEditor(t, "changed", "1")
	m := titleEditModel(t, "draft title")

	m, path := pressCtrlG(t, m)

	assert.True(t, m.ui.Modes.IsEdit())
	assert.Equal(t, "draft title", m.ui.Edit.input.Value())
	assert.Contains(t, m.ui.Screen.StatusMsg, "Editor error")
	assert.NoFileExists(t, path)
}

func TestCtrlG_DescriptionTakesTheEditorTextAndEscStillCancels(t *testing.T) {
	ed := useFakeEditor(t, "one\n\ntwo\n", "0")
	p := sampleProject()
	p.Categories[0].Tasks[0].Description = "saved"
	m := newTestModel(t, p)
	_ = m.startDescriptionInlineEdit(0, 0)
	m.ui.DescriptionEdit.textarea.InsertString(" unsaved")

	m, _ = pressCtrlG(t, m)

	assert.Equal(t, "saved unsaved", ed.received(t))
	assert.Equal(t, modes.ModeDescriptionEdit, m.ui.Modes.Current())
	ta := m.ui.DescriptionEdit.textarea
	assert.Equal(t, "one\n\ntwo", ta.Value(), "the editor's final newline is dropped")
	assert.False(t, ta.HasSelection(), "ctrl+g no longer selects all")
	assert.Equal(t, 2, ta.Line())
	assert.Equal(t, 3, ta.Column())

	next, _ := m.Update(keyPress("esc"))
	cancelled := asModel(next)
	assert.Equal(t, "saved", cancelled.project.Categories[0].Tasks[0].Description)
}

func TestCtrlG_DescriptionSavesWithEnter(t *testing.T) {
	useFakeEditor(t, "from the editor", "0")
	p := sampleProject()
	m := newTestModel(t, p)
	_ = m.startDescriptionInlineEdit(0, 0)

	m, _ = pressCtrlG(t, m)
	next, _ := m.Update(keyPress("enter"))
	saved := asModel(next)

	assert.True(t, saved.ui.Modes.IsNormal())
	assert.Equal(t, "from the editor", saved.project.Categories[0].Tasks[0].Description)
}
