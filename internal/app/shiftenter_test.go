package app

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/app/selection"
)

// decodeKey runs raw terminal bytes through the same decoder bubbletea v2 uses
// and returns the tea.KeyPressMsg string the app matches bindings against.
func decodeKey(t *testing.T, raw string) string {
	t.Helper()
	dec := &uv.EventDecoder{}
	n, ev := dec.Decode([]byte(raw))
	require.Greater(t, n, 0, "decoder consumed no bytes for %q", raw)
	kp, ok := ev.(uv.KeyPressEvent)
	require.True(t, ok, "expected KeyPressEvent, got %T for %q", ev, raw)
	return tea.KeyPressMsg(kp).String()
}

// TestShiftEnterWireEncodings pins the strings real terminals deliver when the
// user presses Shift+Enter. kitty and ghostty send ESC+CR, which decodes to
// "alt+enter"; Kitty-protocol terminals may instead send a true "shift+enter".
// The description binding must accept every form here.
func TestShiftEnterWireEncodings(t *testing.T) {
	assert.Equal(t, "alt+enter", decodeKey(t, "\x1b\r"), "kitty/ghostty send ESC+CR for Shift+Enter")
	assert.Equal(t, "shift+enter", decodeKey(t, "\x1b[13;2u"), "Kitty CSI-u modified Enter")
	assert.Equal(t, "shift+enter", decodeKey(t, "\x1b[27;2;13~"), "xterm modifyOtherKeys modified Enter")
}

// TestDescriptionBindingAcceptsShiftEnterForms is the regression guard: every
// wire encoding of a Shift+Enter press must route to the description action.
// Regressed because the binding only listened for "shift+enter" while the
// user's terminals deliver "alt+enter".
func TestDescriptionBindingAcceptsShiftEnterForms(t *testing.T) {
	for _, key := range []string{"shift+enter", "alt+enter"} {
		p := sampleProject()
		p.Categories[0].Tasks[0].Description = "Some details"
		m := newTestModel(t, p)
		selectFirstTask(m)

		m.dispatchNormalKey(key)

		pos, ok := m.selectedPosition()
		require.True(t, ok)
		assert.Equal(t, selection.FocusDescription, pos.Kind,
			"%q must jump to the description row", key)
	}
}

// TestPlainEnterEditsTitle guards that the plain Enter path is unchanged: it
// edits the title rather than touching the description.
func TestPlainEnterEditsTitle(t *testing.T) {
	assert.Equal(t, "enter", decodeKey(t, "\r"))

	p := sampleProject()
	p.Categories[0].Tasks[0].Description = "Some details"
	m := newTestModel(t, p)
	selectFirstTask(m)

	m.dispatchNormalKey("enter")
	assert.True(t, m.ui.Modes.IsEdit(), "plain enter edits the task title")
}

func selectFirstTask(m *model) {
	m.ui.Selection.SelectByPredicate(func(pp selection.Position) bool {
		return pp.Kind == selection.FocusTask && pp.CategoryIndex == 0 && pp.TaskIndex == 0
	})
}
