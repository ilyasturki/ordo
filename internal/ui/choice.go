package ui

import (
	"strings"
	"unicode"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// ChoiceOption is one row of a Choice. Key is the mnemonic underlined in the
// label; Aliases pick the row too without being shown.
type ChoiceOption struct {
	Label    string
	Key      string
	Aliases  []string
	Detail   string
	Danger   bool
	Disabled bool
	// Match highlights a filter's query in the label, in place of the mnemonic.
	Match string
	// Lead draws a styled mark before the label, given whether the row is
	// selected; its plain width is LeadWidth.
	Lead      func(selected bool) string
	LeadWidth int
}

// Choice is the one dialog every chooser and confirm draws: ↑/↓ and j/k move,
// enter picks, esc and q cancel, a mnemonic or (Numbered) a digit picks at once.
type Choice struct {
	Title    string
	Body     []string
	Options  []ChoiceOption
	Selected int
	Numbered bool
	// Note is muted text under the rows, about the selected one.
	Note []string
}

type ChoiceAction int

const (
	ChoiceUnhandled ChoiceAction = iota
	ChoiceMoved
	ChoicePicked
	ChoiceCancelled
)

// ChoiceReservedKeys move or cancel in every Choice, so none is a mnemonic.
var ChoiceReservedKeys = []string{"j", "k", "q", "g", "G"}

// Handle applies key and reports what it did; on ChoicePicked the int is the
// picked option. A key it doesn't use is ChoiceUnhandled, left to the caller.
func (c *Choice) Handle(key string) (ChoiceAction, int) {
	switch key {
	case "esc", "q":
		return ChoiceCancelled, -1
	case "down", "j", "ctrl+n", "tab":
		c.move(1)
		return ChoiceMoved, c.Selected
	case "up", "k", "ctrl+p", "shift+tab":
		c.move(-1)
		return ChoiceMoved, c.Selected
	case "home", "g":
		c.Selected = -1
		c.move(1)
		return ChoiceMoved, c.Selected
	case "end", "G":
		c.Selected = len(c.Options)
		c.move(-1)
		return ChoiceMoved, c.Selected
	case "enter":
		if c.enabled(c.Selected) {
			return ChoicePicked, c.Selected
		}
		return ChoiceMoved, c.Selected
	}
	if i := c.digitIndex(key); i >= 0 {
		if c.enabled(i) {
			c.Selected = i
			return ChoicePicked, i
		}
		return ChoiceMoved, c.Selected
	}
	for i, o := range c.Options {
		if o.Key == key || containsKey(o.Aliases, key) {
			if !o.Disabled {
				c.Selected = i
				return ChoicePicked, i
			}
			return ChoiceMoved, c.Selected
		}
	}
	return ChoiceUnhandled, -1
}

func containsKey(keys []string, key string) bool {
	for _, k := range keys {
		if k == key {
			return true
		}
	}
	return false
}

// digitIndex maps 1–9 to the first nine rows and 0 to the tenth.
func (c Choice) digitIndex(key string) int {
	if !c.Numbered || len(key) != 1 || key[0] < '0' || key[0] > '9' {
		return -1
	}
	i := int(key[0]-'0') - 1
	if key == "0" {
		i = 9
	}
	if i >= len(c.Options) {
		return -1
	}
	return i
}

func (c Choice) enabled(i int) bool {
	return i >= 0 && i < len(c.Options) && !c.Options[i].Disabled
}

// move steps over disabled rows and stays put when none is left that way.
func (c *Choice) move(delta int) {
	for i := c.Selected + delta; i >= 0 && i < len(c.Options); i += delta {
		if !c.Options[i].Disabled {
			c.Selected = i
			return
		}
	}
}

// Clamp keeps Selected on an enabled row once the options change.
func (c *Choice) Clamp() {
	if c.enabled(c.Selected) {
		return
	}
	at := min(max(c.Selected, 0), len(c.Options))
	c.Selected = at - 1
	c.move(1)
	if !c.enabled(c.Selected) {
		c.Selected = at
		c.move(-1)
	}
}

// View draws the title, the body and every row width cells wide, unframed.
func (c Choice) View(width int, focused bool) string {
	lines := c.Head(width)
	for i := range c.Options {
		lines = append(lines, c.Row(i, width, focused))
	}
	if len(c.Note) > 0 {
		lines = append(lines, "")
		for _, n := range c.Note {
			lines = append(lines, MutedStyle.Render(wrapTo(n, width)))
		}
	}
	return strings.Join(lines, "\n")
}

// Head is the bold title and the muted body, wrapped to width, then a blank row.
func (c Choice) Head(width int) []string {
	var lines []string
	if c.Title != "" {
		lines = append(lines, strings.Split(DialogTitleStyle.Render(wrapTo(c.Title, width)), "\n")...)
		lines = append(lines, "")
	}
	for _, b := range c.Body {
		lines = append(lines, strings.Split(wrapTo(b, width), "\n")...)
	}
	if len(c.Body) > 0 {
		lines = append(lines, "")
	}
	return lines
}

func wrapTo(s string, width int) string {
	if width <= 0 {
		return s
	}
	return ansi.Wrap(s, width, "")
}

// Row draws option i: a full-width band when selected, red when it is the
// danger choice, its mnemonic underlined.
func (c Choice) Row(i, width int, focused bool) string {
	o := c.Options[i]
	selected := i == c.Selected
	base := lipgloss.NewStyle()
	switch {
	case selected && o.Danger:
		base = lipgloss.NewStyle().Bold(true).Foreground(dangerColor).Reverse(true).Faint(!focused)
	case selected:
		base = GetSelectedStyle(focused)
	case o.Disabled:
		base = MutedStyle
	case o.Danger:
		base = lipgloss.NewStyle().Foreground(dangerColor)
	}
	key := base.Underline(true)
	if !selected && !o.Disabled {
		key = DialogKey(key, false)
	}

	var b strings.Builder
	b.WriteString(base.Render("  "))
	used := 2
	if c.Numbered {
		digit := " "
		if i < 9 {
			digit = string(rune('1' + i))
		} else if i == 9 {
			digit = "0"
		}
		ds := base
		if !selected && !o.Disabled {
			ds = DialogKey(base, false)
		}
		b.WriteString(ds.Render(digit) + base.Render("  "))
		used += 3
	}
	if o.Lead != nil {
		b.WriteString(o.Lead(selected))
		used += o.LeadWidth
	}

	detail := ""
	if o.Detail != "" {
		detail = "  " + o.Detail + "  "
	}
	room := width - used - 2
	if detail != "" && room-ansi.StringWidth(detail)+2 < ansi.StringWidth(o.Label) {
		detail = ""
	}
	if detail != "" {
		room = width - used - ansi.StringWidth(detail)
	}
	if o.Match != "" && !selected {
		b.WriteString(HighlightMatches(ansi.Truncate(o.Label, max(room, 1), Ellipsis), o.Match, base, SearchMatchStyle))
	} else {
		b.WriteString(mnemonicLabel(o.Label, o.Key, max(room, 1), base, key))
	}

	line := b.String()
	if detail != "" {
		gap := width - ansi.StringWidth(line) - ansi.StringWidth(detail)
		line += base.Render(strings.Repeat(" ", max(gap, 0)))
		ds := base
		if !selected {
			ds = MutedStyle
		}
		return line + ds.Render(strings.TrimRight(detail, " ")) + base.Render("  ")
	}
	return PadToStyle(line, width, base)
}

var dangerColor = lipgloss.Color("1")

// mnemonicLabel underlines the first rune of label matching key, ignoring case,
// after truncating label to room cells.
func mnemonicLabel(label, key string, room int, base, keyStyle lipgloss.Style) string {
	label = ansi.Truncate(label, room, Ellipsis)
	at := -1
	if r := []rune(key); len(r) == 1 {
		k := unicode.ToLower(r[0])
		for i, lr := range []rune(label) {
			if unicode.ToLower(lr) == k {
				at = i
				break
			}
		}
	}
	runes := []rune(label)
	if at < 0 {
		return base.Render(label)
	}
	return base.Render(string(runes[:at])) + keyStyle.Render(string(runes[at])) + base.Render(string(runes[at+1:]))
}
