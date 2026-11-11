package app

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/config"
	"github.com/ilyasturki/ordo/internal/domain"
	"github.com/ilyasturki/ordo/internal/ui"
)

// lineTagProject is Cat A [Alpha, separator s1 "mid", note n1, Beta].
func lineTagProject() domain.Project {
	return domain.Project{
		ID: "p1", Name: "P",
		Categories: []domain.Category{{
			ID: "c1", Name: "Cat A",
			Tasks: []domain.Task{
				{ID: "t1", Title: "Alpha", Status: domain.StatusTodo},
				{ID: "s1", Kind: domain.KindSeparator, Title: "mid"},
				{ID: "n1", Kind: domain.KindNote, Title: "ask Sam"},
				{ID: "t2", Title: "Beta", Status: domain.StatusTodo},
			},
		}},
	}
}

func TestTagKeysMarkSeparatorsAndNotes(t *testing.T) {
	for _, id := range []string{"s1", "n1"} {
		t.Run(id, func(t *testing.T) {
			m := newTestModel(t, lineTagProject())
			require.True(t, m.selectRow(id))
			row := func() domain.Task { return *m.taskByID(id) }

			m, _ = pressKeys(m, "t")
			assert.Equal(t, domain.TagGreen, row().TagColor, "t gives the line the first free mark")
			assert.Empty(t, m.taskByID("t2").TagColor, "the tasks below keep theirs")

			m, _ = pressKeys(m, "T")
			require.True(t, m.ui.Modes.IsTagEdit(), "T opens the picker on a line")
			m.handleTagEditKey(keyPress("j"))
			m.handleTagEditKey(keyPress("enter"))
			assert.Equal(t, domain.NextTagColor(domain.TagGreen), row().TagColor, "T picks one")

			m, _ = pressKeys(m, "g", "t")
			require.True(t, m.selectRow("t1"))
			m, _ = pressKeys(m, "p")
			assert.Equal(t, row().TagColor, m.taskByID("t1").TagColor, "gt on a line copies its mark")

			m.taskByID("t2").TagColor = domain.TagCyan
			require.True(t, m.selectRow("t2"))
			m, _ = pressKeys(m, "g", "t")
			require.True(t, m.selectRow(id))
			m, _ = pressKeys(m, "p")
			assert.Equal(t, domain.TagCyan, row().TagColor, "p paints a line")

			m, _ = pressKeys(m, "u")
			assert.Equal(t, domain.NextTagColor(domain.TagGreen), row().TagColor, "one undo step")
		})
	}
}

func TestVisualTagCoversLines(t *testing.T) {
	m := newTestModel(t, lineTagProject())
	require.True(t, m.selectRow("t1"))
	m, _ = pressKeys(m, "v", "j", "j", "j", "t")
	color := m.taskByID("t1").TagColor
	require.NotEmpty(t, color)
	for _, id := range []string{"s1", "n1", "t2"} {
		assert.Equal(t, color, m.taskByID(id).TagColor, "%s takes the block's mark", id)
	}
}

func TestVisualTagAnchoredOnALine(t *testing.T) {
	for _, id := range []string{"s1", "n1"} {
		t.Run(id, func(t *testing.T) {
			m := newTestModel(t, lineTagProject())
			require.True(t, m.selectRow(id))
			m, _ = pressKeys(m, "v", "j", "t")
			color := m.taskByID(id).TagColor
			require.NotEmpty(t, color, "a range from a line tags it")
			next := map[string]string{"s1": "n1", "n1": "t2"}[id]
			assert.Equal(t, color, m.taskByID(next).TagColor)
		})
	}
}

func TestTagBatchNamesLinesByKind(t *testing.T) {
	p := lineTagProject()
	for i := range p.Categories[0].Tasks {
		p.Categories[0].Tasks[i].TagColor = domain.TagBlue
	}
	m := newTestModel(t, p)
	require.True(t, m.selectRow("s1"))
	m.startTagEdit()
	assert.Contains(t, ansi.Strip(m.tagEditView()), "all 4 rows marked")
	m.handleTagEditKey(keyPress("space"))
	m.ui.TagEdit.colorIdx = tagColorIndex(domain.TagCyan)
	m.handleTagEditKey(keyPress("enter"))
	assert.Equal(t, "Tagged 4 row(s)", m.ui.Screen.StatusMsg)
	assert.Equal(t, "separator", rowsNoun(m.project, []string{"s1"}))
	assert.Equal(t, "note", rowsNoun(m.project, []string{"n1"}))
	assert.Equal(t, "task", rowsNoun(m.project, []string{"t1", "t2"}))
}

func TestTaggedLinesDrawTheDot(t *testing.T) {
	p := lineTagProject()
	p.Categories[0].Tasks[1].TagColor = domain.TagGreen
	p.Categories[0].Tasks[2].TagColor = domain.TagGreen
	m := newTestModel(t, p)
	m.ui.Screen.Width = 60

	sep := ansi.Strip(m.renderSeparatorLine(p.Categories[0].Tasks[1], 0, false, true, false, false, false, 60, "", ui.HeaderStyle))
	assert.True(t, strings.HasPrefix(sep, "  ━━ "+ui.TagGlyph+" mid ━"), sep)
	assert.Equal(t, 2+sepContentCap, ansi.StringWidth(sep), "the dot takes room from the rule")

	bare := p.Categories[0].Tasks[1]
	bare.Title = ""
	assert.True(t, strings.HasPrefix(ansi.Strip(m.renderSeparatorLine(bare, 0, false, true, false, false, false, 60, "", ui.HeaderStyle)), "  ━━ "+ui.TagGlyph+" ━"))
	assert.NotContains(t, ansi.Strip(m.renderSeparatorLine(bare, 0, true, true, false, false, false, 60, "", ui.HeaderStyle)), "mid")

	note := ansi.Strip(m.renderNoteLine(p.Categories[0].Tasks[2], 0, false, true, false, false, false))
	assert.Equal(t, "   -  "+ui.TagGlyph+" ask Sam", note)

	require.True(t, m.selectRow("n1"))
	assert.Contains(t, ansi.Strip(strings.Join(m.infoLines(), "\n")), "Tag:      "+ui.TagGlyph+" Green")
}

func TestTaggedNoteLayoutMatchesRender(t *testing.T) {
	for _, display := range []string{config.StatusDisplayIcons, config.StatusDisplayText} {
		for n := 1; n <= 30; n++ {
			p := lineTagProject()
			p.Categories[0].Tasks[2].TagColor = domain.TagMagenta
			p.Categories[0].Tasks[2].Title = strings.TrimSpace(strings.Repeat("word ", n))
			m := newTestModel(t, p)
			m.ui.Screen.Width = 50
			_ = m.deps.CfgManager.Update(func(cfg *config.Config) { cfg.StatusDisplay = display })
			m.invalidateLayout()
			item := itemFor(t, m, LayoutSeparator, 2)
			drawn := strings.Split(ansi.Strip(m.renderLayoutItem(item)), "\n")
			assert.Equal(t, item.Height, len(drawn), "%s n=%d", display, n)
			column := taskTitleColumn(domain.Task{TagColor: domain.TagMagenta}, display, 0)
			for _, line := range drawn[1:] {
				assert.Equal(t, strings.Repeat(" ", column)+"word", line[:column+4], "%s n=%d: wrapped lines hang under the text", display, n)
			}
		}
	}
}
