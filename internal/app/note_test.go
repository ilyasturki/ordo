package app

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/app/selection"
	"github.com/ilyasturki/ordo/internal/config"
	"github.com/ilyasturki/ordo/internal/domain"
)

// noteProject is Cat A [Alpha (Alpha1), note n1, Beta].
func noteProject() domain.Project {
	return domain.Project{
		ID: "p1", Name: "P",
		Categories: []domain.Category{{
			ID: "c1", Name: "Cat A",
			Tasks: []domain.Task{
				{ID: "t1", Title: "Alpha", Status: domain.StatusTodo},
				{ID: "t1a", ParentID: "t1", Title: "Alpha1", Status: domain.StatusTodo},
				{ID: "n1", Kind: domain.KindNote, Title: "ask Sam before merging"},
				{ID: "t2", Title: "Beta", Status: domain.StatusTodo},
			},
		}},
	}
}

func (m *model) selectRow(id string) bool {
	return m.ui.Selection.SelectByPredicate(func(p selection.Position) bool { return p.ID == id })
}

func TestNoteIsALineRow(t *testing.T) {
	m := newTestModel(t, noteProject())
	require.True(t, m.selectRow("n1"))
	pos, _ := m.selectedPosition()
	assert.Equal(t, selection.FocusSeparator, pos.Kind, "a note takes the separator's handling")

	m, _ = pressKeys(m, ">")
	assert.Equal(t, "A note doesn't nest", m.ui.Screen.StatusMsg)
	assert.Equal(t, "> ask Sam before merging", m.copyTextForPosition(pos))
	assert.Contains(t, ansi.Strip(strings.Join(m.infoLines(), "\n")), "Note Info")
}

func TestAddNoteBelowTheBlock(t *testing.T) {
	m := newTestModel(t, noteProject())
	require.True(t, m.selectRow("t1a"))
	depth := m.ui.History.UndoDepth()

	m, _ = pressKeys(m, "_")
	require.True(t, m.ui.Modes.IsEdit(), "_ opens the note's editor at once")
	assert.Contains(t, ansi.Strip(m.View().Content), "Enter note text...")
	m, _ = pressKeys(m, "c", "a", "l", "l", "enter")

	tasks := m.project.Categories[0].Tasks
	require.Len(t, tasks, 5)
	assert.True(t, tasks[2].IsNote(), "a note below a subtask lands after its parent's block")
	assert.Equal(t, "call", tasks[2].Title)
	assert.Empty(t, tasks[2].ParentID)
	assert.True(t, m.ui.Modes.IsNormal())
	assert.Equal(t, depth+1, m.ui.History.UndoDepth(), "adding a note is one undo step")

	m, _ = pressKeys(m, "u")
	assert.Len(t, m.project.Categories[0].Tasks, 4)
}

func TestAddNoteLeftEmptyIsDropped(t *testing.T) {
	for _, key := range []string{"esc", "enter"} {
		t.Run(key, func(t *testing.T) {
			m := newTestModel(t, noteProject())
			require.True(t, m.selectRow("t2"))
			depth := m.ui.History.UndoDepth()
			m, _ = pressKeys(m, "_", " ", key)
			assert.Len(t, m.project.Categories[0].Tasks, 4)
			assert.Equal(t, depth, m.ui.History.UndoDepth())
			assert.True(t, m.ui.Modes.IsNormal())
		})
	}
}

func TestAddNoteRefusedUnderAFilter(t *testing.T) {
	m := newTestModel(t, noteProject())
	m.ui.Filter = NewFilterState()
	m.ui.Filter.statuses = map[string]bool{domain.StatusTodo: true}
	m.rebuildPositions()
	require.True(t, m.selectRow("t2"))
	for _, p := range m.ui.Selection.Positions() {
		require.NotEqual(t, "n1", p.ID, "a filter hides notes")
	}

	m, _ = pressKeys(m, "_")
	assert.Equal(t, "Clear the filter to add a note", m.ui.Screen.StatusMsg)
	assert.Len(t, m.project.Categories[0].Tasks, 4)
}

func TestEditNoteNeverEmptiesIt(t *testing.T) {
	m := newTestModel(t, noteProject())
	require.True(t, m.selectRow("n1"))

	m, _ = pressKeys(m, "enter")
	m.ui.Edit.input.SetValue("ask Sam")
	m, _ = pressKeys(m, "enter")
	assert.Equal(t, "ask Sam", m.project.Categories[0].Tasks[2].Title)

	m, _ = pressKeys(m, "enter")
	m.ui.Edit.input.SetValue("  ")
	m, _ = pressKeys(m, "enter")
	assert.Equal(t, "ask Sam", m.project.Categories[0].Tasks[2].Title, "an emptied note keeps its text")
	assert.Len(t, m.project.Categories[0].Tasks, 4)
}

func TestDeleteNoteAtOnce(t *testing.T) {
	m := newTestModel(t, noteProject())
	require.True(t, m.selectRow("n1"))
	m, _ = pressKeys(m, "d")
	assert.True(t, m.ui.Modes.IsNormal(), "a note deletes with no confirm, like a separator")
	assert.Len(t, m.project.Categories[0].Tasks, 3)
}

func TestNoteRendersADashUnderTheStatusGlyph(t *testing.T) {
	for display, lead := range map[string]string{
		config.StatusDisplayIcons: "   -  ",
		config.StatusDisplayText:  "       -      ",
	} {
		t.Run(display, func(t *testing.T) {
			p := noteProject()
			p.Categories[0].Tasks[2].Title = strings.Repeat("word ", 40)
			m := newTestModel(t, p)
			m.ui.Screen.Width = 40
			_ = m.deps.CfgManager.Update(func(cfg *config.Config) { cfg.StatusDisplay = display })
			m.invalidateLayout()

			task := ansi.Strip(m.renderTaskLine(0, 3, false, m.ui.Screen.Width, true, false, false, false))
			titleColumn := strings.Index(task, "Beta")
			require.Equal(t, len(lead), titleColumn, "the note's text starts where titles start")
			assert.Equal(t, "-", lead[strings.Index(task, "[")+len(statusLabel("", display))/2+1:][:1], "the dash sits under the status glyph")

			item := itemFor(t, m, LayoutSeparator, 2)
			rendered := m.renderNoteLine(p.Categories[0].Tasks[2], 0, false, true, false, false, false)
			lines := strings.Split(ansi.Strip(rendered), "\n")
			assert.Len(t, lines, 3, "a long note clamps like a title")
			assert.Equal(t, item.Height, len(lines), "the layout reserves the rows the note draws")
			assert.Equal(t, lead+"word", lines[0][:len(lead)+4])
			for _, line := range lines[1:] {
				assert.Equal(t, strings.Repeat(" ", titleColumn)+"word", line[:titleColumn+4], "a wrapped line hangs under the text")
			}
			assert.NotContains(t, rendered, "[")
			assert.NotContains(t, rendered, "━")
			assert.True(t, strings.HasSuffix(lines[2], "…"))

			require.True(t, m.selectRow("n1"))
			m, _ = pressKeys(m, "enter")
			editor := strings.Split(ansi.Strip(m.renderEditNoteLine(m.project.Categories[0].Tasks[2], 0)), "\n")[0]
			assert.Equal(t, ">"+lead[1:]+"word", editor[:len(lead)+4], "the editor keeps the dash and the column")
		})
	}
}

func TestNoteLayoutMatchesRender(t *testing.T) {
	for _, display := range []string{config.StatusDisplayIcons, config.StatusDisplayText} {
		for _, cut := range []bool{false, true} {
			for n := 1; n <= 40; n++ {
				p := noteProject()
				p.Categories[0].Tasks[2].Title = strings.TrimSpace(strings.Repeat("word ", n))
				m := newTestModel(t, p)
				m.ui.Screen.Width = 50
				_ = m.deps.CfgManager.Update(func(cfg *config.Config) { cfg.StatusDisplay = display })
				if cut {
					m.ui.Clipboard = ClipboardState{IsCut: true, TaskIDs: []string{"n1"}}
				}
				m.invalidateLayout()
				item := itemFor(t, m, LayoutSeparator, 2)
				drawn := strings.Count(m.renderLayoutItem(item), "\n") + 1
				assert.Equal(t, item.Height, drawn, "%s cut=%v n=%d", display, cut, n)
			}
		}
	}
}

func TestExpandShowsANoteInFull(t *testing.T) {
	p := noteProject()
	p.Categories[0].Tasks[2].Title = strings.Repeat("word ", 40)
	m := newTestModel(t, p)
	m.ui.Screen.Width = 40

	require.True(t, m.selectRow("n1"))
	m.toggleExpandRow()
	assert.Equal(t, "n1", m.ui.Screen.ExpandedTaskID, "ze shows a note in full")
	assert.Greater(t, itemFor(t, m, LayoutSeparator, 2).Height, 3)
}

func TestVisualCopyWritesNoteLines(t *testing.T) {
	m := newTestModel(t, noteProject())
	require.True(t, m.selectRow("n1"))
	m.enterVisualMode()
	m.visualMoveCursor(1)

	assert.Equal(t, "> ask Sam before merging\n- [ ] Beta", m.visualMarkdownText(m.visualSelectedPositions(), true))
}
