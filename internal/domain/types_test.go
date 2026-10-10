package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTask_IncreasePriority(t *testing.T) {
	t.Run("increases from trivial to low", func(t *testing.T) {
		task := Task{Priority: PriorityTrivial}
		changed := task.IncreasePriority()
		assert.True(t, changed)
		assert.Equal(t, PriorityLow, task.Priority)
	})

	t.Run("increases from low to medium", func(t *testing.T) {
		task := Task{Priority: PriorityLow}
		changed := task.IncreasePriority()
		assert.True(t, changed)
		assert.Equal(t, PriorityMedium, task.Priority)
	})

	t.Run("increases from medium to high", func(t *testing.T) {
		task := Task{Priority: PriorityMedium}
		changed := task.IncreasePriority()
		assert.True(t, changed)
		assert.Equal(t, PriorityHigh, task.Priority)
	})

	t.Run("increases from high to critical", func(t *testing.T) {
		task := Task{Priority: PriorityHigh}
		changed := task.IncreasePriority()
		assert.True(t, changed)
		assert.Equal(t, PriorityCritical, task.Priority)
	})

	t.Run("cannot increase from critical", func(t *testing.T) {
		task := Task{Priority: PriorityCritical}
		changed := task.IncreasePriority()
		assert.False(t, changed)
		assert.Equal(t, PriorityCritical, task.Priority)
	})

	t.Run("empty priority increases as medium", func(t *testing.T) {
		task := Task{Priority: ""}
		changed := task.IncreasePriority()
		assert.True(t, changed)
		assert.Equal(t, PriorityHigh, task.Priority)
	})
}

func TestTask_DecreasePriority(t *testing.T) {
	t.Run("decreases from critical to high", func(t *testing.T) {
		task := Task{Priority: PriorityCritical}
		changed := task.DecreasePriority()
		assert.True(t, changed)
		assert.Equal(t, PriorityHigh, task.Priority)
	})

	t.Run("decreases from high to medium", func(t *testing.T) {
		task := Task{Priority: PriorityHigh}
		changed := task.DecreasePriority()
		assert.True(t, changed)
		assert.Equal(t, PriorityMedium, task.Priority)
	})

	t.Run("decreases from medium to low", func(t *testing.T) {
		task := Task{Priority: PriorityMedium}
		changed := task.DecreasePriority()
		assert.True(t, changed)
		assert.Equal(t, PriorityLow, task.Priority)
	})

	t.Run("decreases from low to trivial", func(t *testing.T) {
		task := Task{Priority: PriorityLow}
		changed := task.DecreasePriority()
		assert.True(t, changed)
		assert.Equal(t, PriorityTrivial, task.Priority)
	})

	t.Run("cannot decrease from trivial", func(t *testing.T) {
		task := Task{Priority: PriorityTrivial}
		changed := task.DecreasePriority()
		assert.False(t, changed)
		assert.Equal(t, PriorityTrivial, task.Priority)
	})

	t.Run("empty priority decreases as medium", func(t *testing.T) {
		task := Task{Priority: ""}
		changed := task.DecreasePriority()
		assert.True(t, changed)
		assert.Equal(t, PriorityLow, task.Priority)
	})
}

func TestTask_CycleStatus(t *testing.T) {
	t.Run("cycles from todo to in_progress", func(t *testing.T) {
		task := Task{Status: StatusTodo}
		changed := task.CycleStatus()
		assert.True(t, changed)
		assert.Equal(t, StatusInProgress, task.Status)
	})

	t.Run("cycles from in_progress to completed", func(t *testing.T) {
		task := Task{Status: StatusInProgress}
		changed := task.CycleStatus()
		assert.True(t, changed)
		assert.Equal(t, StatusCompleted, task.Status)
	})

	t.Run("cycles from completed to cancelled", func(t *testing.T) {
		task := Task{Status: StatusCompleted}
		changed := task.CycleStatus()
		assert.True(t, changed)
		assert.Equal(t, StatusCancelled, task.Status)
	})

	t.Run("cycles from cancelled to todo", func(t *testing.T) {
		task := Task{Status: StatusCancelled}
		changed := task.CycleStatus()
		assert.True(t, changed)
		assert.Equal(t, StatusTodo, task.Status)
	})
}

func TestTask_CycleStatusReverse(t *testing.T) {
	t.Run("cycles from todo to cancelled", func(t *testing.T) {
		task := Task{Status: StatusTodo}
		changed := task.CycleStatusReverse()
		assert.True(t, changed)
		assert.Equal(t, StatusCancelled, task.Status)
	})

	t.Run("cycles from cancelled to completed", func(t *testing.T) {
		task := Task{Status: StatusCancelled}
		changed := task.CycleStatusReverse()
		assert.True(t, changed)
		assert.Equal(t, StatusCompleted, task.Status)
	})

	t.Run("cycles from completed to in_progress", func(t *testing.T) {
		task := Task{Status: StatusCompleted}
		changed := task.CycleStatusReverse()
		assert.True(t, changed)
		assert.Equal(t, StatusInProgress, task.Status)
	})

	t.Run("cycles from in_progress to todo", func(t *testing.T) {
		task := Task{Status: StatusInProgress}
		changed := task.CycleStatusReverse()
		assert.True(t, changed)
		assert.Equal(t, StatusTodo, task.Status)
	})
}

func TestCategory_AddTask(t *testing.T) {
	t.Run("adds task to category", func(t *testing.T) {
		cat := Category{Tasks: []Task{}}
		task := Task{ID: "test-1", Title: "Test Task"}
		cat.AddTask(task)
		assert.Len(t, cat.Tasks, 1)
		assert.Equal(t, "test-1", cat.Tasks[0].ID)
	})
}

func TestCategory_InsertTask(t *testing.T) {
	t.Run("inserts task at specified index", func(t *testing.T) {
		cat := Category{Tasks: []Task{
			{ID: "1"}, {ID: "3"},
		}}
		task := Task{ID: "2"}
		cat.InsertTask(1, task)
		assert.Len(t, cat.Tasks, 3)
		assert.Equal(t, "1", cat.Tasks[0].ID)
		assert.Equal(t, "2", cat.Tasks[1].ID)
		assert.Equal(t, "3", cat.Tasks[2].ID)
	})

	t.Run("inserts at beginning with index 0", func(t *testing.T) {
		cat := Category{Tasks: []Task{{ID: "2"}}}
		task := Task{ID: "1"}
		cat.InsertTask(0, task)
		assert.Len(t, cat.Tasks, 2)
		assert.Equal(t, "1", cat.Tasks[0].ID)
		assert.Equal(t, "2", cat.Tasks[1].ID)
	})

	t.Run("appends at end for out of range index", func(t *testing.T) {
		cat := Category{Tasks: []Task{{ID: "1"}}}
		task := Task{ID: "2"}
		cat.InsertTask(100, task)
		assert.Len(t, cat.Tasks, 2)
		assert.Equal(t, "2", cat.Tasks[1].ID)
	})

	t.Run("inserts at end for negative index (clamped)", func(t *testing.T) {
		cat := Category{Tasks: []Task{{ID: "1"}}}
		task := Task{ID: "2"}
		cat.InsertTask(-1, task)
		assert.Len(t, cat.Tasks, 2)
		assert.Equal(t, "2", cat.Tasks[1].ID)
	})
}

func TestCategory_RemoveTask(t *testing.T) {
	t.Run("removes task at valid index", func(t *testing.T) {
		cat := Category{Tasks: []Task{
			{ID: "1"}, {ID: "2"}, {ID: "3"},
		}}
		err := cat.RemoveTask(1)
		require.NoError(t, err)
		assert.Len(t, cat.Tasks, 2)
		assert.Equal(t, "1", cat.Tasks[0].ID)
		assert.Equal(t, "3", cat.Tasks[1].ID)
	})

	t.Run("returns error for negative index", func(t *testing.T) {
		cat := Category{Tasks: []Task{{ID: "1"}}}
		err := cat.RemoveTask(-1)
		assert.Error(t, err)
	})

	t.Run("returns error for index out of range", func(t *testing.T) {
		cat := Category{Tasks: []Task{{ID: "1"}}}
		err := cat.RemoveTask(5)
		assert.Error(t, err)
	})
}

func TestProject_AddCategory(t *testing.T) {
	t.Run("adds category to project", func(t *testing.T) {
		proj := Project{Categories: []Category{}}
		cat := Category{ID: "cat-1", Name: "Test"}
		proj.AddCategory(cat)
		assert.Len(t, proj.Categories, 1)
		assert.Equal(t, "cat-1", proj.Categories[0].ID)
	})
}

func TestProject_InsertCategory(t *testing.T) {
	t.Run("inserts category at specified index", func(t *testing.T) {
		proj := Project{Categories: []Category{
			{ID: "1"}, {ID: "3"},
		}}
		cat := Category{ID: "2"}
		proj.InsertCategory(1, cat)
		assert.Len(t, proj.Categories, 3)
		assert.Equal(t, "1", proj.Categories[0].ID)
		assert.Equal(t, "2", proj.Categories[1].ID)
		assert.Equal(t, "3", proj.Categories[2].ID)
	})

	t.Run("appends at end for out of range index", func(t *testing.T) {
		proj := Project{Categories: []Category{{ID: "1"}}}
		cat := Category{ID: "2"}
		proj.InsertCategory(100, cat)
		assert.Len(t, proj.Categories, 2)
		assert.Equal(t, "2", proj.Categories[1].ID)
	})

	t.Run("inserts at end for negative index (clamped)", func(t *testing.T) {
		proj := Project{Categories: []Category{{ID: "1"}}}
		cat := Category{ID: "2"}
		proj.InsertCategory(-1, cat)
		assert.Len(t, proj.Categories, 2)
		assert.Equal(t, "2", proj.Categories[1].ID)
	})
}

func TestProject_RemoveCategory(t *testing.T) {
	t.Run("removes category at valid index", func(t *testing.T) {
		proj := Project{Categories: []Category{
			{ID: "1"}, {ID: "2"}, {ID: "3"},
		}}
		err := proj.RemoveCategory(1)
		require.NoError(t, err)
		assert.Len(t, proj.Categories, 2)
		assert.Equal(t, "1", proj.Categories[0].ID)
		assert.Equal(t, "3", proj.Categories[1].ID)
	})

	t.Run("returns error for negative index", func(t *testing.T) {
		proj := Project{Categories: []Category{{ID: "1"}}}
		err := proj.RemoveCategory(-1)
		assert.Error(t, err)
	})

	t.Run("returns error for index out of range", func(t *testing.T) {
		proj := Project{Categories: []Category{{ID: "1"}}}
		err := proj.RemoveCategory(5)
		assert.Error(t, err)
	})
}

func TestProject_ReverseCategories(t *testing.T) {
	t.Run("reverses order of multiple categories", func(t *testing.T) {
		proj := Project{Categories: []Category{
			{ID: "1"}, {ID: "2"}, {ID: "3"},
		}}
		proj.ReverseCategories()
		assert.Equal(t, []string{"3", "2", "1"},
			[]string{proj.Categories[0].ID, proj.Categories[1].ID, proj.Categories[2].ID})
	})

	t.Run("keeps task order within each category", func(t *testing.T) {
		proj := Project{Categories: []Category{
			{ID: "1", Tasks: []Task{{ID: "a"}, {ID: "b"}}},
			{ID: "2", Tasks: []Task{{ID: "c"}}},
		}}
		proj.ReverseCategories()
		assert.Equal(t, "2", proj.Categories[0].ID)
		assert.Equal(t, "1", proj.Categories[1].ID)
		assert.Equal(t, []string{"a", "b"},
			[]string{proj.Categories[1].Tasks[0].ID, proj.Categories[1].Tasks[1].ID})
	})

	t.Run("is its own inverse", func(t *testing.T) {
		proj := Project{Categories: []Category{
			{ID: "1"}, {ID: "2"}, {ID: "3"}, {ID: "4"},
		}}
		proj.ReverseCategories()
		proj.ReverseCategories()
		assert.Equal(t, []string{"1", "2", "3", "4"},
			[]string{proj.Categories[0].ID, proj.Categories[1].ID, proj.Categories[2].ID, proj.Categories[3].ID})
	})

	t.Run("no-op for zero or one category", func(t *testing.T) {
		empty := Project{Categories: []Category{}, UpdatedAt: "before"}
		empty.ReverseCategories()
		assert.Empty(t, empty.Categories)
		assert.Equal(t, "before", empty.UpdatedAt, "UpdatedAt untouched when nothing moves")

		single := Project{Categories: []Category{{ID: "1"}}, UpdatedAt: "before"}
		single.ReverseCategories()
		assert.Len(t, single.Categories, 1)
		assert.Equal(t, "before", single.UpdatedAt, "UpdatedAt untouched when nothing moves")
	})
}

func TestCategory_AggregateStatus(t *testing.T) {
	t.Run("empty category returns empty string", func(t *testing.T) {
		cat := Category{Tasks: []Task{}}
		assert.Equal(t, "", cat.AggregateStatus())
	})

	t.Run("all todo returns todo", func(t *testing.T) {
		cat := Category{Tasks: []Task{
			{Status: StatusTodo}, {Status: StatusTodo},
		}}
		assert.Equal(t, StatusTodo, cat.AggregateStatus())
	})

	t.Run("all completed returns completed", func(t *testing.T) {
		cat := Category{Tasks: []Task{
			{Status: StatusCompleted}, {Status: StatusCompleted},
		}}
		assert.Equal(t, StatusCompleted, cat.AggregateStatus())
	})

	t.Run("all cancelled returns completed", func(t *testing.T) {
		cat := Category{Tasks: []Task{
			{Status: StatusCancelled}, {Status: StatusCancelled},
		}}
		assert.Equal(t, StatusCompleted, cat.AggregateStatus())
	})

	t.Run("completed and cancelled mix returns completed", func(t *testing.T) {
		cat := Category{Tasks: []Task{
			{Status: StatusCompleted}, {Status: StatusCancelled},
		}}
		assert.Equal(t, StatusCompleted, cat.AggregateStatus())
	})

	t.Run("any in_progress returns in_progress", func(t *testing.T) {
		cat := Category{Tasks: []Task{
			{Status: StatusTodo}, {Status: StatusInProgress}, {Status: StatusCompleted},
		}}
		assert.Equal(t, StatusInProgress, cat.AggregateStatus())
	})

	t.Run("mix of todo and completed returns in_progress", func(t *testing.T) {
		cat := Category{Tasks: []Task{
			{Status: StatusTodo}, {Status: StatusCompleted},
		}}
		assert.Equal(t, StatusInProgress, cat.AggregateStatus())
	})

	t.Run("mix of todo and cancelled returns in_progress", func(t *testing.T) {
		cat := Category{Tasks: []Task{
			{Status: StatusTodo}, {Status: StatusCancelled},
		}}
		assert.Equal(t, StatusInProgress, cat.AggregateStatus())
	})
}

func TestTask_SetStatus(t *testing.T) {
	t.Run("sets status and updates timestamp", func(t *testing.T) {
		task := Task{Status: StatusTodo}
		err := task.SetStatus(StatusInProgress)
		require.NoError(t, err)
		assert.Equal(t, StatusInProgress, task.Status)
		assert.NotEmpty(t, task.UpdatedAt)
	})

	t.Run("sets completion date when completed", func(t *testing.T) {
		task := Task{Status: StatusTodo}
		err := task.SetStatus(StatusCompleted)
		require.NoError(t, err)
		assert.NotEmpty(t, task.CompletionDate)
	})

	t.Run("clears completion date when not completed", func(t *testing.T) {
		task := Task{Status: StatusCompleted, CompletionDate: "2024-01-01"}
		err := task.SetStatus(StatusTodo)
		require.NoError(t, err)
		assert.Empty(t, task.CompletionDate)
	})

	t.Run("returns error for invalid status", func(t *testing.T) {
		task := Task{Status: StatusTodo}
		err := task.SetStatus("invalid")
		assert.Error(t, err)
	})
}

func TestTask_SetPriority(t *testing.T) {
	t.Run("sets priority and updates timestamp", func(t *testing.T) {
		task := Task{Priority: ""}
		err := task.SetPriority(PriorityHigh)
		require.NoError(t, err)
		assert.Equal(t, PriorityHigh, task.Priority)
		assert.NotEmpty(t, task.UpdatedAt)
	})

	t.Run("accepts critical and trivial", func(t *testing.T) {
		task := Task{Priority: ""}
		require.NoError(t, task.SetPriority(PriorityCritical))
		assert.Equal(t, PriorityCritical, task.Priority)
		require.NoError(t, task.SetPriority(PriorityTrivial))
		assert.Equal(t, PriorityTrivial, task.Priority)
	})

	t.Run("returns error for invalid priority", func(t *testing.T) {
		task := Task{Priority: ""}
		err := task.SetPriority("invalid")
		assert.Error(t, err)
	})
}

func TestTask_CyclePriority(t *testing.T) {
	// none → critical → high → medium → low → trivial → none
	want := []string{
		PriorityCritical,
		PriorityHigh,
		PriorityMedium,
		PriorityLow,
		PriorityTrivial,
		"",
	}
	task := Task{Priority: ""}
	for i, expected := range want {
		changed := task.CyclePriority()
		assert.True(t, changed, "step %d should report a change", i)
		assert.Equal(t, expected, task.Priority, "step %d", i)
	}
}

func TestTask_CycleTag(t *testing.T) {
	t.Run("walks the palette then back to none", func(t *testing.T) {
		task := Task{}
		want := []string{TagGreen, TagBlue, TagMagenta, TagCyan,
			TagGreenDiamond, TagBlueDiamond, TagMagentaDiamond, TagCyanDiamond, ""}
		for _, expected := range want {
			task.CycleTag()
			assert.Equal(t, expected, task.TagColor)
		}
	})

	t.Run("stamps UpdatedAt", func(t *testing.T) {
		task := Task{}
		task.CycleTag()
		assert.NotEmpty(t, task.UpdatedAt)
	})
}

func TestNextTagColor(t *testing.T) {
	assert.Equal(t, TagGreen, NextTagColor(""))
	assert.Equal(t, TagBlue, NextTagColor(TagGreen))
	assert.Equal(t, TagMagenta, NextTagColor(TagBlue))
	assert.Equal(t, TagCyan, NextTagColor(TagMagenta))
	assert.Equal(t, TagGreenDiamond, NextTagColor(TagCyan))
	assert.Equal(t, "", NextTagColor(TagCyanDiamond))
	assert.Equal(t, "", NextTagColor("bogus"))
}

func TestValidateTagColor(t *testing.T) {
	for _, c := range append([]string{""}, TagColorCycle...) {
		assert.NoError(t, ValidateTagColor(c))
	}
	assert.Error(t, ValidateTagColor("red"))
	assert.Error(t, ValidateTagColor("red-diamond"))
	assert.Error(t, ValidateTagColor("2"))
}

func TestTask_SetTagColor(t *testing.T) {
	t.Run("sets a valid color", func(t *testing.T) {
		task := Task{}
		require.NoError(t, task.SetTagColor(TagBlue))
		assert.Equal(t, TagBlue, task.TagColor)
		assert.NotEmpty(t, task.UpdatedAt)
	})

	t.Run("clears the tag", func(t *testing.T) {
		task := Task{TagColor: TagBlue}
		require.NoError(t, task.SetTagColor(""))
		assert.Equal(t, "", task.TagColor)
	})

	t.Run("rejects an invalid color", func(t *testing.T) {
		task := Task{}
		assert.Error(t, task.SetTagColor("chartreuse"))
	})
}
