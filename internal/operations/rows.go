package operations

import (
	"errors"
	"slices"

	"github.com/ilyasturki/ordo/internal/domain"
)

// LocateTask finds a task or separator by ID anywhere in the project.
func LocateTask(p *domain.Project, id string) (catIdx, taskIdx int, err error) {
	for ci := range p.Categories {
		for ti, t := range p.Categories[ci].Tasks {
			if t.ID == id {
				return ci, ti, nil
			}
		}
	}
	return -1, -1, domain.ErrTaskNotFound
}

// TaskRun resolves ids to the run of whole sibling blocks [lo, hi) they cover in one category.
func TaskRun(p *domain.Project, ids []string) (catIdx, lo, hi int, err error) {
	catIdx = -1
	idxs := make([]int, 0, len(ids))
	for _, id := range ids {
		ci, ti, err := LocateTask(p, id)
		if err != nil {
			return 0, 0, 0, err
		}
		if catIdx >= 0 && ci != catIdx {
			return 0, 0, 0, refuse("Can't: selection spans multiple categories")
		}
		catIdx = ci
		idxs = append(idxs, ti)
	}
	if catIdx < 0 {
		return 0, 0, 0, ErrNoChange
	}
	lo, hi, err = domain.CoverBlocks(p.Categories[catIdx].Tasks, idxs)
	switch {
	case errors.Is(err, domain.ErrNotContiguous):
		return 0, 0, 0, refuse("Can't: selection isn't contiguous (rows hidden by filter)")
	case err != nil:
		return 0, 0, 0, refuse("Can't: the selection mixes levels")
	}
	return catIdx, lo, hi, nil
}

// SettleTasks nests rows for an insert at index of tasks and returns the index to insert at.
func SettleTasks(tasks, rows []domain.Task, index int, parentID string) int {
	if domain.SettleParents(rows, parentID) || parentID == "" {
		return index
	}
	for i, t := range tasks {
		if t.ID == parentID {
			return domain.BlockEnd(tasks, i)
		}
	}
	return index
}

// MoveRows lifts the ids' blocks and inserts them at index of the category (counted before the lift).
func MoveRows(p *domain.Project, ids []string, categoryID string, index int, parentID string) ([]domain.Task, error) {
	dstIdx, err := p.FindCategoryByID(categoryID)
	if err != nil {
		return nil, err
	}
	picked := make(map[string]bool, len(ids))
	for _, id := range ids {
		picked[id] = true
	}
	// Project order, not the caller's: a row nested since it was picked follows its parent.
	var moved []domain.Task
	for _, cat := range p.Categories {
		moved = append(moved, domain.PickBlocks(cat.Tasks, picked)...)
	}
	if len(moved) == 0 {
		return nil, ErrNoChange
	}
	parents := make([]string, len(moved))
	lifted := make(map[string]bool, len(moved))
	for i, t := range moved {
		parents[i] = t.ParentID
		lifted[t.ID] = true
	}
	dst := &p.Categories[dstIdx]
	index = max(0, min(index, len(dst.Tasks)))
	if parentID != "" && !lifted[parentID] {
		pi := slices.IndexFunc(dst.Tasks, func(t domain.Task) bool { return t.ID == parentID })
		switch {
		case pi < 0:
			return nil, domain.ErrTaskNotFound
		case dst.Tasks[pi].IsSeparator():
			return nil, refuse("A separator doesn't nest")
		case dst.Tasks[pi].ParentID != "":
			return nil, refuse("Already a subtask: nesting goes one level deep")
		}
		index = max(pi+1, min(index, domain.BlockEnd(dst.Tasks, pi)))
	} else if parentID == "" && index < len(dst.Tasks) && dst.Tasks[index].ParentID != "" {
		index = domain.BlockStart(dst.Tasks, index)
	}
	index = SettleTasks(dst.Tasks, moved, index, parentID)
	for _, t := range moved {
		if t.ParentID == t.ID {
			return nil, refuse("Can't nest a task under itself")
		}
	}
	now := domain.NowTimestamp()
	for i := range moved {
		if moved[i].ParentID != parents[i] {
			moved[i].UpdatedAt = now
		}
	}
	for _, t := range dst.Tasks[:max(0, min(index, len(dst.Tasks)))] {
		if lifted[t.ID] {
			index--
		}
	}
	for i := range p.Categories {
		cat := &p.Categories[i]
		kept := cat.Tasks[:0]
		for _, t := range cat.Tasks {
			if !lifted[t.ID] {
				kept = append(kept, t)
			}
		}
		if len(kept) != len(cat.Tasks) {
			cat.Tasks = kept
			cat.UpdatedAt = now
		}
	}
	index = max(0, min(index, len(dst.Tasks)))
	dst.Tasks = slices.Insert(dst.Tasks, index, moved...)
	dst.UpdatedAt, p.UpdatedAt = now, now
	return moved, nil
}

// DeleteTasks removes each id with its subtasks, or frees the subtasks when keepChildren.
func DeleteTasks(p *domain.Project, ids []string, keepChildren bool) ([]domain.Task, error) {
	var removed []domain.Task
	for _, id := range ids {
		ci, ti, err := LocateTask(p, id)
		if err != nil {
			continue
		}
		cat := &p.Categories[ci]
		if keepChildren {
			cat.ReleaseSubtasks(ti)
		}
		removed = append(removed, slices.Clone(cat.Tasks[ti:domain.BlockEnd(cat.Tasks, ti)])...)
		cat.RemoveTaskBlock(ti)
	}
	if len(removed) == 0 {
		return nil, ErrNoChange
	}
	p.UpdatedAt = domain.NowTimestamp()
	return removed, nil
}

// NestTasks nests the ids' run under the block above it, or un-nests it; returns the new parent's ID.
func NestTasks(p *domain.Project, ids []string, in bool, v View) (string, error) {
	catIdx, lo, hi, err := TaskRun(p, ids)
	if err != nil {
		return "", err
	}
	cat := &p.Categories[catIdx]
	if in && lo > 0 && v.Filtering() && !v.Shown(*cat)[domain.BlockStart(cat.Tasks, lo-1)] {
		return "", refuse("Can't nest: the filter hides the row above")
	}
	nested := cat.Tasks[lo].ParentID != ""
	if in {
		err = cat.IndentTasks(lo, hi)
	} else {
		_, err = cat.OutdentTasks(lo, hi)
	}
	if err != nil {
		return "", NestRefusal(err, nested, "task", "subtask", "subtasks")
	}
	p.UpdatedAt = domain.NowTimestamp()
	if in {
		return cat.Tasks[lo].ParentID, nil
	}
	return "", nil
}

// ShiftTasks moves the ids' run one shown sibling block in dir, into the next category at an edge.
func ShiftTasks(p *domain.Project, ids []string, dir int, v View) error {
	catIdx, lo, hi, err := TaskRun(p, ids)
	if err != nil {
		return err
	}
	cat := &p.Categories[catIdx]
	nested := cat.Tasks[lo].ParentID != ""
	a, b, ok := nextShownBlocks(*cat, v.Shown(*cat), lo, hi, dir)
	if !ok {
		if nested {
			return EdgeRefusal(dir, true)
		}
		return shiftTasksAcross(p, v, catIdx, lo, hi, dir)
	}
	if dir > 0 {
		domain.SwapRanges(cat.Tasks, lo, hi, b)
	} else {
		domain.SwapRanges(cat.Tasks, a, lo, hi)
	}
	now := domain.NowTimestamp()
	cat.UpdatedAt, p.UpdatedAt = now, now
	return nil
}

// nextShownBlocks spans [a, b) from the run through the first shown sibling block in dir.
func nextShownBlocks(cat domain.Category, shown []bool, lo, hi, dir int) (int, int, bool) {
	if dir > 0 {
		head := domain.LastBlockStart(cat.Tasks, lo, hi)
		for {
			nlo, nhi, ok := domain.SiblingBlock(cat.Tasks, head, 1)
			if !ok {
				return 0, 0, false
			}
			if slices.Contains(shown[nlo:nhi], true) {
				return hi, nhi, true
			}
			head = nlo
		}
	}
	head := lo
	for {
		nlo, _, ok := domain.SiblingBlock(cat.Tasks, head, -1)
		if !ok {
			return 0, 0, false
		}
		if slices.Contains(shown[nlo:head], true) {
			return nlo, lo, true
		}
		head = nlo
	}
}

// shiftTasksAcross expects tasks [lo, hi) of category catIdx to be top-level blocks.
func shiftTasksAcross(p *domain.Project, v View, catIdx, lo, hi, dir int) error {
	dstIdx := catIdx + dir
	if dstIdx < 0 || dstIdx >= len(p.Categories) {
		return EdgeRefusal(dir, false)
	}
	src := &p.Categories[catIdx]
	dst := &p.Categories[dstIdx]
	if v.Folded(dst.ID) || (dst.ParentID != "" && v.Folded(dst.ParentID)) {
		return refuse("Can't move into a folded category")
	}
	if !v.CategoryVisible(*dst) {
		return refuse("Can't move: the filter would hide it there")
	}
	insertAt := 0
	if dir < 0 {
		insertAt = len(dst.Tasks)
	}
	block := slices.Clone(src.Tasks[lo:hi])
	src.Tasks = slices.Delete(src.Tasks, lo, hi)
	dst.Tasks = slices.Insert(dst.Tasks, insertAt, block...)
	now := domain.NowTimestamp()
	src.UpdatedAt, dst.UpdatedAt, p.UpdatedAt = now, now, now
	return nil
}
