package app

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/app/selection"
	"github.com/ilyasturki/ordo/internal/domain"
	"github.com/ilyasturki/ordo/internal/merge"
)

// eagerHistory is the undo history as it was before the carry waited for a pop: every outside
// change merged into every step at once.
type eagerHistory struct{ undo, redo, pending []domain.Project }

func (h *eagerHistory) rebase(mine, merged domain.Project) {
	for _, stack := range [][]domain.Project{h.undo, h.redo, h.pending} {
		for i := range stack {
			stack[i] = merge.Three(mine, stack[i], merged).Project
		}
	}
}

type reshaper struct {
	r    *rand.Rand
	next int
}

func (s *reshaper) edit(p domain.Project) domain.Project {
	p = cloneProject(p)
	for range 1 + s.r.IntN(3) {
		c := &p.Categories[s.r.IntN(len(p.Categories))]
		switch op := s.r.IntN(5); {
		case op == 0 || len(c.Tasks) == 0:
			s.next++
			t := domain.Task{ID: fmt.Sprintf("n%d", s.next), Title: fmt.Sprintf("New %d", s.next), Status: domain.StatusTodo}
			c.Tasks = slices.Insert(c.Tasks, s.r.IntN(len(c.Tasks)+1), t)
		case op == 1:
			i := s.r.IntN(len(c.Tasks))
			c.Tasks = slices.Delete(c.Tasks, i, i+1)
		case op == 2:
			i := s.r.IntN(len(c.Tasks))
			t := c.Tasks[i]
			c.Tasks = slices.Delete(c.Tasks, i, i+1)
			c.Tasks = slices.Insert(c.Tasks, s.r.IntN(len(c.Tasks)+1), t)
		case op == 3:
			c.Tasks[s.r.IntN(len(c.Tasks))].Title = fmt.Sprintf("Title %d", s.r.IntN(4))
		default:
			c.Tasks[s.r.IntN(len(c.Tasks))].Priority = domain.PriorityOrder[s.r.IntN(len(domain.PriorityOrder))]
		}
	}
	return p
}

func TestUndoAfterOutsideChangesMatchesTheEagerRebase(t *testing.T) {
	s := &reshaper{r: rand.New(rand.NewPCG(3, 5))}
	m := benchModel(benchProject(3, 6), false)
	var ref eagerHistory
	capped := false
	for step := range 2400 {
		op := s.r.IntN(20)
		if step%800 > 620 {
			op = 0
		}
		switch {
		case op < 8:
			merged := s.edit(m.project)
			ref.rebase(m.project, merged)
			m.ui.History.rebase(m.project, merged)
			m.project = cloneProject(merged)
		case op < 12:
			if len(ref.undo) >= historyLimit {
				ref.undo = ref.undo[1:]
			}
			ref.undo = append(ref.undo, cloneProject(m.project))
			ref.pending, ref.redo = ref.redo, nil
			m.recordHistory()
			if op == 11 {
				ref.undo = ref.undo[:len(ref.undo)-1]
				ref.redo, ref.pending = ref.pending, nil
				m.discardLastHistory()
				continue
			}
			m.project = s.edit(m.project)
		case op < 17:
			if len(ref.undo) == 0 {
				continue
			}
			ref.redo = append(ref.redo, cloneProject(m.project))
			want := ref.undo[len(ref.undo)-1]
			ref.undo, ref.pending = ref.undo[:len(ref.undo)-1], nil
			m.undo()
			require.Equal(t, want, m.project, "undo at step %d", step)
		default:
			if len(ref.redo) == 0 {
				continue
			}
			ref.undo = append(ref.undo, cloneProject(m.project))
			want := ref.redo[len(ref.redo)-1]
			ref.redo, ref.pending = ref.redo[:len(ref.redo)-1], nil
			m.redo()
			require.Equal(t, want, m.project, "redo at step %d", step)
		}
		require.LessOrEqual(t, len(m.ui.History.changes), historyLimit)
		capped = capped || len(m.ui.History.changes) == historyLimit
	}
	for len(ref.undo) > 0 {
		want := ref.undo[len(ref.undo)-1]
		ref.undo = ref.undo[:len(ref.undo)-1]
		m.undo()
		require.Equal(t, want, m.project, "undo to the start")
	}
	assert.True(t, capped, "a long run of outside changes reaches the cap")
}

func TestUndoAndRedoSurviveSeveralOutsideChanges(t *testing.T) {
	m, web, p := sharedModel(t)
	picked := tasksIn(p, 2)
	m.selectItem(selection.FocusTask, picked[0].ID)
	hit(m, "l")
	onWeb(t, web, p.ID, func(p *domain.Project) { retitle(p, picked[1].ID, "Web one") })
	tick(t, m)
	m.selectItem(selection.FocusTask, picked[0].ID)
	hit(m, "l")
	raised, _ := findTask(m.project, picked[0].ID)
	onWeb(t, web, p.ID, func(p *domain.Project) { retitle(p, picked[0].ID, "Web two") })
	tick(t, m)

	hit(m, "u", "u")

	disk := onDisk(t, web, p.ID)
	got, _ := findTask(disk, picked[0].ID)
	assert.Equal(t, picked[0].Priority, got.Priority)
	assert.Equal(t, "Web two", got.Title)
	got, _ = findTask(disk, picked[1].ID)
	assert.Equal(t, "Web one", got.Title)

	hit(m, "ctrl+r", "ctrl+r")

	disk = onDisk(t, web, p.ID)
	got, _ = findTask(disk, picked[0].ID)
	assert.Equal(t, raised.Priority, got.Priority)
	assert.Equal(t, "Web two", got.Title)
	got, _ = findTask(disk, picked[1].ID)
	assert.Equal(t, "Web one", got.Title)
}
