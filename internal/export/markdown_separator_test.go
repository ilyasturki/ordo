package export

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/domain"
)

func TestExportSeparators(t *testing.T) {
	project := domain.Project{
		Name: "Sep",
		Categories: []domain.Category{
			{
				Name: "Work",
				Tasks: []domain.Task{
					{Title: "before", Status: domain.StatusTodo},
					{Kind: domain.KindSeparator, Title: "v0.9"},
					{Title: "after", Status: domain.StatusTodo},
					{Kind: domain.KindSeparator},
				},
			},
		},
	}

	var buf bytes.Buffer
	require.NoError(t, ExportMarkdown(project, &buf, Options{}))
	out := buf.String()

	assert.Contains(t, out, "### v0.9")
	assert.Contains(t, out, "\n---\n")
	// Separators must not be emitted as task list items.
	assert.NotContains(t, out, "[ ] v0.9")
}

func TestImportSeparators(t *testing.T) {
	md := strings.Join([]string{
		"# Sep",
		"",
		"## Work",
		"",
		"- [ ] before",
		"### v0.9",
		"- [x] after",
		"---",
		"- [ ] tail",
	}, "\n")

	project, err := ImportMarkdown(strings.NewReader(md), "")
	require.NoError(t, err)
	require.Len(t, project.Categories, 1)

	tasks := project.Categories[0].Tasks
	require.Len(t, tasks, 5)

	assert.False(t, tasks[0].IsSeparator())
	assert.Equal(t, "before", tasks[0].Title)

	assert.True(t, tasks[1].IsSeparator())
	assert.Equal(t, "v0.9", tasks[1].Title)

	assert.False(t, tasks[2].IsSeparator())
	assert.Equal(t, "after", tasks[2].Title)
	assert.Equal(t, domain.StatusCompleted, tasks[2].Status)

	assert.True(t, tasks[3].IsSeparator())
	assert.Equal(t, "", tasks[3].Title)

	assert.Equal(t, "tail", tasks[4].Title)
}

func TestSeparatorRoundTrip(t *testing.T) {
	project := domain.Project{
		Name: "RT",
		Categories: []domain.Category{
			{
				Name: "C",
				Tasks: []domain.Task{
					{Title: "a", Status: domain.StatusTodo},
					{Kind: domain.KindSeparator, Title: "milestone"},
					{Title: "b", Status: domain.StatusInProgress},
					{Kind: domain.KindSeparator},
					{Title: "c", Status: domain.StatusCompleted},
				},
			},
		},
	}

	var buf bytes.Buffer
	require.NoError(t, ExportMarkdown(project, &buf, Options{}))

	reimported, err := ImportMarkdown(&buf, "")
	require.NoError(t, err)
	require.Len(t, reimported.Categories, 1)

	got := reimported.Categories[0].Tasks
	require.Len(t, got, 5)

	kinds := make([]bool, len(got))
	for i, tk := range got {
		kinds[i] = tk.IsSeparator()
	}
	assert.Equal(t, []bool{false, true, false, true, false}, kinds)
	assert.Equal(t, "milestone", got[1].Title)
	assert.Equal(t, "", got[3].Title)
}
