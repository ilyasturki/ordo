package operations

import (
	"errors"
	"slices"
	"strconv"

	"github.com/ilyasturki/ordo/internal/domain"
)

// Undo lists the replayable ops that put back what op changed, given the project before and after
// op applied; nil when it has none. op is replayable, as Perform returns it. A project op's undo
// goes to the project op went to: the queue entry names it, not the op. For project.create, before
// is the zero Project.
func Undo(before, after domain.Project, op Op) []Op {
	switch op.Action {
	case "task.add", "task.insert", "separator.add":
		if op.New != "" {
			return []Op{{Action: "task.delete", IDs: []string{op.New}}}
		}
	case "task.title", "task.description", "task.attach":
		return undoText(before, after, op)
	case "status.set", "priority.set", "tag.set", "estimate.set":
		return undoValues(before, after, op)
	case "task.delete":
		return undoDelete(before, after)
	case "task.place", "task.move":
		return landing(after, undoMove(before, op.IDs))
	case "category.add":
		if op.New != "" {
			return []Op{{Action: "category.delete", Cat: op.New}}
		}
	case "category.rename":
		if ci, err := before.FindCategoryByID(op.Cat); err == nil {
			return []Op{{Action: "category.rename", Cat: op.Cat, Title: before.Categories[ci].Name}}
		}
	case "project.create":
		return []Op{{Action: "project.delete"}}
	case "project.rename":
		return []Op{{Action: "project.rename", Title: before.Name}}
	}
	return nil
}

func rowOf(p *domain.Project, id string) (domain.Task, bool) {
	ci, ti, err := LocateTask(p, id)
	if err != nil {
		return domain.Task{}, false
	}
	return p.Categories[ci].Tasks[ti], true
}

func undoText(before, after domain.Project, op Op) []Op {
	if len(op.IDs) == 0 {
		return nil
	}
	id := op.IDs[0]
	old, wasThere := rowOf(&before, id)
	now, isThere := rowOf(&after, id)
	if !wasThere || !isThere {
		return nil
	}
	if op.Action == "task.title" {
		return []Op{{Action: "task.title", IDs: []string{id}, Title: old.Title, Base: now.Title}}
	}
	return []Op{{Action: "task.description", IDs: []string{id}, Text: old.Description, Base: now.Description}}
}

// undoValues sets each row back to its old value, one op per value in the order rows show them.
func undoValues(before, after domain.Project, op Op) []Op {
	var ops []Op
	at := make(map[string]int)
	done := make(map[string]bool)
	for _, id := range op.IDs {
		old, ok := rowOf(&before, id)
		now, _ := rowOf(&after, id)
		value := valueOf(op.Action, old)
		if !ok || done[id] || old.IsSeparator() || value == valueOf(op.Action, now) {
			continue
		}
		done[id] = true
		i, seen := at[value]
		if !seen {
			i, at[value] = len(ops), len(ops)
			ops = append(ops, Op{Action: op.Action, Value: value})
		}
		ops[i].IDs = append(ops[i].IDs, id)
	}
	return ops
}

func valueOf(action string, t domain.Task) string {
	switch action {
	case "status.set":
		return t.Status
	case "priority.set":
		return t.Priority
	case "tag.set":
		return t.TagColor
	}
	return strconv.Itoa(t.EstimateMinutes)
}

// undoDelete restores each deleted block above the row that now follows its place. A parent deleted
// with keep comes back alone, then takes back its freed children and any deleted with it, in order.
func undoDelete(before, after domain.Project) []Op {
	parentIn := make(map[string]string)
	for _, c := range after.Categories {
		for _, t := range c.Tasks {
			parentIn[t.ID] = t.ParentID
		}
	}
	left := func(id string) bool {
		_, ok := parentIn[id]
		return ok
	}
	var ops []Op
	for _, c := range before.Categories {
		tasks := c.Tasks
		restore := func(parent string, from int, rows []domain.Task) Op {
			stays := func(t domain.Task) bool {
				p, ok := parentIn[t.ID]
				return ok && p == parent
			}
			return Op{Action: "task.restore", Cat: c.ID, Parent: parent, Before: nextSibling(tasks, from, parent, stays), Rows: rows}
		}
		for i := 0; i < len(tasks); i++ {
			t := tasks[i]
			if left(t.ID) {
				continue
			}
			if t.ParentID != "" {
				ops = append(ops, restore(t.ParentID, i+1, []domain.Task{t}))
				continue
			}
			end := domain.BlockEnd(tasks, i)
			children := tasks[i+1 : end]
			if !slices.ContainsFunc(children, func(c domain.Task) bool { return left(c.ID) }) {
				ops = append(ops, restore("", end, slices.Clone(tasks[i:end])))
			} else {
				ops = append(ops, restore("", i+1, []domain.Task{t}))
				for _, child := range children {
					if left(child.ID) {
						ops = append(ops, Op{Action: "task.place", IDs: []string{child.ID}, Cat: c.ID, Parent: t.ID})
					} else {
						ops = append(ops, Op{Action: "task.restore", Cat: c.ID, Parent: t.ID, Rows: []domain.Task{child}})
					}
				}
			}
			i = end - 1
		}
	}
	return ops
}

// undoMove places each moved block back above the old sibling that follows it and stayed put,
// in their old order so blocks sharing that sibling land back in order.
func undoMove(before domain.Project, ids []string) []Op {
	picked := make(map[string]bool, len(ids))
	for _, id := range ids {
		picked[id] = true
	}
	moved := make(map[string]bool)
	for _, c := range before.Categories {
		for _, t := range domain.PickBlocks(c.Tasks, picked) {
			moved[t.ID] = true
		}
	}
	var ops []Op
	for _, c := range before.Categories {
		for i := 0; i < len(c.Tasks); {
			t, end := c.Tasks[i], i+1
			if picked[t.ID] {
				end = domain.BlockEnd(c.Tasks, i)
				stays := func(s domain.Task) bool { return s.ParentID == t.ParentID && !moved[s.ID] }
				ops = append(ops, Op{Action: "task.place", IDs: []string{t.ID}, Cat: c.ID, Parent: t.ParentID, Before: nextSibling(c.Tasks, end, t.ParentID, stays)})
			}
			i = end
		}
	}
	return ops
}

// nextSibling is the first row of tasks[from:] at parent's level that stays. At the top level it
// looks past children too: deleting their parent with keep frees them there.
func nextSibling(tasks []domain.Task, from int, parent string, stays func(domain.Task) bool) string {
	for _, t := range tasks[from:] {
		if parent != "" && t.ParentID != parent {
			break
		}
		if stays(t) {
			return t.ID
		}
	}
	return ""
}

// landing keeps the ops that change something, applied in turn on a copy of after.
func landing(after domain.Project, ops []Op) []Op {
	p := Clone(after)
	var out []Op
	for _, op := range ops {
		if _, err := Apply(&p, op); !errors.Is(err, ErrNoChange) {
			out = append(out, op)
		}
	}
	return out
}
