package app

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/app/selection"
	"github.com/ilyasturki/ordo/internal/attachments"
	"github.com/ilyasturki/ordo/internal/clipboard"
	"github.com/ilyasturki/ordo/internal/config"
	"github.com/ilyasturki/ordo/internal/domain"
)

// imagesOutModel is a release category whose crash report and its subtask
// carry screenshots: the shape of a list handed to an agent.
func imagesOutModel(t *testing.T) (m *model, crash, theme string) {
	t.Helper()
	store := attachments.NewStore(t.TempDir())
	crashRef, err := store.Add(pngBytes(t, 4, 4))
	require.NoError(t, err)
	themeRef, err := store.Add(pngBytes(t, 8, 8))
	require.NoError(t, err)
	project := domain.Project{
		ID:   "p1",
		Name: "ordo",
		Categories: []domain.Category{{
			ID:   "c1",
			Name: "v0.12.0",
			Tasks: []domain.Task{
				{ID: "crash", Title: "Crash when saving a folded category", Status: domain.StatusTodo,
					Description: "Happens after `zc` then `w`.\n" + attachments.Line("Screenshot 2026-10-11 10.12.03", crashRef)},
				{ID: "theme", ParentID: "crash", Title: "Check it on the light theme", Status: domain.StatusTodo,
					Description: attachments.Line("light.png", themeRef)},
				{ID: "docs", Title: "Document the fold keys", Status: domain.StatusTodo},
			},
		}},
	}
	m = newTestModel(t, project)
	m.deps.Attachments = store
	crash, _ = store.Path(crashRef)
	theme, _ = store.Path(themeRef)
	return m, crash, theme
}

func TestCopy_PointsImagesAtTheirFile(t *testing.T) {
	m, crash, _ := imagesOutModel(t)
	task := selection.Position{Kind: selection.FocusTask, CategoryIndex: 0, TaskIndex: 0}
	want := "Crash when saving a folded category\n\nHappens after `zc` then `w`.\n![Screenshot 2026-10-11 10.12.03](" + crash + ")"
	assert.Equal(t, want, m.copyTextForPosition(task))

	task.Kind = selection.FocusDescription
	assert.Equal(t, "Happens after `zc` then `w`.\n![Screenshot 2026-10-11 10.12.03]("+crash+")", m.copyTextForPosition(task))
}

func TestCopy_PastesBackWithTheStoredReference(t *testing.T) {
	t.Setenv(clipboard.EnvFile, filepath.Join(t.TempDir(), "clipboard"))
	m, _, _ := imagesOutModel(t)
	original := m.project.Categories[0].Tasks[0].Description

	selectRow(t, m, selection.FocusTask, "crash")
	send(t, m, m.copySelected()())
	selectRow(t, m, selection.FocusTask, "docs")
	runAll(t, m, m.paste())

	tasks := m.project.Categories[0].Tasks
	require.Len(t, tasks, 5)
	assert.Equal(t, original, tasks[3].Description, "the project keeps the relative form")
}

func TestVisualCopy_ListsEachImageUnderItsBullet(t *testing.T) {
	m, crash, theme := imagesOutModel(t)
	selectRow(t, m, selection.FocusTask, "crash")
	m.enterVisualMode()
	m.visualMoveCursor(2)

	text := m.visualMarkdownText(m.visualSelectedPositions(), false)
	assert.Equal(t, "- Crash when saving a folded category\n"+
		"  ![Screenshot 2026-10-11 10.12.03]("+crash+")\n"+
		"  - Check it on the light theme\n"+
		"    ![light.png]("+theme+")\n"+
		"- Document the fold keys", text)

	for _, path := range regexp.MustCompile(`\]\(([^)]+)\)`).FindAllStringSubmatch(text, -1) {
		_, err := os.Stat(path[1])
		assert.NoError(t, err)
	}
}

func TestVisualCopy_SelectedDescriptionCarriesItsImagesOnce(t *testing.T) {
	m, crash, theme := imagesOutModel(t)
	m.ui.Screen.ExpandDescriptions = true
	m.rebuildPositions()
	selectRow(t, m, selection.FocusTask, "crash")
	m.enterVisualMode()
	m.visualMoveCursor(1)

	text := m.visualMarkdownText(m.visualSelectedPositions(), true)
	assert.Equal(t, "- [ ] Crash when saving a folded category\n"+
		"  Happens after `zc` then `w`.\n"+
		"  ![Screenshot 2026-10-11 10.12.03]("+crash+")\n"+
		"  - [ ] Check it on the light theme\n"+
		"    ![light.png]("+theme+")", text)
}

func TestTaskCommand_GetsTheImagePaths(t *testing.T) {
	m, crash, theme := imagesOutModel(t)
	bound, err := bindTaskCommands([]config.TaskCommand{shCommand("w", "", `printf '%s' "$ORDO_TASK_IMAGES" | tr '\n' '|'`)})
	require.NoError(t, err)
	m.deps.setTaskCommands(bound)
	tasks := m.project.Categories[0].Tasks
	tasks[0].Description += "\n" + tasks[1].Description
	selectRow(t, m, selection.FocusTask, "crash")

	m, cmd := pressKeys(m, "w")
	m = runCmd(t, m, cmd)
	assert.Equal(t, crash+"|"+theme, m.ui.Screen.StatusMsg)
}

func TestCopyCategory_PointsImagesAtTheirFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "clipboard")
	t.Setenv(clipboard.EnvFile, path)
	m, crash, _ := imagesOutModel(t)
	selectRow(t, m, selection.FocusCategory, "c1")

	send(t, m, m.copyCategoryContent()())
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(got), "    ![Screenshot 2026-10-11 10.12.03]("+crash+")\n")
	assert.NotContains(t, string(got), "](attachments/")
}
