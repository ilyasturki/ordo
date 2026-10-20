package app

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/app/modes"
	"github.com/ilyasturki/ordo/internal/app/selection"
	"github.com/ilyasturki/ordo/internal/domain"
)

var ctrlC = tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}

func TestCtrlC_InlineEditsSaveThenQuit(t *testing.T) {
	cases := map[string]struct {
		open  func(m *model)
		check func(t *testing.T, p domain.Project)
	}{
		"title": {
			open: func(m *model) { m.selectItem(selection.FocusTask, "t1"); m.startEditing() },
			check: func(t *testing.T, p domain.Project) {
				assert.Equal(t, "Typed before ctrl+c", p.Categories[0].Tasks[0].Title)
			},
		},
		"add task": {
			open: func(m *model) { m.selectItem(selection.FocusTask, "t1"); m.startAddingTask() },
			check: func(t *testing.T, p domain.Project) {
				var titles []string
				for _, task := range p.Categories[0].Tasks {
					titles = append(titles, task.Title)
				}
				assert.Contains(t, titles, "Typed before ctrl+c")
			},
		},
		"category": {
			open: func(m *model) { m.selectItem(selection.FocusCategory, "c1"); m.startEditing() },
			check: func(t *testing.T, p domain.Project) {
				assert.Equal(t, "Typed before ctrl+c", p.Categories[0].Name)
			},
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			m := newTestModel(t, sampleProject())
			c.open(m)
			require.True(t, m.ui.Modes.IsEdit())
			m.ui.Edit.input.SetValue("Typed before ctrl+c")

			after, cmd := m.handleKeyMsg(ctrlC)

			require.NotNil(t, cmd)
			assert.IsType(t, tea.QuitMsg{}, cmd())
			c.check(t, after.(model).project)
		})
	}
}

func TestCtrlC_RefusedInlineEditQuitsOnTheSecondPress(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.selectItem(selection.FocusCategory, "c1")
	m.startEditing()
	m.ui.Edit.input.SetValue("Cat B")

	next, cmd := m.handleKeyMsg(ctrlC)
	assert.Nil(t, cmd, "the taken name keeps the editor open with its merge offer")
	after := next.(model)
	assert.True(t, after.ui.Modes.IsEdit())

	_, cmd = after.handleKeyMsg(ctrlC)
	require.NotNil(t, cmd)
	assert.IsType(t, tea.QuitMsg{}, cmd())
	assert.Equal(t, "Cat A", after.project.Categories[0].Name)
}

func TestCtrlC_QuitsFromEveryMode(t *testing.T) {
	for mode := modes.ModeNormal; mode <= modes.ModeConfirmLeave; mode++ {
		if mode == modes.ModeExternalEdit {
			continue
		}
		m := newTestModel(t, sampleProject())
		m.ui.Modes = modes.NewMachine(mode)

		_, cmd := m.handleKeyMsg(ctrlC)

		require.NotNil(t, cmd, "mode %d", mode)
		assert.IsType(t, tea.QuitMsg{}, cmd(), "mode %d", mode)
	}
}

func TestCtrlC_PickerAddInputQuits(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Modes = modes.NewMachine(modes.ModeProjectPicker)
	after, _ := m.handleKeyMsg(tea.KeyPressMsg{Code: 'a', Text: "a"})

	_, cmd := after.(model).handleKeyMsg(ctrlC)

	require.NotNil(t, cmd)
	assert.IsType(t, tea.QuitMsg{}, cmd())
}

func TestCtrlC_ExternalEditIgnores(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Modes = modes.NewMachine(modes.ModeExternalEdit)

	_, cmd := m.handleKeyMsg(ctrlC)

	assert.Nil(t, cmd)
}

func TestCtrlC_DescriptionEditSavesThenQuits(t *testing.T) {
	p := sampleProject()
	p.Categories[0].Tasks[0].Description = "original"
	m := newTestModel(t, p)
	_ = m.startDescriptionInlineEdit(0, 0)
	m.ui.DescriptionEdit.textarea.SetValue("a long description\nworth keeping")

	after, cmd := m.handleKeyMsg(ctrlC)

	require.NotNil(t, cmd)
	assert.IsType(t, tea.QuitMsg{}, cmd())
	assert.Equal(t, "a long description\nworth keeping", after.(model).project.Categories[0].Tasks[0].Description)
}
