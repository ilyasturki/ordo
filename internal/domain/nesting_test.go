package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// nestedTasks specs: "id", "id<parent", or "-" for a separator.
func nestedTasks(specs ...string) []Task {
	tasks := make([]Task, len(specs))
	for i, s := range specs {
		if s == "-" {
			tasks[i] = Task{ID: "sep", Kind: KindSeparator}
			continue
		}
		id, parent, _ := cutParent(s)
		tasks[i] = Task{ID: id, ParentID: parent, Title: id, Status: StatusTodo}
	}
	return tasks
}

func cutParent(s string) (id, parent string, nested bool) {
	for i := range s {
		if s[i] == '<' {
			return s[:i], s[i+1:], true
		}
	}
	return s, "", false
}

func taskShape(tasks []Task) []string {
	out := make([]string, len(tasks))
	for i, t := range tasks {
		out[i] = t.ID
		if t.ParentID != "" {
			out[i] += "<" + t.ParentID
		}
	}
	return out
}

func nestedCategories(specs ...string) []Category {
	cats := make([]Category, len(specs))
	for i, s := range specs {
		id, parent, _ := cutParent(s)
		cats[i] = Category{ID: id, Name: id, ParentID: parent}
	}
	return cats
}

func categoryShape(cats []Category) []string {
	out := make([]string, len(cats))
	for i, c := range cats {
		out[i] = c.ID
		if c.ParentID != "" {
			out[i] += "<" + c.ParentID
		}
	}
	return out
}

func TestBlockEndAndParentIndex(t *testing.T) {
	tasks := nestedTasks("a", "a1<a", "a2<a", "b", "-", "c", "c1<c")
	assert.Equal(t, 3, BlockEnd(tasks, 0), "a parent's block holds its subtasks")
	assert.Equal(t, 2, BlockEnd(tasks, 1), "a subtask is a block of its own")
	assert.Equal(t, 4, BlockEnd(tasks, 3))
	assert.Equal(t, 7, BlockEnd(tasks, 5))

	assert.Equal(t, -1, ParentIndex(tasks, 0))
	assert.Equal(t, 0, ParentIndex(tasks, 2))
	assert.Equal(t, 5, ParentIndex(tasks, 6))
	assert.Equal(t, 0, BlockStart(tasks, 2))
	assert.Equal(t, 3, BlockStart(tasks, 3))
	assert.True(t, HasChildren(tasks, 0))
	assert.False(t, HasChildren(tasks, 3))
}

func TestIndentTask(t *testing.T) {
	t.Run("nests under the task above as its last subtask", func(t *testing.T) {
		c := Category{Tasks: nestedTasks("a", "a1<a", "b")}
		require.NoError(t, c.IndentTasks(2, 3))
		assert.Equal(t, []string{"a", "a1<a", "b<a"}, taskShape(c.Tasks))
	})
	t.Run("refuses the first task", func(t *testing.T) {
		c := Category{Tasks: nestedTasks("a", "b")}
		assert.ErrorIs(t, c.IndentTasks(0, 1), ErrNothingToNest)
	})
	t.Run("refuses a subtask", func(t *testing.T) {
		c := Category{Tasks: nestedTasks("a", "a1<a", "a2<a")}
		assert.ErrorIs(t, c.IndentTasks(2, 3), ErrNestDepth)
	})
	t.Run("refuses a parent", func(t *testing.T) {
		c := Category{Tasks: nestedTasks("a", "b", "b1<b")}
		assert.ErrorIs(t, c.IndentTasks(1, 2), ErrNestDepth)
	})
	t.Run("refuses under a separator", func(t *testing.T) {
		c := Category{Tasks: nestedTasks("a", "-", "b")}
		assert.ErrorIs(t, c.IndentTasks(2, 3), ErrNothingToNest)
	})
	t.Run("a separator never nests", func(t *testing.T) {
		c := Category{Tasks: nestedTasks("a", "-")}
		assert.ErrorIs(t, c.IndentTasks(1, 2), ErrSeparatorNest)
	})
}

func TestOutdentTask(t *testing.T) {
	t.Run("lands right after the parent's block", func(t *testing.T) {
		c := Category{Tasks: nestedTasks("a", "a1<a", "a2<a", "a3<a", "b")}
		at, err := c.OutdentTasks(1, 2)
		require.NoError(t, err)
		assert.Equal(t, 3, at)
		assert.Equal(t, []string{"a", "a2<a", "a3<a", "a1", "b"}, taskShape(c.Tasks))
	})
	t.Run("refuses a top-level task", func(t *testing.T) {
		c := Category{Tasks: nestedTasks("a", "b")}
		_, err := c.OutdentTasks(1, 2)
		assert.ErrorIs(t, err, ErrNotNested)
	})
}

func TestNestingRuns(t *testing.T) {
	t.Run("several tasks nest under the one above", func(t *testing.T) {
		c := Category{Tasks: nestedTasks("a", "b", "c", "d")}
		require.NoError(t, c.IndentTasks(1, 3))
		assert.Equal(t, []string{"a", "b<a", "c<a", "d"}, taskShape(c.Tasks))
	})
	t.Run("several subtasks come out together, in order", func(t *testing.T) {
		c := Category{Tasks: nestedTasks("a", "a1<a", "a2<a", "a3<a", "b")}
		at, err := c.OutdentTasks(1, 3)
		require.NoError(t, err)
		assert.Equal(t, 2, at)
		assert.Equal(t, []string{"a", "a3<a", "a1", "a2", "b"}, taskShape(c.Tasks))
	})
	t.Run("a run with a parent in it refuses to nest", func(t *testing.T) {
		c := Category{Tasks: nestedTasks("a", "b", "c", "c1<c")}
		assert.ErrorIs(t, c.IndentTasks(1, 4), ErrNestDepth)
	})
	t.Run("a run across two parents refuses to come out", func(t *testing.T) {
		c := Category{Tasks: nestedTasks("a", "a1<a", "b", "b1<b")}
		_, err := c.OutdentTasks(1, 4)
		assert.ErrorIs(t, err, ErrMixedLevels)
	})
}

func TestCoverBlocks(t *testing.T) {
	tasks := nestedTasks("a", "a1<a", "a2<a", "b", "c", "c1<c")
	lo, hi, err := CoverBlocks(tasks, []int{0, 3})
	require.NoError(t, err)
	assert.Equal(t, [2]int{0, 4}, [2]int{lo, hi}, "a parent brings its subtasks")

	_, _, err = CoverBlocks(tasks, []int{1, 2})
	assert.NoError(t, err, "subtasks of one parent")

	_, _, err = CoverBlocks(tasks, []int{2, 3})
	assert.ErrorIs(t, err, ErrMixedLevels)

	_, _, err = CoverBlocks(tasks, []int{0, 4})
	assert.ErrorIs(t, err, ErrNotContiguous)
}

func TestMoveTaskBlock(t *testing.T) {
	t.Run("a parent hops the next sibling block with its subtasks", func(t *testing.T) {
		c := Category{Tasks: nestedTasks("a", "a1<a", "b", "b1<b", "b2<b", "c")}
		at, ok := c.MoveTaskBlock(0, 1)
		require.True(t, ok)
		assert.Equal(t, 3, at)
		assert.Equal(t, []string{"b", "b1<b", "b2<b", "a", "a1<a", "c"}, taskShape(c.Tasks))

		at, ok = c.MoveTaskBlock(at, -1)
		require.True(t, ok)
		assert.Equal(t, 0, at)
		assert.Equal(t, []string{"a", "a1<a", "b", "b1<b", "b2<b", "c"}, taskShape(c.Tasks))
	})
	t.Run("a subtask moves among its siblings only", func(t *testing.T) {
		c := Category{Tasks: nestedTasks("a", "a1<a", "a2<a", "b")}
		at, ok := c.MoveTaskBlock(1, 1)
		require.True(t, ok)
		assert.Equal(t, 2, at)
		assert.Equal(t, []string{"a", "a2<a", "a1<a", "b"}, taskShape(c.Tasks))

		_, ok = c.MoveTaskBlock(2, 1)
		assert.False(t, ok, "the last subtask stays in its parent")
		_, ok = c.MoveTaskBlock(1, -1)
		assert.False(t, ok, "the first subtask stays in its parent")
	})
	t.Run("a separator is a sibling block", func(t *testing.T) {
		c := Category{Tasks: nestedTasks("a", "a1<a", "-")}
		at, ok := c.MoveTaskBlock(2, -1)
		require.True(t, ok)
		assert.Equal(t, 0, at)
		assert.Equal(t, []string{"sep", "a", "a1<a"}, taskShape(c.Tasks))
	})
}

func TestNormalizeNesting(t *testing.T) {
	t.Run("regroups a stray child after its parent", func(t *testing.T) {
		got := normalizeOutline(nestedTasks("a1<a", "b", "a", "b1<b"))
		assert.Equal(t, []string{"b", "b1<b", "a", "a1<a"}, taskShape(got))
	})
	t.Run("an orphan becomes top-level in place", func(t *testing.T) {
		got := normalizeOutline(nestedTasks("a", "x<gone", "b"))
		assert.Equal(t, []string{"a", "x", "b"}, taskShape(got))
	})
	t.Run("a grandchild becomes top-level", func(t *testing.T) {
		got := normalizeOutline(nestedTasks("a", "b<a", "c<b"))
		assert.Equal(t, []string{"a", "b<a", "c"}, taskShape(got))
	})
	t.Run("a cycle unwinds to top-level", func(t *testing.T) {
		got := normalizeOutline(nestedTasks("a<b", "b<a"))
		assert.Equal(t, []string{"a", "b"}, taskShape(got))
	})
	t.Run("separators never nest", func(t *testing.T) {
		tasks := nestedTasks("-", "a<sep", "b")
		tasks[2].ParentID = "a"
		tasks = append(tasks, Task{ID: "s2", Kind: KindSeparator, ParentID: "b"})
		got := normalizeOutline(tasks)
		assert.Equal(t, []string{"sep", "a", "b<a", "s2"}, taskShape(got))
	})
	t.Run("a valid outline is unchanged", func(t *testing.T) {
		tasks := nestedTasks("a", "a1<a", "-", "b", "b1<b")
		assert.Equal(t, taskShape(tasks), taskShape(normalizeOutline(tasks)))
	})
	t.Run("categories follow the same rules", func(t *testing.T) {
		p := Project{Categories: nestedCategories("s<v", "v", "w", "t<s")}
		p.Categories[1].Tasks = nestedTasks("a1<a", "a")
		NormalizeNesting(&p)
		assert.Equal(t, []string{"v", "s<v", "w", "t"}, categoryShape(p.Categories))
		assert.Equal(t, []string{"a", "a1<a"}, taskShape(p.Categories[0].Tasks))
	})
}

func TestRequiredSchema(t *testing.T) {
	p := Project{Categories: nestedCategories("v", "w")}
	p.Categories[0].Tasks = nestedTasks("a", "b")
	assert.Equal(t, 1, RequiredSchema(p))

	p.Categories[0].Tasks[1].ParentID = "a"
	assert.Equal(t, 2, RequiredSchema(p))

	p.Categories[0].Tasks[1].ParentID = ""
	p.Categories[1].ParentID = "v"
	assert.Equal(t, 2, RequiredSchema(p))
}

func TestCheckSchemaReadsVersionTwo(t *testing.T) {
	assert.NoError(t, CheckSchema(Project{Schema: 2}))
	assert.ErrorIs(t, CheckSchema(Project{Schema: 3}), ErrSchemaTooNew)
}

func TestCategoryNamesAreUniqueAmongSiblings(t *testing.T) {
	p := Project{Categories: nestedCategories("v1", "ui<v1", "v2", "ui<v2", "ui")}
	assert.Equal(t, -1, CategoryNameConflict(p.Categories, "v2", "api", -1))
	assert.Equal(t, 3, CategoryNameConflict(p.Categories, "v2", " UI ", -1))
	assert.Equal(t, -1, CategoryNameConflict(p.Categories, "v2", "ui", 3), "a category doesn't clash with itself")
	assert.Equal(t, 4, CategoryNameConflict(p.Categories, "", "ui", -1))

	assert.NoError(t, p.RenameCategory(1, "UI"), "renaming to its own name in another case")
	assert.ErrorIs(t, p.RenameCategory(0, "V2"), ErrDuplicateCategoryName)
	assert.NoError(t, p.RenameCategory(1, "v2"), "a sub-category may share a top-level name")
}

func TestAddCategoryUnder(t *testing.T) {
	p := Project{Categories: nestedCategories("v1", "ui<v1", "v2")}
	cat, err := p.AddCategoryUnder("v1", "api")
	require.NoError(t, err)
	assert.Equal(t, "v1", cat.ParentID)
	assert.Equal(t, []string{"v1", "ui<v1", cat.ID + "<v1", "v2"}, categoryShape(p.Categories))

	_, err = p.AddCategoryUnder("v1", "UI")
	assert.ErrorIs(t, err, ErrDuplicateCategoryName)
	_, err = p.AddCategoryUnder("ui", "deeper")
	assert.ErrorIs(t, err, ErrNestDepth)
	_, err = p.AddCategoryUnder("v2", "ui")
	assert.NoError(t, err, "the same name is fine under another parent")
}

func TestIndentOutdentCategory(t *testing.T) {
	t.Run("nests under the top-level category above", func(t *testing.T) {
		p := Project{Categories: nestedCategories("v1", "ui<v1", "v2", "ui")}
		require.NoError(t, p.IndentCategories(3, 4))
		assert.Equal(t, []string{"v1", "ui<v1", "v2", "ui<v2"}, categoryShape(p.Categories))
	})
	t.Run("refuses a name its new siblings carry", func(t *testing.T) {
		p := Project{Categories: nestedCategories("v1", "ui<v1", "ui")}
		assert.ErrorIs(t, p.IndentCategories(2, 3), ErrDuplicateCategoryName)
		assert.Equal(t, []string{"v1", "ui<v1", "ui"}, categoryShape(p.Categories))
	})
	t.Run("refuses a category with sub-categories", func(t *testing.T) {
		p := Project{Categories: nestedCategories("v1", "v2", "a<v2")}
		assert.ErrorIs(t, p.IndentCategories(1, 2), ErrNestDepth)
	})
	t.Run("outdent lands after the parent's block", func(t *testing.T) {
		p := Project{Categories: nestedCategories("v1", "a<v1", "b<v1", "v2")}
		at, err := p.OutdentCategories(1, 2)
		require.NoError(t, err)
		assert.Equal(t, 2, at)
		assert.Equal(t, []string{"v1", "b<v1", "a", "v2"}, categoryShape(p.Categories))
	})
	t.Run("outdent refuses a taken top-level name", func(t *testing.T) {
		p := Project{Categories: nestedCategories("v1", "v2<v1", "v2")}
		_, err := p.OutdentCategories(1, 2)
		assert.ErrorIs(t, err, ErrDuplicateCategoryName)
	})
}

func TestReverseCategoriesKeepsBlocks(t *testing.T) {
	p := Project{Categories: nestedCategories("v1", "a<v1", "b<v1", "v2", "v3", "c<v3")}
	p.ReverseCategories()
	assert.Equal(t, []string{"v3", "c<v3", "v2", "v1", "a<v1", "b<v1"}, categoryShape(p.Categories))
	p.ReverseCategories()
	assert.Equal(t, []string{"v1", "a<v1", "b<v1", "v2", "v3", "c<v3"}, categoryShape(p.Categories))
}

func TestMoveTaskAcrossCategoriesTakesSubtasks(t *testing.T) {
	p := Project{Categories: nestedCategories("v1", "v2")}
	p.Categories[0].Tasks = nestedTasks("a", "a1<a", "a2<a", "b")
	p.Categories[1].Tasks = nestedTasks("x")

	require.NoError(t, p.MoveTask(0, 0, 1))
	assert.Equal(t, []string{"b"}, taskShape(p.Categories[0].Tasks))
	assert.Equal(t, []string{"x", "a", "a1<a", "a2<a"}, taskShape(p.Categories[1].Tasks))

	require.NoError(t, p.MoveTask(1, 2, 0))
	assert.Equal(t, []string{"b", "a1"}, taskShape(p.Categories[0].Tasks), "a lone subtask lands top-level")
}

func TestProgressAndTreeCounts(t *testing.T) {
	c := Category{Tasks: nestedTasks("a", "a1<a", "a2<a", "a3<a", "b")}
	c.Tasks[1].Status = StatusCompleted
	c.Tasks[2].Status = StatusCancelled
	done, total := c.ChildProgress(0)
	assert.Equal(t, 2, done)
	assert.Equal(t, 3, total)
	done, total = c.ChildProgress(4)
	assert.Zero(t, done)
	assert.Zero(t, total)

	p := Project{Categories: nestedCategories("v", "s<v", "w")}
	p.Categories[1].Tasks = nestedTasks("x")
	p.Categories[1].Tasks[0].Status = StatusInProgress
	assert.Equal(t, 1, p.TreeStatusCounts(0).InProgress)
	assert.Equal(t, StatusInProgress, p.TreeAggregateStatus(0))
	assert.Equal(t, "", p.Categories[0].AggregateStatus(), "the parent's own list is empty")
	assert.Equal(t, "", p.TreeAggregateStatus(2))
}

func TestReleaseAndRemoveBlocks(t *testing.T) {
	c := Category{Tasks: nestedTasks("a", "a1<a", "a2<a", "b")}
	c.ReleaseSubtasks(0)
	require.NoError(t, c.RemoveTask(0))
	assert.Equal(t, []string{"a1", "a2", "b"}, taskShape(c.Tasks))

	c = Category{Tasks: nestedTasks("a", "a1<a", "b")}
	c.RemoveTaskBlock(0)
	assert.Equal(t, []string{"b"}, taskShape(c.Tasks))

	p := Project{Categories: nestedCategories("v", "ui<v", "w", "ui")}
	assert.ErrorIs(t, p.ReleaseSubCategories(0), ErrDuplicateCategoryName)
	p.Categories[3].Name = "other"
	require.NoError(t, p.ReleaseSubCategories(0))
	assert.Equal(t, []string{"v", "ui", "w", "ui"}, categoryShape(p.Categories))

	p = Project{Categories: nestedCategories("v", "ui<v", "w")}
	p.RemoveCategoryBlock(0)
	assert.Equal(t, []string{"w"}, categoryShape(p.Categories))
}

func TestCategoryPath(t *testing.T) {
	p := Project{Categories: nestedCategories("v0.2", "ui/ux<v0.2")}
	assert.Equal(t, "v0.2", p.CategoryPath(0))
	assert.Equal(t, "v0.2 / ui/ux", p.CategoryPath(1))

	assert.Equal(t, [][2]string{{"v0.2", "ui/ux"}, {"v0.2 / ui", "ux"}}, SplitCategoryPath("v0.2 / ui/ux"))
	assert.Empty(t, SplitCategoryPath("plain"))
	assert.Empty(t, SplitCategoryPath("/edge/"))
}

func TestAnIDLessItemHasNoChildren(t *testing.T) {
	tasks := nestedTasks("a", "b", "c")
	tasks[0].ID = ""
	assert.Equal(t, 1, BlockEnd(tasks, 0))
	assert.Equal(t, []string{"b", "c"}, taskShape(PickBlocks(tasks, map[string]bool{"b": true, "c": true})))

	p := Project{Categories: nestedCategories("", "h", "m")}
	require.NoError(t, p.MergeCategory(0, 2))
	assert.Equal(t, []string{"h", "m"}, categoryShape(p.Categories))
}

func TestPickBlocksKeepsTheOutlineOrder(t *testing.T) {
	tasks := nestedTasks("x", "b", "a<b", "c")
	picked := map[string]bool{"a": true, "b": true}
	assert.Equal(t, []string{"b", "a<b"}, taskShape(PickBlocks(tasks, picked)))
}

func TestMoveTaskRestampsASubtaskItUnnests(t *testing.T) {
	p := Project{Categories: nestedCategories("v1", "v2")}
	p.Categories[0].Tasks = nestedTasks("a", "a1<a")
	require.NoError(t, p.MoveTask(0, 1, 1))
	assert.NotEmpty(t, p.Categories[1].Tasks[0].UpdatedAt)
}

func TestTreeTaskCountTakesEveryStatus(t *testing.T) {
	p := Project{Categories: nestedCategories("v", "s<v")}
	p.Categories[0].Tasks = nestedTasks("a", "-")
	p.Categories[1].Tasks = nestedTasks("b")
	p.Categories[1].Tasks[0].Status = "blocked"
	assert.Equal(t, 2, p.TreeTaskCount(0))
}
