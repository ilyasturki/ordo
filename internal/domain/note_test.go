package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLineMakesANote(t *testing.T) {
	note, err := NewLine(KindNote, "ask Sam first")
	require.NoError(t, err)
	assert.True(t, note.IsNote())
	assert.False(t, note.IsTask())
	assert.False(t, note.IsSeparator())
	assert.Equal(t, "ask Sam first", note.Title)
	assert.Empty(t, note.Status)
	assert.NotEmpty(t, note.ID)
}

func TestNotesAreLeftOutLikeSeparators(t *testing.T) {
	p := Project{Categories: []Category{{Tasks: []Task{
		{ID: "a", Title: "one", Status: StatusCompleted},
		{ID: "n", Kind: KindNote, Title: "one"},
		{ID: "b", Title: "two", Status: StatusCompleted, TagColor: TagGreen},
	}}}}
	cat := &p.Categories[0]

	assert.Equal(t, 2, cat.TaskCount())
	assert.Equal(t, 2, cat.StatusCounts().Total())
	assert.Equal(t, StatusCompleted, cat.AggregateStatus())
	_, ti, err := p.FindTask("one")
	require.NoError(t, err)
	assert.Equal(t, 0, ti, "a note never matches a task selector")
	_, _, err = p.FindTask("n")
	assert.ErrorIs(t, err, ErrTaskNotFound)
	assert.Equal(t, "", (&Category{Tasks: []Task{{Kind: KindNote}}}).AggregateStatus())
}

func TestNotesNeverNest(t *testing.T) {
	c := Category{Tasks: []Task{{ID: "a", Title: "A"}, {ID: "n", Kind: KindNote, Title: "N"}}}
	assert.ErrorIs(t, c.IndentTasks(1, 2), ErrSeparatorNest)

	c = Category{Tasks: []Task{{ID: "n", Kind: KindNote, Title: "N"}, {ID: "a", Title: "A"}}}
	assert.ErrorIs(t, c.IndentTasks(1, 2), ErrNothingToNest)

	got := normalizeOutline([]Task{
		{ID: "a", Title: "A"},
		{ID: "n", Kind: KindNote, Title: "N", ParentID: "a"},
		{ID: "b", Title: "B", ParentID: "n"},
	})
	assert.Equal(t, []string{"a", "n", "b"}, taskShape(got))
}
