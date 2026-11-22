package app

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/domain"
)

// sectionsProject is Cat A [t1, s1, n1, t2 done, s2, t3, n2, s4] and Cat B [s3, t4].
func sectionsProject() domain.Project {
	return domain.Project{
		ID: "p1", Name: "P",
		Categories: []domain.Category{
			{ID: "c1", Name: "Cat A", Tasks: []domain.Task{
				{ID: "t1", Title: "one", Status: domain.StatusTodo},
				{ID: "s1", Kind: domain.KindSeparator, Title: "done ones"},
				{ID: "n1", Kind: domain.KindNote, Title: "about them"},
				{ID: "t2", Title: "two", Status: domain.StatusCompleted, TagColor: domain.TagBlue},
				{ID: "s2", Kind: domain.KindSeparator, Title: "open ones"},
				{ID: "t3", Title: "three", Status: domain.StatusTodo},
				{ID: "n2", Kind: domain.KindNote, Title: "under three"},
				{ID: "s4", Kind: domain.KindSeparator, Title: "later"},
			}},
			{ID: "c2", Name: "Cat B", Tasks: []domain.Task{
				{ID: "s3", Kind: domain.KindSeparator},
				{ID: "t4", Title: "four", Status: domain.StatusTodo},
			}},
		},
	}
}

func shownIDs(p domain.Project, f FilterState) []string {
	var out []string
	for _, cat := range p.Categories {
		for i, shown := range shownRows(cat, &f, nil) {
			if shown {
				out = append(out, cat.Tasks[i].ID)
			}
		}
	}
	return out
}

func TestLinesFollowTheirSectionUnderTheFilter(t *testing.T) {
	cases := []struct {
		name  string
		set   func(p *domain.Project, f *FilterState)
		shown []string
	}{
		{"status", func(_ *domain.Project, f *FilterState) { f.statuses[domain.StatusTodo] = true },
			[]string{"t1", "s2", "t3", "n2", "s3", "t4"}},
		{"another status", func(_ *domain.Project, f *FilterState) { f.statuses[domain.StatusCompleted] = true },
			[]string{"s1", "n1", "t2"}},
		{"priority matches no line by itself", func(_ *domain.Project, f *FilterState) { f.priorities[domain.PriorityHigh] = true },
			nil},
		{"a line's own tag", func(p *domain.Project, f *FilterState) {
			f.tags[domain.TagGreen] = true
			p.Categories[0].Tasks[7].TagColor = domain.TagGreen
			p.Categories[0].Tasks[6].TagColor = domain.TagGreen
		}, []string{"n2", "s4"}},
		{"its own tag under a status too", func(p *domain.Project, f *FilterState) {
			f.statuses[domain.StatusTodo] = true
			f.tags[domain.TagGreen] = true
			p.Categories[0].Tasks[7].TagColor = domain.TagGreen
		}, []string{"s4"}},
		{"untagged matches no line by itself", func(_ *domain.Project, f *FilterState) { f.tags[""] = true },
			[]string{"t1", "s2", "t3", "n2", "s3", "t4"}},
		{"a category alone keeps all its lines", func(_ *domain.Project, f *FilterState) { f.categories["c1"] = true },
			[]string{"t1", "s1", "n1", "t2", "s2", "t3", "n2", "s4"}},
		{"a tagged line outside the kept category", func(p *domain.Project, f *FilterState) {
			f.categories["c1"] = true
			f.tags[domain.TagGreen] = true
			p.Categories[1].Tasks[0].TagColor = domain.TagGreen
		}, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := sectionsProject()
			f := NewFilterState()
			c.set(&p, &f)
			assert.Equal(t, c.shown, shownIDs(p, f))
		})
	}
}

func TestLinesShowWithNoFilterOrDuringAPeek(t *testing.T) {
	p := sectionsProject()
	f := NewFilterState()
	f.statuses[domain.StatusTodo] = true
	f.startPeek("c1")
	assert.Equal(t, []string{"t1", "s2", "t3", "n2", "s3", "t4"}, shownIDs(p, f), "the category's peek leaves its sections' lines closed")
	f.startSectionPeek("c1", "s1")
	assert.Equal(t, []string{"t1", "n1", "t2", "s2", "t3", "n2", "s3", "t4"}, shownIDs(p, f), "a section's peek shows its rows under its line")
	f.endPeek()
	f.off = true
	assert.Len(t, shownIDs(p, f), 10)
}

func TestASectionTheFilterHidesIsOneLine(t *testing.T) {
	m := newTestModel(t, sectionsProject())
	m.ui.Filter.statuses[domain.StatusTodo] = true
	m.rebuildPositions()
	assert.Equal(t, []string{"project:", "category:c1", "task:t1", "section:s1", "separator:s2", "task:t3", "separator:n2", "section:s4", "category:c2", "separator:s3", "task:t4"}, rowIDs(m))
	cat := m.project.Categories[0]
	assert.Equal(t, 2, sectionHidden(cat, &m.ui.Filter, 1), "a note and a task")
	assert.Equal(t, 0, sectionHidden(cat, &m.ui.Filter, 7), "an empty section")
	assert.Equal(t, 0, hiddenByFilter(cat, &m.ui.Filter), "t2 counts on its section's line")
	lines := screen(m)
	assert.Contains(t, lineWith(t, lines, "done ones · 2 hidden"), "── done ones · 2 hidden ─")
	assert.Contains(t, lineWith(t, lines, "later"), "── later ─")
	assert.NotContains(t, strings.Join(lines, "\n"), "hidden by the filter")
}

func TestEveryHiddenTaskCountsOnce(t *testing.T) {
	p := sectionsProject()
	p.Categories[0].Tasks = append(p.Categories[0].Tasks[:6], append([]domain.Task{{ID: "t5", Title: "five", Status: domain.StatusCompleted}}, p.Categories[0].Tasks[6:]...)...)
	f := NewFilterState()
	f.statuses[domain.StatusTodo] = true
	cat := p.Categories[0]
	shown := filterRows(cat, &f)
	inLines := 0
	for sep, end := range hiddenSections(cat, &f, shown) {
		for _, task := range cat.Tasks[sep+1 : end] {
			if task.IsTask() {
				inLines++
			}
		}
	}
	assert.Equal(t, 1, inLines, "t2 behind its section's line")
	assert.Equal(t, 1, hiddenByFilter(cat, &f), "t5 among shown rows")

	f.startPeek("c1")
	assert.Equal(t, []string{"t1", "s2", "t3", "t5", "n2"}, shownIDs(domain.Project{Categories: []domain.Category{cat}}, f), "the category's peek shows t5 alone")
}

func TestAddLineUnderAFilterWhenItWouldShow(t *testing.T) {
	m := newTestModel(t, sectionsProject())
	m.ui.Filter.statuses[domain.StatusTodo] = true
	m.rebuildPositions()

	require.True(t, m.selectRow("t1"))
	m, _ = pressKeys(m, "-")
	assert.Equal(t, "Clear the filter to add a separator here", m.ui.Screen.StatusMsg, "no shown task would follow it")
	assert.Len(t, m.project.Categories[0].Tasks, 8)

	depth := m.ui.History.UndoDepth()
	require.True(t, m.selectRow("s2"))
	m, _ = pressKeys(m, "-")
	require.Len(t, m.project.Categories[0].Tasks, 9, "t3 follows it")
	assert.True(t, m.project.Categories[0].Tasks[5].IsSeparator())
	assert.Equal(t, depth+1, m.ui.History.UndoDepth())
	assert.Equal(t, "separator:"+m.project.Categories[0].Tasks[5].ID, cursorItem(t, m))

	require.True(t, m.selectRow("t3"))
	m, _ = pressKeys(m, "_", "o", "k", "enter")
	assert.Equal(t, "ok", m.project.Categories[0].Tasks[7].Title, "a note in a shown section")
	assert.True(t, m.project.Categories[0].Tasks[7].IsNote())

	require.True(t, m.selectRow("t4"))
	m, _ = pressKeys(m, "_")
	assert.True(t, m.ui.Modes.IsEdit())
}

func TestUntaggingAMatchedLineHoldsItUntilTheCursorLeaves(t *testing.T) {
	p := sectionsProject()
	p.Categories[0].Tasks[7].TagColor = domain.TagGreen
	m := newTestModel(t, p)
	m.ui.Filter.tags[domain.TagGreen] = true
	m.rebuildPositions()
	require.True(t, m.selectRow("s4"))

	m, _ = pressKeys(m, "t")
	require.NotEqual(t, domain.TagGreen, m.taskByID("s4").TagColor)
	assert.Contains(t, rowIDs(m), "separator:s4", "held where the edit left it")
	assert.Equal(t, "Hidden by the filter once you move off it", m.ui.Screen.StatusMsg)

	m, _ = pressKeys(m, "k")
	assert.NotContains(t, rowIDs(m), "separator:s4")
}

func TestAHeldTaskStaysWhileTheCursorIsOnItsDescription(t *testing.T) {
	p := sectionsProject()
	p.Categories[0].Tasks[5].Description = "details"
	m := newTestModel(t, p)
	m.ui.Screen.ExpandDescriptions = true
	m.ui.Filter.statuses[domain.StatusTodo] = true
	m.rebuildPositions()
	require.True(t, m.selectRow("t3"))

	m, _ = pressKeys(m, "space")
	require.NotEqual(t, domain.StatusTodo, m.taskByID("t3").Status)
	m, _ = pressKeys(m, "j")
	require.Equal(t, "description:t3", cursorItem(t, m))
	assert.Contains(t, rowIDs(m), "task:t3", "its own description keeps it held")
}
