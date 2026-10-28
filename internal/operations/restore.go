package operations

import (
	"slices"
	"strings"

	"github.com/ilyasturki/ordo/internal/domain"
)

var errBadRows = refuse("Can't restore those rows")

// restoreRows puts back rows a delete removed, as they were. A row the project holds again stays
// where it is, and the children it had go back last under it.
func restoreRows(p *domain.Project, op Op) (Outcome, error) {
	if err := checkRows(op.Rows); err != nil {
		return Outcome{}, err
	}
	rows := slices.DeleteFunc(slices.Clone(op.Rows), func(t domain.Task) bool {
		_, _, err := LocateTask(p, t.ID)
		return err == nil
	})
	if len(rows) == 0 {
		return Outcome{}, ErrNoChange
	}
	cat, parent, before := op.Cat, op.Parent, op.Before
	if head := op.Rows[0].ID; rows[0].ID != head {
		ci, _, _ := LocateTask(p, head)
		cat, parent, before = p.Categories[ci].ID, head, ""
	}
	ci, err := p.FindCategoryByID(cat)
	if err != nil {
		return Outcome{}, err
	}
	tasks := p.Categories[ci].Tasks
	index := len(tasks)
	pi := slices.IndexFunc(tasks, func(t domain.Task) bool { return t.ID == parent })
	if parent == "" || pi < 0 || tasks[pi].ParentID != "" || tasks[pi].IsSeparator() {
		parent, pi = "", -1
	} else {
		index = domain.BlockEnd(tasks, pi)
	}
	if bi := slices.IndexFunc(tasks, func(t domain.Task) bool { return t.ID == before }); before != "" && bi >= 0 {
		index = bi
	}
	if pi >= 0 {
		index = max(pi+1, min(index, domain.BlockEnd(tasks, pi)))
	} else if index < len(tasks) && tasks[index].ParentID != "" {
		index = domain.BlockStart(tasks, index)
	}
	rows[0].ParentID = ""
	index = SettleTasks(tasks, rows, index, parent)
	c := &p.Categories[ci]
	c.Tasks = slices.Insert(c.Tasks, index, rows...)
	now := domain.NowTimestamp()
	c.UpdatedAt, p.UpdatedAt = now, now
	return Outcome{Msg: plural(len(rows), "1 row restored", "%d rows restored"), Row: rows[0].ID}, nil
}

// checkRows takes the rows of a restore as untrusted, like an imported file: each one well formed,
// and every row after the first one of its children.
func checkRows(rows []domain.Task) error {
	if len(rows) == 0 {
		return ErrNoChange
	}
	if err := domain.ValidateProjectText(domain.Project{Categories: []domain.Category{{Tasks: rows}}}); err != nil {
		return err
	}
	seen := make(map[string]bool, len(rows))
	for i, t := range rows {
		if domain.ValidateID(t.ID) != nil || seen[t.ID] {
			return errBadRows
		}
		seen[t.ID] = true
		if i > 0 && (t.ParentID != rows[0].ID || t.IsSeparator() || rows[0].IsSeparator()) {
			return errBadRows
		}
		if err := checkRow(t); err != nil {
			return err
		}
	}
	return nil
}

func checkRow(t domain.Task) error {
	switch {
	case t.Kind != "" && !t.IsSeparator():
		return ErrInvalidKind
	case t.EstimateMinutes < 0:
		return ErrNegativeEstimate
	case !t.IsSeparator() && strings.TrimSpace(t.Title) == "":
		return ErrTitleRequired
	}
	if t.Status != "" || !t.IsSeparator() {
		if err := domain.ValidateStatus(t.Status); err != nil {
			return err
		}
	}
	if err := domain.ValidatePriority(t.Priority); err != nil {
		return err
	}
	return domain.ValidateTagColor(t.TagColor)
}
