package operations

import (
	"slices"
	"strings"

	"github.com/ilyasturki/ordo/internal/domain"
)

// TaskFields carries the writable fields of a task. It is the single unit the
// CLI and web front doors both build and hand to CreateTask, so the two
// surfaces create tasks through identical logic.
type TaskFields struct {
	Title       string
	Status      string
	Priority    string
	Estimate    int
	Description string
	TagColor    string
	// Kind selects the row to build: empty for an ordinary task,
	// domain.KindSeparator for a divider. A separator carries only a Title (its
	// label, which may be empty); every other field must be unset.
	Kind string
	// ParentID makes the task the last subtask of that top-level task.
	ParentID string
}

// isSeparatorKind reports whether f describes a separator row.
func (f TaskFields) isSeparatorKind() bool { return f.Kind == domain.KindSeparator }

// validateSeparatorFields rejects the fields a separator does not have, so a
// client can't smuggle a status or an estimate onto a divider.
func validateSeparatorFields(f TaskFields) error {
	if f.Status != "" || f.Priority != "" || f.Estimate != 0 ||
		f.Description != "" || f.TagColor != "" || f.ParentID != "" {
		return ErrSeparatorFieldNotAllowed
	}
	return nil
}

// CreateTask builds a task from f and appends it to the category identified by
// categoryID. An empty Status defaults to todo; Status and Priority, when set,
// are applied through the domain setters so their side effects (UpdatedAt,
// CompletionDate) and validation hold. Returns ErrTitleRequired, an invalid
// status/priority error, or domain.ErrCategoryNotFound.
func CreateTask(p *domain.Project, categoryID string, f TaskFields) (domain.Task, error) {
	return CreateTaskAfter(p, categoryID, f, "")
}

// validateTaskText screens the free-text fields of a write.
//
// This is the single gate every writer shares — the TUI and the CLI both
// build a TaskFields and hand it here — which is why the check lives at this
// layer rather than in each front door. Titles render on one line and reject
// every control character; descriptions are legitimately multi-line, so they
// keep newlines and tabs and reject the rest.
func validateTaskText(f TaskFields) error {
	if err := domain.ValidateLine(f.Title); err != nil {
		return err
	}
	return domain.ValidateMultiline(f.Description)
}

func CreateTaskAfter(
	p *domain.Project,
	categoryID string,
	f TaskFields,
	afterTaskID string,
) (domain.Task, error) {
	return createTask(p, categoryID, f, afterTaskID, false)
}

// CreateTaskAtTop puts the new row first in its category, where the TUI's a lands on a header.
func CreateTaskAtTop(p *domain.Project, categoryID string, f TaskFields) (domain.Task, error) {
	return createTask(p, categoryID, f, "", true)
}

func createTask(p *domain.Project, categoryID string, f TaskFields, afterTaskID string, top bool) (domain.Task, error) {
	switch f.Kind {
	case "", domain.KindSeparator:
	default:
		return domain.Task{}, ErrInvalidKind
	}
	// A separator's label is optional (an unlabeled one renders as a plain
	// rule), so the title requirement applies only to real tasks.
	if !f.isSeparatorKind() && strings.TrimSpace(f.Title) == "" {
		return domain.Task{}, ErrTitleRequired
	}
	if err := validateTaskText(f); err != nil {
		return domain.Task{}, err
	}
	if f.isSeparatorKind() {
		if err := validateSeparatorFields(f); err != nil {
			return domain.Task{}, err
		}
	}
	if f.Estimate < 0 {
		return domain.Task{}, ErrNegativeEstimate
	}
	idx, err := p.FindCategoryByID(categoryID)
	if err != nil {
		return domain.Task{}, err
	}
	cat := &p.Categories[idx]

	// Resolve the anchor before building the row so a bad id fails without
	// mutating anything.
	insertAt := len(cat.Tasks)
	if top {
		insertAt = 0
	}
	parentID := ""
	switch {
	case f.ParentID != "":
		parent, err := cat.FindTaskByID(f.ParentID)
		if err != nil {
			return domain.Task{}, err
		}
		if cat.Tasks[parent].IsSeparator() {
			return domain.Task{}, domain.ErrSeparatorNest
		}
		if cat.Tasks[parent].ParentID != "" {
			return domain.Task{}, domain.ErrNestDepth
		}
		insertAt, parentID = domain.BlockEnd(cat.Tasks, parent), f.ParentID
	case afterTaskID != "":
		anchor, err := cat.FindTaskByID(afterTaskID)
		if err != nil {
			return domain.Task{}, err
		}
		insertAt, parentID = domain.BlockEnd(cat.Tasks, anchor), cat.Tasks[anchor].ParentID
		if f.isSeparatorKind() && parentID != "" {
			insertAt, parentID = domain.BlockEnd(cat.Tasks, domain.BlockStart(cat.Tasks, anchor)), ""
		}
	}

	if f.isSeparatorKind() {
		sep, err := domain.NewSeparator()
		if err != nil {
			return domain.Task{}, err
		}
		sep.Title = strings.TrimSpace(f.Title)
		cat.InsertTask(insertAt, sep)
		return sep, nil
	}

	task, err := domain.NewTask(f.Title)
	if err != nil {
		return domain.Task{}, err
	}
	if f.Status != "" {
		if err := task.SetStatus(f.Status); err != nil {
			return domain.Task{}, err
		}
	}
	if f.Priority != "" {
		if err := task.SetPriority(f.Priority); err != nil {
			return domain.Task{}, err
		}
	}
	task.SetEstimate(f.Estimate)
	task.Description = f.Description
	task.ParentID = parentID
	if f.TagColor != "" {
		if err := task.SetTagColor(f.TagColor); err != nil {
			return domain.Task{}, err
		}
	}

	cat.InsertTask(insertAt, task)
	return task, nil
}

// findTask resolves categoryID→category and taskID→task within p, returning a
// pointer to the stored task so callers mutate it in place. Returns
// domain.ErrCategoryNotFound or domain.ErrTaskNotFound.
func findTask(p *domain.Project, categoryID, taskID string) (*domain.Task, error) {
	cidx, err := p.FindCategoryByID(categoryID)
	if err != nil {
		return nil, err
	}
	tidx, err := p.Categories[cidx].FindTaskByID(taskID)
	if err != nil {
		return nil, err
	}
	return &p.Categories[cidx].Tasks[tidx], nil
}

// findRealTask is findTask restricted to ordinary tasks. The field setters go
// through it so every door into a task's status/priority/estimate refuses a
// separator, not just the partial-update path.
func findRealTask(p *domain.Project, categoryID, taskID string) (*domain.Task, error) {
	task, err := findTask(p, categoryID, taskID)
	if err != nil {
		return nil, err
	}
	if task.IsSeparator() {
		return nil, ErrSeparatorFieldNotAllowed
	}
	return task, nil
}

// SetTaskStatus applies status to the identified task through domain.SetStatus,
// so completion-date bookkeeping and validation hold.
func SetTaskStatus(p *domain.Project, categoryID, taskID, status string) (domain.Task, error) {
	task, err := findRealTask(p, categoryID, taskID)
	if err != nil {
		return domain.Task{}, err
	}
	if err := task.SetStatus(status); err != nil {
		return domain.Task{}, err
	}
	return *task, nil
}

// TaskUpdate carries a partial task edit: a nil field is left untouched. The
// pointers exist so a caller can tell an omitted field from one explicitly
// set to its zero value — clearing a description and not touching it are
// different requests.
type TaskUpdate struct {
	Title       *string
	Status      *string
	Priority    *string
	Estimate    *int
	Description *string
}

// UpdateTask applies a partial edit to the identified task, routing each field
// through the domain setters where they exist so validation and the
// UpdatedAt/CompletionDate bookkeeping hold. An update with no fields set
// returns the task unchanged along with ErrNoChange, so an empty PATCH doesn't
// bump the project's timestamp. Returns ErrTitleRequired, ErrNegativeEstimate,
// an invalid status/priority error, domain.ErrCategoryNotFound, or
// domain.ErrTaskNotFound.
func UpdateTask(p *domain.Project, categoryID, taskID string, u TaskUpdate) (domain.Task, error) {
	task, err := findTask(p, categoryID, taskID)
	if err != nil {
		return domain.Task{}, err
	}
	// A separator is a divider, not a task: the only thing it has is its label,
	// and that label may legitimately be empty (an unlabeled separator renders
	// as a plain rule). Both rules differ from the task path, so split here
	// before the ordinary validation runs.
	if task.IsSeparator() {
		return updateSeparator(task, u)
	}

	// Validate everything up front: a rejected field must not leave the task
	// half-updated, since the caller saves whatever the project holds.
	var title string
	if u.Title != nil {
		title = strings.TrimSpace(*u.Title)
		if title == "" {
			return domain.Task{}, ErrTitleRequired
		}
		if err := domain.ValidateLine(title); err != nil {
			return domain.Task{}, err
		}
	}
	if u.Description != nil {
		if err := domain.ValidateMultiline(*u.Description); err != nil {
			return domain.Task{}, err
		}
	}
	if u.Estimate != nil && *u.Estimate < 0 {
		return domain.Task{}, ErrNegativeEstimate
	}
	if u.Status != nil {
		if err := domain.ValidateStatus(*u.Status); err != nil {
			return domain.Task{}, err
		}
	}
	if u.Priority != nil {
		if err := domain.ValidatePriority(*u.Priority); err != nil {
			return domain.Task{}, err
		}
	}

	if u.Title == nil && u.Status == nil && u.Priority == nil && u.Estimate == nil && u.Description == nil {
		return *task, ErrNoChange
	}

	if u.Title != nil {
		task.Title = title
		task.UpdatedAt = domain.NowTimestamp()
	}
	if u.Description != nil {
		// Match CreateTask: trailing newlines from a multi-line editor are noise.
		task.Description = strings.TrimRight(*u.Description, "\n")
		task.UpdatedAt = domain.NowTimestamp()
	}
	if u.Status != nil {
		if err := task.SetStatus(*u.Status); err != nil {
			return domain.Task{}, err
		}
	}
	if u.Priority != nil {
		if err := task.SetPriority(*u.Priority); err != nil {
			return domain.Task{}, err
		}
	}
	if u.Estimate != nil {
		task.SetEstimate(*u.Estimate)
	}
	return *task, nil
}

// updateSeparator applies the only edit a separator accepts: its label. Any
// other field in u is rejected rather than ignored, so a client that thinks it
// set a status finds out it didn't.
func updateSeparator(sep *domain.Task, u TaskUpdate) (domain.Task, error) {
	if u.Status != nil || u.Priority != nil || u.Estimate != nil || u.Description != nil {
		return domain.Task{}, ErrSeparatorFieldNotAllowed
	}
	if u.Title == nil {
		return *sep, ErrNoChange
	}
	// Empty is allowed here (unlike a task): it clears the label back to a
	// plain rule.
	label := strings.TrimSpace(*u.Title)
	if err := domain.ValidateLine(label); err != nil {
		return domain.Task{}, err
	}
	if label == sep.Title {
		return *sep, ErrNoChange
	}
	sep.Title = label
	sep.UpdatedAt = domain.NowTimestamp()
	return *sep, nil
}

// DeleteTask removes the identified task, with its subtasks, from its category
// and returns the task as it was, so callers can report or undo the deletion.
func DeleteTask(p *domain.Project, categoryID, taskID string) (domain.Task, error) {
	cidx, err := p.FindCategoryByID(categoryID)
	if err != nil {
		return domain.Task{}, err
	}
	cat := &p.Categories[cidx]
	tidx, err := cat.FindTaskByID(taskID)
	if err != nil {
		return domain.Task{}, err
	}
	task := cat.Tasks[tidx]
	cat.RemoveTaskBlock(tidx)
	return task, nil
}

// SetTaskPriority applies priority to the identified task through
// domain.SetPriority.
func SetTaskPriority(p *domain.Project, categoryID, taskID, priority string) (domain.Task, error) {
	task, err := findRealTask(p, categoryID, taskID)
	if err != nil {
		return domain.Task{}, err
	}
	if err := task.SetPriority(priority); err != nil {
		return domain.Task{}, err
	}
	return *task, nil
}

// SetTaskEstimate sets the identified task's estimate in minutes.
func SetTaskEstimate(p *domain.Project, categoryID, taskID string, minutes int) (domain.Task, error) {
	if minutes < 0 {
		return domain.Task{}, ErrNegativeEstimate
	}
	task, err := findRealTask(p, categoryID, taskID)
	if err != nil {
		return domain.Task{}, err
	}
	task.SetEstimate(minutes)
	return *task, nil
}

// MoveTaskWithinCategory moves by delta sibling blocks; off the end is ErrNoChange, so no save.
func MoveTaskWithinCategory(p *domain.Project, categoryID, taskID string, delta int) (domain.Task, error) {
	cidx, err := p.FindCategoryByID(categoryID)
	if err != nil {
		return domain.Task{}, err
	}
	cat := &p.Categories[cidx]
	tidx, err := cat.FindTaskByID(taskID)
	if err != nil {
		return domain.Task{}, err
	}
	before, stamp := slices.Clone(cat.Tasks), cat.UpdatedAt
	at := tidx
	for range max(delta, -delta) {
		var ok bool
		if at, ok = cat.MoveTaskBlock(at, sign(delta)); !ok {
			cat.Tasks, cat.UpdatedAt = before, stamp
			return cat.Tasks[tidx], ErrNoChange
		}
	}
	return cat.Tasks[at], nil
}

func sign(n int) int {
	if n < 0 {
		return -1
	}
	return 1
}

// MoveTaskToCategory moves the identified task out of srcCatID and into
// dstCatID. Propagates domain.MoveTask's error when the two categories are the
// same or an id does not resolve.
func MoveTaskToCategory(p *domain.Project, srcCatID, taskID, dstCatID string) (domain.Task, error) {
	srcIdx, err := p.FindCategoryByID(srcCatID)
	if err != nil {
		return domain.Task{}, err
	}
	tidx, err := p.Categories[srcIdx].FindTaskByID(taskID)
	if err != nil {
		return domain.Task{}, err
	}
	dstIdx, err := p.FindCategoryByID(dstCatID)
	if err != nil {
		return domain.Task{}, err
	}
	task := p.Categories[srcIdx].Tasks[tidx]
	if err := p.MoveTask(srcIdx, tidx, dstIdx); err != nil {
		return domain.Task{}, err
	}
	return task, nil
}
