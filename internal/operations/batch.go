package operations

import (
	"strings"

	"github.com/ilyasturki/ordo/internal/domain"
)

func realTasks(p *domain.Project, ids []string) []*domain.Task {
	var out []*domain.Task
	for _, id := range ids {
		ci, ti, err := LocateTask(p, id)
		if err != nil || p.Categories[ci].Tasks[ti].IsSeparator() {
			continue
		}
		out = append(out, &p.Categories[ci].Tasks[ti])
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

// CycleTags gives every task the mark after the first one's; an untagged first takes the
// first mark no open task beside the batch uses.
func CycleTags(p *domain.Project, ids []string) (string, error) {
	tasks := realTasks(p, ids)
	if len(tasks) == 0 {
		return "", ErrNoChange
	}
	next := domain.NextTagColor(tasks[0].TagColor)
	if tasks[0].TagColor == "" {
		next = domain.FreeTagColor(tasksAround(p, ids))
	}
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

// SetTags sets color ("" clears) on every task among ids and counts the ones it changed.
func SetTags(p *domain.Project, ids []string, color string) (int, error) {
	if err := domain.ValidateTagColor(color); err != nil {
		return 0, err
	}
	color = domain.CanonicalTagColor(color)
	changed := 0
	for _, t := range realTasks(p, ids) {
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
