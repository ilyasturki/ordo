package app

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ilyasturki/ordo/internal/domain"
)

func TestParseTaskEdit(t *testing.T) {
	t.Run("title only", func(t *testing.T) {
		title, desc := parseTaskEdit("Just a title")
		assert.Equal(t, "Just a title", title)
		assert.Empty(t, desc)
	})

	t.Run("title with trailing newline only", func(t *testing.T) {
		title, desc := parseTaskEdit("Just a title\n")
		assert.Equal(t, "Just a title", title)
		assert.Empty(t, desc)
	})

	t.Run("single newline separates title from description", func(t *testing.T) {
		title, desc := parseTaskEdit("Title\nBody text")
		assert.Equal(t, "Title", title)
		assert.Equal(t, "Body text", desc)
	})

	t.Run("blank line between title and description (git-commit style)", func(t *testing.T) {
		title, desc := parseTaskEdit("Title\n\nBody text")
		assert.Equal(t, "Title", title)
		assert.Equal(t, "Body text", desc)
	})

	t.Run("multi-paragraph description", func(t *testing.T) {
		title, desc := parseTaskEdit("Title\n\nPara one.\n\nPara two.")
		assert.Equal(t, "Title", title)
		assert.Equal(t, "Para one.\n\nPara two.", desc)
	})

	t.Run("trims surrounding blank lines from description", func(t *testing.T) {
		title, desc := parseTaskEdit("Title\n\n\nBody\n\n\n")
		assert.Equal(t, "Title", title)
		assert.Equal(t, "Body", desc)
	})

	t.Run("whitespace-only description is treated as empty", func(t *testing.T) {
		title, desc := parseTaskEdit("Title\n   \n\t\n")
		assert.Equal(t, "Title", title)
		assert.Empty(t, desc)
	})

	t.Run("leading blank lines before title are skipped", func(t *testing.T) {
		title, desc := parseTaskEdit("\n\nTitle\nBody")
		assert.Equal(t, "Title", title)
		assert.Equal(t, "Body", desc)
	})

	t.Run("crlf line endings are handled", func(t *testing.T) {
		title, desc := parseTaskEdit("Title\r\n\r\nBody")
		assert.Equal(t, "Title", title)
		assert.Equal(t, "Body", desc)
	})

	t.Run("empty content returns empty", func(t *testing.T) {
		title, desc := parseTaskEdit("")
		assert.Empty(t, title)
		assert.Empty(t, desc)
	})
}

func TestFormatTaskForEdit(t *testing.T) {
	t.Run("title only when no description", func(t *testing.T) {
		got := formatTaskForEdit(domain.Task{Title: "Just title"})
		assert.Equal(t, "Just title", got)
	})

	t.Run("uses blank line separator for readability", func(t *testing.T) {
		got := formatTaskForEdit(domain.Task{Title: "T", Description: "Body"})
		assert.Equal(t, "T\n\nBody", got)
	})

	t.Run("round-trips through parseTaskEdit", func(t *testing.T) {
		task := domain.Task{Title: "Hello", Description: "Line one.\n\nLine two."}
		title, desc := parseTaskEdit(formatTaskForEdit(task))
		assert.Equal(t, task.Title, title)
		assert.Equal(t, task.Description, desc)
	})
}
