package ui

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

var (
	// Selection styles use Reverse so the terminal swaps fg/bg natively at
	// render time — this tracks live light/dark theme changes, which
	// lipgloss.AdaptiveColor cannot (background detection is cached once at
	// startup via sync.Once and never re-queried).
	HeaderStyle   = lipgloss.NewStyle().Bold(true)
	MutedStyle    = lipgloss.NewStyle().Faint(true)
	CategoryStyle = lipgloss.NewStyle().Bold(true)
	// SeparatorStyle draws in-category divider rules at full foreground
	// contrast (an empty style = the terminal's default foreground).
	SeparatorStyle   = lipgloss.NewStyle()
	SelectedStyle    = lipgloss.NewStyle().Bold(true).Reverse(true)
	StatusLineStyle  = lipgloss.NewStyle().Faint(true)
	HelpDialogStyle  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2)
	DialogTitleStyle = lipgloss.NewStyle().Bold(true)
	DialogHintStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	SuccessStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	FilterTagStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true)
	VisualTagStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("0")).Background(lipgloss.Color("3")).Bold(true)

	// VisualSelectedStyle highlights rows that are part of an active visual
	// range. Distinct from the reverse-based cursor style so the user can
	// tell at a glance they're in visual mode.
	VisualSelectedStyle          = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("0")).Background(lipgloss.Color("3"))
	UnfocusedVisualSelectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("0")).Background(lipgloss.Color("3")).Faint(true)

	// VisualCursorStyle marks the cursor row inside an active visual range.
	// Reverse swaps fg/bg natively so the cursor reads as a distinct block
	// against the yellow range band — same visual language as the normal-mode
	// cursor, which makes it obvious which end `j`/`k` extends and which end
	// `o` swaps to.
	VisualCursorStyle          = lipgloss.NewStyle().Bold(true).Reverse(true)
	UnfocusedVisualCursorStyle = lipgloss.NewStyle().Bold(true).Reverse(true).Faint(true)

	// CutBadgeStyle marks rows whose item is staged for cut/paste.
	CutBadgeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true)

	// Search highlight: matched substrings during `/` search read as black on
	// yellow — the conventional search-highlight look, legible on both light and
	// dark terminals (the ANSI palette tracks the active theme). The current
	// match adds bold + underline so it stands out from the other matches and
	// from the cursor's reverse band.
	SearchMatchStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("0")).Background(lipgloss.Color("3"))
	SearchCurrentMatchStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("0")).Background(lipgloss.Color("3")).Bold(true).Underline(true)

	// Description text mirrors its task title's color treatment (see
	// TaskTitleStyle) with italic added; the faint left bar glyph gives it a
	// markdown-blockquote look that keeps it distinct from the title row.
	DescriptionBarStyle = lipgloss.NewStyle().Faint(true)

	// Shortcut bar (lazygit-style footer): keys read brighter, labels read
	// quieter so the eye can hop between them. Faint avoids any explicit
	// color, which keeps the bar usable on both light and dark terminals.
	ShortcutKeyStyle   = lipgloss.NewStyle().Bold(true)
	ShortcutLabelStyle = lipgloss.NewStyle().Faint(true)
	ShortcutSepStyle   = lipgloss.NewStyle().Faint(true)

	// A frameless dialog: Padding(2,3) equals HelpDialogStyle's border plus
	// padding, so a panel fills the same rectangle.
	PanelStyle = lipgloss.NewStyle().Padding(2, 3)
	// The same panel owning the whole terminal: no bottom padding, so the footer
	// rests on the last row.
	FullPanelStyle = lipgloss.NewStyle().Padding(1, 0, 0, 2)
)

// ANSI 0–6 so it tracks the terminal theme; 16–255 and hex do not.
var dialogKeyColor = lipgloss.Color("6")

// Focused rows are Reverse; a foreground color inside that band renders as a
// pasted block.
func DialogKey(base lipgloss.Style, focused bool) lipgloss.Style {
	if focused {
		return base
	}
	return base.Foreground(dialogKeyColor)
}

// CutMark is the badge appended to rows whose item is pending a cut/paste.
const CutMark = " ✂"

// ApplyCut decorates a style so cut rows read as "ghosted/in flight": faint
// plus italic so the differentiation survives even when the row is also
// selected (which inverts colors) or completed (which is already faint).
func ApplyCut(s lipgloss.Style) lipgloss.Style {
	return s.Faint(true).Italic(true)
}

func GetVisualSelectedStyle(focused bool) lipgloss.Style {
	if focused {
		return VisualSelectedStyle
	}
	return UnfocusedVisualSelectedStyle
}

func GetVisualCursorStyle(focused bool) lipgloss.Style {
	if focused {
		return VisualCursorStyle
	}
	return UnfocusedVisualCursorStyle
}

func StatusStyle(status string) lipgloss.Style {
	switch status {
	case "in_progress":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("4"))
	case "completed":
		return lipgloss.NewStyle().Faint(true)
	case "cancelled":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	}
}

func priorityColor(priority string) (color.Color, bool) {
	switch priority {
	case "critical", "high":
		return lipgloss.Color("1"), true
	case "low", "trivial":
		return lipgloss.Color("6"), true
	default:
		return nil, false
	}
}

// tagColor maps a stored tag color name to a basic ANSI color. Like the
// priority palette these are 0–6 so the terminal keeps them legible on both
// light and dark themes (ANSI 16–255 and hex do not track the theme).
func tagColor(name string) (color.Color, bool) {
	switch name {
	case "green":
		return lipgloss.Color("2"), true
	case "blue":
		return lipgloss.Color("4"), true
	case "magenta":
		return lipgloss.Color("5"), true
	case "cyan":
		return lipgloss.Color("6"), true
	default:
		return nil, false
	}
}

// TagDot is the glyph drawn before a tagged task's title.
const TagDot = "●"

// TagDotVisible reports whether name is a renderable tag color — i.e. whether a
// dot should be drawn (and its column reserved) at all.
func TagDotVisible(name string) bool {
	_, ok := tagColor(name)
	return ok
}

// TagDotStyle returns the style for the tag dot and whether it renders. faint
// mirrors the priority icon's completed/cancelled dimming.
func TagDotStyle(name string, faint bool) (lipgloss.Style, bool) {
	c, ok := tagColor(name)
	if !ok {
		return lipgloss.NewStyle(), false
	}
	s := lipgloss.NewStyle().Foreground(c)
	if faint {
		s = s.Faint(true)
	}
	return s, true
}

// TagBlockStyle returns the style for a tag rendered as a filled color block on
// a highlighted row: Reverse turns the tag color into the cell background so the
// tag reads as part of the reversed selection bar instead of a clashing
// foreground island. Reverse-based so it tracks the terminal theme like the rest
// of the selection.
func TagBlockStyle(name string) (lipgloss.Style, bool) {
	c, ok := tagColor(name)
	if !ok {
		return lipgloss.NewStyle(), false
	}
	return lipgloss.NewStyle().Foreground(c).Reverse(true), true
}

// TagSegmentText is the plain leading tag text drawn before a task title: the
// dot, an optional label, and a trailing space — or "" when untagged. Both the
// renderer and the layout/column math call this so the two never drift.
func TagSegmentText(name, label string) string {
	if !TagDotVisible(name) {
		return ""
	}
	if label != "" {
		return TagDot + " " + label + " "
	}
	return TagDot + " "
}

// PriorityStyle returns the text style for a task title with the given
// priority. Color is applied only when colorMode is "full".
func PriorityStyle(priority, colorMode string) lipgloss.Style {
	if colorMode != "full" {
		return lipgloss.NewStyle()
	}
	if c, ok := priorityColor(priority); ok {
		return lipgloss.NewStyle().Foreground(c)
	}
	return lipgloss.NewStyle()
}

// PriorityIconStyle returns the style for the priority icon. Color is applied
// when colorMode is "full" or "icon".
func PriorityIconStyle(priority, colorMode string) lipgloss.Style {
	if colorMode != "full" && colorMode != "icon" {
		return lipgloss.NewStyle()
	}
	if c, ok := priorityColor(priority); ok {
		return lipgloss.NewStyle().Foreground(c)
	}
	return lipgloss.NewStyle()
}

// PriorityIconBlockStyle returns the style for the priority icon rendered as a
// filled color block on a highlighted row, mirroring TagBlockStyle: Reverse
// turns the priority color into the cell background so the icon keeps its color
// as part of the reversed selection bar instead of vanishing into it. Color is
// applied when colorMode is "full" or "icon"; the bool reports whether a
// colored block renders at all.
func PriorityIconBlockStyle(priority, colorMode string) (lipgloss.Style, bool) {
	if colorMode != "full" && colorMode != "icon" {
		return lipgloss.NewStyle(), false
	}
	if c, ok := priorityColor(priority); ok {
		return lipgloss.NewStyle().Foreground(c).Reverse(true), true
	}
	return lipgloss.NewStyle(), false
}

func TaskTitleStyle(priority, status, colorMode string) lipgloss.Style {
	base := PriorityStyle(priority, colorMode)
	if status == "completed" || status == "cancelled" {
		return base.Faint(true)
	}
	return base
}

// PriorityIcon returns the glyph drawn before a task title for the given
// priority. The critical/trivial glyphs are single-cell arrows (U+21C8/U+21CA)
// so every priority row aligns identically — unlike the double-triangle emoji
// (U+23EB/EC), whose column advance varies by terminal.
func PriorityIcon(priority string) string {
	switch priority {
	case "critical":
		return "⇈"
	case "high":
		return "▲"
	case "low":
		return "▼"
	case "trivial":
		return "⇊"
	default:
		return ""
	}
}

// Unfocused selection style (dimmed reverse — same visual language as focused,
// just quieter; lets the terminal swap fg/bg so it tracks light/dark themes).
var UnfocusedSelectedStyle = lipgloss.NewStyle().Bold(true).Reverse(true).Faint(true)

// Unfocused cursor style for edit mode (dimmed reverse)
var UnfocusedCursorStyle = lipgloss.NewStyle().Reverse(true).Faint(true)

func GetSelectedStyle(focused bool) lipgloss.Style {
	if focused {
		return SelectedStyle
	}
	return UnfocusedSelectedStyle
}

func GetSelectedStatusStyle(status string, focused bool) lipgloss.Style {
	return GetSelectedStyle(focused)
}

func GetSelectedPriorityStyle(priority string, focused bool) lipgloss.Style {
	return GetSelectedStyle(focused)
}

func GetCursorStyle(focused bool) lipgloss.Style {
	if focused {
		return SelectedStyle
	}
	return UnfocusedCursorStyle
}
