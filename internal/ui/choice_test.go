package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
)

func deleteChoice() Choice {
	return Choice{
		Title: "Delete task?",
		Options: []ChoiceOption{
			{Label: "Delete", Key: "d", Aliases: []string{"y"}, Danger: true},
			{Label: "Keep subtasks", Key: "s"},
			{Label: "Cancel", Key: "c", Aliases: []string{"n"}},
		},
	}
}

func TestChoice_ArrowsAndJKMoveWithinBounds(t *testing.T) {
	c := deleteChoice()
	for _, key := range []string{"down", "j", "ctrl+n"} {
		c.Selected = 0
		act, i := c.Handle(key)
		assert.Equal(t, ChoiceMoved, act, key)
		assert.Equal(t, 1, i, key)
	}
	c.Selected = 2
	c.Handle("down")
	assert.Equal(t, 2, c.Selected, "the last row stays the last")
	for _, key := range []string{"up", "k", "ctrl+p"} {
		c.Selected = 2
		c.Handle(key)
		assert.Equal(t, 1, c.Selected, key)
	}
	c.Handle("g")
	assert.Equal(t, 0, c.Selected)
	c.Handle("G")
	assert.Equal(t, 2, c.Selected)
	c.Handle("home")
	assert.Equal(t, 0, c.Selected)
	c.Handle("end")
	assert.Equal(t, 2, c.Selected)
}

func TestChoice_EnterPicksTheSelectedRow(t *testing.T) {
	c := deleteChoice()
	c.Selected = 1
	act, i := c.Handle("enter")
	assert.Equal(t, ChoicePicked, act)
	assert.Equal(t, 1, i)
}

func TestChoice_EscAndQCancel(t *testing.T) {
	for _, key := range []string{"esc", "q"} {
		c := deleteChoice()
		act, _ := c.Handle(key)
		assert.Equal(t, ChoiceCancelled, act, key)
	}
}

func TestChoice_MnemonicsAndAliasesPickAtOnce(t *testing.T) {
	for key, want := range map[string]int{"d": 0, "y": 0, "s": 1, "c": 2, "n": 2} {
		c := deleteChoice()
		act, i := c.Handle(key)
		assert.Equal(t, ChoicePicked, act, key)
		assert.Equal(t, want, i, key)
		assert.Equal(t, want, c.Selected, key)
	}
}

func TestChoice_MnemonicsMatchCase(t *testing.T) {
	c := Choice{Options: []ChoiceOption{{Label: "Mine", Key: "m"}, {Label: "All mine", Key: "a", Aliases: []string{"M"}}}}
	_, i := c.Handle("M")
	assert.Equal(t, 1, i)
	_, i = c.Handle("m")
	assert.Equal(t, 0, i)
}

func TestChoice_UnknownKeysAreLeftToTheCaller(t *testing.T) {
	c := deleteChoice()
	act, _ := c.Handle("space")
	assert.Equal(t, ChoiceUnhandled, act)
	act, _ = c.Handle("1")
	assert.Equal(t, ChoiceUnhandled, act, "digits pick only in a numbered choice")
}

func TestChoice_DigitsPickByPosition(t *testing.T) {
	c := Choice{Numbered: true}
	for range 10 {
		c.Options = append(c.Options, ChoiceOption{Label: "x"})
	}
	act, i := c.Handle("3")
	assert.Equal(t, ChoicePicked, act)
	assert.Equal(t, 2, i)
	_, i = c.Handle("0")
	assert.Equal(t, 9, i, "0 is the tenth row")
	c.Options = c.Options[:4]
	act, _ = c.Handle("7")
	assert.Equal(t, ChoiceUnhandled, act, "a digit past the rows does nothing")
}

func TestChoice_DisabledRowsAreSkippedAndNeverPicked(t *testing.T) {
	c := Choice{Options: []ChoiceOption{{Label: "Mine", Key: "m"}, {Label: "Both", Key: "b", Disabled: true}, {Label: "Theirs", Key: "t"}}}
	c.Handle("down")
	assert.Equal(t, 2, c.Selected, "down steps over the disabled row")
	act, _ := c.Handle("b")
	assert.NotEqual(t, ChoicePicked, act)
	c.Selected = 1
	c.Clamp()
	assert.NotEqual(t, 1, c.Selected)
}

func TestChoice_RowsAreTheFixedWidth(t *testing.T) {
	c := deleteChoice()
	c.Options[1].Detail = "current"
	for _, line := range strings.Split(c.View(40, true), "\n")[2:] {
		assert.Equal(t, 40, ansi.StringWidth(line), line)
	}
}

func TestChoice_ViewShowsNoFooter(t *testing.T) {
	view := ansi.Strip(deleteChoice().View(40, true))
	lines := strings.Split(view, "\n")
	assert.Equal(t, "Delete task?", lines[0])
	assert.Equal(t, "Cancel", strings.TrimSpace(lines[len(lines)-1]), "the last row is the last option")
	assert.NotContains(t, view, "esc")
}

func TestChoice_MnemonicIsUnderlinedInItsLabel(t *testing.T) {
	c := deleteChoice()
	row := c.Row(1, 40, true)
	assert.Contains(t, row, "Keep ")
	idx := strings.Index(row, "Keep ")
	assert.Contains(t, row[idx:], "\x1b[4", "the s of subtasks is underlined")
	assert.Equal(t, "  Keep subtasks", strings.TrimRight(ansi.Strip(row), " "))
}

func TestChoice_LongLabelsAreCutToTheWidth(t *testing.T) {
	c := Choice{Options: []ChoiceOption{{Label: strings.Repeat("long ", 30), Key: "l"}}}
	assert.Equal(t, 30, ansi.StringWidth(c.Row(0, 30, true)))
}

func TestChoice_NumberedRowsLeadWithTheirDigit(t *testing.T) {
	c := Choice{Numbered: true, Options: []ChoiceOption{{Label: "Feature"}, {Label: "Fix"}}}
	assert.Equal(t, "  2  Fix", strings.TrimRight(ansi.Strip(c.Row(1, 30, true)), " "))
}
