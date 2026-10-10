package modes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewMachine(t *testing.T) {
	t.Run("initializes with given mode", func(t *testing.T) {
		m := NewMachine(ModeNormal)
		assert.Equal(t, ModeNormal, m.Current())
		assert.True(t, m.IsNormal())
	})

	t.Run("initializes with edit mode", func(t *testing.T) {
		m := NewMachine(ModeEdit)
		assert.Equal(t, ModeEdit, m.Current())
		assert.True(t, m.IsEdit())
	})
}

func TestMachine_TransitionTo(t *testing.T) {
	t.Run("normal can transition to any mode", func(t *testing.T) {
		m := NewMachine(ModeNormal)
		assert.True(t, m.TransitionTo(ModeEdit))
		assert.True(t, m.IsEdit())

		m = NewMachine(ModeNormal)
		assert.True(t, m.TransitionTo(ModeHelp))
		assert.True(t, m.IsHelp())

		m = NewMachine(ModeNormal)
		assert.True(t, m.TransitionTo(ModeConfirmDelete))
		assert.True(t, m.IsConfirmDelete())

		m = NewMachine(ModeNormal)
		assert.True(t, m.TransitionTo(ModeOptions))
		assert.True(t, m.IsOptions())

		m = NewMachine(ModeNormal)
		assert.True(t, m.TransitionTo(ModeProjectPicker))
		assert.True(t, m.IsProjectPicker())
	})

	t.Run("edit can only transition to normal", func(t *testing.T) {
		m := NewMachine(ModeEdit)
		assert.False(t, m.TransitionTo(ModeHelp))
		assert.True(t, m.IsEdit())

		assert.True(t, m.TransitionTo(ModeNormal))
		assert.True(t, m.IsNormal())
	})

	t.Run("help can only transition to normal", func(t *testing.T) {
		m := NewMachine(ModeHelp)
		assert.False(t, m.TransitionTo(ModeEdit))
		assert.True(t, m.IsHelp())

		assert.True(t, m.TransitionTo(ModeNormal))
		assert.True(t, m.IsNormal())
	})

	t.Run("confirm delete can only transition to normal", func(t *testing.T) {
		m := NewMachine(ModeConfirmDelete)
		assert.False(t, m.TransitionTo(ModeEdit))
		assert.True(t, m.IsConfirmDelete())

		assert.True(t, m.TransitionTo(ModeNormal))
		assert.True(t, m.IsNormal())
	})
}

func TestMachine_CanPerformAction(t *testing.T) {
	t.Run("normal mode allows all actions", func(t *testing.T) {
		m := NewMachine(ModeNormal)
		assert.True(t, m.CanPerformAction(ActionNavigate))
		assert.True(t, m.CanPerformAction(ActionToggleTask))
		assert.True(t, m.CanPerformAction(ActionDeleteItem))
		assert.True(t, m.CanPerformAction(ActionEditItem))
		assert.True(t, m.CanPerformAction(ActionChangePriority))
	})

	t.Run("edit mode blocks all actions", func(t *testing.T) {
		m := NewMachine(ModeEdit)
		assert.False(t, m.CanPerformAction(ActionNavigate))
		assert.False(t, m.CanPerformAction(ActionToggleTask))
		assert.False(t, m.CanPerformAction(ActionDeleteItem))
	})

	t.Run("help mode only allows toggle help", func(t *testing.T) {
		m := NewMachine(ModeHelp)
		assert.False(t, m.CanPerformAction(ActionNavigate))
		assert.True(t, m.CanPerformAction(ActionOpenHelp))
	})

	t.Run("options mode blocks all actions", func(t *testing.T) {
		m := NewMachine(ModeOptions)
		assert.False(t, m.CanPerformAction(ActionNavigate))
		assert.False(t, m.CanPerformAction(ActionEditItem))
	})
}

func TestMachine_ToggleHelp(t *testing.T) {
	t.Run("toggles from normal to help", func(t *testing.T) {
		m := NewMachine(ModeNormal)
		m.ToggleHelp()
		assert.True(t, m.IsHelp())
	})

	t.Run("toggles from help to normal", func(t *testing.T) {
		m := NewMachine(ModeHelp)
		m.ToggleHelp()
		assert.True(t, m.IsNormal())
	})

	t.Run("does nothing from other modes", func(t *testing.T) {
		m := NewMachine(ModeEdit)
		m.ToggleHelp()
		assert.True(t, m.IsEdit())
	})
}

func TestMachine_ConvenienceMethods(t *testing.T) {
	t.Run("ToEdit", func(t *testing.T) {
		m := NewMachine(ModeNormal)
		assert.True(t, m.ToEdit())
		assert.True(t, m.IsEdit())
	})

	t.Run("ToHelp", func(t *testing.T) {
		m := NewMachine(ModeNormal)
		assert.True(t, m.ToHelp())
		assert.True(t, m.IsHelp())
	})

	t.Run("ToConfirmDelete", func(t *testing.T) {
		m := NewMachine(ModeNormal)
		assert.True(t, m.ToConfirmDelete())
		assert.True(t, m.IsConfirmDelete())
	})

	t.Run("ToOptions", func(t *testing.T) {
		m := NewMachine(ModeNormal)
		assert.True(t, m.ToOptions())
		assert.True(t, m.IsOptions())
	})

	t.Run("ToProjectPicker", func(t *testing.T) {
		m := NewMachine(ModeNormal)
		assert.True(t, m.ToProjectPicker())
		assert.True(t, m.IsProjectPicker())
	})

	t.Run("ToNormal always works", func(t *testing.T) {
		m := NewMachine(ModeEdit)
		m.ToNormal()
		assert.True(t, m.IsNormal())
	})
}

func TestMachine_PasteChooserReturnsWhereItOpened(t *testing.T) {
	m := NewMachine(ModeEdit)
	assert.True(t, m.ToPasteChooser())
	assert.True(t, m.IsPasteChooser())
	assert.False(t, m.CanPerformAction(ActionDeleteItem))
	assert.False(t, m.ToHelp())
	assert.True(t, m.ToEdit())

	m = NewMachine(ModeNormal)
	assert.True(t, m.ToPasteChooser())
	m.ToNormal()
	assert.True(t, m.IsNormal())

	m = NewMachine(ModeSearch)
	assert.False(t, m.ToPasteChooser())
}
