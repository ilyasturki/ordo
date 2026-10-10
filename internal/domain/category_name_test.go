package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func namedCats(names ...string) []Category {
	cats := make([]Category, len(names))
	for i, n := range names {
		cats[i] = Category{ID: n, Name: n}
	}
	return cats
}

func TestCategoryNameConflict(t *testing.T) {
	cats := namedCats("v0.2.0", "next")
	assert.Equal(t, 0, CategoryNameConflict(cats, "", "  V0.2.0 ", -1))
	assert.Equal(t, -1, CategoryNameConflict(cats, "", "v0.2.0", 0), "a category never conflicts with itself")
	assert.Equal(t, 0, CategoryNameConflict(cats, "", "v0.2.0", 1))
	assert.Equal(t, -1, CategoryNameConflict(cats, "", "v0.3.0", -1))
}

func TestUniqueCategoryName(t *testing.T) {
	assert.Equal(t, "v0.3.0", UniqueCategoryName(namedCats("v0.2.0"), "", "v0.3.0"))
	assert.Equal(t, "v0.2.0 (2)", UniqueCategoryName(namedCats("v0.2.0"), "", "v0.2.0"))
	assert.Equal(t, "v0.2.0 (3)", UniqueCategoryName(namedCats("v0.2.0", "V0.2.0 (2)"), "", "v0.2.0"))
	assert.Equal(t, "v0.2.0", UniqueCategoryName(namedCats("v0.2.0"), "v1", "v0.2.0"), "only siblings count")
}

func TestRenameAndAddShareTheRule(t *testing.T) {
	p := Project{Categories: namedCats("v0.2.0", "next")}
	assert.ErrorIs(t, p.RenameCategory(1, " V0.2.0"), ErrDuplicateCategoryName)
	_, err := p.AddCategoryNamed("NEXT")
	assert.ErrorIs(t, err, ErrDuplicateCategoryName)
	assert.NoError(t, p.RenameCategory(0, "V0.2.0"), "renaming to a case variant of itself is fine")
}

func TestMergeCategory(t *testing.T) {
	t.Run("moves tasks under a separator named after the source", func(t *testing.T) {
		p := Project{Categories: []Category{
			{ID: "dst", Name: "v0.2.0", Tasks: []Task{{ID: "a"}}},
			{ID: "src", Name: " next ", Tasks: []Task{{ID: "b"}, {ID: "c"}}},
		}}
		require.NoError(t, p.MergeCategory(1, 0))

		require.Len(t, p.Categories, 1)
		tasks := p.Categories[0].Tasks
		require.Len(t, tasks, 4)
		assert.Equal(t, "a", tasks[0].ID)
		assert.True(t, tasks[1].IsSeparator())
		assert.Equal(t, "next", tasks[1].Title)
		assert.Equal(t, "b", tasks[2].ID)
		assert.Equal(t, "c", tasks[3].ID)
	})

	t.Run("an empty source just goes", func(t *testing.T) {
		p := Project{Categories: []Category{
			{ID: "src", Name: "v0.2.0"},
			{ID: "dst", Name: "v0.2.0", Tasks: []Task{{ID: "a"}}},
		}}
		require.NoError(t, p.MergeCategory(0, 1))
		require.Len(t, p.Categories, 1)
		assert.Equal(t, "dst", p.Categories[0].ID)
		assert.Len(t, p.Categories[0].Tasks, 1)
	})

	t.Run("sub-categories join the target's, merging by name", func(t *testing.T) {
		p := Project{Categories: []Category{
			{ID: "dst", Name: "v0.2.0"},
			{ID: "ui", Name: "ui", ParentID: "dst", Tasks: []Task{{ID: "a"}}},
			{ID: "src", Name: "v0.2.0b"},
			{ID: "ui2", Name: "UI", ParentID: "src", Tasks: []Task{{ID: "b"}}},
			{ID: "api", Name: "api", ParentID: "src", Tasks: []Task{{ID: "c"}}},
			{ID: "next", Name: "next"},
		}}
		require.NoError(t, p.MergeCategory(2, 0))

		var shape []string
		for _, c := range p.Categories {
			shape = append(shape, c.ID+"<"+c.ParentID)
		}
		assert.Equal(t, []string{"dst<", "ui<dst", "api<dst", "next<"}, shape)
		ui := p.Categories[1].Tasks
		require.Len(t, ui, 3)
		assert.Equal(t, "a", ui[0].ID)
		assert.Equal(t, "UI", ui[1].Title)
		assert.Equal(t, "b", ui[2].ID)
	})

	t.Run("rejects bad indexes", func(t *testing.T) {
		p := Project{Categories: namedCats("a", "b")}
		assert.Error(t, p.MergeCategory(0, 0))
		assert.Error(t, p.MergeCategory(0, 2))
		assert.Error(t, p.MergeCategory(-1, 1))
	})
}
