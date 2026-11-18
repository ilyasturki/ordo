package merge

import (
	"fmt"
	"maps"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/domain"
)

// oldThree is Three as it stood before the linear rewrite, kept to prove the rewrite changes no result.
func oldThree(base, mine, theirs domain.Project) Result {
	var res Result
	merged, clashes := mergeFields(projectFields, base, mine, theirs)
	for _, f := range clashes {
		res.Collisions = append(res.Collisions, Collision{
			Row: "Project", Field: f.label, Mine: f.value(mine), Theirs: f.value(theirs),
			take: func(p *domain.Project) { f.take(p, mine) },
		})
	}

	tasks, taskClashes := mergeTasks(base, mine, theirs)
	cats, catClashes := mergeCategories(base, mine, theirs, tasks)
	res.Collisions = append(res.Collisions, catClashes...)
	res.Collisions = append(res.Collisions, taskClashes...)
	foldTwinCategories(cats, base, mine, theirs, tasks)

	keepCats := make(map[string]bool, len(cats))
	for id := range cats {
		keepCats[id] = true
	}
	merged.Categories = make([]domain.Category, 0, len(cats))
	for _, id := range oldMergeOrder(
		idsOf(base.Categories, categoryID), idsOf(mine.Categories, categoryID), idsOf(theirs.Categories, categoryID), keepCats,
	) {
		cat := cats[id]
		cat.Tasks = oldPlaceTasks(id, base, mine, theirs, tasks)
		merged.Categories = append(merged.Categories, cat)
	}
	domain.NormalizeNesting(&merged)
	res.Project = merged
	return res
}

func oldPlaceTasks(cat string, base, mine, theirs domain.Project, tasks map[string]placedTask) []domain.Task {
	keep := make(map[string]bool)
	for id, row := range tasks {
		if row.cat == cat {
			keep[id] = true
		}
	}
	rowsIn := func(p domain.Project) []string {
		i := slices.IndexFunc(p.Categories, func(c domain.Category) bool { return c.ID == cat })
		if i < 0 {
			return nil
		}
		return idsOf(p.Categories[i].Tasks, taskID)
	}
	mineRows, theirRows := rowsIn(mine), rowsIn(theirs)
	for _, id := range slices.Sorted(maps.Keys(keep)) {
		if !slices.Contains(mineRows, id) && !slices.Contains(theirRows, id) {
			mineRows = append(mineRows, id)
		}
	}
	order := oldMergeOrder(rowsIn(base), mineRows, theirRows, keep)
	out := make([]domain.Task, 0, len(order))
	for _, id := range order {
		out = append(out, tasks[id].task)
	}
	return out
}

func oldBefore(list []string, id string) string {
	if i := slices.Index(list, id); i > 0 {
		return list[i-1]
	}
	return ""
}

func oldMergeOrder(base, mine, theirs []string, keep map[string]bool) []string {
	out := make([]string, 0, len(keep))
	for _, id := range theirs {
		if keep[id] {
			out = append(out, id)
		}
	}
	shared := func(list []string) []string {
		return slices.DeleteFunc(slices.Clone(list), func(id string) bool {
			return !slices.Contains(base, id) || !slices.Contains(mine, id) || !slices.Contains(theirs, id)
		})
	}
	sharedBase, sharedMine, sharedTheirs := shared(base), shared(mine), shared(theirs)
	for i, id := range mine {
		if !keep[id] {
			continue
		}
		if slices.Contains(sharedMine, id) {
			was := oldBefore(sharedBase, id)
			if oldBefore(sharedMine, id) == was || oldBefore(sharedTheirs, id) != was {
				continue
			}
		} else if slices.Contains(theirs, id) {
			continue
		}
		if at := slices.Index(out, id); at >= 0 {
			out = slices.Delete(out, at, at+1)
		}
		at := 0
		for j := i - 1; j >= 0; j-- {
			if k := slices.Index(out, mine[j]); k >= 0 {
				at = k + 1
				break
			}
		}
		out = slices.Insert(out, at, id)
	}
	return out
}

type shaper struct {
	r    *rand.Rand
	next int
}

func (s *shaper) id(prefix string) string {
	s.next++
	return fmt.Sprintf("%s%d", prefix, s.next)
}

func (s *shaper) base() domain.Project {
	p := domain.Project{ID: "p", Name: "P"}
	for range s.r.IntN(5) {
		cat := domain.Category{ID: s.id("c"), Name: fmt.Sprintf("Cat %d", s.r.IntN(4))}
		for range s.r.IntN(9) {
			t := row(s.id("t"), fmt.Sprintf("Task %d", s.next))
			if n := len(cat.Tasks); n > 0 && s.r.IntN(4) == 0 {
				t.ParentID = cat.Tasks[n-1].ID
			}
			cat.Tasks = append(cat.Tasks, t)
		}
		p.Categories = append(p.Categories, cat)
	}
	domain.NormalizeNesting(&p)
	return p
}

// edit applies a handful of random edits of every kind the merge reads: fields, adds, deletes, moves.
func (s *shaper) edit(p domain.Project, shared []string) domain.Project {
	p = Clone(p)
	for range s.r.IntN(6) {
		if len(p.Categories) == 0 || s.r.IntN(8) == 0 {
			name := fmt.Sprintf("Cat %d", s.r.IntN(4))
			id := s.id("c")
			if len(shared) > 0 && s.r.IntN(2) == 0 {
				id = shared[s.r.IntN(len(shared))]
			}
			if !slices.ContainsFunc(p.Categories, func(c domain.Category) bool { return c.ID == id }) {
				p.Categories = slices.Insert(p.Categories, s.r.IntN(len(p.Categories)+1), domain.Category{ID: id, Name: name})
			}
			continue
		}
		ci := s.r.IntN(len(p.Categories))
		cat := &p.Categories[ci]
		switch op := s.r.IntN(9); {
		case op == 0:
			p.Categories = slices.Delete(p.Categories, ci, ci+1)
		case op == 1 && len(p.Categories) > 1:
			c := p.Categories[ci]
			p.Categories = slices.Delete(p.Categories, ci, ci+1)
			p.Categories = slices.Insert(p.Categories, s.r.IntN(len(p.Categories)+1), c)
		case op == 2:
			cat.Name = fmt.Sprintf("Cat %d", s.r.IntN(4))
		case len(cat.Tasks) == 0 || op == 3:
			t := row(s.id("t"), fmt.Sprintf("Added %d", s.next))
			if len(shared) > 0 && s.r.IntN(3) == 0 {
				t.ID = shared[s.r.IntN(len(shared))]
			}
			if !slices.ContainsFunc(cat.Tasks, func(x domain.Task) bool { return x.ID == t.ID }) {
				cat.Tasks = slices.Insert(cat.Tasks, s.r.IntN(len(cat.Tasks)+1), t)
			}
		case op == 4:
			ti := s.r.IntN(len(cat.Tasks))
			cat.Tasks = slices.Delete(cat.Tasks, ti, ti+1)
		case op == 5:
			ti := s.r.IntN(len(cat.Tasks))
			t := cat.Tasks[ti]
			cat.Tasks = slices.Delete(cat.Tasks, ti, ti+1)
			cat.Tasks = slices.Insert(cat.Tasks, s.r.IntN(len(cat.Tasks)+1), t)
		case op == 6:
			ti := s.r.IntN(len(cat.Tasks))
			t := cat.Tasks[ti]
			cat.Tasks = slices.Delete(cat.Tasks, ti, ti+1)
			to := &p.Categories[s.r.IntN(len(p.Categories))]
			to.Tasks = slices.Insert(to.Tasks, s.r.IntN(len(to.Tasks)+1), t)
		case op == 7:
			cat.Tasks[s.r.IntN(len(cat.Tasks))].Title = fmt.Sprintf("Title %d", s.r.IntN(3))
		default:
			cat.Tasks[s.r.IntN(len(cat.Tasks))].Priority = domain.PriorityOrder[s.r.IntN(len(domain.PriorityOrder))]
		}
	}
	domain.NormalizeNesting(&p)
	return p
}

type seenCollision struct{ Task, Cat, Row, Field, Mine, Theirs string }

func seen(cs []Collision) []seenCollision {
	out := make([]seenCollision, len(cs))
	for i, c := range cs {
		out[i] = seenCollision{c.Task, c.Cat, c.Row, c.Field, c.Mine, c.Theirs}
	}
	return out
}

func TestThreeMatchesTheQuadraticMerge(t *testing.T) {
	s := &shaper{r: rand.New(rand.NewPCG(7, 11))}
	for i := range 20000 {
		base := s.base()
		shared := []string{s.id("c"), s.id("t")}
		mine, theirs := s.edit(base, shared), s.edit(base, shared)
		want, got := oldThree(base, mine, theirs), Three(base, mine, theirs)
		require.Equal(t, want.Project, got.Project, "case %d", i)
		require.Equal(t, seen(want.Collisions), seen(got.Collisions), "case %d", i)
	}
}
