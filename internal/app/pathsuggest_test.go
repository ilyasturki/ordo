package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/app/selection"
)

func pathTree(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, "alpha"), 0o755))
	for _, f := range []string{"alpha/inner.md", "alpine.txt", "beta", ".hidden"} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, f), nil, 0o644))
	}
	require.NoError(t, os.Symlink(filepath.Join(dir, "alpha"), filepath.Join(dir, "linked")))
	return dir
}

func pathKey(k string) tea.KeyPressMsg {
	switch k {
	case "tab":
		return tea.KeyPressMsg{Code: tea.KeyTab}
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "backspace":
		return tea.KeyPressMsg{Code: tea.KeyBackspace}
	case "ctrl+n", "ctrl+p", "ctrl+_":
		return tea.KeyPressMsg{Code: rune(k[len(k)-1]), Mod: tea.ModCtrl}
	}
	r := []rune(k)[0]
	return tea.KeyPressMsg{Code: r, Text: k}
}

// sendField runs each key through Update, delivering the directory reads it
// starts as the runtime would; a key longer than one rune that names no key
// is typed rune by rune.
func sendField(t *testing.T, m *model, keys ...string) *model {
	t.Helper()
	for _, k := range keys {
		if pathKey(k).Text != "" && len([]rune(k)) > 1 {
			for _, r := range k {
				m = sendField(t, m, string(r))
			}
			continue
		}
		next, _ := m.Update(pathKey(k))
		m = deliverPathReads(t, nextModel(next))
	}
	return m
}

func nextModel(next tea.Model) *model {
	m := asModel(next)
	return &m
}

func deliverPathReads(t *testing.T, m *model) *model {
	t.Helper()
	for {
		var dirs []string
		for dir := range m.ui.Edit.paths.reading {
			dirs = append(dirs, dir)
		}
		for dir := range m.ui.DescriptionEdit.paths.reading {
			dirs = append(dirs, dir)
		}
		if len(dirs) == 0 {
			return m
		}
		for _, dir := range dirs {
			next, _ := m.Update(readPathDir(dir)())
			m = nextModel(next)
		}
	}
}

func pathNames(s pathSuggest) []string {
	if !s.open() {
		return nil
	}
	out := make([]string, len(s.items))
	for i, e := range s.items {
		out[i] = e.name
		if e.dir {
			out[i] += "/"
		}
	}
	return out
}

func editingTask(t *testing.T, title string) *model {
	t.Helper()
	p := sampleProject()
	p.Categories[0].Tasks[0].Title = title
	m := newTestModel(t, p)
	m.ui.Selection.MoveTo(2)
	m.startEditing()
	require.True(t, m.ui.Modes.IsEdit())
	return m
}

func TestPathBeforeCaret(t *testing.T) {
	cases := []struct {
		text      string
		caret     int
		wantPath  string
		wantStart int
		wantOK    bool
	}{
		{"see /etc/ho", 11, "/etc/ho", 4, true},
		{"see ~/", 6, "~/", 4, true},
		{"(./a", 4, "./a", 1, true},
		{"../", 3, "../", 0, true},
		{"/", 1, "/", 0, true},
		{"/etc/hosts and", 14, "", 0, false},
		{"/etc/hosts", 4, "/etc", 0, true},
		{"and/or", 6, "", 0, false},
		{"https://x.io/a", 14, "", 0, false},
		{"plain", 5, "", 0, false},
	}
	for _, tc := range cases {
		path, start, ok := pathBeforeCaret([]rune(tc.text), tc.caret)
		assert.Equal(t, tc.wantOK, ok, tc.text)
		assert.Equal(t, tc.wantPath, path, tc.text)
		assert.Equal(t, tc.wantStart, start, tc.text)
	}
}

func TestPathSuggest_OpensFiltersAndCompletes(t *testing.T) {
	dir := pathTree(t)
	m := editingTask(t, "see")
	m = sendField(t, m, " "+dir+"/")
	assert.Equal(t, []string{"alpha/", "alpine.txt", "beta", "linked/"}, pathNames(m.ui.Edit.paths), "a dot only shows hidden files")

	m = sendField(t, m, "al")
	assert.Equal(t, []string{"alpha/", "alpine.txt"}, pathNames(m.ui.Edit.paths))

	m = sendField(t, m, "down")
	assert.Equal(t, "alpine.txt", m.ui.Edit.paths.items[m.ui.Edit.paths.sel].name)
	m = sendField(t, m, "ctrl+p")
	assert.Equal(t, "alpha", m.ui.Edit.paths.items[m.ui.Edit.paths.sel].name)
	m = sendField(t, m, "up")
	assert.Equal(t, "alpine.txt", m.ui.Edit.paths.items[m.ui.Edit.paths.sel].name, "the selection wraps")
	m = sendField(t, m, "ctrl+n")

	m = sendField(t, m, "tab")
	assert.Equal(t, "see "+dir+"/alpha/", m.ui.Edit.input.Value())
	assert.Equal(t, []string{"inner.md"}, pathNames(m.ui.Edit.paths), "the list follows into the directory")

	m = sendField(t, m, "tab")
	assert.Equal(t, "see "+dir+"/alpha/inner.md", m.ui.Edit.input.Value())
	assert.False(t, m.ui.Edit.paths.open(), "a file ends the list")

	m = sendField(t, m, "ctrl+_")
	assert.Equal(t, "see "+dir+"/alpha/", m.ui.Edit.input.Value(), "undo takes a completion back")

	m = sendField(t, m, " x")
	assert.False(t, m.ui.Edit.paths.open(), "a word that is no path closes it")
	m = sendField(t, m, "enter")
	assert.Equal(t, "see "+dir+"/alpha/ x", m.project.Categories[0].Tasks[0].Title)
}

func TestPathSuggest_HiddenFilesAfterADot(t *testing.T) {
	dir := pathTree(t)
	m := sendField(t, editingTask(t, ""), dir+"/.")
	assert.Equal(t, []string{".hidden"}, pathNames(m.ui.Edit.paths))
}

func TestPathSuggest_HomeAndWorkingDirectory(t *testing.T) {
	dir := pathTree(t)
	t.Setenv("HOME", dir)
	m := sendField(t, editingTask(t, ""), "~/be")
	assert.Equal(t, []string{"beta"}, pathNames(m.ui.Edit.paths))

	t.Chdir(filepath.Join(dir, "alpha"))
	m = sendField(t, editingTask(t, ""), "./")
	assert.Equal(t, []string{"inner.md"}, pathNames(m.ui.Edit.paths))
	m = sendField(t, editingTask(t, ""), "../li")
	assert.Equal(t, []string{"linked/"}, pathNames(m.ui.Edit.paths))
}

func TestPathSuggest_EscClosesThenCancels(t *testing.T) {
	dir := pathTree(t)
	m := sendField(t, editingTask(t, "keep"), " "+dir+"/al")
	require.True(t, m.ui.Edit.paths.open())

	m = sendField(t, m, "esc")
	assert.True(t, m.ui.Modes.IsEdit(), "the first esc only closes the list")
	assert.False(t, m.ui.Edit.paths.open())
	m = sendField(t, m, "p")
	assert.False(t, m.ui.Edit.paths.open(), "typing on keeps a dismissed list closed")
	m = sendField(t, m, "down")
	assert.False(t, m.ui.Edit.paths.open())

	m = sendField(t, m, "esc")
	assert.True(t, m.ui.Modes.IsNormal(), "the second esc cancels the edit")
	assert.Equal(t, "keep", m.project.Categories[0].Tasks[0].Title)
}

func TestPathSuggest_ReadsOffTheKeyPath(t *testing.T) {
	dir := pathTree(t)
	m := editingTask(t, "")
	for _, r := range dir + "/" {
		next, _ := m.Update(pathKey(string(r)))
		m = nextModel(next)
	}
	assert.False(t, m.ui.Edit.paths.open(), "nothing shows before the read lands")
	assert.True(t, m.ui.Edit.paths.reading[dir], "the directory is read by a command")
	m = sendField(t, m, "esc")
	assert.True(t, m.ui.Modes.IsNormal(), "a list still loading takes no esc")

	m = sendField(t, editingTask(t, ""), dir+"/al")
	listings := len(m.ui.Edit.paths.listings)
	m = sendField(t, m, "backspace", "p")
	assert.Equal(t, listings, len(m.ui.Edit.paths.listings), "a listed directory is not read again")
}

func TestPathSuggest_NotInNames(t *testing.T) {
	dir := pathTree(t)
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(1)
	m.startEditing()
	require.Equal(t, selection.FocusCategory, m.ui.Edit.itemType)
	m = sendField(t, m, dir+"/")
	assert.False(t, m.ui.Edit.paths.open())
}

func TestPathSuggest_InTheDescription(t *testing.T) {
	dir := pathTree(t)
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(2)
	m.startDescriptionInlineEdit(0, 0)
	require.True(t, m.ui.Modes.IsDescriptionEdit())

	m = sendField(t, m, "see "+dir+"/b")
	assert.Equal(t, []string{"beta"}, pathNames(m.ui.DescriptionEdit.paths))
	assert.Contains(t, m.renderView(), "beta")
	m = sendField(t, m, "tab")
	assert.Equal(t, "see "+dir+"/beta", m.ui.DescriptionEdit.textarea.Value())

	m = sendField(t, m, "backspace", "esc")
	assert.True(t, m.ui.Modes.IsDescriptionEdit())
	m = sendField(t, m, "esc")
	assert.True(t, m.ui.Modes.IsNormal())
}

func TestPathSuggest_ListShiftsNoRow(t *testing.T) {
	dir := pathTree(t)
	m := sendField(t, editingTask(t, "x"), " "+dir+"/al")
	require.True(t, m.ui.Edit.paths.open())
	open := strings.Split(m.renderView(), "\n")
	assert.Contains(t, strings.Join(open, "\n"), "alpine.txt")

	m = sendField(t, m, "esc")
	closed := strings.Split(m.renderView(), "\n")
	require.Len(t, open, len(closed))
	editRow := -1
	for i, line := range closed {
		if strings.Contains(line, "/al") {
			editRow = i
			break
		}
	}
	require.GreaterOrEqual(t, editRow, 0)
	for i := 0; i <= editRow; i++ {
		assert.Equal(t, closed[i], open[i], "line %d above the list", i)
	}
	last := len(closed) - 1
	assert.Equal(t, closed[last], open[last], "the bottom bar stays")
}
