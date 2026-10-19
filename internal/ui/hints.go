package ui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// Hint is a single key→action pair shown in a footer hint line (dialog
// footer or the bottom shortcut bar).
type Hint struct {
	Key   string
	Label string
	// Active draws the hint in the filter color, for a mode that is on.
	Active bool
}

const hintSeparator = "  ·  "

// RenderHints styles each hint with a bold key and faint label, joined by a
// faint separator. Used for dialog footer rows where the full line always
// fits the dialog width — no truncation.
func RenderHints(hints []Hint) string {
	if len(hints) == 0 {
		return ""
	}
	parts := make([]string, len(hints))
	for i, h := range hints {
		parts[i] = formatHint(h)
	}
	return strings.Join(parts, ShortcutSepStyle.Render(hintSeparator))
}

// RenderHintsToWidth is RenderHints capped to width: trailing hints are
// dropped until the plain line fits. If even the first hint overflows, it's
// hard-truncated. Used by the bottom shortcut bar which lives outside any
// dialog and must fit the full terminal width.
func RenderHintsToWidth(hints []Hint, width int) string {
	if len(hints) == 0 {
		return ""
	}
	if width <= 0 {
		return RenderHints(hints)
	}

	plain := make([]string, len(hints))
	styled := make([]string, len(hints))
	for i, h := range hints {
		plain[i] = h.Key + " " + h.Label
		styled[i] = formatHint(h)
	}
	sep := ShortcutSepStyle.Render(hintSeparator)

	for n := len(plain); n > 0; n-- {
		if ansi.StringWidth(strings.Join(plain[:n], hintSeparator)) <= width {
			return strings.Join(styled[:n], sep)
		}
	}
	return ansi.Truncate(styled[0], width, "")
}

func formatHint(h Hint) string {
	if h.Active {
		return FilterTagStyle.Render(h.Key + " " + h.Label)
	}
	return ShortcutKeyStyle.Render(h.Key) + " " + ShortcutLabelStyle.Render(h.Label)
}
