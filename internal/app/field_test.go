package app

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fieldKey builds the key a terminal delivers for name, as bubbletea names it.
func fieldKey(t *testing.T, name string) tea.KeyPressMsg {
	t.Helper()
	var msg tea.KeyPressMsg
	if len([]rune(name)) == 1 {
		msg = tea.KeyPressMsg{Code: []rune(name)[0], Text: name}
	} else {
		parts := strings.Split(name, "+")
		last := parts[len(parts)-1]
		if last == "" {
			last = "+"
		}
		for _, mod := range parts[:len(parts)-1] {
			switch mod {
			case "ctrl":
				msg.Mod |= tea.ModCtrl
			case "alt":
				msg.Mod |= tea.ModAlt
			case "shift":
				msg.Mod |= tea.ModShift
			}
		}
		switch last {
		case "backspace":
			msg.Code = tea.KeyBackspace
		case "delete":
			msg.Code = tea.KeyDelete
		case "left":
			msg.Code = tea.KeyLeft
		default:
			msg.Code = []rune(last)[0]
		}
	}
	want := name
	if name == " " {
		msg.Code, want = tea.KeySpace, "space"
	}
	require.Equal(t, want, msg.String())
	return msg
}

func freshKillRing(t *testing.T) {
	t.Helper()
	killRing = ""
	t.Cleanup(func() { killRing = "" })
}

// fieldUnderTest drives one widget kind through the same keys.
type fieldUnderTest struct {
	name  string
	start func(value string, cursor int) fieldBuffer
	press func(t *testing.T, keys ...string)
}

func fieldKinds() []func() fieldUnderTest {
	line := func() fieldUnderTest {
		f := newTextInput()
		f.Focus()
		return fieldUnderTest{
			name: "textinput",
			start: func(value string, cursor int) fieldBuffer {
				f.SetValue(value)
				f.SetCursor(cursor)
				return &f
			},
			press: func(t *testing.T, keys ...string) {
				for _, k := range keys {
					f, _ = f.Update(fieldKey(t, k))
				}
			},
		}
	}
	area := func() fieldUnderTest {
		ta := textarea.New()
		ta.SetWidth(80)
		ta.Focus()
		f := areaField{Model: ta}
		return fieldUnderTest{
			name: "textarea",
			start: func(value string, cursor int) fieldBuffer {
				f.restore(fieldSnap{value: value, col: cursor})
				return &f
			},
			press: func(t *testing.T, keys ...string) {
				for _, k := range keys {
					f, _ = f.Update(fieldKey(t, k))
				}
			},
		}
	}
	return []func() fieldUnderTest{line, area}
}

func typedKeys(s string) []string {
	keys := make([]string, 0, len(s))
	for _, r := range s {
		keys = append(keys, string(r))
	}
	return keys
}

func TestFieldKeys(t *testing.T) {
	cases := []struct {
		name    string
		value   string
		cursor  int
		keys    []string
		want    string
		wantCol int
	}{
		{"ctrl+t swaps around the cursor", "one two three", 6, []string{"ctrl+t"}, "one tow three", 7},
		{"ctrl+t at the end swaps the last two", "abc", 3, []string{"ctrl+t"}, "acb", 3},
		{"ctrl+t at the start does nothing", "abc", 0, []string{"ctrl+t"}, "abc", 0},
		{"alt+t drags the word before past the next", "one two three", 6, []string{"alt+t"}, "two one three", 7},
		{"alt+t at the end swaps the last two words", "one two three", 13, []string{"alt+t"}, "one three two", 13},
		{"alt+t with one word does nothing", "one", 1, []string{"alt+t"}, "one", 1},
		{"alt+u upcases to the word's end", "one two three", 5, []string{"alt+u"}, "one tWO three", 7},
		{"alt+l downcases the next word", "ONE TWO THREE", 3, []string{"alt+l"}, "ONE two THREE", 7},
		{"alt+c capitalizes the next word", "one tWO three", 3, []string{"alt+c"}, "one Two three", 7},
		{"alt+d on a word's last letter keeps the next word", "one two three", 6, []string{"alt+d"}, "one tw three", 6},
		{"alt+d before a word kills the space and the word", "one two three", 3, []string{"alt+d"}, "one three", 3},
		{"alt+< goes to the start", "one two", 5, []string{"alt+<"}, "one two", 0},
		{"alt+> goes to the end", "one two", 2, []string{"alt+>"}, "one two", 7},
		{"ctrl+y yanks what ctrl+w killed", "one two three", 13, []string{"ctrl+w", "alt+<", "ctrl+y"}, "threeone two ", 5},
		{"ctrl+y yanks what ctrl+k killed", "one two three", 4, []string{"ctrl+k", "alt+<", "ctrl+y"}, "two threeone ", 9},
		{"ctrl+y yanks what ctrl+u killed", "one two three", 8, []string{"ctrl+u", "alt+>", "ctrl+y"}, "threeone two ", 13},
		{"ctrl+y yanks what alt+d killed", "one two three", 3, []string{"alt+d", "alt+>", "ctrl+y"}, "one three two", 13},
		{"ctrl+y yanks what alt+backspace killed", "one two", 7, []string{"alt+backspace", "alt+<", "ctrl+y"}, "twoone ", 3},
		{"kills in a row yank as one", "one two three", 13, []string{"ctrl+w", "ctrl+w", "ctrl+y"}, "one two three", 13},
		{"a kill after a motion replaces the last", "one two three", 13, []string{"ctrl+w", "ctrl+b", "ctrl+w", "alt+>", "ctrl+y"}, "one  two", 8},
		{"ctrl+y with nothing killed does nothing", "one", 3, []string{"ctrl+y"}, "one", 3},
		{"ctrl+_ undoes a kill and puts the cursor back", "one two three", 4, []string{"ctrl+k", "alt+<", "ctrl+_"}, "one two three", 4},
		{"ctrl+/ undoes too", "one two", 7, []string{"ctrl+w", "ctrl+/"}, "one two", 7},
		{"ctrl+shift+- undoes too", "one two", 7, []string{"ctrl+w", "ctrl+shift+-"}, "one two", 7},
		{"undo takes back a typed word", "", 0, append(typedKeys("foo bar"), "ctrl+_"), "foo ", 4},
		{"undo takes back each typed word", "", 0, append(typedKeys("foo bar"), "ctrl+_", "ctrl+_"), "", 0},
		{"undo takes back a run of backspaces", "abc", 3, []string{"backspace", "backspace", "backspace", "ctrl+_"}, "abc", 3},
		{"a motion ends a typed run", "", 0, append(typedKeys("ab"), "ctrl+b", "x", "ctrl+_"), "ab", 1},
		{"undo takes back a yank", "one", 3, []string{"ctrl+w", "ctrl+y", "ctrl+y", "ctrl+_"}, "one", 3},
		{"undo takes back a transpose", "abc", 3, []string{"ctrl+t", "ctrl+_"}, "abc", 3},
		{"undo takes back a case change", "one", 0, []string{"alt+u", "ctrl+_"}, "one", 0},
		{"undo with nothing to undo does nothing", "one", 1, []string{"ctrl+_"}, "one", 1},
	}
	for _, kind := range fieldKinds() {
		for _, c := range cases {
			f := kind()
			t.Run(f.name+"/"+c.name, func(t *testing.T) {
				freshKillRing(t)
				buf := f.start(c.value, c.cursor)
				f.press(t, c.keys...)
				got := buf.snap()
				assert.Equal(t, c.want, got.value)
				assert.Equal(t, c.wantCol, got.col)
			})
		}
	}
}

func TestFieldUndoIsCappedAndDropsTheOldest(t *testing.T) {
	f := newTextInput()
	f.Focus()
	for i := 0; i < fieldUndoDepth+5; i++ {
		f, _ = f.Update(fieldKey(t, "a"))
		f, _ = f.Update(fieldKey(t, "ctrl+b"))
		f, _ = f.Update(fieldKey(t, "ctrl+e"))
	}
	assert.Len(t, f.edits.undo, fieldUndoDepth)
	for range fieldUndoDepth {
		f, _ = f.Update(fieldKey(t, "ctrl+_"))
	}
	assert.Equal(t, strings.Repeat("a", 5), f.Value())
}

func TestAreaFieldUndoRestoresTheCursorLine(t *testing.T) {
	freshKillRing(t)
	ta := textarea.New()
	ta.SetWidth(80)
	ta.Focus()
	f := areaField{Model: ta}
	f.restore(fieldSnap{value: "first line\nsecond line\nthird", line: 1, col: 7})
	require.Equal(t, 1, f.Line())
	require.Equal(t, 7, f.Column())

	for _, k := range []string{"ctrl+k", "alt+<", "ctrl+_"} {
		f, _ = f.Update(fieldKey(t, k))
	}
	assert.Equal(t, "first line\nsecond line\nthird", f.Value())
	assert.Equal(t, 1, f.Line())
	assert.Equal(t, 7, f.Column())
}

func TestAreaFieldLineKeysStayOnTheCursorLine(t *testing.T) {
	ta := textarea.New()
	ta.SetWidth(80)
	ta.Focus()
	f := areaField{Model: ta}
	f.restore(fieldSnap{value: "one two\nthree four", line: 1, col: 10})

	f, _ = f.Update(fieldKey(t, "alt+t"))
	assert.Equal(t, "one two\nfour three", f.Value())
	f, _ = f.Update(fieldKey(t, "ctrl+a"))
	f, _ = f.Update(fieldKey(t, "ctrl+t"))
	assert.Equal(t, "one two\nfour three", f.Value(), "ctrl+t at a line's start doesn't reach the line above")
}

func TestAKillInOneFieldYanksInAnother(t *testing.T) {
	freshKillRing(t)
	m := newTestModel(t, sampleProject())
	selectFirstTask(m)
	m.startEditing()
	require.True(t, m.ui.Modes.IsEdit())
	m.ui.Edit.input.SetValue("keep this word")
	m.ui.Edit.input.CursorEnd()
	m.handleKeyMsg(fieldKey(t, "ctrl+w"))
	m.cancelEditing()

	m.startSearch()
	require.True(t, m.ui.Modes.IsSearch())
	m.handleKeyMsg(fieldKey(t, "ctrl+y"))
	assert.Equal(t, "word", m.ui.Search.input.Value())
}

func TestTitleEditUndoesThroughTheKeyHandler(t *testing.T) {
	freshKillRing(t)
	m := newTestModel(t, sampleProject())
	selectFirstTask(m)
	m.startEditing()
	require.True(t, m.ui.Modes.IsEdit())
	m.ui.Edit.input.SetValue("buy milk today")
	m.ui.Edit.input.SetCursor(8)

	m.handleKeyMsg(fieldKey(t, "ctrl+k"))
	assert.Equal(t, "buy milk", m.ui.Edit.input.Value())
	m.handleKeyMsg(fieldKey(t, "ctrl+_"))
	assert.Equal(t, "buy milk today", m.ui.Edit.input.Value())
	assert.Equal(t, 8, m.ui.Edit.input.Position())
}

func TestDescriptionEditUndoesThroughTheKeyHandler(t *testing.T) {
	freshKillRing(t)
	p := sampleProject()
	p.Categories[0].Tasks[0].Description = "line one\nline two"
	m := newTestModel(t, p)
	selectFirstTask(m)
	require.NotNil(t, m.startDescriptionInlineEdit(0, 0))

	m.handleKeyMsg(fieldKey(t, "ctrl+w"))
	assert.Equal(t, "line one\nline ", m.ui.DescriptionEdit.textarea.Value())
	m.handleKeyMsg(fieldKey(t, "ctrl+a"))
	m.handleKeyMsg(fieldKey(t, "ctrl+y"))
	assert.Equal(t, "line one\ntwoline ", m.ui.DescriptionEdit.textarea.Value())
	m.handleKeyMsg(fieldKey(t, "ctrl+_"))
	m.handleKeyMsg(fieldKey(t, "ctrl+_"))
	assert.Equal(t, "line one\nline two", m.ui.DescriptionEdit.textarea.Value())
	assert.Equal(t, 1, m.ui.DescriptionEdit.textarea.Line())
	assert.Equal(t, 8, m.ui.DescriptionEdit.textarea.Column())
}

func TestHelpFilterKeepsItsPagingKeys(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Modes.ToHelp()
	m.startHelpFilter()
	for _, k := range typedKeys("a") {
		m.handleKeyMsg(fieldKey(t, k))
	}
	m.handleKeyMsg(fieldKey(t, "ctrl+d"))
	assert.Equal(t, "a", m.ui.Help.Filter.Value(), "ctrl+d pages instead of deleting")
	m.handleKeyMsg(fieldKey(t, "ctrl+u"))
	assert.Equal(t, "a", m.ui.Help.Filter.Value(), "ctrl+u pages instead of killing")
	m.handleKeyMsg(fieldKey(t, "ctrl+w"))
	assert.Equal(t, "", m.ui.Help.Filter.Value())
	m.handleKeyMsg(fieldKey(t, "ctrl+_"))
	assert.Equal(t, "a", m.ui.Help.Filter.Value())
}
