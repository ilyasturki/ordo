package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"phasionary/internal/app/selection"
	"phasionary/internal/clipboard"
	"phasionary/internal/config"
	"phasionary/internal/domain"
)

// copyProject has a task carrying a description so the FocusDescription row is
// meaningful to copy from.
func copyProject() domain.Project {
	return domain.Project{
		ID:   "p1",
		Name: "My Project",
		Categories: []domain.Category{{
			ID:   "c1",
			Name: "Cat A",
			Tasks: []domain.Task{
				{ID: "t1", Title: "Alpha", Description: "the body of alpha", Status: domain.StatusTodo},
				{ID: "sep", Title: "— divider —", Status: domain.StatusTodo, Kind: domain.KindSeparator},
				{ID: "t2", Title: "Bravo", Status: domain.StatusTodo},
			},
		}},
	}
}

func TestCopyTextForPosition(t *testing.T) {
	m := newTestModel(t, copyProject())

	cases := []struct {
		name string
		pos  selection.Position
		want string
	}{
		{"project", selection.Position{Kind: selection.FocusProject}, "My Project"},
		{"category", selection.Position{Kind: selection.FocusCategory, CategoryIndex: 0}, "Cat A"},
		{"task", selection.Position{Kind: selection.FocusTask, CategoryIndex: 0, TaskIndex: 0}, "Alpha\n\nthe body of alpha"},
		{"description", selection.Position{Kind: selection.FocusDescription, CategoryIndex: 0, TaskIndex: 0}, "the body of alpha"},
		{"separator", selection.Position{Kind: selection.FocusSeparator, CategoryIndex: 0, TaskIndex: 1}, "— divider —"},
		{"task without description", selection.Position{Kind: selection.FocusTask, CategoryIndex: 0, TaskIndex: 2}, "Bravo"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, m.copyTextForPosition(tc.pos))
		})
	}
}

// Regression: pressing `y` while focused on a description row must copy the
// description body, not an empty string.
func TestCopySelected_CopiesDescriptionBody(t *testing.T) {
	m := newTestModel(t, copyProject())
	// Description rows only exist as navigable positions when expanded.
	m.ui.Screen.ExpandDescriptions = true
	m.rebuildPositions()
	m.ui.Selection.SelectByPredicate(func(p selection.Position) bool {
		return p.Kind == selection.FocusDescription && p.CategoryIndex == 0 && p.TaskIndex == 0
	})

	pos, ok := m.selectedPosition()
	if !ok || pos.Kind != selection.FocusDescription {
		t.Fatalf("expected a focused description row, got %+v ok=%v", pos, ok)
	}
	assert.Equal(t, "the body of alpha", m.copyTextForPosition(pos))

	// A description copy is text-only: it must not stash a paste-able task.
	m.copySelected()
	assert.Empty(t, m.ui.Clipboard.Tasks)
}

func TestClipboardResultMsg_ReportsTheCopy(t *testing.T) {
	cases := []struct {
		name string
		msg  clipboardResultMsg
		want string
	}{
		{"plain", clipboardResultMsg{}, "Copied!"},
		{"labelled", clipboardResultMsg{label: "URL: x"}, "Copied URL: x"},
		{"failure", clipboardResultMsg{err: assert.AnError}, "Copy failed: " + assert.AnError.Error()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := newTestModel(t, copyProject())
			updated, _ := m.Update(tc.msg)
			assert.Equal(t, tc.want, updated.(model).ui.Screen.StatusMsg)
		})
	}
}

func TestCopy_GoesOutAsOSC52ByDefault(t *testing.T) {
	t.Setenv(clipboard.EnvFile, "")
	t.Setenv("PATH", t.TempDir())
	m := newTestModel(t, copyProject())
	require.True(t, m.selectTaskIdx(0, 0))

	cmd := m.copySelected()

	require.NotNil(t, cmd)
	assert.Contains(t, runSequence(t, cmd()), tea.SetClipboard("Alpha\n\nthe body of alpha")())
}

func TestCopy_RunsWlCopyWhenAskedTo(t *testing.T) {
	out := filepath.Join(t.TempDir(), "copied")
	fakeWlCopy(t, "cat > "+out)
	m := newTestModel(t, copyProject())
	require.True(t, m.selectTaskIdx(0, 0))
	require.NoError(t, m.deps.CfgManager.Update(func(cfg *config.Config) { cfg.CopyVia = config.CopyViaWlCopy }))

	msg := m.copySelected()()

	assert.Equal(t, clipboardResultMsg{text: "Alpha\n\nthe body of alpha", ofCopy: true}, msg)
	got, err := os.ReadFile(out)
	require.NoError(t, err)
	assert.Equal(t, "Alpha\n\nthe body of alpha", string(got))
}

func TestCopy_FallsBackToOSC52WithoutWlCopy(t *testing.T) {
	t.Setenv(clipboard.EnvFile, "")
	t.Setenv("PATH", t.TempDir())
	m := newTestModel(t, copyProject())
	require.True(t, m.selectTaskIdx(0, 0))
	require.NoError(t, m.deps.CfgManager.Update(func(cfg *config.Config) { cfg.CopyVia = config.CopyViaWlCopy }))

	msg := m.copySelected()()

	assert.Contains(t, runSequence(t, msg), tea.SetClipboard("Alpha\n\nthe body of alpha")())
}

func runSequence(t *testing.T, msg tea.Msg) []tea.Msg {
	t.Helper()
	v := reflect.ValueOf(msg)
	require.Equal(t, reflect.Slice, v.Kind(), "got %T", msg)
	var msgs []tea.Msg
	for i := range v.Len() {
		msgs = append(msgs, v.Index(i).Interface().(tea.Cmd)())
	}
	return msgs
}

// fakeWlCopy stands body in for wl-copy (pbcopy on macOS), behind a socket name no compositor answers to.
func fakeWlCopy(t *testing.T, body string) {
	t.Helper()
	sh, err := exec.LookPath("sh")
	require.NoError(t, err)
	cat, err := exec.LookPath("cat")
	require.NoError(t, err)
	dir := t.TempDir()
	script := "#!" + sh + "\nPATH=" + filepath.Dir(cat) + "\n" + body + "\n"
	for _, name := range []string{"wl-copy", "pbcopy"} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(script), 0o755))
	}
	t.Setenv(clipboard.EnvFile, "")
	t.Setenv("PATH", dir)
	t.Setenv("WAYLAND_DISPLAY", "wayland-test")
}

func TestClipboardFile_CarriesCopiesBothWays(t *testing.T) {
	path := filepath.Join(t.TempDir(), "clipboard")
	t.Setenv(clipboard.EnvFile, path)
	m := newTestModel(t, copyProject())
	require.True(t, m.selectTaskIdx(0, 0))

	send(t, m, m.copySelected()())
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "Alpha\n\nthe body of alpha", string(got))

	require.NoError(t, os.WriteFile(path, []byte("from the browser"), 0o600))
	require.True(t, m.selectTaskIdx(0, 2))
	send(t, m, m.paste()())
	assert.Equal(t, []string{"Alpha", "— divider —", "Bravo", "from the browser"}, taskTitles(m.project.Categories[0]))
}
