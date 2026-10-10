package domain

import (
	"errors"
	"strings"
	"unicode/utf8"
)

// Text limits and validation.
//
// Every string a user stores here eventually reaches a terminal — the TUI
// renders it and the CLI prints it. A terminal treats
// some byte sequences as commands rather than text: ESC (0x1B) opens an escape
// sequence, and in UTF-8 mode U+009B acts as a bare CSI. A title carrying those
// bytes is not data the terminal displays, it is instructions the terminal
// obeys — OSC 52, for instance, writes the reader's clipboard.
//
// So control characters are rejected at the write boundary rather than escaped
// at each sink: one gate the CLI, TUI and import all pass through beats
// several that each have to stay correct forever.

// Length is deliberately not one of the limits: a single-line field may be as
// long as the user cares to type. What a long one must not do is push the
// screen out of reach, and that is bounded where it arises — the list renders
// such a field into at most ui.MaxLineRows rows, whatever its length.

var (
	// ErrControlCharacters is returned when text carries control characters a
	// terminal would interpret rather than display.
	ErrControlCharacters = errors.New("text must not contain control characters")

	// ErrInvalidUTF8 is returned when text is not valid UTF-8. Invalid bytes
	// reach the terminal undecoded, so they get the same treatment.
	ErrInvalidUTF8 = errors.New("text must be valid UTF-8")
)

// isControl reports whether r is a control character: C0 (below space), DEL, or
// C1 (U+0080–U+009F). Newline and tab are C0 but are handled by the callers
// that allow them, so they are not special-cased here.
func isControl(r rune) bool {
	return r < 0x20 || r == 0x7F || (r >= 0x80 && r <= 0x9F)
}

// ValidateLine checks a single-line field: valid UTF-8 and no control
// characters at all (a title has no legitimate use for a newline or a tab).
// Length is unbounded.
func ValidateLine(s string) error {
	if !utf8.ValidString(s) {
		return ErrInvalidUTF8
	}
	for _, r := range s {
		if isControl(r) {
			return ErrControlCharacters
		}
	}
	return nil
}

// ValidateMultiline checks a field that may span lines (a task description).
// Newline and tab are legitimate here — interior newlines are preserved by
// design — so only the remaining control characters are rejected.
func ValidateMultiline(s string) error {
	if !utf8.ValidString(s) {
		return ErrInvalidUTF8
	}
	for _, r := range s {
		if r == '\n' || r == '\t' {
			continue
		}
		if isControl(r) {
			return ErrControlCharacters
		}
	}
	return nil
}

// StripControl removes control characters from s, keeping newline and tab when
// allowNewlines is set. Invalid UTF-8 sequences are dropped by the range loop's
// decoding, which yields RuneError for them.
//
// This is the repair path for data that predates validation, not a substitute
// for it: writes reject, loads strip. A project written before these checks
// existed still opens, just without the bytes that would drive the terminal.
//
// Note that this defangs escape sequences rather than deleting them: only the
// control byte is removed, so "\x1b[2J" becomes the literal text "[2J". That is
// deliberate. Recognising whole sequences would need a full ANSI parser, and
// getting that parser subtly wrong is how this class of bug survives — whereas
// removing the byte that makes the terminal listen cannot be partially correct.
// The leftover is ugly, and inert.
func StripControl(s string, allowNewlines bool) string {
	if !needsStrip(s, allowNewlines) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for i, r := range s {
		// range yields RuneError both for an undecodable byte and for a real
		// U+FFFD; only the former decodes as a single byte, so the width is what
		// tells them apart. Dropping a legitimate U+FFFD would corrupt text that
		// was never dangerous.
		if r == utf8.RuneError {
			if _, size := utf8.DecodeRuneInString(s[i:]); size <= 1 {
				continue
			}
		}
		if allowNewlines && (r == '\n' || r == '\t') {
			b.WriteRune(r)
			continue
		}
		if isControl(r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// ValidateProjectText checks every string a project carries.
//
// This is the import gate. A project file received from someone else is the one
// genuinely untrusted input this tool accepts, and unlike an interactive write
// it arrives fully formed — bypassing the operations layer where the per-field
// checks live. So it is validated wholesale here instead, and rejected rather
// than repaired: a file that fails is one the user should know about.
//
// Every string field is checked, not just the obvious ones. Timestamps, enum
// fields and IDs all reach a terminal somewhere (`--long` prints IDs, the TUI
// renders dates), and a field being enum-like in normal use is no guarantee
// about what a hand-written file puts there.
func ValidateProjectText(p Project) error {
	lines := []string{p.ID, p.Name, p.CreatedAt, p.UpdatedAt}
	for _, c := range p.Categories {
		lines = append(lines, c.ID, c.Name, c.ParentID, c.CreatedAt, c.UpdatedAt)
		for _, t := range c.Tasks {
			lines = append(lines,
				t.ID, t.ParentID, t.Title, t.Status, t.CreatedAt, t.UpdatedAt,
				t.Priority, t.CompletionDate, t.Kind, t.TagColor,
			)
			if err := ValidateMultiline(t.Description); err != nil {
				return err
			}
		}
	}
	for _, s := range lines {
		if err := ValidateLine(s); err != nil {
			return err
		}
	}
	return nil
}

// StripProjectText removes control characters from every string a project
// carries, in place.
//
// This is the load path's counterpart to ValidateProjectText: projects written
// before these checks existed, or edited by hand, still open — just without the
// bytes that would drive the terminal. Nothing here truncates: a control
// character was never legitimate content, whereas the text around it is.
func StripProjectText(p *Project) {
	p.ID = StripControl(p.ID, false)
	p.Name = StripControl(p.Name, false)
	p.CreatedAt = StripControl(p.CreatedAt, false)
	p.UpdatedAt = StripControl(p.UpdatedAt, false)
	for i := range p.Categories {
		c := &p.Categories[i]
		c.ID = StripControl(c.ID, false)
		c.Name = StripControl(c.Name, false)
		c.ParentID = StripControl(c.ParentID, false)
		c.CreatedAt = StripControl(c.CreatedAt, false)
		c.UpdatedAt = StripControl(c.UpdatedAt, false)
		for j := range c.Tasks {
			t := &c.Tasks[j]
			t.ID = StripControl(t.ID, false)
			t.ParentID = StripControl(t.ParentID, false)
			t.Title = StripControl(t.Title, false)
			t.Status = StripControl(t.Status, false)
			t.CreatedAt = StripControl(t.CreatedAt, false)
			t.UpdatedAt = StripControl(t.UpdatedAt, false)
			t.Priority = StripControl(t.Priority, false)
			t.CompletionDate = StripControl(t.CompletionDate, false)
			t.Kind = StripControl(t.Kind, false)
			t.TagColor = StripControl(t.TagColor, false)
			t.Description = StripControl(t.Description, true)
		}
	}
}

// needsStrip reports whether s contains anything StripControl would remove, so
// the common case (clean text) returns the original string without allocating.
func needsStrip(s string, allowNewlines bool) bool {
	if !utf8.ValidString(s) {
		return true
	}
	for _, r := range s {
		if allowNewlines && (r == '\n' || r == '\t') {
			continue
		}
		if isControl(r) {
			return true
		}
	}
	return false
}
