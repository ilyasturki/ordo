package export

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/domain"
)

func TestExportMarkdown(t *testing.T) {
	t.Run("exports project with all status types", func(t *testing.T) {
		project := domain.Project{
			Name: "Test Project",
			Categories: []domain.Category{
				{
					Name: "Features",
					Tasks: []domain.Task{
						{Title: "Todo task", Status: domain.StatusTodo},
						{Title: "Done task", Status: domain.StatusCompleted},
						{Title: "Cancelled task", Status: domain.StatusCancelled},
						{Title: "In progress task", Status: domain.StatusInProgress},
					},
				},
			},
		}

		var buf bytes.Buffer
		err := ExportMarkdown(project, &buf)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "# Test Project")
		assert.Contains(t, output, "## Features")
		assert.Contains(t, output, "- [ ] Todo task")
		assert.Contains(t, output, "- [x] Done task")
		assert.Contains(t, output, "- [-] Cancelled task")
		assert.Contains(t, output, "- [~] In progress task")
	})

	t.Run("exports priorities", func(t *testing.T) {
		project := domain.Project{
			Name: "Priority Test",
			Categories: []domain.Category{
				{
					Name: "Tasks",
					Tasks: []domain.Task{
						{Title: "High priority", Status: domain.StatusTodo, Priority: domain.PriorityHigh},
						{Title: "Medium priority", Status: domain.StatusTodo, Priority: domain.PriorityMedium},
						{Title: "Low priority", Status: domain.StatusTodo, Priority: domain.PriorityLow},
						{Title: "No priority", Status: domain.StatusTodo},
					},
				},
			},
		}

		var buf bytes.Buffer
		err := ExportMarkdown(project, &buf)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "- [ ] High priority (high)")
		assert.Contains(t, output, "- [ ] Medium priority (medium)")
		assert.Contains(t, output, "- [ ] Low priority (low)")
		assert.Contains(t, output, "- [ ] No priority\n")
	})

	t.Run("exports multiple categories", func(t *testing.T) {
		project := domain.Project{
			Name: "Multi Category",
			Categories: []domain.Category{
				{Name: "Feature", Tasks: []domain.Task{{Title: "Feature 1", Status: domain.StatusTodo}}},
				{Name: "Fix", Tasks: []domain.Task{{Title: "Bug fix", Status: domain.StatusCompleted}}},
			},
		}

		var buf bytes.Buffer
		err := ExportMarkdown(project, &buf)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "## Feature")
		assert.Contains(t, output, "## Fix")
		assert.Contains(t, output, "- [ ] Feature 1")
		assert.Contains(t, output, "- [x] Bug fix")
	})

	t.Run("exports empty categories", func(t *testing.T) {
		project := domain.Project{
			Name: "Empty Categories",
			Categories: []domain.Category{
				{Name: "Empty", Tasks: []domain.Task{}},
			},
		}

		var buf bytes.Buffer
		err := ExportMarkdown(project, &buf)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "## Empty")
	})
}

func TestImportMarkdown(t *testing.T) {
	t.Run("imports basic markdown", func(t *testing.T) {
		md := `# My Project

## Features

- [ ] Task one
- [x] Task two
- [-] Task three
- [~] Task four
`
		project, err := ImportMarkdown(strings.NewReader(md), "")
		require.NoError(t, err)

		assert.Equal(t, "My Project", project.Name)
		require.Len(t, project.Categories, 1)
		assert.Equal(t, "Features", project.Categories[0].Name)
		require.Len(t, project.Categories[0].Tasks, 4)

		assert.Equal(t, "Task one", project.Categories[0].Tasks[0].Title)
		assert.Equal(t, domain.StatusTodo, project.Categories[0].Tasks[0].Status)

		assert.Equal(t, "Task two", project.Categories[0].Tasks[1].Title)
		assert.Equal(t, domain.StatusCompleted, project.Categories[0].Tasks[1].Status)

		assert.Equal(t, "Task three", project.Categories[0].Tasks[2].Title)
		assert.Equal(t, domain.StatusCancelled, project.Categories[0].Tasks[2].Status)

		assert.Equal(t, "Task four", project.Categories[0].Tasks[3].Title)
		assert.Equal(t, domain.StatusInProgress, project.Categories[0].Tasks[3].Status)
	})

	t.Run("imports priorities", func(t *testing.T) {
		md := `# Priority Test

## Tasks

- [ ] High task (high)
- [ ] Medium task (medium)
- [ ] Low task (low)
- [ ] No priority task
`
		project, err := ImportMarkdown(strings.NewReader(md), "")
		require.NoError(t, err)

		require.Len(t, project.Categories[0].Tasks, 4)
		assert.Equal(t, "High task", project.Categories[0].Tasks[0].Title)
		assert.Equal(t, domain.PriorityHigh, project.Categories[0].Tasks[0].Priority)

		assert.Equal(t, "Medium task", project.Categories[0].Tasks[1].Title)
		assert.Equal(t, domain.PriorityMedium, project.Categories[0].Tasks[1].Priority)

		assert.Equal(t, "Low task", project.Categories[0].Tasks[2].Title)
		assert.Equal(t, domain.PriorityLow, project.Categories[0].Tasks[2].Priority)

		assert.Equal(t, "No priority task", project.Categories[0].Tasks[3].Title)
		assert.Empty(t, project.Categories[0].Tasks[3].Priority)
	})

	t.Run("imports multiple categories", func(t *testing.T) {
		md := `# Multi

## Feature

- [ ] Feature 1

## Fix

- [x] Bug fix
`
		project, err := ImportMarkdown(strings.NewReader(md), "")
		require.NoError(t, err)

		require.Len(t, project.Categories, 2)
		assert.Equal(t, "Feature", project.Categories[0].Name)
		assert.Equal(t, "Fix", project.Categories[1].Name)
	})

	t.Run("uses provided name over parsed name", func(t *testing.T) {
		md := `# Parsed Name

## Tasks

- [ ] Task
`
		project, err := ImportMarkdown(strings.NewReader(md), "Override Name")
		require.NoError(t, err)

		assert.Equal(t, "Override Name", project.Name)
	})

	t.Run("uses default name when none provided", func(t *testing.T) {
		md := `## Tasks

- [ ] Task
`
		project, err := ImportMarkdown(strings.NewReader(md), "")
		require.NoError(t, err)

		assert.Equal(t, "Imported Project", project.Name)
	})

	t.Run("generates IDs for all entities", func(t *testing.T) {
		md := `# Test

## Category

- [ ] Task
`
		project, err := ImportMarkdown(strings.NewReader(md), "")
		require.NoError(t, err)

		assert.NotEmpty(t, project.ID)
		assert.NotEmpty(t, project.Categories[0].ID)
		assert.NotEmpty(t, project.Categories[0].Tasks[0].ID)
	})

	t.Run("sets timestamps", func(t *testing.T) {
		md := `# Test

## Category

- [ ] Task
`
		project, err := ImportMarkdown(strings.NewReader(md), "")
		require.NoError(t, err)

		assert.NotEmpty(t, project.CreatedAt)
		assert.NotEmpty(t, project.UpdatedAt)
		assert.NotEmpty(t, project.Categories[0].CreatedAt)
		assert.NotEmpty(t, project.Categories[0].Tasks[0].CreatedAt)
	})

	t.Run("sets completion date for completed tasks", func(t *testing.T) {
		md := `# Test

## Category

- [x] Completed task
- [ ] Todo task
`
		project, err := ImportMarkdown(strings.NewReader(md), "")
		require.NoError(t, err)

		assert.NotEmpty(t, project.Categories[0].Tasks[0].CompletionDate)
		assert.Empty(t, project.Categories[0].Tasks[1].CompletionDate)
	})

	t.Run("ignores content between project header and first category", func(t *testing.T) {
		md := `# Test

Some notes here.

## Tasks

- [ ] Task
`
		project, err := ImportMarkdown(strings.NewReader(md), "")
		require.NoError(t, err)

		assert.Equal(t, "Test", project.Name)
		require.Len(t, project.Categories, 1)
		assert.Equal(t, "Tasks", project.Categories[0].Name)
	})

	t.Run("ignores tasks outside categories", func(t *testing.T) {
		md := `# Test

- [ ] Orphan task

## Tasks

- [ ] Valid task
`
		project, err := ImportMarkdown(strings.NewReader(md), "")
		require.NoError(t, err)

		require.Len(t, project.Categories, 1)
		require.Len(t, project.Categories[0].Tasks, 1)
		assert.Equal(t, "Valid task", project.Categories[0].Tasks[0].Title)
	})
}

func TestRoundTrip(t *testing.T) {
	t.Run("export then import preserves data", func(t *testing.T) {
		original := domain.Project{
			Name: "Round Trip",
			Categories: []domain.Category{
				{
					Name: "Feature",
					Tasks: []domain.Task{
						{Title: "Task A", Status: domain.StatusTodo, Priority: domain.PriorityHigh},
						{Title: "Task B", Status: domain.StatusCompleted},
						{Title: "Task C", Status: domain.StatusInProgress, Priority: domain.PriorityLow},
					},
				},
				{
					Name: "Fix",
					Tasks: []domain.Task{
						{Title: "Bug X", Status: domain.StatusCancelled, Priority: domain.PriorityMedium},
					},
				},
			},
		}

		var buf bytes.Buffer
		err := ExportMarkdown(original, &buf)
		require.NoError(t, err)

		imported, err := ImportMarkdown(&buf, "")
		require.NoError(t, err)

		assert.Equal(t, original.Name, imported.Name)
		require.Len(t, imported.Categories, len(original.Categories))

		for i, origCat := range original.Categories {
			impCat := imported.Categories[i]
			assert.Equal(t, origCat.Name, impCat.Name)
			require.Len(t, impCat.Tasks, len(origCat.Tasks))

			for j, origTask := range origCat.Tasks {
				impTask := impCat.Tasks[j]
				assert.Equal(t, origTask.Title, impTask.Title)
				assert.Equal(t, origTask.Status, impTask.Status)
				assert.Equal(t, origTask.Priority, impTask.Priority)
			}
		}
	})
}

func TestDescriptionRoundTrip(t *testing.T) {
	t.Run("exports description as indented body", func(t *testing.T) {
		project := domain.Project{
			Name: "Desc",
			Categories: []domain.Category{
				{
					Name: "Tasks",
					Tasks: []domain.Task{
						{Title: "With body", Status: domain.StatusTodo, Description: "Line one.\nLine two."},
						{Title: "No body", Status: domain.StatusTodo},
					},
				},
			},
		}

		var buf bytes.Buffer
		require.NoError(t, ExportMarkdown(project, &buf))
		output := buf.String()

		assert.Contains(t, output, "- [ ] With body\n    Line one.\n    Line two.\n")
		assert.Contains(t, output, "- [ ] No body\n")
	})

	t.Run("imports description from indented body", func(t *testing.T) {
		md := `# Project

## Tasks

- [ ] First task
    Description line one.
    Description line two.
- [x] Second task
- [ ] Third task
    Has its own body.
`
		project, err := ImportMarkdown(strings.NewReader(md), "")
		require.NoError(t, err)
		require.Len(t, project.Categories[0].Tasks, 3)

		assert.Equal(t, "Description line one.\nDescription line two.", project.Categories[0].Tasks[0].Description)
		assert.Empty(t, project.Categories[0].Tasks[1].Description)
		assert.Equal(t, "Has its own body.", project.Categories[0].Tasks[2].Description)
	})

	t.Run("imports multi-paragraph description", func(t *testing.T) {
		md := `# Project

## Tasks

- [ ] Task
    Paragraph one.

    Paragraph two.
`
		project, err := ImportMarkdown(strings.NewReader(md), "")
		require.NoError(t, err)

		assert.Equal(t, "Paragraph one.\n\nParagraph two.", project.Categories[0].Tasks[0].Description)
	})

	t.Run("round-trips descriptions", func(t *testing.T) {
		original := domain.Project{
			Name: "RT",
			Categories: []domain.Category{
				{
					Name: "Cat",
					Tasks: []domain.Task{
						{Title: "Single line", Status: domain.StatusTodo, Description: "Just one line."},
						{Title: "Multi line", Status: domain.StatusTodo, Description: "First.\nSecond."},
						{Title: "Paragraphs", Status: domain.StatusTodo, Description: "Para A.\n\nPara B."},
						{Title: "No desc", Status: domain.StatusTodo},
					},
				},
			},
		}

		var buf bytes.Buffer
		require.NoError(t, ExportMarkdown(original, &buf))

		imported, err := ImportMarkdown(&buf, "")
		require.NoError(t, err)
		require.Len(t, imported.Categories[0].Tasks, 4)
		for i, want := range original.Categories[0].Tasks {
			assert.Equal(t, want.Description, imported.Categories[0].Tasks[i].Description,
				"task %d (%s)", i, want.Title)
		}
	})
}

func TestStatusToMarker(t *testing.T) {
	tests := []struct {
		status   string
		expected string
	}{
		{domain.StatusTodo, " "},
		{domain.StatusCompleted, "x"},
		{domain.StatusCancelled, "-"},
		{domain.StatusInProgress, "~"},
		{"unknown", " "},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			assert.Equal(t, tt.expected, statusToMarker(tt.status))
		})
	}
}

func TestMarkerToStatus(t *testing.T) {
	tests := []struct {
		marker   string
		expected string
	}{
		{" ", domain.StatusTodo},
		{"x", domain.StatusCompleted},
		{"-", domain.StatusCancelled},
		{"~", domain.StatusInProgress},
		{"?", domain.StatusTodo},
	}

	for _, tt := range tests {
		t.Run(tt.marker, func(t *testing.T) {
			assert.Equal(t, tt.expected, markerToStatus(tt.marker))
		})
	}
}

func TestImportMarkdown_MergesRepeatedHeadings(t *testing.T) {
	md := "# P\n\n## v0.2.0\n\n- [ ] one\n\n## next\n\n- [ ] two\n\n## V0.2.0 \n\n### later\n- [x] three\n"
	imported, err := ImportMarkdown(strings.NewReader(md), "")
	require.NoError(t, err)

	require.Len(t, imported.Categories, 2)
	first := imported.Categories[0]
	assert.Equal(t, "v0.2.0", first.Name)
	require.Len(t, first.Tasks, 3)
	assert.Equal(t, "one", first.Tasks[0].Title)
	assert.True(t, first.Tasks[1].IsSeparator())
	assert.Equal(t, "three", first.Tasks[2].Title)
	assert.Equal(t, "next", imported.Categories[1].Name)
}

func TestExportMarkdown_TagLeavesTitleAlone(t *testing.T) {
	project := domain.Project{
		Name: "Tagged",
		Categories: []domain.Category{{
			Name: "Feature",
			Tasks: []domain.Task{
				{Title: "Ship it", Status: domain.StatusTodo, Priority: domain.PriorityHigh, TagColor: domain.TagCyan},
			},
		}},
	}
	var buf bytes.Buffer
	require.NoError(t, ExportMarkdown(project, &buf))
	assert.Contains(t, buf.String(), "- [ ] Ship it (high)\n")
}

// Titles used to lose a trailing " #word" to the removed tag-label suffix.
func TestImportMarkdown_TrailingHashtagStaysInTitle(t *testing.T) {
	md := "# P\n\n## Feature\n\n- [ ] Fix #123 in parser (high)\n- [x] Ship it #urgent\n"
	imported, err := ImportMarkdown(strings.NewReader(md), "")
	require.NoError(t, err)

	tasks := imported.Categories[0].Tasks
	assert.Equal(t, "Fix #123 in parser", tasks[0].Title)
	assert.Equal(t, domain.PriorityHigh, tasks[0].Priority)
	assert.Equal(t, "Ship it #urgent", tasks[1].Title)
	assert.Equal(t, "", tasks[1].TagColor)
}
