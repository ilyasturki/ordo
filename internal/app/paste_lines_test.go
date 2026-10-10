package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"phasionary/internal/clipboard"
	"phasionary/internal/domain"
	"phasionary/internal/export"
)

func task(title, status string) pastedRow {
	return pastedRow{title: title, status: status}
}

func sep(label string) pastedRow {
	return pastedRow{title: label, separator: true}
}

func TestParsePastedRows(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   string
		want []pastedRow
	}{
		{"every marker shape is a task", "- foo\r\n* bar\n1. baz\n2) qux\n- [x] done\n- [ ] open\n\n• dot", []pastedRow{
			task("foo", domain.StatusTodo),
			task("bar", domain.StatusTodo),
			task("baz", domain.StatusTodo),
			task("qux", domain.StatusTodo),
			task("done", domain.StatusCompleted),
			task("open", domain.StatusTodo),
			task("dot", domain.StatusTodo),
		}},
		{"no markers at all is tasks too", "  plain  \nalso plain\n", []pastedRow{
			task("plain", domain.StatusTodo),
			task("also plain", domain.StatusTodo),
		}},
		{"the app's own status markers round-trip", "- [~] doing\n- [-] dropped", []pastedRow{
			task("doing", domain.StatusInProgress),
			task("dropped", domain.StatusCancelled),
		}},
		{"bare lines beside bulleted ones group them", "Design the API\n- write handlers\n- write tests\nShip it", []pastedRow{
			sep("Design the API"),
			task("write handlers", domain.StatusTodo),
			task("write tests", domain.StatusTodo),
			sep("Ship it"),
		}},
		{"a bare checkbox marks the line on its own", "Groceries\n[ ] milk\n[x] bread", []pastedRow{
			sep("Groceries"),
			task("milk", domain.StatusTodo),
			task("bread", domain.StatusCompleted),
		}},
		{"headings and rules do not make a paste mixed", "## Backend\n- [ ] handlers\n- [x] auth\n---", []pastedRow{
			sep("Backend"),
			task("handlers", domain.StatusTodo),
			task("auth", domain.StatusCompleted),
			sep(""),
		}},
		{"indented list items stay rows of their own", "- parent\n  - child\n    - grandchild", []pastedRow{
			task("parent", domain.StatusTodo),
			task("child", domain.StatusTodo),
			task("grandchild", domain.StatusTodo),
		}},
		{"indented lines under a task are its description", "- [x] parent\n    first\n\n      deeper\n    ---\n\n- next", []pastedRow{
			{title: "parent", status: domain.StatusCompleted, description: "first\n\n  deeper\n---"},
			task("next", domain.StatusTodo),
		}},
		{"a blank line before the description still folds", "Title\n\n  body", []pastedRow{
			{title: "Title", status: domain.StatusTodo, description: "body"},
		}},
		{"lines indented alike are siblings", "  plain\n  also plain", []pastedRow{
			task("plain", domain.StatusTodo),
			task("also plain", domain.StatusTodo),
		}},
		{"a separator takes no description", "## Group\n  note\n- a", []pastedRow{
			sep("Group"),
			sep("note"),
			task("a", domain.StatusTodo),
		}},
		{"an indented list item after a description ends it", "- a\n  about a\n  - b", []pastedRow{
			{title: "a", status: domain.StatusTodo, description: "about a"},
			task("b", domain.StatusTodo),
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, parsePastedRows(tc.in))
		})
	}
}

func TestParsePastedRows_TheAppsOwnCopiesComeBackWhole(t *testing.T) {
	cat := domain.Category{Name: "Sprint", Tasks: []domain.Task{
		{ID: "a", Title: "Write handlers", Status: domain.StatusInProgress, Description: "GET and PUT\n\n- one list item"},
		{ID: "s", Kind: domain.KindSeparator, Title: "Later"},
		{ID: "b", Title: "Ship", Status: domain.StatusTodo, Description: "on friday\n---\nafter review"},
	}}

	assert.Equal(t, []pastedRow{
		sep("Sprint"),
		{title: "Write handlers", status: domain.StatusInProgress, description: "GET and PUT"},
		task("one list item", domain.StatusTodo),
		sep("Later"),
		{title: "Ship", status: domain.StatusTodo, description: "on friday\n---\nafter review"},
	}, parsePastedRows(export.ExportCategoryMarkdown(cat)),
		"Y: descriptions fold back, a list inside one still splits off")

	m := newTestModelWithDescription(t)
	require.True(t, m.selectTaskIdx(0, 0))
	m.enterVisualMode()
	m.visualMoveCursor(2)
	assert.Equal(t, []pastedRow{
		{title: "Alpha", status: domain.StatusTodo, description: "alpha details"},
		task("Beta", domain.StatusTodo),
	}, parsePastedRows(m.visualMarkdownText(m.visualSelectedPositions(), false)), "visual y")
}

func TestPasteClipboardLines_KeepsFoldedDescriptions(t *testing.T) {
	m := newTestModel(t, sampleProject())
	require.True(t, m.selectTaskIdx(0, 0))

	m.resolvePaste(clipboardLinesMsg{text: "- [ ] handlers\n    GET and PUT\n- auth"})
	require.True(t, m.ui.Modes.IsPasteChooser())
	send(t, m, key("enter"))

	tasks := m.project.Categories[0].Tasks
	require.Len(t, tasks, 5)
	assert.Equal(t, "handlers", tasks[1].Title)
	assert.Equal(t, "GET and PUT", tasks[1].Description)
	assert.Equal(t, "auth", tasks[2].Title)
}

// startAddTaskAt lands the cursor on (catIdx, taskIdx) and opens the add flow,
// so the pending blank task sits at taskIdx+1.
func startAddTaskAt(t *testing.T, m *model, catIdx, taskIdx int) {
	t.Helper()
	require.True(t, m.selectTaskIdx(catIdx, taskIdx))
	m.startAddingTask()
	require.True(t, m.ui.Modes.IsEdit())
}

func TestPasteLinesInEdit_CreatesOneTaskPerLine(t *testing.T) {
	m := newTestModel(t, sampleProject())
	startAddTaskAt(t, m, 0, 0) // pending task at index 1
	m.ui.Edit.input.SetValue("pre")

	handled := m.pasteLinesInEdit("- [x] First\n- Second\n")

	require.True(t, handled)
	tasks := m.project.Categories[0].Tasks
	require.Len(t, tasks, 5)
	assert.Equal(t, "pre First", tasks[1].Title, "the typed text prepends to the first row")
	assert.Equal(t, domain.StatusCompleted, tasks[1].Status)
	assert.Equal(t, "Second", tasks[2].Title)
	assert.Equal(t, domain.StatusTodo, tasks[2].Status)
	assert.Equal(t, "t2", tasks[3].ID, "existing tasks shift below the batch")

	assert.True(t, m.ui.Modes.IsNormal())
	pos, ok := m.selectedPosition()
	require.True(t, ok)
	assert.Equal(t, "Second", m.project.Categories[0].Tasks[pos.TaskIndex].Title)
	assert.Equal(t, "Added 2 tasks", m.ui.Screen.StatusMsg)
}

func TestPasteLinesInEdit_MixedPasteConvertsPendingRowToSeparator(t *testing.T) {
	m := newTestModel(t, sampleProject())
	startAddTaskAt(t, m, 0, 0)

	require.True(t, m.pasteLinesInEdit("Design the API\n- write handlers\n- write tests\nShip it"))

	tasks := m.project.Categories[0].Tasks
	require.Len(t, tasks, 7)
	assert.True(t, tasks[1].IsSeparator())
	assert.Equal(t, "Design the API", tasks[1].Title)
	assert.Empty(t, tasks[1].Status, "a converted row carries no status")
	assert.Equal(t, "write handlers", tasks[2].Title)
	assert.False(t, tasks[2].IsSeparator())
	assert.Equal(t, "write tests", tasks[3].Title)
	assert.True(t, tasks[4].IsSeparator())
	assert.Equal(t, "Ship it", tasks[4].Title)

	assert.Equal(t, "Added 2 tasks, 2 separators", m.ui.Screen.StatusMsg)
	pos, ok := m.selectedPosition()
	require.True(t, ok)
	assert.Equal(t, "Ship it", m.project.Categories[0].Tasks[pos.TaskIndex].Title,
		"the cursor lands on the last row even when it is a separator")
}

func TestPasteLinesInEdit_SingleLineFallsThrough(t *testing.T) {
	m := newTestModel(t, sampleProject())
	startAddTaskAt(t, m, 0, 0)

	assert.False(t, m.pasteLinesInEdit("just one line"))
	assert.False(t, m.pasteLinesInEdit("trailing newline\n"))
	assert.True(t, m.ui.Modes.IsEdit(), "single-line pastes stay in the inline editor")
}

func TestPasteLinesInEdit_SplitsAnExistingTitle(t *testing.T) {
	m := newTestModel(t, sampleProject())
	require.True(t, m.selectTaskIdx(0, 0))
	require.NoError(t, m.project.Categories[0].Tasks[0].SetStatus(domain.StatusInProgress))
	first := m.project.Categories[0].Tasks[0].Title
	m.startEditing() // editing t1, not adding

	require.True(t, m.pasteLinesInEdit("- a\n- b"))

	tasks := m.project.Categories[0].Tasks
	require.Len(t, tasks, 4)
	assert.Equal(t, first+" a", tasks[0].Title, "the first row merges into the edited title")
	assert.Equal(t, domain.StatusInProgress, tasks[0].Status,
		"merging a line rewrites the title alone, not the status the task already had")
	assert.Equal(t, "b", tasks[1].Title)
	assert.True(t, m.ui.Modes.IsNormal())
	assert.Equal(t, "Added 1 task", m.ui.Screen.StatusMsg)

	m.undo()

	require.Len(t, m.project.Categories[0].Tasks, 3, "the batch undoes in one step")
	assert.Equal(t, first, m.project.Categories[0].Tasks[0].Title)
	assert.Equal(t, domain.StatusInProgress, m.project.Categories[0].Tasks[0].Status)
}

func TestPasteLinesInEdit_FirstRowsDescriptionJoinsTheEditedTask(t *testing.T) {
	m := newTestModel(t, sampleProject())
	require.True(t, m.selectTaskIdx(0, 0))
	m.project.Categories[0].Tasks[0].Description = "already here"
	m.startEditing()

	require.True(t, m.pasteLinesInEdit("- a\n  about a\n- b"))

	assert.Equal(t, "already here\nabout a", m.project.Categories[0].Tasks[0].Description)
	assert.Equal(t, "b", m.project.Categories[0].Tasks[1].Title)
}

func TestPasteLinesInEdit_LeadingSeparatorSpareTheEditedTask(t *testing.T) {
	m := newTestModel(t, sampleProject())
	require.True(t, m.selectTaskIdx(0, 0))
	m.project.Categories[0].Tasks[0].Description = "keep me"
	first := m.project.Categories[0].Tasks[0].Title
	m.startEditing()

	require.True(t, m.pasteLinesInEdit("Design the API\n- write handlers"))

	tasks := m.project.Categories[0].Tasks
	require.Len(t, tasks, 5)
	assert.Equal(t, first, tasks[0].Title, "the edited task keeps its title")
	assert.False(t, tasks[0].IsSeparator())
	assert.Equal(t, "keep me", tasks[0].Description, "and its description")
	assert.True(t, tasks[1].IsSeparator())
	assert.Equal(t, "Design the API", tasks[1].Title)
	assert.Equal(t, "write handlers", tasks[2].Title)
}

func TestPasteLinesInEdit_UndoesAsOneBatch(t *testing.T) {
	m := newTestModel(t, sampleProject())
	startAddTaskAt(t, m, 0, 0)
	require.True(t, m.pasteLinesInEdit("a\nb\nc"))
	require.Len(t, m.project.Categories[0].Tasks, 6)

	m.undo()

	assert.Len(t, m.project.Categories[0].Tasks, 3)
	for _, task := range m.project.Categories[0].Tasks {
		assert.NotEmpty(t, task.ID)
	}
	// Guard: no leftover selection on a vanished row.
	_, ok := m.selectedPosition()
	assert.True(t, ok)
}

func TestPasteClipboardLines_InsertsBelowSelection(t *testing.T) {
	m := newTestModel(t, sampleProject())
	require.True(t, m.selectTaskIdx(0, 0))

	m.resolvePaste(clipboardLinesMsg{text: "## Backend\n- [x] handlers\n- auth"})
	require.True(t, m.ui.Modes.IsPasteChooser(), "several rows ask how to paste")
	send(t, m, key("enter"))

	tasks := m.project.Categories[0].Tasks
	require.Len(t, tasks, 6)
	assert.True(t, tasks[1].IsSeparator())
	assert.Equal(t, "Backend", tasks[1].Title)
	assert.Equal(t, "handlers", tasks[2].Title)
	assert.Equal(t, domain.StatusCompleted, tasks[2].Status)
	assert.Equal(t, "auth", tasks[3].Title)
	assert.Equal(t, "Added 2 tasks, 1 separator", m.ui.Screen.StatusMsg)

	m.undo()
	assert.Len(t, m.project.Categories[0].Tasks, 3)
}

func TestPasteClipboardLines_SingleLineBecomesOneTask(t *testing.T) {
	m := newTestModel(t, sampleProject())
	require.True(t, m.selectTaskIdx(0, 0))

	m.resolvePaste(clipboardLinesMsg{text: "  buy milk  "})

	assert.Equal(t, "buy milk", m.project.Categories[0].Tasks[1].Title)
	assert.Equal(t, "Added 1 task", m.ui.Screen.StatusMsg)
}

func TestPasteClipboardLines_EmptyOrFailedClipboard(t *testing.T) {
	m := newTestModel(t, sampleProject())
	require.True(t, m.selectTaskIdx(0, 0))

	m.resolvePaste(clipboardLinesMsg{text: "   \n\n"})
	assert.Equal(t, "Nothing to paste", m.ui.Screen.StatusMsg)

	m.resolvePaste(clipboardLinesMsg{err: assert.AnError})
	assert.Equal(t, "Nothing to paste", m.ui.Screen.StatusMsg, "the tools fail on an empty clipboard too")

	m.resolvePaste(clipboardLinesMsg{err: clipboard.ErrNoBackend})
	assert.Equal(t, "Can't read the clipboard: paste with the terminal instead", m.ui.Screen.StatusMsg)

	assert.Len(t, m.project.Categories[0].Tasks, 3, "nothing was inserted")
}
