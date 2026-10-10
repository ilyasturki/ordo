package domain

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The payloads here are the ones a terminal actually acts on, not arbitrary
// low bytes: OSC 52 writes the reader's clipboard, CSI moves the cursor and
// erases, and U+009B is a single-rune CSI that survives naive "strip ESC"
// filters. If a validator lets any of these through, watching a project list
// scroll past is enough to be attacked.
const (
	osc52Clipboard = "\x1b]52;c;cm0gLXJm\x07"
	csiEraseScreen = "\x1b[2J"
	c1CSI          = "K"
)

func TestValidateLine(t *testing.T) {
	t.Run("accepts ordinary text", func(t *testing.T) {
		assert.NoError(t, ValidateLine("Ship the release"))
		assert.NoError(t, ValidateLine("café — naïve 日本語 🎉"))
		assert.NoError(t, ValidateLine(""))
	})

	t.Run("rejects terminal control sequences", func(t *testing.T) {
		for name, payload := range map[string]string{
			"OSC 52 clipboard write": osc52Clipboard,
			"CSI erase screen":       csiEraseScreen,
			"bare C1 CSI":            c1CSI,
			"lone ESC":               "\x1b",
			"embedded in text":       "Ship it" + osc52Clipboard + " today",
			"DEL":                    "a\x7fb",
			"NUL":                    "a\x00b",
		} {
			assert.ErrorIs(t, ValidateLine(payload), ErrControlCharacters, name)
		}
	})

	t.Run("rejects newline and tab", func(t *testing.T) {
		// A single-line field has no legitimate use for either, and a newline is
		// what turns one stored title into thousands of rendered rows.
		assert.ErrorIs(t, ValidateLine("a\nb"), ErrControlCharacters)
		assert.ErrorIs(t, ValidateLine("a\tb"), ErrControlCharacters)
	})

	t.Run("rejects invalid UTF-8", func(t *testing.T) {
		assert.ErrorIs(t, ValidateLine("bad \xff\xfe"), ErrInvalidUTF8)
	})

	t.Run("does not cap length", func(t *testing.T) {
		// A title may be as long as the user cares to type. What a long one must
		// not do is push the screen out of reach, and that is bounded at render
		// time (ui.WrapClamped), not here.
		assert.NoError(t, ValidateLine(strings.Repeat("a", 100_000)))
		assert.NoError(t, ValidateLine(strings.Repeat("é", 100_000)))
	})
}

func TestValidateMultiline(t *testing.T) {
	t.Run("keeps newline and tab", func(t *testing.T) {
		assert.NoError(t, ValidateMultiline("line one\nline two\n\tindented"))
	})

	t.Run("still rejects other control characters", func(t *testing.T) {
		assert.ErrorIs(t, ValidateMultiline("notes\n"+osc52Clipboard), ErrControlCharacters)
		assert.ErrorIs(t, ValidateMultiline(c1CSI), ErrControlCharacters)
	})

	t.Run("is not length-capped", func(t *testing.T) {
		assert.NoError(t, ValidateMultiline(strings.Repeat("a", 100_000)))
	})
}

func TestStripControl(t *testing.T) {
	t.Run("removes control characters, defanging the sequence", func(t *testing.T) {
		// Only the control byte goes; the printable remainder stays as literal
		// text. "]52;c;..." without its ESC is something the terminal displays
		// rather than obeys, which is the whole point.
		assert.Equal(t, "Ship it]52;c;cm0gLXJm today",
			StripControl("Ship it"+osc52Clipboard+" today", false))
		assert.Equal(t, "ab", StripControl("a\x00b", false))
	})

	t.Run("keeps newline and tab only when asked", func(t *testing.T) {
		assert.Equal(t, "ab", StripControl("a\nb", false))
		assert.Equal(t, "a\nb", StripControl("a\nb", true))
	})

	t.Run("returns clean input unchanged", func(t *testing.T) {
		const clean = "café 日本語 🎉"
		assert.Equal(t, clean, StripControl(clean, false))
	})

	t.Run("preserves a genuine replacement character", func(t *testing.T) {
		// U+FFFD is legitimate content and decodes as three bytes, whereas an
		// undecodable byte also surfaces as U+FFFD but decodes as one. Only the
		// latter should be dropped.
		assert.Equal(t, "a�b", StripControl("a�b", false))
		assert.Equal(t, "ab", StripControl("a\xffb", false))
	})
}

func TestValidateProjectText(t *testing.T) {
	withTitle := func(title string) Project {
		return Project{
			Name: "Roadmap",
			Categories: []Category{{
				Name:  "Backlog",
				Tasks: []Task{{Title: title}},
			}},
		}
	}

	t.Run("accepts a clean project", func(t *testing.T) {
		assert.NoError(t, ValidateProjectText(withTitle("Ship the release")))
	})

	t.Run("rejects control characters anywhere", func(t *testing.T) {
		assert.ErrorIs(t, ValidateProjectText(withTitle(osc52Clipboard)), ErrControlCharacters)

		p := withTitle("fine")
		p.Name = "Road" + csiEraseScreen + "map"
		assert.ErrorIs(t, ValidateProjectText(p), ErrControlCharacters)

		p = withTitle("fine")
		p.Categories[0].Name = c1CSI
		assert.ErrorIs(t, ValidateProjectText(p), ErrControlCharacters)

		// Fields that are enum-like or generated in normal use are still
		// attacker-controlled in a hand-written file, and they render too.
		p = withTitle("fine")
		p.Categories[0].Tasks[0].Status = osc52Clipboard
		assert.ErrorIs(t, ValidateProjectText(p), ErrControlCharacters)

		p = withTitle("fine")
		p.Categories[0].Tasks[0].TagColor = osc52Clipboard
		assert.ErrorIs(t, ValidateProjectText(p), ErrControlCharacters)
	})

	t.Run("allows multi-line descriptions", func(t *testing.T) {
		p := withTitle("fine")
		p.Categories[0].Tasks[0].Description = "first\nsecond\n\tthird"
		assert.NoError(t, ValidateProjectText(p))

		p.Categories[0].Tasks[0].Description = "first\n" + osc52Clipboard
		assert.ErrorIs(t, ValidateProjectText(p), ErrControlCharacters)
	})

	t.Run("allows over-long text", func(t *testing.T) {
		assert.NoError(t, ValidateProjectText(withTitle(strings.Repeat("a", 100_000))))
	})
}

func TestStripProjectText(t *testing.T) {
	p := Project{
		ID:   "abc",
		Name: "Road" + csiEraseScreen + "map",
		Categories: []Category{{
			Name: "Back" + osc52Clipboard + "log",
			Tasks: []Task{{
				Title:       "Ship" + osc52Clipboard,
				Description: "keep\nthe newline" + csiEraseScreen,
				TagColor:    c1CSI + "green",
			}},
		}},
	}
	StripProjectText(&p)

	// Every control byte is gone, leaving the printable remainder as inert
	// text — the terminal displays "[2J" instead of clearing the screen.
	assert.Equal(t, "Road[2Jmap", p.Name)
	assert.Equal(t, "Back]52;c;cm0gLXJmlog", p.Categories[0].Name)
	assert.Equal(t, "Ship]52;c;cm0gLXJm", p.Categories[0].Tasks[0].Title)
	assert.Equal(t, "Kgreen", p.Categories[0].Tasks[0].TagColor)
	// Descriptions keep their line structure; only the escape byte goes.
	assert.Equal(t, "keep\nthe newline[2J", p.Categories[0].Tasks[0].Description)

	// Stripping must leave the result acceptable to the write-path validator,
	// or a loaded project could not be saved back.
	require.NoError(t, ValidateProjectText(p))
}

func TestValidateID(t *testing.T) {
	t.Run("accepts generated IDs", func(t *testing.T) {
		id, err := NewID()
		require.NoError(t, err)
		assert.NoError(t, ValidateID(id))
	})

	t.Run("accepts hand-written IDs", func(t *testing.T) {
		// Import keeps a readable ID from a hand-written file, so the charset is
		// deliberately wider than what NewID emits.
		for _, id := range []string{"work", "p1", "my_project-2", "A"} {
			assert.NoError(t, ValidateID(id), id)
		}
	})

	t.Run("rejects path traversal", func(t *testing.T) {
		// Each of these was a working escape from the data directory: the ID is
		// interpolated straight into <dir>/<id>.json.
		for _, id := range []string{
			"..",
			"../../etc/passwd",
			"../../ESCAPED_OUTSIDE",
			"..%2f..%2fx",
			"/etc/passwd",
			"a/b",
			"a\\b",
			".hidden",
			"a.json",
		} {
			assert.ErrorIs(t, ValidateID(id), ErrInvalidID, id)
		}
	})

	t.Run("rejects empty, over-long and exotic input", func(t *testing.T) {
		assert.ErrorIs(t, ValidateID(""), ErrInvalidID)
		assert.ErrorIs(t, ValidateID(strings.Repeat("a", MaxIDLen+1)), ErrInvalidID)
		assert.ErrorIs(t, ValidateID("a\x00b"), ErrInvalidID)
		assert.ErrorIs(t, ValidateID("id with spaces"), ErrInvalidID)
		assert.ErrorIs(t, ValidateID("café"), ErrInvalidID)
	})
}
