package operations

import (
	"errors"
	"slices"
	"testing"

	"github.com/ilyasturki/ordo/internal/domain"
)

// outline builds categories from specs: "cat:" opens one, ">x" is a subtask, "--x" a separator.
func outline(t *testing.T, specs ...string) *domain.Project {
	t.Helper()
	p, err := domain.NewProject("Test")
	if err != nil {
		t.Fatal(err)
	}
	parent := ""
	for _, s := range specs {
		switch {
		case len(s) > 4 && s[:4] == "cat:":
			cat, err := domain.NewCategory(s[4:])
			if err != nil {
				t.Fatal(err)
			}
			cat.ID = s[4:]
			p.Categories = append(p.Categories, cat)
			continue
		}
		cat := &p.Categories[len(p.Categories)-1]
		var task domain.Task
		switch {
		case len(s) > 2 && s[:2] == "--":
			task, _ = domain.NewSeparator()
			s = s[2:]
			task.Title = s
			parent = ""
		case s[0] == '>':
			s = s[1:]
			task, _ = domain.NewTask(s)
			task.ParentID = parent
		default:
			task, _ = domain.NewTask(s)
			parent = s
		}
		task.ID = s
		cat.Tasks = append(cat.Tasks, task)
	}
	return &p
}

// layout renders every category as "cat:" then its rows, in the outline spec form.
func layout(p *domain.Project) []string {
	var out []string
	for _, c := range p.Categories {
		out = append(out, "cat:"+c.ID)
		for _, task := range c.Tasks {
			switch {
			case task.IsSeparator():
				out = append(out, "--"+task.ID)
			case task.ParentID != "":
				out = append(out, ">"+task.ID)
			default:
				out = append(out, task.ID)
			}
		}
	}
	return out
}

func wantLayout(t *testing.T, p *domain.Project, want ...string) {
	t.Helper()
	if got := layout(p); !slices.Equal(got, want) {
		t.Fatalf("layout\n got %v\nwant %v", got, want)
	}
}

func wantRefusal(t *testing.T, err error, msg string) {
	t.Helper()
	var r *Refusal
	if !errors.As(err, &r) || r.Msg != msg {
		t.Fatalf("err = %v, want refusal %q", err, msg)
	}
}

func TestTaskRunRefusesScatteredSelections(t *testing.T) {
	p := outline(t, "cat:a", "x", "y", "z", "cat:b", "w")
	_, _, _, err := TaskRun(p, []string{"x", "w"})
	wantRefusal(t, err, "Can't: selection spans multiple categories")
	_, _, _, err = TaskRun(p, []string{"x", "z"})
	wantRefusal(t, err, "Can't: selection isn't contiguous (rows hidden by filter)")
	if _, _, _, err := TaskRun(p, []string{"ghost"}); !errors.Is(err, domain.ErrTaskNotFound) {
		t.Fatalf("unknown id: err = %v", err)
	}
	ci, lo, hi, err := TaskRun(p, []string{"z", "y"})
	if err != nil || ci != 0 || lo != 1 || hi != 3 {
		t.Fatalf("run = %d [%d,%d) %v", ci, lo, hi, err)
	}
}

func TestMoveRowsCarriesSubtasksAcrossCategories(t *testing.T) {
	p := outline(t, "cat:a", "p", ">c", "q", "cat:b", "r")
	rows, err := MoveRows(p, []string{"p"}, "b", 1, "")
	if err != nil || len(rows) != 2 {
		t.Fatalf("rows = %v, err = %v", rows, err)
	}
	wantLayout(t, p, "cat:a", "q", "cat:b", "r", "p", ">c")
}

func TestMoveRowsCountsTheIndexBeforeTheLift(t *testing.T) {
	p := outline(t, "cat:a", "x", "y", "z")
	if _, err := MoveRows(p, []string{"x"}, "a", 3, ""); err != nil {
		t.Fatal(err)
	}
	wantLayout(t, p, "cat:a", "y", "z", "x")
}

func TestMoveRowsNestsUnderAParent(t *testing.T) {
	p := outline(t, "cat:a", "p", ">c", "x")
	if _, err := MoveRows(p, []string{"x"}, "a", 1, "p"); err != nil {
		t.Fatal(err)
	}
	wantLayout(t, p, "cat:a", "p", ">x", ">c")
	if _, err := MoveRows(p, []string{"ghost"}, "a", 0, ""); !errors.Is(err, ErrNoChange) {
		t.Fatalf("gone rows: err = %v", err)
	}
}

func TestMoveRowsRefusesATaskUnderItself(t *testing.T) {
	p := outline(t, "cat:a", "p", "x")
	_, err := MoveRows(p, []string{"p"}, "a", 1, "p")
	wantRefusal(t, err, "Can't nest a task under itself")
	wantLayout(t, p, "cat:a", "p", "x")
}

func TestDeleteTasksTakesOrFreesSubtasks(t *testing.T) {
	p := outline(t, "cat:a", "p", ">c", "x")
	removed, err := DeleteTasks(p, []string{"p"}, true)
	if err != nil || len(removed) != 1 {
		t.Fatalf("removed = %v, err = %v", removed, err)
	}
	wantLayout(t, p, "cat:a", "c", "x")

	p = outline(t, "cat:a", "p", ">c", "x")
	removed, err = DeleteTasks(p, []string{"p", "c"}, false)
	if err != nil || len(removed) != 2 {
		t.Fatalf("removed = %v, err = %v", removed, err)
	}
	wantLayout(t, p, "cat:a", "x")
	if _, err := DeleteTasks(p, []string{"p"}, false); !errors.Is(err, ErrNoChange) {
		t.Fatalf("second delete: err = %v", err)
	}
}

func TestNestTasksPhrasesItsRefusals(t *testing.T) {
	p := outline(t, "cat:a", "p", ">c", "x", "--s")
	_, err := NestTasks(p, []string{"p"}, true, AllShown)
	wantRefusal(t, err, "Nothing above to nest under")
	_, err = NestTasks(p, []string{"c"}, true, AllShown)
	wantRefusal(t, err, "Already a subtask: nesting goes one level deep")
	_, err = NestTasks(p, []string{"x"}, false, AllShown)
	wantRefusal(t, err, "Already at the top level")
	_, err = NestTasks(p, []string{"s"}, true, AllShown)
	wantRefusal(t, err, "A separator doesn't nest")

	parent, err := NestTasks(p, []string{"x"}, true, AllShown)
	if err != nil || parent != "p" {
		t.Fatalf("parent = %q, err = %v", parent, err)
	}
	wantLayout(t, p, "cat:a", "p", ">c", ">x", "--s")
	if _, err := NestTasks(p, []string{"c"}, false, AllShown); err != nil {
		t.Fatal(err)
	}
	wantLayout(t, p, "cat:a", "p", ">x", "c", "--s")
}

type hiding struct {
	allShown
	hidden map[string]bool
	folded map[string]bool
}

func (h hiding) Shown(cat domain.Category) []bool {
	shown := make([]bool, len(cat.Tasks))
	for i, t := range cat.Tasks {
		shown[i] = !h.hidden[t.ID]
	}
	return shown
}

func (h hiding) Filtering() bool       { return len(h.hidden) > 0 }
func (h hiding) Folded(id string) bool { return h.folded[id] }

func TestNestTasksRefusesUnderAHiddenRow(t *testing.T) {
	p := outline(t, "cat:a", "p", "x")
	_, err := NestTasks(p, []string{"x"}, true, hiding{hidden: map[string]bool{"p": true}})
	wantRefusal(t, err, "Can't nest: the filter hides the row above")
}

func TestShiftTasksSkipsHiddenBlocks(t *testing.T) {
	p := outline(t, "cat:a", "x", "h", "y")
	if err := ShiftTasks(p, []string{"x"}, 1, hiding{hidden: map[string]bool{"h": true}}); err != nil {
		t.Fatal(err)
	}
	wantLayout(t, p, "cat:a", "h", "y", "x")
}

func TestShiftTasksCrossesIntoTheNextCategory(t *testing.T) {
	p := outline(t, "cat:a", "x", "cat:b", "y")
	if err := ShiftTasks(p, []string{"x"}, 1, AllShown); err != nil {
		t.Fatal(err)
	}
	wantLayout(t, p, "cat:a", "cat:b", "x", "y")
	if err := ShiftTasks(p, []string{"y"}, -1, AllShown); err != nil {
		t.Fatal(err)
	}
	if err := ShiftTasks(p, []string{"y"}, -1, AllShown); err != nil {
		t.Fatal(err)
	}
	wantLayout(t, p, "cat:a", "y", "cat:b", "x")
	wantRefusal(t, ShiftTasks(p, []string{"y"}, -1, AllShown), "Already at the top")
	wantRefusal(t, ShiftTasks(p, []string{"y"}, 1, hiding{folded: map[string]bool{"b": true}}),
		"Can't move into a folded category")
}

func TestShiftTasksKeepsASubtaskInsideItsParent(t *testing.T) {
	p := outline(t, "cat:a", "p", ">c", "cat:b")
	wantRefusal(t, ShiftTasks(p, []string{"c"}, 1, AllShown), "Already at the bottom of its parent")
}

func TestCycleStatusStepsTheBatchFromItsFirstTask(t *testing.T) {
	p := outline(t, "cat:a", "x", "y", "--s")
	p.Categories[0].Tasks[1].Status = domain.StatusCompleted
	next, n, err := CycleStatus(p, []string{"x", "y", "s"}, false)
	if err != nil || next != domain.StatusInProgress || n != 2 {
		t.Fatalf("next = %q, n = %d, err = %v", next, n, err)
	}
	if _, _, err := CycleStatus(p, []string{"s"}, false); !errors.Is(err, ErrNoChange) {
		t.Fatalf("separator alone: err = %v", err)
	}
}

func TestCycleTagsPicksAFreeMarkForAnUntaggedBatch(t *testing.T) {
	p := outline(t, "cat:a", "x", "y", "z")
	p.Categories[0].Tasks[2].TagColor = domain.TagGreen
	color, err := CycleTags(p, []string{"x", "y"})
	if err != nil || color == "" || color == domain.TagGreen {
		t.Fatalf("color = %q, err = %v", color, err)
	}
	if p.Categories[0].Tasks[0].TagColor != color || p.Categories[0].Tasks[1].TagColor != color {
		t.Fatal("batch not tagged together")
	}
	if color, _ = CycleTags(p, []string{"x"}); color != domain.NextTagColor(p.Categories[0].Tasks[1].TagColor) {
		t.Fatalf("tagged task stepped to %q", color)
	}
}

func TestSetTagsCountsOnlyChanges(t *testing.T) {
	p := outline(t, "cat:a", "x", "y")
	p.Categories[0].Tasks[0].TagColor = domain.TagBlue
	if n, err := SetTags(p, []string{"x", "y"}, domain.TagBlue); err != nil || n != 1 {
		t.Fatalf("n = %d, err = %v", n, err)
	}
	if _, err := SetTags(p, []string{"x"}, domain.TagBlue); !errors.Is(err, ErrNoChange) {
		t.Fatalf("err = %v", err)
	}
	if _, err := SetTags(p, []string{"x"}, "orange"); err == nil {
		t.Fatal("bad color accepted")
	}
}

func TestEveryActionHasASurface(t *testing.T) {
	seen := make(map[string]bool)
	for _, a := range Actions {
		if a.Surface&Both == 0 || a.Label == "" || seen[a.ID] {
			t.Errorf("action %+v", a)
		}
		seen[a.ID] = true
	}
}

func TestMoveRowsKeepsTheOutlineValid(t *testing.T) {
	p := outline(t, "cat:a", "x", ">c", "y", "--s", "z")
	wantRefusal(t, func() error { _, err := MoveRows(p, []string{"y"}, "a", 2, "c"); return err }(), "Already a subtask: nesting goes one level deep")
	wantRefusal(t, func() error { _, err := MoveRows(p, []string{"z"}, "a", 4, "s"); return err }(), "A separator doesn't nest")
	if _, err := MoveRows(p, []string{"y"}, "a", 1, ""); err != nil {
		t.Fatal(err)
	}
	wantLayout(t, p, "cat:a", "y", "x", ">c", "--s", "z")
	if _, err := MoveRows(p, []string{"z"}, "a", 0, "x"); err != nil {
		t.Fatal(err)
	}
	wantLayout(t, p, "cat:a", "y", "x", ">z", ">c", "--s")
}
