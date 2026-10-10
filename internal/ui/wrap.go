package ui

import (
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"
)

// MaxLineRows caps how many wrapped rows a single-line field — a task title, a
// category name — may occupy in the list.
//
// Titles carry no length limit, so the bound has to live here instead: it holds
// even on the focused row, so j/k past a long title never shifts the rows below.
const MaxLineRows = 3

// Ellipsis marks text elided at either end of a clamped or scrolled window,
// wherever one is drawn.
const Ellipsis = "…"

// WrapClamped wraps text to width columns and returns at most maxRows lines,
// marking elision with a leading and/or trailing ellipsis.
//
// focus is a byte offset the window should contain — where a search match starts
// (see FirstMatchIndex), or negative for "start at the beginning". A field long
// enough to be clamped has its window moved onto focus, so a match past the cap
// is still visible.
//
// Every step works in byte offsets and walks at most a screenful of runes, so
// cost is O(maxRows*width) rather than O(len(text)): a megabyte title wraps as
// fast as a short one. Callers pass plain text and style the lines themselves.
func WrapClamped(text string, width, maxRows, focus int) []string {
	width = max(width, 1)
	maxRows = max(maxRows, 1)
	budget := maxRows * width

	start := 0
	if focus > 0 {
		// The latest a window may start and still fill every row — 0 for text
		// that already fits, where there is nothing to move the window onto.
		if latest := retreat(text, len(text), budget); latest > 0 {
			start = min(retreat(text, focus, width), latest)
			start = min(snapToWord(text, start, width), focus, latest)
		}
	}
	// A row of slack past the budget, so word breaks can't leave the last line
	// short of material.
	end := advance(text, start, budget+width)
	window := text[start:end]
	lines := strings.Split(ansi.Wrap(window, width, ""), "\n")

	// Word breaks make the window wrap to more rows than there is room for, so
	// pick which of them to show: the ones around focus, or the first maxRows.
	first := 0
	if len(lines) > maxRows && focus > start {
		lead := focusLine(window, lines, focus-start) - 1
		first = min(max(lead, 0), len(lines)-maxRows)
	}
	shown := lines[first:min(first+maxRows, len(lines))]

	if start > 0 || first > 0 {
		shown[0] = Ellipsis + ansi.Truncate(shown[0], width-1, "")
	}
	if end < len(text) || first+len(shown) < len(lines) {
		// Truncate to width-1 and append rather than passing the ellipsis as
		// Truncate's tail: a last line that happens to fill the row exactly is
		// not longer than the limit, so Truncate would leave it unmarked.
		i := len(shown) - 1
		shown[i] = ansi.Truncate(shown[i], width-1, "") + Ellipsis
	}
	return shown
}

// focusLine reports which of lines holds the byte at offset rel within window.
// It walks the wrap the way ansi.Wrap produced it — a line's bytes, then the
// spaces the break swallowed — so the answer is exact rather than estimated.
func focusLine(window string, lines []string, rel int) int {
	pos := 0
	for i, line := range lines {
		pos += len(line)
		if rel < pos {
			return i
		}
		for pos < len(window) && window[pos] == ' ' {
			pos++
		}
	}
	return len(lines) - 1
}

// CountClamped reports how many rows WrapClamped produces for text. The count
// does not depend on focus, so the layout can size a row without knowing where a
// search match sits inside it — which is what keeps the reservation and the
// render in agreement.
func CountClamped(text string, width, maxRows int) int {
	return len(WrapClamped(text, width, maxRows, -1))
}

// retreat returns the byte offset n runes before i.
func retreat(s string, i, n int) int {
	for range n {
		if i <= 0 {
			return 0
		}
		_, size := utf8.DecodeLastRuneInString(s[:i])
		i -= size
	}
	return i
}

// advance returns the byte offset n runes after i, or len(s) if the text ends
// first.
func advance(s string, i, n int) int {
	for range n {
		if i >= len(s) {
			return len(s)
		}
		_, size := utf8.DecodeRuneInString(s[i:])
		i += size
	}
	return i
}

// snapToWord moves a window's start forward to just past the next space within
// the following span bytes, so the window doesn't open mid-word. Returns i
// unchanged when there is no space that close.
func snapToWord(s string, i, span int) int {
	if j := strings.IndexByte(s[i:min(i+span, len(s))], ' '); j >= 0 {
		return i + j + 1
	}
	return i
}

type LineSpan struct {
	Text  string
	Start int // byte offset into the source
}

// WrapSpans wraps text to width columns and returns every row it takes, with no
// row cap. ansi.Wrap eats the whitespace it breaks on, so the rows do not
// concatenate back to text and each carries where it starts instead.
func WrapSpans(text string, width int) []LineSpan {
	width = max(width, 1)
	lines := strings.Split(ansi.Wrap(text, width, ""), "\n")
	spans := make([]LineSpan, 0, len(lines))
	pos := 0
	for _, line := range lines {
		spans = appendSplit(spans, LineSpan{Text: line, Start: pos}, width)
		pos += len(line)
		for pos < len(text) && strings.ContainsRune(" \t\r\n", rune(text[pos])) {
			pos++
		}
	}
	return spans
}

// appendSplit adds span, cut into rows that fit width: ansi.Wrap can leave an
// over-wide run of spaces on one row.
func appendSplit(spans []LineSpan, span LineSpan, width int) []LineSpan {
	if ansi.StringWidth(span.Text) <= width {
		return append(spans, span)
	}
	start, w := 0, 0
	for i, r := range span.Text {
		rw := ansi.StringWidth(string(r))
		if w+rw > width {
			spans = append(spans, LineSpan{Text: span.Text[start:i], Start: span.Start + start})
			start, w = i, 0
		}
		w += rw
	}
	return append(spans, LineSpan{Text: span.Text[start:], Start: span.Start + start})
}
