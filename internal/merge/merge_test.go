package merge

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/domain"
)

func row(id, title string) domain.Task {
	return domain.Task{ID: id, Title: title, Status: domain.StatusTodo, UpdatedAt: "2026-01-01T00:00:00Z"}
}

func project(rows ...domain.Task) domain.Project {
	return domain.Project{ID: "p", Name: "P", Categories: []domain.Category{{ID: "c", Name: "C", Tasks: rows}}}
}

func edit(p domain.Project, fn func(c *domain.Category)) domain.Project {
	out := Clone(p)
	fn(&out.Categories[0])
	return out
}

func titles(p domain.Project) []string {
	var out []string
	for _, c := range p.Categories {
		for _, t := range c.Tasks {
			out = append(out, t.Title)
		}
	}
	return out
}

func TestDifferentRowsAndFieldsMergeWithoutACollision(t *testing.T) {
	base := project(row("a", "A"), row("b", "B"))
	mine := edit(base, func(c *domain.Category) { c.Tasks[0].Title = "A mine" })
	theirs := edit(base, func(c *domain.Category) {
		c.Tasks[0].Status = domain.StatusCompleted
		c.Tasks[1].Title = "B theirs"
	})

	res := Three(base, mine, theirs)

	assert.Empty(t, res.Collisions)
	assert.Equal(t, []string{"A mine", "B theirs"}, titles(res.Project))
	assert.Equal(t, domain.StatusCompleted, res.Project.Categories[0].Tasks[0].Status)
}

func TestTheSameValueOnBothSidesIsNoCollision(t *testing.T) {
	base := project(row("a", "A"))
	both := edit(base, func(c *domain.Category) { c.Tasks[0].Title = "Same" })
	assert.Empty(t, Three(base, both, both).Collisions)
}

func TestSameFieldCollidesAndHoldsTheirsUntilResolved(t *testing.T) {
	base := project(row("a", "A"))
	mine := edit(base, func(c *domain.Category) { c.Tasks[0].Title = "Mine" })
	theirs := edit(base, func(c *domain.Category) { c.Tasks[0].Title = "Theirs" })

	res := Three(base, mine, theirs)

	require.Len(t, res.Collisions, 1)
	got := res.Collisions[0]
	assert.Equal(t, "title", got.Field)
	assert.Equal(t, "Mine", got.Mine)
	assert.Equal(t, "Theirs", got.Theirs)
	assert.Equal(t, []string{"Theirs"}, titles(res.Project))

	kept, err := res.Resolve([]Pick{PickMine})
	require.NoError(t, err)
	assert.Equal(t, []string{"Mine"}, titles(kept))

	both, err := res.Resolve([]Pick{PickBoth})
	require.NoError(t, err)
	assert.Equal(t, []string{"Theirs", "Mine"}, titles(both))
	assert.NotEqual(t, "a", both.Categories[0].Tasks[1].ID)
	assert.Equal(t, []string{"Theirs"}, titles(res.Project), "resolving leaves the result as it was")
}

func TestOneCopyPerRowHoweverManyFieldsCollide(t *testing.T) {
	base := project(row("a", "A"))
	mine := edit(base, func(c *domain.Category) { c.Tasks[0].Title, c.Tasks[0].Description = "Mine", "mine" })
	theirs := edit(base, func(c *domain.Category) { c.Tasks[0].Title, c.Tasks[0].Description = "Theirs", "theirs" })

	res := Three(base, mine, theirs)
	require.Len(t, res.Collisions, 2)
	both, err := res.Resolve([]Pick{PickBoth, PickBoth})
	require.NoError(t, err)
	assert.Equal(t, []string{"Theirs", "Mine"}, titles(both))
}

func TestStatusTakesItsCompletionDateAlong(t *testing.T) {
	base := project(row("a", "A"))
	mine := edit(base, func(c *domain.Category) {
		c.Tasks[0].Status, c.Tasks[0].CompletionDate = domain.StatusCompleted, "2026-02-02T00:00:00Z"
	})
	res := Three(base, mine, base)
	assert.Equal(t, "2026-02-02T00:00:00Z", res.Project.Categories[0].Tasks[0].CompletionDate)
}

func TestAnEditOutlivesADelete(t *testing.T) {
	base := project(row("a", "A"), row("b", "B"))
	edited := edit(base, func(c *domain.Category) { c.Tasks[0].Title = "A edited" })
	deleted := edit(base, func(c *domain.Category) { c.Tasks = c.Tasks[1:] })

	assert.Equal(t, []string{"A edited", "B"}, titles(Three(base, edited, deleted).Project))
	assert.Equal(t, []string{"A edited", "B"}, titles(Three(base, deleted, edited).Project))
	assert.Equal(t, []string{"B"}, titles(Three(base, deleted, base).Project))
	assert.Equal(t, []string{"B"}, titles(Three(base, base, deleted).Project))
}

func TestRowsAddedOnBothSidesAllStayNearTheirNeighbours(t *testing.T) {
	base := project(row("a", "A"), row("b", "B"))
	mine := edit(base, func(c *domain.Category) { c.Tasks = []domain.Task{c.Tasks[0], row("m", "M"), c.Tasks[1]} })
	theirs := edit(base, func(c *domain.Category) { c.Tasks = append(c.Tasks, row("t", "T")) })

	assert.Equal(t, []string{"A", "M", "B", "T"}, titles(Three(base, mine, theirs).Project))
}

func TestAMoveLandsAmongRowsTheirsAdded(t *testing.T) {
	base := project(row("a", "A"), row("b", "B"), row("c", "C"), row("d", "D"))
	mine := edit(base, func(c *domain.Category) {
		c.Tasks = []domain.Task{c.Tasks[0], c.Tasks[3], c.Tasks[1], c.Tasks[2]}
	})
	theirs := edit(base, func(c *domain.Category) {
		c.Tasks = []domain.Task{c.Tasks[0], c.Tasks[1], row("x", "X"), c.Tasks[2], c.Tasks[3]}
	})

	assert.Equal(t, []string{"A", "D", "B", "X", "C"}, titles(Three(base, mine, theirs).Project))
}

func TestMoveToAnotherCategoryMergesWithAnEdit(t *testing.T) {
	base := domain.Project{ID: "p", Name: "P", Categories: []domain.Category{
		{ID: "c1", Name: "One", Tasks: []domain.Task{row("a", "A")}},
		{ID: "c2", Name: "Two", Tasks: []domain.Task{}},
	}}
	mine := Clone(base)
	mine.Categories[1].Tasks = mine.Categories[0].Tasks
	mine.Categories[0].Tasks = []domain.Task{}
	theirs := Clone(base)
	theirs.Categories[0].Tasks[0].Title = "A theirs"

	got := Three(base, mine, theirs).Project

	assert.Empty(t, got.Categories[0].Tasks)
	require.Len(t, got.Categories[1].Tasks, 1)
	assert.Equal(t, "A theirs", got.Categories[1].Tasks[0].Title)
}

func TestACategoryDeletedWhileARowWasAddedStays(t *testing.T) {
	base := domain.Project{ID: "p", Name: "P", Categories: []domain.Category{
		{ID: "c1", Name: "One", Tasks: []domain.Task{}},
		{ID: "c2", Name: "Two", Tasks: []domain.Task{}},
	}}
	mine := Clone(base)
	mine.Categories = mine.Categories[1:]
	theirs := Clone(base)
	theirs.Categories[0].Tasks = []domain.Task{row("n", "New")}

	got := Three(base, mine, theirs).Project

	require.Len(t, got.Categories, 2)
	assert.Equal(t, []string{"New"}, titles(got))
}

func TestCategoriesAddedUnderOneNameJoin(t *testing.T) {
	base := domain.Project{ID: "p", Name: "P", Categories: []domain.Category{}}
	mine := Clone(base)
	mine.Categories = []domain.Category{{ID: "m", Name: "v2", Tasks: []domain.Task{row("a", "A")}}}
	theirs := Clone(base)
	theirs.Categories = []domain.Category{{ID: "t", Name: "V2 ", Tasks: []domain.Task{row("b", "B")}}}

	got := Three(base, mine, theirs).Project

	require.Len(t, got.Categories, 1)
	assert.Equal(t, "t", got.Categories[0].ID)
	assert.ElementsMatch(t, []string{"A", "B"}, titles(got))
}

func TestACopyOfAParentLandsAfterItsSubtasks(t *testing.T) {
	sub := row("s", "Sub")
	sub.ParentID = "a"
	base := project(row("a", "A"), sub)
	mine := edit(base, func(c *domain.Category) { c.Tasks[0].Title = "Mine" })
	theirs := edit(base, func(c *domain.Category) { c.Tasks[0].Title = "Theirs" })

	both, err := Three(base, mine, theirs).Resolve([]Pick{PickBoth})
	require.NoError(t, err)
	assert.Equal(t, []string{"Theirs", "Sub", "Mine"}, titles(both))
}

func TestProjectNameCollidesWithNoCopyOnOffer(t *testing.T) {
	base := project()
	mine, theirs := Clone(base), Clone(base)
	mine.Name, theirs.Name = "Mine", "Theirs"

	res := Three(base, mine, theirs)
	require.Len(t, res.Collisions, 1)
	assert.False(t, res.Collisions[0].CanCopy())
	p, err := res.Resolve([]Pick{PickMine})
	require.NoError(t, err)
	assert.Equal(t, "Mine", p.Name)
}

func TestAGroupSetOnOneSideSurvivesAnEditOnTheOther(t *testing.T) {
	base := project(row("a", "A"))
	mine := Clone(base)
	mine.Group = "Work"
	theirs := edit(base, func(c *domain.Category) { c.Tasks[0].Title = "A theirs" })

	res := Three(base, mine, theirs)
	assert.Empty(t, res.Collisions)
	assert.Equal(t, "Work", res.Project.Group)
	assert.Equal(t, []string{"A theirs"}, titles(res.Project))

	other := Clone(base)
	other.Group = "Perso"
	res = Three(base, mine, other)
	require.Len(t, res.Collisions, 1)
	assert.Equal(t, "group", res.Collisions[0].Field)
	assert.False(t, Same(base, mine))
}

func TestHoldsNeedsTheRowItself(t *testing.T) {
	base := project(row("a", "A"), row("b", "B"))
	wrote := edit(base, func(c *domain.Category) {
		c.Tasks[0].Status, c.Tasks[0].UpdatedAt = domain.StatusCompleted, "2026-03-03T10:00:00Z"
		c.Tasks[1].Title, c.Tasks[1].UpdatedAt = "B mine", "2026-03-03T10:00:01Z"
	})

	built := edit(wrote, func(c *domain.Category) { c.Tasks[1].Title = "B agent" })
	assert.True(t, Holds(base, wrote, built))

	twin := edit(base, func(c *domain.Category) {
		c.Tasks[0].Status, c.Tasks[0].UpdatedAt = domain.StatusCompleted, "2026-03-03T09:00:00Z"
	})
	assert.False(t, Holds(base, wrote, twin), "the same value at another time is another writer")
	assert.False(t, Holds(base, wrote, base))
}

func TestSameIgnoresStamps(t *testing.T) {
	a := project(row("a", "A"))
	b := edit(a, func(c *domain.Category) { c.Tasks[0].UpdatedAt = "2030-01-01T00:00:00Z" })
	b.UpdatedAt, b.Schema = "2030-01-01T00:00:00Z", 2
	assert.True(t, Same(a, b))
	assert.False(t, Same(a, edit(a, func(c *domain.Category) { c.Tasks[0].Title = "B" })))
}
