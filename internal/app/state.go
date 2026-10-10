package app

import (
	"charm.land/bubbles/v2/textinput"

	"phasionary/internal/app/selection"
	"phasionary/internal/domain"
	"phasionary/internal/ui"
)

var filterStatuses = []string{
	domain.StatusTodo,
	domain.StatusInProgress,
	domain.StatusCompleted,
	domain.StatusCancelled,
}

// filterPriorities lists the priority filter rows: every level highest-first,
// followed by a trailing "" "none" bucket matching tasks with no priority.
var filterPriorities = append(append([]string{}, domain.PriorityOrder...), "")

// filterTagColors lists the tag filter rows: the palette followed by a trailing
// "" "untagged" bucket, which matches tasks that carry no tag color.
var filterTagColors = append(append([]string{}, domain.TagColorCycle...), "")

type FilterView int

const (
	FilterViewHub FilterView = iota
	FilterViewStatus
	FilterViewPriority
	FilterViewCategory
	FilterViewTag
)

const (
	FilterHubStatus int = iota
	FilterHubPriority
	FilterHubCategory
	FilterHubTag
	FilterHubClearAll
)

const filterHubCount = 5

type FilterState struct {
	view             FilterView
	hubSelected      int
	statusSelected   int
	prioritySelected int
	categorySelected int
	tagSelected      int
	statuses         map[string]bool
	priorities       map[string]bool
	categories       map[string]bool
	tags             map[string]bool
}

func NewFilterState() FilterState {
	return FilterState{
		view:       FilterViewHub,
		statuses:   make(map[string]bool),
		priorities: make(map[string]bool),
		categories: make(map[string]bool),
		tags:       make(map[string]bool),
	}
}

func (f *FilterState) View() FilterView {
	return f.view
}

func (f *FilterState) SetView(v FilterView) {
	f.view = v
}

func (f *FilterState) ResetToHub() {
	f.view = FilterViewHub
}

func (f *FilterState) HubSelected() int {
	return f.hubSelected
}

func (f *FilterState) Selected() int {
	switch f.view {
	case FilterViewStatus:
		return f.statusSelected
	case FilterViewPriority:
		return f.prioritySelected
	case FilterViewCategory:
		return f.categorySelected
	case FilterViewTag:
		return f.tagSelected
	default:
		return f.hubSelected
	}
}

func (f *FilterState) MoveUp() {
	switch f.view {
	case FilterViewHub:
		if f.hubSelected > 0 {
			f.hubSelected--
		}
	case FilterViewStatus:
		if f.statusSelected > 0 {
			f.statusSelected--
		}
	case FilterViewPriority:
		if f.prioritySelected > 0 {
			f.prioritySelected--
		}
	case FilterViewCategory:
		if f.categorySelected > 0 {
			f.categorySelected--
		}
	case FilterViewTag:
		if f.tagSelected > 0 {
			f.tagSelected--
		}
	}
}

func (f *FilterState) MoveDown(catCount int) {
	switch f.view {
	case FilterViewHub:
		if f.hubSelected < filterHubCount-1 {
			f.hubSelected++
		}
	case FilterViewStatus:
		if f.statusSelected < len(filterStatuses)-1 {
			f.statusSelected++
		}
	case FilterViewPriority:
		if f.prioritySelected < len(filterPriorities)-1 {
			f.prioritySelected++
		}
	case FilterViewCategory:
		if catCount > 0 && f.categorySelected < catCount-1 {
			f.categorySelected++
		}
	case FilterViewTag:
		if f.tagSelected < len(filterTagColors)-1 {
			f.tagSelected++
		}
	}
}

func (f *FilterState) ToggleSelected(categories []domain.Category) {
	switch f.view {
	case FilterViewStatus:
		if f.statusSelected >= 0 && f.statusSelected < len(filterStatuses) {
			toggleSetMember(f.statuses, filterStatuses[f.statusSelected])
		}
	case FilterViewPriority:
		if f.prioritySelected >= 0 && f.prioritySelected < len(filterPriorities) {
			toggleSetMember(f.priorities, filterPriorities[f.prioritySelected])
		}
	case FilterViewCategory:
		if f.categorySelected >= 0 && f.categorySelected < len(categories) {
			toggleSetMember(f.categories, categories[f.categorySelected].ID)
		}
	case FilterViewTag:
		if f.tagSelected >= 0 && f.tagSelected < len(filterTagColors) {
			toggleSetMember(f.tags, filterTagColors[f.tagSelected])
		}
	}
}

func toggleSetMember(set map[string]bool, key string) {
	if set[key] {
		delete(set, key)
	} else {
		set[key] = true
	}
}

func (f *FilterState) IsStatusEnabled(status string) bool       { return f.statuses[status] }
func (f *FilterState) IsPriorityEnabled(priority string) bool   { return f.priorities[priority] }
func (f *FilterState) IsCategoryEnabled(categoryID string) bool { return f.categories[categoryID] }
func (f *FilterState) IsTagEnabled(color string) bool           { return f.tags[color] }

func (f *FilterState) StatusCount() int   { return len(f.statuses) }
func (f *FilterState) PriorityCount() int { return len(f.priorities) }
func (f *FilterState) CategoryCount() int { return len(f.categories) }
func (f *FilterState) TagCount() int      { return len(f.tags) }

func (f *FilterState) HasActiveFilter() bool {
	return len(f.statuses) > 0 || len(f.priorities) > 0 || len(f.categories) > 0 || len(f.tags) > 0
}

// CategoryVisible reports whether the category filter keeps the category
// categoryID, whose parent category is parentCategoryID or "".
func (f *FilterState) CategoryVisible(categoryID, parentCategoryID string) bool {
	return len(f.categories) == 0 || f.categories[categoryID] ||
		(parentCategoryID != "" && f.categories[parentCategoryID])
}

// TaskVisible judges a task of the category categoryID, whose parent category
// is parentCategoryID or "": picking a category in the filter picks its
// sub-categories too.
func (f *FilterState) TaskVisible(task domain.Task, categoryID, parentCategoryID string) bool {
	if !f.HasActiveFilter() {
		return true
	}
	if len(f.statuses) > 0 && !f.statuses[task.Status] {
		return false
	}
	if len(f.priorities) > 0 && !f.priorities[task.Priority] {
		return false
	}
	if !f.CategoryVisible(categoryID, parentCategoryID) {
		return false
	}
	// The "untagged" row is keyed by "", which an untagged task's empty TagColor
	// matches directly.
	if len(f.tags) > 0 && !f.tags[task.TagColor] {
		return false
	}
	return true
}

func (f *FilterState) ClearAll() {
	f.statuses = make(map[string]bool)
	f.priorities = make(map[string]bool)
	f.categories = make(map[string]bool)
	f.tags = make(map[string]bool)
	f.statusSelected = 0
	f.prioritySelected = 0
	f.categorySelected = 0
	f.tagSelected = 0
}

type OptionsState struct {
	selectedOption int
}

type ProjectPickerState struct {
	projects     []domain.Project
	selected     int  // index into projects; meaningful only when !onNew
	onNew        bool // the pinned "+ New Project" affordance is selected
	scrollOffset int
	isAdding     bool
	input        textinput.Model
	// filtering narrows the list to projects whose name matches query (fzf
	// style). While filtering, allProjects holds the full ordered set that
	// projects is derived from; query is the live filter text, filter its input.
	filtering   bool
	filter      textinput.Model
	query       string
	allProjects []domain.Project
	fullscreen  bool
}

func (p *ProjectPickerState) reset() {
	*p = ProjectPickerState{}
}

type ConfirmDeleteKind int

const (
	ConfirmDeleteSelection ConfirmDeleteKind = iota
	ConfirmDeleteProject
	ConfirmDeleteVisualRange
)

type ConfirmDeleteState struct {
	Kind        ConfirmDeleteKind
	ProjectID   string
	TaskIDs     []string
	CategoryIDs []string
}

func (c *ConfirmDeleteState) reset() {
	c.Kind = ConfirmDeleteSelection
	c.ProjectID = ""
	c.TaskIDs = nil
	c.CategoryIDs = nil
}

func (p *ProjectPickerState) isOnNewProject() bool {
	return p.onNew
}

// virtualIndex flattens the cursor onto a single list where 0 is the pinned
// New Project row and 1..n are the projects, which makes relative moves (j/k,
// paging) uniform across the boundary.
func (p *ProjectPickerState) virtualIndex() int {
	if p.onNew {
		return 0
	}
	return p.selected + 1
}

// setVirtual maps a virtual index back onto the (onNew, selected) pair,
// clamping to range. Index 0 (or an empty list) lands on New Project.
func (p *ProjectPickerState) setVirtual(v int) {
	n := len(p.projects)
	if v <= 0 || n == 0 {
		p.onNew = true
		p.selected = 0
		return
	}
	p.onNew = false
	p.selected = v - 1
	if p.selected >= n {
		p.selected = n - 1
	}
}

func (p *ProjectPickerState) startAdding() {
	p.isAdding = true
	p.input = textinput.New()
	p.input.Focus()
}

func (p *ProjectPickerState) cancelAdding() {
	p.isAdding = false
	p.input = textinput.Model{}
}

// startFiltering enters the type-to-filter sub-mode, snapshotting the full
// ordered list so clearing the filter restores it. The cursor moves onto the
// first project (never the pinned New Project row, which isn't a filter target).
func (p *ProjectPickerState) startFiltering() {
	p.filtering = true
	p.allProjects = p.projects
	p.query = ""
	p.filter = textinput.New()
	p.filter.Focus()
	p.onNew = false
	p.selected = 0
	p.scrollOffset = 0
}

// cancelFiltering leaves the filter sub-mode and restores the full project list,
// keeping the highlighted project selected when it survives the restore.
func (p *ProjectPickerState) cancelFiltering(visible int) {
	var keep string
	if p.selected < len(p.projects) {
		keep = p.projects[p.selected].ID
	}
	p.filtering = false
	p.filter = textinput.Model{}
	p.query = ""
	p.projects = p.allProjects
	p.allProjects = nil
	p.onNew = false
	p.selected = 0
	for i, pr := range p.projects {
		if pr.ID == keep {
			p.selected = i
			break
		}
	}
	p.ensureVisible(visible)
}

type FoldState struct {
	folded map[string]bool
}

func NewFoldState() FoldState {
	return FoldState{
		folded: make(map[string]bool),
	}
}

func NewFoldStateFrom(foldedIDs []string) FoldState {
	folded := make(map[string]bool, len(foldedIDs))
	for _, id := range foldedIDs {
		folded[id] = true
	}
	return FoldState{folded: folded}
}

func (f *FoldState) FoldedIDs() []string {
	ids := make([]string, 0, len(f.folded))
	for id := range f.folded {
		ids = append(ids, id)
	}
	return ids
}

func (f *FoldState) IsFolded(categoryID string) bool {
	return f.folded[categoryID]
}

func (f *FoldState) Toggle(categoryID string) {
	if f.folded[categoryID] {
		delete(f.folded, categoryID)
	} else {
		f.folded[categoryID] = true
	}
}

func (f *FoldState) FoldAll(categoryIDs []string) {
	for _, id := range categoryIDs {
		f.folded[id] = true
	}
}

func (f *FoldState) UnfoldAll() {
	f.folded = make(map[string]bool)
}

func (f *FoldState) HasFolded() bool {
	return len(f.folded) > 0
}

type EditState struct {
	input     textinput.Model
	isAdding  bool
	newItemID string
	itemType  selection.FocusKind
	wrap      editWrapCache
	// err is shown under the edited row until the text changes.
	err string
	// mergeInto is the ID of the category whose name the edit collides with
	// while the merge offer is open.
	mergeInto string
}

type editWrapCache struct {
	value     string
	available int
	rows      []ui.LineSpan
}

// wrapFor reuses the last wrap: a keystroke lays the buffer out several times
// over, and an editor's wrap is linear in the buffer.
func (e *EditState) wrapFor(value string, available int) []ui.LineSpan {
	if c := e.wrap; c.rows != nil && c.available == available && c.value == value {
		return c.rows
	}
	e.wrap = editWrapCache{value: value, available: available, rows: ui.WrapSpans(value, available)}
	return e.wrap.rows
}

func (e *EditState) reset() {
	e.input = textinput.New()
	e.isAdding = false
	e.newItemID = ""
	e.itemType = selection.FocusProject
	e.wrap = editWrapCache{}
	e.err = ""
	e.mergeInto = ""
}

func newEditState(value string, isAdding bool, newID string, kind selection.FocusKind) EditState {
	ti := textinput.New()
	// No CharLimit: a title may be any length; the row it renders into is what
	// stays bounded.
	ti.SetValue(value)
	ti.SetCursor(len([]rune(value)))
	ti.Focus()
	return EditState{
		input:     ti,
		isAdding:  isAdding,
		newItemID: newID,
		itemType:  kind,
	}
}
