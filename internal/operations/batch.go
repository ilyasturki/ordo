package operations

import (
	"strings"

	"github.com/ilyasturki/ordo/internal/domain"
)

func realTasks(p *domain.Project, ids []string) []*domain.Task {
	var out []*domain.Task
	for _, id := range ids {
		ci, ti, err := LocateTask(p, id)
		if err != nil || !p.Categories[ci].Tasks[ti].IsTask() {
			continue
		}
		out = append(out, &p.Categories[ci].Tasks[ti])
	}
	return out
}

// taggedRows is realTasks with separators and notes, which carry a mark too.
func taggedRows(p *domain.Project, ids []string) []*domain.Task {
	var out []*domain.Task
	for _, id := range ids {
		if ci, ti, err := LocateTask(p, id); err == nil {
			out = append(out, &p.Categories[ci].Tasks[ti])
		}
	}
	return out
}

// CycleStatus sets every task to the status after the first one's, so a batch steps together.
func CycleStatus(p *domain.Project, ids []string, reverse bool) (string, int, error) {
	tasks := realTasks(p, ids)
	if len(tasks) == 0 {
		return "", 0, ErrNoChange
	}
	next := domain.NextStatus(tasks[0].Status, reverse)
	n, err := SetStatuses(p, ids, next)
	return next, n, err
}

// SetStatuses sets status on every task among ids and counts the ones it changed.
func SetStatuses(p *domain.Project, ids []string, status string) (int, error) {
	if err := domain.ValidateStatus(status); err != nil {
		return 0, err
	}
	changed := 0
	for _, t := range realTasks(p, ids) {
		if t.Status == status {
			continue
		}
		_ = t.SetStatus(status)
		changed++
	}
	if changed == 0 {
		return 0, ErrNoChange
	}
	return changed, nil
}

// SetPriorities sets priority on every task among ids and counts the ones it changed.
func SetPriorities(p *domain.Project, ids []string, priority string) (int, error) {
	if err := domain.ValidatePriority(priority); err != nil {
		return 0, err
	}
	priority = domain.CanonicalPriority(priority)
	changed := 0
	for _, t := range realTasks(p, ids) {
		if t.Priority == priority {
			continue
		}
		_ = t.SetPriority(priority)
		changed++
	}
	if changed == 0 {
		return 0, ErrNoChange
	}
	return changed, nil
}

// StepPriority raises (up) or lowers one task's priority a notch.
func StepPriority(p *domain.Project, id string, up bool) error {
	tasks := realTasks(p, []string{id})
	if len(tasks) == 0 {
		return ErrNoChange
	}
	changed := tasks[0].DecreasePriority
	if up {
		changed = tasks[0].IncreasePriority
	}
	if !changed() {
		return ErrNoChange
	}
	return nil
}

// CycleTags gives every row the mark after the first one's, in a round that starts at the
// first mark no open task beside the batch uses and ends on no tag.
func CycleTags(p *domain.Project, ids []string) (string, error) {
	tasks := taggedRows(p, ids)
	if len(tasks) == 0 {
		return "", ErrNoChange
	}
	next := domain.RotateTagColor(tasks[0].TagColor, domain.FreeTagColor(tasksAround(p, ids)))
	for _, t := range tasks {
		_ = t.SetTagColor(next)
	}
	return next, nil
}

// tasksAround lists the tasks of every category ids touch, minus ids.
func tasksAround(p *domain.Project, ids []string) []domain.Task {
	picked := make(map[string]bool, len(ids))
	cats := make(map[int]bool)
	for _, id := range ids {
		picked[id] = true
		if ci, _, err := LocateTask(p, id); err == nil {
			cats[ci] = true
		}
	}
	var out []domain.Task
	for ci := range cats {
		for _, t := range p.Categories[ci].Tasks {
			if !picked[t.ID] {
				out = append(out, t)
			}
		}
	}
	return out
}

// SetTags sets color ("" clears) on every row among ids and counts the ones it changed.
func SetTags(p *domain.Project, ids []string, color string) (int, error) {
	if err := domain.ValidateTagColor(color); err != nil {
		return 0, err
	}
	color = domain.CanonicalTagColor(color)
	changed := 0
	for _, t := range taggedRows(p, ids) {
		if t.TagColor == color {
			continue
		}
		_ = t.SetTagColor(color)
		changed++
	}
	if changed == 0 {
		return 0, ErrNoChange
	}
	return changed, nil
}

// SetKinds turns every task among ids into a note (kind "note") or every note into a todo task
// (kind ""), refusing them all when one task holds what a note can't; separators stay.
func SetKinds(p *domain.Project, ids []string, kind string) (int, error) {
	if kind != "" && kind != domain.KindNote {
		return 0, ErrInvalidKind
	}
	var rows []*domain.Task
	for _, t := range taggedRows(p, ids) {
		if !t.IsSeparator() && t.Kind != kind {
			rows = append(rows, t)
		}
	}
	if len(rows) == 0 {
		return 0, ErrNoChange
	}
	if kind == domain.KindNote {
		for _, t := range rows {
			if err := noteRefusal(p, *t); err != nil {
				return 0, err
			}
		}
	}
	for _, t := range rows {
		t.Kind, t.Status, t.Priority, t.EstimateMinutes, t.CompletionDate = kind, "", "", 0, ""
		if kind == "" {
			t.Status = domain.StatusTodo
		}
		t.UpdatedAt = domain.NowTimestamp()
	}
	return len(rows), nil
}

func noteRefusal(p *domain.Project, t domain.Task) error {
	switch {
	case t.ParentID != "":
		return refuse("%s is a subtask, and a note doesn't nest: un-nest it first", quote(t.Title))
	case strings.Contains(t.Description, "!["):
		return refuse("%s has images, and a note holds one line of text", quote(t.Title))
	case t.Description != "":
		return refuse("%s has a description, and a note holds one line of text", quote(t.Title))
	}
	ci, _, _ := LocateTask(p, t.ID)
	for _, c := range p.Categories[ci].Tasks {
		if c.ParentID == t.ID {
			return refuse("%s has subtasks, and a note doesn't nest", quote(t.Title))
		}
	}
	return nil
}

// SetEstimates sets minutes on every task among ids and counts the ones it changed.
func SetEstimates(p *domain.Project, ids []string, minutes int) (int, error) {
	if minutes < 0 {
		return 0, ErrNegativeEstimate
	}
	changed := 0
	for _, t := range realTasks(p, ids) {
		if t.EstimateMinutes == minutes {
			continue
		}
		t.SetEstimate(minutes)
		changed++
	}
	if changed == 0 {
		return 0, ErrNoChange
	}
	return changed, nil
}

func SetCategoryEstimate(p *domain.Project, categoryID string, minutes int) error {
	if minutes < 0 {
		return ErrNegativeEstimate
	}
	idx, err := p.FindCategoryByID(categoryID)
	if err != nil {
		return err
	}
	if p.Categories[idx].EstimateMinutes == minutes {
		return ErrNoChange
	}
	p.Categories[idx].SetEstimate(minutes)
	return nil
}

// JoinDescription appends text as a new line of a description.
func JoinDescription(existing, text string) string {
	existing = strings.TrimRight(existing, "\n")
	if existing == "" || text == "" {
		return existing + text
	}
	return existing + "\n" + text
}

func PrependDescription(existing, text string) string {
	existing = strings.Trim(existing, "\n")
	if existing == "" || text == "" {
		return text + existing
	}
	return text + "\n\n" + existing
}

func SplitTitle(p *domain.Project, id, title, rest string) (domain.Task, error) {
	tasks := realTasks(p, []string{id})
	if len(tasks) == 0 {
		return domain.Task{}, domain.ErrTaskNotFound
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return domain.Task{}, ErrTitleRequired
	}
	if err := domain.ValidateLine(title); err != nil {
		return domain.Task{}, err
	}
	t := tasks[0]
	t.Title = title
	t.Description = PrependDescription(t.Description, strings.TrimSpace(rest))
	t.UpdatedAt = domain.NowTimestamp()
	return *t, nil
}

// AppendDescription adds text as the last line of the task's description.
func AppendDescription(p *domain.Project, id, text string) (domain.Task, error) {
	tasks := realTasks(p, []string{id})
	if len(tasks) == 0 {
		return domain.Task{}, domain.ErrTaskNotFound
	}
	t := tasks[0]
	t.Description = JoinDescription(t.Description, text)
	t.UpdatedAt = domain.NowTimestamp()
	return *t, nil
}
