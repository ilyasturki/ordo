package export

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/domain"
)

func TestExportNotes(t *testing.T) {
	project := domain.Project{Name: "N", Categories: []domain.Category{{Name: "Work", Tasks: []domain.Task{
		{Title: "before", Status: domain.StatusTodo, Description: "> quoted in the details"},
		{Kind: domain.KindNote, Title: "ask Sam first"},
	}}}}

	var buf bytes.Buffer
	require.NoError(t, ExportMarkdown(project, &buf, Options{}))
	assert.Contains(t, buf.String(), "\n> ask Sam first\n")
	assert.Contains(t, buf.String(), "\n    > quoted in the details\n")
	assert.Equal(t, "> ask Sam first", LineMarkdown(project.Categories[0].Tasks[1]))
}

func TestImportNotes(t *testing.T) {
	md := strings.Join([]string{
		"# N",
		"## Work",
		"- [ ] before",
		"    > quoted in the details",
		">ask Sam first",
		"  - [ ] not a subtask of the note",
		">",
		"> second",
	}, "\n")

	project, err := ImportMarkdown(strings.NewReader(md), "")
	require.NoError(t, err)
	tasks := project.Categories[0].Tasks
	require.Len(t, tasks, 4)
	assert.Equal(t, "> quoted in the details", tasks[0].Description)
	assert.True(t, tasks[1].IsNote())
	assert.Equal(t, "ask Sam first", tasks[1].Title)
	assert.Empty(t, tasks[1].Status)
	assert.Empty(t, tasks[2].ParentID, "a note parents nothing")
	assert.True(t, tasks[3].IsNote(), "an empty quote line is dropped")
	assert.Equal(t, "second", tasks[3].Title)
}

func TestNoteRoundTrip(t *testing.T) {
	project := domain.Project{Name: "RT", Categories: []domain.Category{{Name: "C", Tasks: []domain.Task{
		{Title: "a", Status: domain.StatusTodo},
		{Kind: domain.KindNote, Title: "a note > with a quote"},
		{Kind: domain.KindSeparator, Title: "later"},
		{Kind: domain.KindNote, Title: "### not a separator"},
	}}}}

	var buf bytes.Buffer
	require.NoError(t, ExportMarkdown(project, &buf, Options{}))
	back, err := ImportMarkdown(&buf, "")
	require.NoError(t, err)

	got := back.Categories[0].Tasks
	require.Len(t, got, 4)
	for i, want := range project.Categories[0].Tasks {
		assert.Equal(t, want.Kind, got[i].Kind)
		assert.Equal(t, want.Title, got[i].Title)
	}
}
