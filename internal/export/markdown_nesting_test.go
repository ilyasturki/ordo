package export

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"phasionary/internal/domain"
)

func nestedFixture() domain.Project {
	return domain.Project{
		Name: "Versions",
		Categories: []domain.Category{
			{ID: "v2", Name: "v0.2.0", Tasks: []domain.Task{
				{ID: "loose", Title: "Loose", Status: domain.StatusTodo},
			}},
			{ID: "ui", Name: "ui/ux", ParentID: "v2", Tasks: []domain.Task{
				{ID: "timer", Title: "Rest timer", Status: domain.StatusInProgress, Priority: domain.PriorityHigh, Description: "why\n  - keep this indent"},
				{ID: "pick", ParentID: "timer", Title: "Pick a default", Status: domain.StatusCompleted, Description: "first line\n\nafter a gap"},
				{ID: "sound", ParentID: "timer", Title: "Add a sound", Status: domain.StatusTodo},
				{ID: "sep", Kind: domain.KindSeparator, Title: "later"},
				{ID: "sets", Title: "Fix sets", Status: domain.StatusTodo},
			}},
			{ID: "v1", Name: "v0.1.0", Tasks: []domain.Task{
				{ID: "old", Title: "Old", Status: domain.StatusCompleted},
			}},
			{ID: "api", Name: "api", ParentID: "v1"},
		},
	}
}

func outline(p domain.Project) []string {
	var out []string
	for _, c := range p.Categories {
		line := "## " + c.Name
		if c.ParentID != "" {
			line = "  " + line
		}
		out = append(out, line)
		for _, t := range c.Tasks {
			line := "- " + t.Status + " " + t.Title + " (" + t.Priority + ") " + strings.ReplaceAll(t.Description, "\n", "|")
			if t.IsSeparator() {
				line = "-- " + t.Title
			}
			if t.ParentID != "" {
				line = "  " + line
			}
			out = append(out, line)
		}
	}
	return out
}

func TestExportNesting(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, ExportMarkdown(nestedFixture(), &buf))
	out := buf.String()

	assert.Contains(t, out, "## v0.2.0\n")
	assert.Contains(t, out, "## v0.2.0 / ui\\/ux\n", "a sub-category's heading carries its path, a slash in a name escaped")
	assert.Contains(t, out, "- [~] Rest timer (high)\n    why\n      - keep this indent\n  - [x] Pick a default\n      first line\n\n      after a gap\n  - [ ] Add a sound\n### later\n")
	assert.Contains(t, out, "## v0.1.0 / api\n")
}

func TestNestingRoundTrip(t *testing.T) {
	project := nestedFixture()
	var buf bytes.Buffer
	require.NoError(t, ExportMarkdown(project, &buf))

	back, err := ImportMarkdown(&buf, "")
	require.NoError(t, err)
	assert.Equal(t, outline(project), outline(back))
	assert.Equal(t, 2, domain.RequiredSchema(back))
}

func TestCategoryMarkdownTakesTheAreas(t *testing.T) {
	project := nestedFixture()
	out := CategoryMarkdown(project, 0)
	assert.True(t, strings.HasPrefix(out, "## v0.2.0\n"))
	assert.Contains(t, out, "## v0.2.0 / ui\\/ux")
	assert.NotContains(t, out, "v0.1.0")
	assert.True(t, strings.HasPrefix(CategoryMarkdown(project, 1), "## v0.2.0 / ui\\/ux\n"))
}

func TestImportNestingRules(t *testing.T) {
	t.Run("a path heading names a sub-category only under an earlier category", func(t *testing.T) {
		md := "# P\n\n## A / B\n\n## A\n\n## A / C\n\n- [ ] in c\n"
		p, err := ImportMarkdown(strings.NewReader(md), "")
		require.NoError(t, err)
		require.Len(t, p.Categories, 3)
		assert.Equal(t, "A / B", p.Categories[0].Name, "no A yet, so the whole heading is the name")
		assert.Equal(t, "C", p.Categories[2].Name)
		assert.Equal(t, p.Categories[1].ID, p.Categories[2].ParentID)
	})
	t.Run("a sub-category heading later in the file joins its parent's block", func(t *testing.T) {
		md := "## A\n## B\n## A / x\n## A / x\n- [ ] once\n"
		p, err := ImportMarkdown(strings.NewReader(md), "")
		require.NoError(t, err)
		var names []string
		for i := range p.Categories {
			names = append(names, p.CategoryPath(i))
		}
		assert.Equal(t, []string{"A", "A / x", "B"}, names)
	})
	t.Run("a subtask line needs a task above it", func(t *testing.T) {
		md := "## A\n  - [ ] orphan\n- [ ] parent\n  - [ ] child\n### sep\n  - [ ] after sep\n"
		p, err := ImportMarkdown(strings.NewReader(md), "")
		require.NoError(t, err)
		tasks := p.Categories[0].Tasks
		require.Len(t, tasks, 5)
		assert.Empty(t, tasks[0].ParentID)
		assert.Equal(t, tasks[1].ID, tasks[2].ParentID)
		assert.Empty(t, tasks[4].ParentID)
	})
	t.Run("a slash inside a top-level name is no path", func(t *testing.T) {
		project := domain.Project{Name: "P", Categories: []domain.Category{
			{ID: "ci", Name: "CI", Tasks: []domain.Task{{ID: "a", Title: "a", Status: domain.StatusTodo}}},
			{ID: "cd", Name: "CD", ParentID: "ci", Tasks: []domain.Task{{ID: "b", Title: "b", Status: domain.StatusTodo}}},
			{ID: "cicd", Name: "CI/CD", Tasks: []domain.Task{{ID: "c", Title: "c", Status: domain.StatusTodo}}},
		}}
		var buf bytes.Buffer
		require.NoError(t, ExportMarkdown(project, &buf))
		back, err := ImportMarkdown(&buf, "")
		require.NoError(t, err)
		assert.Equal(t, outline(project), outline(back))
	})
	t.Run("a name that reads like a path stays one", func(t *testing.T) {
		project := domain.Project{Name: "P", Categories: []domain.Category{
			{ID: "a", Name: "A", Tasks: []domain.Task{{ID: "a1", Title: "a", Status: domain.StatusTodo}}},
			{ID: "b", Name: "B", ParentID: "a", Tasks: []domain.Task{{ID: "b1", Title: "b", Status: domain.StatusTodo}}},
			{ID: "ab", Name: "A / B", Tasks: []domain.Task{{ID: "c1", Title: "c", Status: domain.StatusTodo}}},
			{ID: "odd", Name: `back\/slash`, Tasks: []domain.Task{{ID: "d1", Title: "d", Status: domain.StatusTodo}}},
		}}
		var buf bytes.Buffer
		require.NoError(t, ExportMarkdown(project, &buf))
		assert.Contains(t, buf.String(), "## A \\/ B\n")
		back, err := ImportMarkdown(&buf, "")
		require.NoError(t, err)
		assert.Equal(t, outline(project), outline(back))
	})
	t.Run("a hand-written path nests only under a known category", func(t *testing.T) {
		md := "## A\n- [ ] a\n## A / B\n- [ ] b\n## X / Y\n- [ ] c\n"
		p, err := ImportMarkdown(strings.NewReader(md), "")
		require.NoError(t, err)
		require.Len(t, p.Categories, 3)
		assert.Equal(t, p.Categories[0].ID, p.Categories[1].ParentID)
		assert.Equal(t, "B", p.Categories[1].Name)
		assert.Equal(t, "X / Y", p.Categories[2].Name)
		assert.Empty(t, p.Categories[2].ParentID)
	})
	t.Run("a checkbox inside a description stays description", func(t *testing.T) {
		md := "## A\n- [ ] parent\n    - [ ] not a subtask\n"
		p, err := ImportMarkdown(strings.NewReader(md), "")
		require.NoError(t, err)
		require.Len(t, p.Categories[0].Tasks, 1)
		assert.Equal(t, "- [ ] not a subtask", p.Categories[0].Tasks[0].Description)
	})
}
