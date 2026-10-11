package app

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/app/modes"
)

var ctrlC = tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}

func TestCtrlC_QuitsFromEveryMode(t *testing.T) {
	for mode := modes.ModeNormal; mode <= modes.ModeChangelog; mode++ {
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
