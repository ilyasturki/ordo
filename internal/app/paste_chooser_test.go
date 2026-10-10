package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"phasionary/internal/app/modes"
	"phasionary/internal/app/selection"
	"phasionary/internal/clipboard"
	"phasionary/internal/domain"
)

func key(s string) tea.KeyPressMsg {
	switch s {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	}
	if rest, ok := strings.CutPrefix(s, "ctrl+"); ok {
		return tea.KeyPressMsg{Code: []rune(rest)[0], Mod: tea.ModCtrl}
	}
	return tea.KeyPressMsg{Code: []rune(s)[0], Text: s}
}

// send runs msg through Update the way the program would, keeping the result.
func send(t *testing.T, m *model, msg tea.Msg) tea.Cmd {
	t.Helper()
	next, cmd := m.Update(msg)
	switch v := next.(type) {
	case model:
		*m = v
	case *model:
		*m = *v
	}
	return cmd
}

const threeLines = "Groceries\n- milk\n- [x] bread"

func TestPasteChooser_OpensFromEveryRoute(t *testing.T) {
	t.Run("p reading the clipboard", func(t *testing.T) {
		m := newTestModel(t, sampleProject())
		require.True(t, m.selectTaskIdx(0, 0))
		require.NotNil(t, send(t, m, key("p")))
		send(t, m, clipboardLinesMsg{text: threeLines})
		assert.True(t, m.ui.Modes.IsPasteChooser())
		assert.False(t, m.ui.PasteChooser.fromEdit)
	})
	t.Run("a terminal paste in normal mode", func(t *testing.T) {
		m := newTestModel(t, sampleProject())
		require.True(t, m.selectTaskIdx(0, 0))
		send(t, m, tea.PasteMsg{Content: threeLines})
		assert.True(t, m.ui.Modes.IsPasteChooser())
	})
	t.Run("ctrl+v in the title editor", func(t *testing.T) {
		m := newTestModel(t, sampleProject())
		require.True(t, m.selectTaskIdx(0, 0))
		m.startEditing()
		require.NotNil(t, send(t, m, key("ctrl+v")), "ctrl+v reads through internal/clipboard")
		assert.Equal(t, "Alpha", m.ui.Edit.input.Value(), "and not through the textinput's own paste")
		send(t, m, editorClipboardMsg{mode: modes.ModeEdit, text: threeLines})
		assert.True(t, m.ui.Modes.IsPasteChooser())
		assert.True(t, m.ui.PasteChooser.fromEdit)
	})
	t.Run("a terminal paste in the title editor", func(t *testing.T) {
		m := newTestModel(t, sampleProject())
		require.True(t, m.selectTaskIdx(0, 0))
		m.startAddingTask()
		send(t, m, tea.PasteMsg{Content: threeLines})
		assert.True(t, m.ui.Modes.IsPasteChooser())
		assert.True(t, m.ui.PasteChooser.fromEdit)
		assert.True(t, m.editorOpen(), "the editor stays drawn behind the chooser")
	})
}

func TestPasteChooser_SingleRowSkipsIt(t *testing.T) {
	m := newTestModel(t, sampleProject())
	require.True(t, m.selectTaskIdx(0, 0))
	send(t, m, tea.PasteMsg{Content: "buy milk\n"})
	assert.True(t, m.ui.Modes.IsNormal())
	assert.Equal(t, "buy milk", m.project.Categories[0].Tasks[1].Title)

	m.startEditing()
	m.ui.Edit.input.SetCursor(0)
	send(t, m, tea.PasteMsg{Content: "Pre "})
	assert.True(t, m.ui.Modes.IsEdit())
	assert.Equal(t, "Pre buy milk", m.ui.Edit.input.Value(), "inserted verbatim at the caret")
}

func TestPasteChooser_EditorClipboardReadLandsOnlyWhereItWasAsked(t *testing.T) {
	m := newTestModel(t, sampleProject())
	require.True(t, m.selectTaskIdx(0, 0))
	send(t, m, editorClipboardMsg{mode: modes.ModeEdit, text: threeLines})
	assert.True(t, m.ui.Modes.IsNormal(), "the editor closed before the read came back")
	assert.Len(t, m.project.Categories[0].Tasks, 3)

	m.startEditing()
	send(t, m, editorClipboardMsg{mode: modes.ModeEdit, err: clipboard.ErrTimeout})
	assert.Equal(t, "Can't read the clipboard: paste with the terminal instead", m.ui.Screen.StatusMsg)
}

func TestPaste_ANewerClipboardRetiresTheCopyInThePastesUndoStep(t *testing.T) {
	m := newTestModel(t, sampleProject())
	require.True(t, m.selectTaskIdx(0, 0))
	m.cutSelectedTask()
	landWrite(m)

	require.True(t, m.selectTaskIdx(1, 0))
	m.resolvePaste(clipboardLinesMsg{text: threeLines})
	require.True(t, m.ui.Modes.IsPasteChooser())
	send(t, m, key("esc"))
	assert.True(t, m.isTaskCut("t1"), "a cancelled chooser leaves the cut alone")

	m.resolvePaste(clipboardLinesMsg{text: threeLines})
	send(t, m, key("enter"))
	assert.False(t, m.ui.Clipboard.IsCut, "the paste of newer text drops the stale cut")

	m.undo()
	assert.True(t, m.isTaskCut("t1"), "and undoing the paste brings the cut back")
	assert.Equal(t, []string{"t4", "t5"}, taskIDs(m.project.Categories[1]))
}

func TestPasteChooser_TitleSummarisesTheParse(t *testing.T) {
	m := newTestModel(t, sampleProject())
	require.True(t, m.selectTaskIdx(0, 0))
	m.pasteText("## A\n- one\n  about one\n- two\n---\n- three\n- four", false)

	view := m.pasteChooserView()
	assert.Contains(t, view, "Paste: 4 tasks, 2 separators, 1 description")
}

func TestPasteChooser_AppendOnlyOnATask(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(1) // Cat A
	m.pasteText(threeLines, false)
	assert.Len(t, m.ui.PasteChooser.options, 3)
	send(t, m, key("esc"))

	require.True(t, m.selectTaskIdx(0, 0))
	m.pasteText(threeLines, false)
	require.Len(t, m.ui.PasteChooser.options, 4)
	assert.Equal(t, "append to the description of “Alpha”", m.ui.PasteChooser.options[3].label)
}

func TestPasteChooser_NavigatesWithJKAndEnter(t *testing.T) {
	m := newTestModel(t, sampleProject())
	require.True(t, m.selectTaskIdx(0, 0))
	m.pasteText(threeLines, false)
	send(t, m, key("j"))
	send(t, m, key("j"))
	send(t, m, key("k"))
	send(t, m, key("down"))
	send(t, m, key("enter"))

	assert.True(t, m.ui.Modes.IsNormal())
	assert.Equal(t, "Groceries - milk - [x] bread", m.project.Categories[0].Tasks[1].Title,
		"the third option joins every line as it came")
}

func TestPasteChooser_EscCancels(t *testing.T) {
	m := newTestModel(t, sampleProject())
	require.True(t, m.selectTaskIdx(0, 0))
	m.pasteText(threeLines, false)
	send(t, m, key("esc"))

	assert.True(t, m.ui.Modes.IsNormal())
	assert.Len(t, m.project.Categories[0].Tasks, 3)
	assert.Equal(t, "Paste cancelled", m.ui.Screen.StatusMsg)
	assert.False(t, m.ui.History.CanUndo())
}

const notes = "- [~] Call the bank\n\n  ask about fees\n\n  - and the card\n"

func TestPasteChooser_EachModeFromNormal(t *testing.T) {
	for _, tc := range []struct {
		key   string
		check func(t *testing.T, cat domain.Category)
	}{
		{"t", func(t *testing.T, cat domain.Category) {
			assert.Equal(t, []string{"Alpha", "Call the bank", "and the card", "Beta", "Gamma"}, taskTitles(cat))
			assert.Equal(t, "ask about fees", cat.Tasks[1].Description)
		}},
		{"d", func(t *testing.T, cat domain.Category) {
			assert.Equal(t, []string{"Alpha", "Call the bank", "Beta", "Gamma"}, taskTitles(cat))
			assert.Equal(t, domain.StatusInProgress, cat.Tasks[1].Status, "the title line's marker is read, not kept")
			assert.Equal(t, "ask about fees\n\n- and the card", cat.Tasks[1].Description,
				"blank lines and list items stay in the description")
		}},
		{"o", func(t *testing.T, cat domain.Category) {
			assert.Equal(t, []string{"Alpha", "- [~] Call the bank ask about fees - and the card", "Beta", "Gamma"}, taskTitles(cat))
		}},
		{"a", func(t *testing.T, cat domain.Category) {
			assert.Equal(t, []string{"Alpha", "Beta", "Gamma"}, taskTitles(cat))
			assert.Equal(t, "old\n- [~] Call the bank\n\n  ask about fees\n\n  - and the card", cat.Tasks[0].Description)
		}},
	} {
		t.Run(tc.key, func(t *testing.T) {
			p := sampleProject()
			p.Categories[0].Tasks[0].Description = "old"
			m := newTestModel(t, p)
			require.True(t, m.selectTaskIdx(0, 0))
			before := cloneProject(m.project)

			send(t, m, tea.PasteMsg{Content: notes})
			require.True(t, m.ui.Modes.IsPasteChooser())
			send(t, m, key(tc.key))

			assert.True(t, m.ui.Modes.IsNormal())
			tc.check(t, m.project.Categories[0])
			m.undo()
			assert.Equal(t, before, m.project, "one undo step takes the paste back")
		})
	}
}

func TestPasteChooser_EachModeFromTheEditor(t *testing.T) {
	setup := func(t *testing.T) *model {
		p := sampleProject()
		p.Categories[0].Tasks[0].Description = "old"
		m := newTestModel(t, p)
		require.True(t, m.selectTaskIdx(0, 0))
		m.startEditing()
		m.ui.Edit.input.SetCursor(0)
		send(t, m, tea.PasteMsg{Content: notes})
		require.True(t, m.ui.Modes.IsPasteChooser())
		return m
	}

	t.Run("t splits into tasks and closes the editor", func(t *testing.T) {
		m := setup(t)
		send(t, m, key("t"))
		assert.True(t, m.ui.Modes.IsNormal())
		cat := m.project.Categories[0]
		assert.Equal(t, []string{"Alpha Call the bank", "and the card", "Beta", "Gamma"}, taskTitles(cat))
		assert.Equal(t, "old\nask about fees", cat.Tasks[0].Description)
		m.undo()
		assert.Equal(t, []string{"Alpha", "Beta", "Gamma"}, taskTitles(m.project.Categories[0]))
	})
	t.Run("d puts the first line in the title and the rest in the description", func(t *testing.T) {
		m := setup(t)
		send(t, m, key("d"))
		assert.True(t, m.ui.Modes.IsNormal())
		task := m.project.Categories[0].Tasks[0]
		assert.Equal(t, "Alpha Call the bank", task.Title, "the first line follows the typed title, as with t")
		assert.Equal(t, "Pasted into the title and description", m.ui.Screen.StatusMsg, "an edit adds no task")
		assert.Equal(t, domain.StatusTodo, task.Status, "an edited task keeps its status")
		assert.Equal(t, "old\nask about fees\n\n- and the card", task.Description)
		m.undo()
		assert.Equal(t, "Alpha", m.project.Categories[0].Tasks[0].Title)
		assert.Equal(t, "old", m.project.Categories[0].Tasks[0].Description)
	})
	t.Run("o joins the lines at the caret and keeps editing", func(t *testing.T) {
		m := setup(t)
		send(t, m, key("o"))
		assert.True(t, m.ui.Modes.IsEdit())
		assert.Equal(t, "- [~] Call the bank ask about fees - and the cardAlpha", m.ui.Edit.input.Value())
		assert.False(t, m.ui.History.CanUndo(), "nothing lands until the edit is saved")
	})
	t.Run("a appends to the description and keeps editing", func(t *testing.T) {
		m := setup(t)
		send(t, m, key("a"))
		assert.True(t, m.ui.Modes.IsEdit())
		assert.Equal(t, "Alpha", m.ui.Edit.input.Value())
		assert.Equal(t, "old\n- [~] Call the bank\n\n  ask about fees\n\n  - and the card", m.project.Categories[0].Tasks[0].Description)
		m.cancelEditing()
		m.undo()
		assert.Equal(t, "old", m.project.Categories[0].Tasks[0].Description)
	})
	t.Run("esc goes back to the editor untouched", func(t *testing.T) {
		m := setup(t)
		send(t, m, key("esc"))
		assert.True(t, m.ui.Modes.IsEdit())
		assert.Equal(t, "Alpha", m.ui.Edit.input.Value())
	})
}

func TestPasteChooser_AddFlowStaysOneUndoStep(t *testing.T) {
	for _, k := range []string{"t", "d", "a"} {
		t.Run(k, func(t *testing.T) {
			m := newTestModel(t, sampleProject())
			require.True(t, m.selectTaskIdx(0, 0))
			m.startAddingTask()
			send(t, m, tea.PasteMsg{Content: notes})
			send(t, m, key(k))
			if m.ui.Modes.IsEdit() {
				m.ui.Edit.input.SetValue("Typed")
				m.finishEditing()
			}
			require.Greater(t, len(m.project.Categories[0].Tasks), 3)

			m.undo()
			assert.Equal(t, []string{"Alpha", "Beta", "Gamma"}, taskTitles(m.project.Categories[0]))
			assert.False(t, m.ui.History.CanUndo())
		})
	}
}

func TestPaste_OneLineInputsFlattenIt(t *testing.T) {
	t.Run("category editor", func(t *testing.T) {
		m := newTestModel(t, sampleProject())
		m.ui.Selection.MoveTo(1)
		m.startEditing()
		send(t, m, tea.PasteMsg{Content: " one\ntwo\n"})
		assert.Equal(t, "Cat Aone two", m.ui.Edit.input.Value())
	})
	t.Run("search", func(t *testing.T) {
		m := newTestModel(t, sampleProject())
		m.startSearch()
		send(t, m, tea.PasteMsg{Content: "Gam\n"})
		assert.Equal(t, "Gam", m.ui.Search.input.Value())
		pos, _ := m.selectedPosition()
		assert.Equal(t, "Gamma", m.project.Categories[pos.CategoryIndex].Tasks[pos.TaskIndex].Title,
			"the search follows the pasted query")
	})
	t.Run("tag picker", func(t *testing.T) {
		m := newTestModel(t, sampleProject())
		require.True(t, m.selectTaskIdx(0, 0))
		m.startTagEdit()
		send(t, m, tea.PasteMsg{Content: "needs\nreview"})
		assert.True(t, m.ui.Modes.IsTagEdit())
		assert.Zero(t, m.ui.TagEdit.colorIdx, "a mark picker takes no text")
		assert.Equal(t, "Nothing here takes a paste", m.ui.Screen.StatusMsg)
	})
	t.Run("help filter", func(t *testing.T) {
		m := newTestModel(t, sampleProject())
		m.ui.Modes.ToggleHelp()
		send(t, m, tea.PasteMsg{Content: "undo\n"})
		assert.True(t, m.ui.Help.Filtering, "a paste starts the filter")
		assert.Equal(t, "undo", m.ui.Help.Filter.Value())
	})
	t.Run("picker filter", func(t *testing.T) {
		m := newTestModel(t, sampleProject())
		m.ui.Picker = ProjectPickerState{projects: []domain.Project{{ID: "a", Name: "Home"}, {ID: "b", Name: "Work"}}}
		m.ui.Modes.ToProjectPicker()
		m.ui.Picker.startFiltering()
		send(t, m, tea.PasteMsg{Content: "wo\n"})
		assert.Equal(t, "wo", m.ui.Picker.query)
		require.Len(t, m.ui.Picker.projects, 1)
		assert.Equal(t, "Work", m.ui.Picker.projects[0].Name)
	})
	t.Run("task command prompt", func(t *testing.T) {
		m := newCommandModel(t, shCommand("m", "when", "true"))
		m, _ = pressKeys(m, "m")
		require.True(t, m.ui.Modes.IsCommandPrompt())
		send(t, m, tea.PasteMsg{Content: "next\nfriday\n"})
		assert.Equal(t, "next friday", m.ui.TaskCommand.input.Value())
	})
	t.Run("visual mode says why it drops it", func(t *testing.T) {
		m := newTestModel(t, sampleProject())
		require.True(t, m.selectTaskIdx(0, 0))
		m.enterVisualMode()
		send(t, m, tea.PasteMsg{Content: "x"})
		assert.Equal(t, "Leave visual mode to paste", m.ui.Screen.StatusMsg)
	})
}

func TestPaste_DescriptionEditorReadsThroughTheClipboardPackage(t *testing.T) {
	m := newTestModel(t, sampleProject())
	require.True(t, m.selectTaskIdx(0, 0))
	m.startDescriptionInlineEdit(0, 0)
	require.True(t, m.ui.Modes.IsDescriptionEdit())

	require.NotNil(t, send(t, m, key("ctrl+v")))
	send(t, m, editorClipboardMsg{mode: modes.ModeDescriptionEdit, text: "line one\nline two"})

	assert.Equal(t, "line one\nline two", m.ui.DescriptionEdit.textarea.Value())
}

func TestPasteChooser_DescriptionModeNeverSavesAnEmptyTitle(t *testing.T) {
	assert.Equal(t, "body", titleAndDescription("## \nbody").title, "a bare heading marker is no title")

	m := newTestModel(t, sampleProject())
	require.True(t, m.selectTaskIdx(0, 0))
	m.startEditing()
	m.ui.Edit.input.SetValue("")
	send(t, m, tea.PasteMsg{Content: "## \n## "})
	require.True(t, m.ui.Modes.IsPasteChooser())
	send(t, m, key("d"))

	assert.True(t, m.ui.Modes.IsEdit(), "the editor stays open")
	assert.Equal(t, "Alpha", m.project.Categories[0].Tasks[0].Title)
	assert.False(t, m.ui.History.CanUndo())
}

func TestPasteChooser_KeepsTheTaskFocused(t *testing.T) {
	m := newTestModelWithDescription(t)
	m.ui.Selection.MoveTo(3) // t1's description row
	m.pasteText(threeLines, false)
	send(t, m, key("a"))

	pos, ok := m.selectedPosition()
	require.True(t, ok)
	assert.Equal(t, selection.FocusDescription, pos.Kind)
	assert.Equal(t, "alpha details\nGroceries\n- milk\n- [x] bread", m.project.Categories[0].Tasks[0].Description)
}
