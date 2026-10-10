package domain

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
)

const (
	StatusTodo       = "todo"
	StatusInProgress = "in_progress"
	StatusCompleted  = "completed"
	StatusCancelled  = "cancelled"

	PriorityCritical = "critical"
	PriorityHigh     = "high"
	PriorityMedium   = "medium"
	PriorityLow      = "low"
	PriorityTrivial  = "trivial"

	// KindSeparator marks a Task that is really a visual divider between tasks
	// within a category. Its Title holds the optional label; Status/Priority/etc.
	// are unused. An absent/empty Kind is an ordinary task.
	KindSeparator = "separator"

	// Tag marks are stored as stable names, not ANSI numbers. Each is a hue the
	// terminal theme tracks (red and yellow already mean priority, status and
	// search) times a glyph: the plain name is a dot, the -diamond name a diamond.
	TagGreen          = "green"
	TagBlue           = "blue"
	TagMagenta        = "magenta"
	TagCyan           = "cyan"
	TagDiamondSuffix  = "-diamond"
	TagGreenDiamond   = TagGreen + TagDiamondSuffix
	TagBlueDiamond    = TagBlue + TagDiamondSuffix
	TagMagentaDiamond = TagMagenta + TagDiamondSuffix
	TagCyanDiamond    = TagCyan + TagDiamondSuffix
)

// TagColorCycle is the canonical mark order `t` steps through and the single
// source of truth the other tag lists derive from: NextTagColor walks it, and
// the picker and filter build their rows by adding the "no tag" sentinel.
var TagColorCycle = []string{
	TagGreen, TagBlue, TagMagenta, TagCyan,
	TagGreenDiamond, TagBlueDiamond, TagMagentaDiamond, TagCyanDiamond,
}

// PriorityOrder lists the priority levels highest-first and is the single source
// of truth the priority lists derive from: ValidatePriority checks membership
// against it, and the filter/completion rows build from it (the filter appends
// the "" no-priority sentinel). The empty "" level is not part of the order. The
// Increase/Decrease/Cycle transition methods keep their own switches because
// they also fold in the "" level with level-specific behavior.
var PriorityOrder = []string{PriorityCritical, PriorityHigh, PriorityMedium, PriorityLow, PriorityTrivial}

var DefaultCategories = []string{"Feature", "Fix", "Ergonomy", "Documentation", "Research"}

var (
	ErrDuplicateCategoryName = errors.New("category name already exists")
	ErrCategoryNotFound      = errors.New("category not found")
	ErrTaskNotFound          = errors.New("task not found")
)

// ProjectSchemaVersion is the version of the on-disk project format this
// binary reads and writes. Files written before versioning existed carry no
// schema field and are read as version 1. Bump it when an older binary would
// misread a file, or would lose data by saving it without a field it does not
// know; never reuse a number.
const ProjectSchemaVersion = 1

// ErrSchemaTooNew is returned when a project file declares a schema version
// newer than this binary understands. Refusing beats loading: saving through a
// partially-understood format would silently drop the fields the newer version
// added. Callers wrap it with the file's identity.
var ErrSchemaTooNew = errors.New("written by a newer phasionary; upgrade phasionary to open it")

// Project is stored as a single JSON file.
type Project struct {
	// Schema versions the on-disk format; see ProjectSchemaVersion. Zero means
	// the file predates versioning and is read as version 1. Saves stamp the
	// current version.
	Schema     int        `json:"schema,omitempty"`
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	CreatedAt  string     `json:"created_at"`
	UpdatedAt  string     `json:"updated_at"`
	Categories []Category `json:"categories"`
}

// CheckSchema reports whether a decoded project's schema version is one this
// binary can safely operate on.
func CheckSchema(p Project) error {
	if p.Schema > ProjectSchemaVersion {
		return fmt.Errorf("schema version %d, this build reads up to %d: %w",
			p.Schema, ProjectSchemaVersion, ErrSchemaTooNew)
	}
	return nil
}

type Category struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at,omitempty"`
	EstimateMinutes int    `json:"estimate_minutes,omitempty"`
	Tasks           []Task `json:"tasks"`
}

type Task struct {
	ID              string `json:"id"`
	Title           string `json:"title"`
	Status          string `json:"status"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
	Priority        string `json:"priority,omitempty"`
	CompletionDate  string `json:"completion_date,omitempty"`
	EstimateMinutes int    `json:"estimate_minutes,omitempty"`
	Description     string `json:"description,omitempty"`
	Kind            string `json:"kind,omitempty"`
	TagColor        string `json:"tag_color,omitempty"`
}

// IsSeparator reports whether this task is a separator row rather than a real
// task. Separator-aware consumers (status tallies, filters, CLI listings)
// should skip these entries.
func (t Task) IsSeparator() bool {
	return t.Kind == KindSeparator
}

var EstimatePresets = []int{0, 15, 30, 60, 120, 240, 480, 960, 1440, 2400}

func NowTimestamp() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func NormalizeName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// MaxIDLen caps an identifier's length so a hand-written or hostile ID cannot
// produce a filename the OS rejects.
const MaxIDLen = 64

// ErrInvalidID is returned when an identifier contains anything outside
// [A-Za-z0-9_-], is empty, or exceeds MaxIDLen.
var ErrInvalidID = errors.New("invalid id")

// ValidateID checks an identifier that will be interpolated into a filename.
//
// The store maps a project ID straight to <dir>/<id>.json, so the ID is a path
// component and an unchecked one escapes the data directory ("../../x") or
// collides with the .lock and .tmp siblings. This is an allowlist rather than a
// denylist of "../" and friends: excluding '/' makes traversal unrepresentable,
// and excluding '.' rules out both ".." and any confusion with those suffixes,
// without needing to enumerate what an attacker might try.
//
// The set is deliberately wider than what NewID emits so that a hand-written
// project imported with a readable ID keeps it.
func ValidateID(id string) error {
	if id == "" || len(id) > MaxIDLen {
		return ErrInvalidID
	}
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-' || r == '_':
		default:
			return ErrInvalidID
		}
	}
	return nil
}

func NewID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	// Format as UUID-like string.
	parts := []string{
		hex.EncodeToString(b[0:4]),
		hex.EncodeToString(b[4:6]),
		hex.EncodeToString(b[6:8]),
		hex.EncodeToString(b[8:10]),
		hex.EncodeToString(b[10:16]),
	}
	return strings.Join(parts, "-"), nil
}

func NewProject(name string) (Project, error) {
	id, err := NewID()
	if err != nil {
		return Project{}, err
	}
	now := NowTimestamp()
	project := Project{
		ID:         id,
		Name:       name,
		CreatedAt:  now,
		UpdatedAt:  now,
		Categories: []Category{},
	}
	return project, nil
}

func NewCategory(name string) (Category, error) {
	id, err := NewID()
	if err != nil {
		return Category{}, err
	}
	now := NowTimestamp()
	return Category{
		ID:        id,
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
		Tasks:     []Task{},
	}, nil
}

func NewTask(title string) (Task, error) {
	id, err := NewID()
	if err != nil {
		return Task{}, err
	}
	now := NowTimestamp()
	return Task{
		ID:        id,
		Title:     title,
		Status:    StatusTodo,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// NewSeparator creates a bare (unlabeled) separator row. The label lives in
// Title and is set later via editing; an empty label renders as a plain rule.
func NewSeparator() (Task, error) {
	id, err := NewID()
	if err != nil {
		return Task{}, err
	}
	now := NowTimestamp()
	return Task{
		ID:        id,
		Kind:      KindSeparator,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func ValidateStatus(status string) error {
	switch status {
	case StatusTodo, StatusInProgress, StatusCompleted, StatusCancelled:
		return nil
	default:
		return errors.New("invalid status")
	}
}

func ValidatePriority(priority string) error {
	if priority == "" || slices.Contains(PriorityOrder, priority) {
		return nil
	}
	return errors.New("invalid priority")
}

func (t *Task) SetStatus(status string) error {
	if err := ValidateStatus(status); err != nil {
		return err
	}
	t.Status = status
	t.UpdatedAt = NowTimestamp()
	if status == StatusCompleted {
		t.CompletionDate = NowTimestamp()
	} else {
		t.CompletionDate = ""
	}
	return nil
}

func (t *Task) SetPriority(priority string) error {
	if err := ValidatePriority(priority); err != nil {
		return err
	}
	t.Priority = priority
	t.UpdatedAt = NowTimestamp()
	return nil
}

func (t *Task) IncreasePriority() bool {
	var newPriority string
	switch t.Priority {
	case PriorityTrivial:
		newPriority = PriorityLow
	case PriorityLow:
		newPriority = PriorityMedium
	case PriorityMedium, "":
		newPriority = PriorityHigh
	case PriorityHigh:
		newPriority = PriorityCritical
	case PriorityCritical:
		return false
	default:
		newPriority = PriorityMedium
	}
	t.Priority = newPriority
	t.UpdatedAt = NowTimestamp()
	return true
}

func (t *Task) DecreasePriority() bool {
	var newPriority string
	switch t.Priority {
	case PriorityCritical:
		newPriority = PriorityHigh
	case PriorityHigh:
		newPriority = PriorityMedium
	case PriorityMedium, "":
		newPriority = PriorityLow
	case PriorityLow:
		newPriority = PriorityTrivial
	case PriorityTrivial:
		return false
	default:
		newPriority = PriorityMedium
	}
	t.Priority = newPriority
	t.UpdatedAt = NowTimestamp()
	return true
}

func (t *Task) SetEstimate(minutes int) {
	t.EstimateMinutes = minutes
	t.UpdatedAt = NowTimestamp()
}

// NextStatus returns the status one step along the cycle
// (todo→in_progress→completed→cancelled→todo), walking it backwards when
// reverse is set. Any unrecognized value resets to todo.
func NextStatus(status string, reverse bool) string {
	if reverse {
		switch status {
		case StatusTodo:
			return StatusCancelled
		case StatusCancelled:
			return StatusCompleted
		case StatusCompleted:
			return StatusInProgress
		case StatusInProgress:
			return StatusTodo
		default:
			return StatusTodo
		}
	}
	switch status {
	case StatusTodo:
		return StatusInProgress
	case StatusInProgress:
		return StatusCompleted
	case StatusCompleted:
		return StatusCancelled
	case StatusCancelled:
		return StatusTodo
	default:
		return StatusTodo
	}
}

func (t *Task) CycleStatus() bool {
	nextStatus := NextStatus(t.Status, false)
	if nextStatus == t.Status {
		return false
	}
	_ = t.SetStatus(nextStatus)
	return true
}

func (t *Task) CyclePriority() bool {
	var next string
	switch t.Priority {
	case "":
		next = PriorityCritical
	case PriorityCritical:
		next = PriorityHigh
	case PriorityHigh:
		next = PriorityMedium
	case PriorityMedium:
		next = PriorityLow
	case PriorityLow:
		next = PriorityTrivial
	case PriorityTrivial:
		next = ""
	default:
		next = ""
	}
	t.Priority = next
	t.UpdatedAt = NowTimestamp()
	return true
}

func ValidateTagColor(color string) error {
	if color == "" || slices.Contains(TagColorCycle, color) {
		return nil
	}
	return errors.New("invalid tag color")
}

// NextTagColor returns the color that follows color in TagColorCycle, wrapping
// from the last color back to "" (no tag). An empty color starts the cycle at
// the first color; any unrecognized value resets to "" (no tag).
func NextTagColor(color string) string {
	if color == "" {
		return TagColorCycle[0]
	}
	for i, c := range TagColorCycle {
		if c == color && i+1 < len(TagColorCycle) {
			return TagColorCycle[i+1]
		}
	}
	return "" // last color, or any unrecognized value, wraps to no tag
}

// FreeTagColor picks a mark for a new batch among tasks: the first mark in
// TagColorCycle that no open (todo or in progress) task uses, or, when every
// mark is taken, the one the fewest open tasks use.
func FreeTagColor(tasks []Task) string {
	open := make(map[string]int)
	for _, t := range tasks {
		if !t.IsSeparator() && t.TagColor != "" &&
			(t.Status == StatusTodo || t.Status == StatusInProgress) {
			open[t.TagColor]++
		}
	}
	best := TagColorCycle[0]
	for _, c := range TagColorCycle {
		if open[c] < open[best] {
			best = c
		}
		if open[c] == 0 {
			return c
		}
	}
	return best
}

// CycleTag advances the tag one step through the palette, wrapping through none.
// Always reports a change.
func (t *Task) CycleTag() bool {
	t.TagColor = NextTagColor(t.TagColor)
	t.UpdatedAt = NowTimestamp()
	return true
}

// SetTagColor sets a validated palette color, or "" to clear the tag.
func (t *Task) SetTagColor(color string) error {
	if err := ValidateTagColor(color); err != nil {
		return err
	}
	t.TagColor = color
	t.UpdatedAt = NowTimestamp()
	return nil
}

func (t *Task) CycleStatusReverse() bool {
	nextStatus := NextStatus(t.Status, true)
	if nextStatus == t.Status {
		return false
	}
	_ = t.SetStatus(nextStatus)
	return true
}

type StatusCounts struct {
	Todo       int
	InProgress int
	Completed  int
	Cancelled  int
}

func (s StatusCounts) Total() int {
	return s.Todo + s.InProgress + s.Completed + s.Cancelled
}

// TaskCount is the number of real tasks in the category, excluding separator
// rows. Use it wherever a "task count" is surfaced to the user.
func (c *Category) TaskCount() int {
	n := 0
	for _, task := range c.Tasks {
		if !task.IsSeparator() {
			n++
		}
	}
	return n
}

func (c *Category) StatusCounts() StatusCounts {
	var counts StatusCounts
	for _, task := range c.Tasks {
		if task.IsSeparator() {
			continue
		}
		switch task.Status {
		case StatusTodo:
			counts.Todo++
		case StatusInProgress:
			counts.InProgress++
		case StatusCompleted:
			counts.Completed++
		case StatusCancelled:
			counts.Cancelled++
		}
	}
	return counts
}

func (p *Project) StatusCounts() StatusCounts {
	var counts StatusCounts
	for _, cat := range p.Categories {
		c := cat.StatusCounts()
		counts.Todo += c.Todo
		counts.InProgress += c.InProgress
		counts.Completed += c.Completed
		counts.Cancelled += c.Cancelled
	}
	return counts
}

func (c *Category) AggregateStatus() string {
	allFinished := true
	hasInProgress := false
	hasFinished := false
	hasTask := false
	for _, t := range c.Tasks {
		if t.IsSeparator() {
			continue
		}
		hasTask = true
		if t.Status == StatusInProgress {
			hasInProgress = true
		}
		if t.Status == StatusCompleted || t.Status == StatusCancelled {
			hasFinished = true
		} else {
			allFinished = false
		}
	}
	if !hasTask {
		return ""
	}
	if hasInProgress {
		return StatusInProgress
	}
	if allFinished {
		return StatusCompleted
	}
	if hasFinished {
		return StatusInProgress
	}
	return StatusTodo
}

func (c *Category) SetEstimate(minutes int) {
	c.EstimateMinutes = minutes
	c.UpdatedAt = NowTimestamp()
}

func (c *Category) AddTask(task Task) {
	c.Tasks = append(c.Tasks, task)
	c.UpdatedAt = NowTimestamp()
}

func (c *Category) InsertTask(index int, task Task) {
	if index < 0 || index > len(c.Tasks) {
		index = len(c.Tasks)
	}
	c.Tasks = append(c.Tasks, Task{})
	copy(c.Tasks[index+1:], c.Tasks[index:])
	c.Tasks[index] = task
	c.UpdatedAt = NowTimestamp()
}

func (c *Category) RemoveTask(index int) error {
	if index < 0 || index >= len(c.Tasks) {
		return errors.New("task index out of range")
	}
	c.Tasks = append(c.Tasks[:index], c.Tasks[index+1:]...)
	c.UpdatedAt = NowTimestamp()
	return nil
}

func (p *Project) AddCategory(cat Category) {
	p.Categories = append(p.Categories, cat)
	p.UpdatedAt = NowTimestamp()
}

func (p *Project) InsertCategory(index int, cat Category) {
	if index < 0 || index > len(p.Categories) {
		index = len(p.Categories)
	}
	p.Categories = append(p.Categories, Category{})
	copy(p.Categories[index+1:], p.Categories[index:])
	p.Categories[index] = cat
	p.UpdatedAt = NowTimestamp()
}

func (p *Project) RemoveCategory(index int) error {
	if index < 0 || index >= len(p.Categories) {
		return errors.New("category index out of range")
	}
	p.Categories = append(p.Categories[:index], p.Categories[index+1:]...)
	p.UpdatedAt = NowTimestamp()
	return nil
}

// CategoryNameConflict is the one category uniqueness rule: it returns the index
// of the category in cats, other than except, whose name matches name trimmed
// and case-insensitively, or -1. Pass except -1 to check a new name.
func CategoryNameConflict(cats []Category, name string, except int) int {
	normalized := NormalizeName(name)
	for i, cat := range cats {
		if i != except && NormalizeName(cat.Name) == normalized {
			return i
		}
	}
	return -1
}

// UniqueCategoryName returns name, or name with the first free " (n)" suffix
// from 2 up when a category in cats already carries it.
func UniqueCategoryName(cats []Category, name string) string {
	candidate := name
	for n := 2; CategoryNameConflict(cats, candidate, -1) >= 0; n++ {
		candidate = fmt.Sprintf("%s (%d)", name, n)
	}
	return candidate
}

// MergeCategory moves the category at src into the one at dst and removes src.
// Its tasks land at the end of dst under a separator labelled with src's name;
// an empty src just goes.
func (p *Project) MergeCategory(src, dst int) error {
	if src < 0 || src >= len(p.Categories) || dst < 0 || dst >= len(p.Categories) {
		return errors.New("category index out of range")
	}
	if src == dst {
		return errors.New("cannot merge a category into itself")
	}
	from := p.Categories[src]
	if len(from.Tasks) > 0 {
		sep, err := NewSeparator()
		if err != nil {
			return err
		}
		sep.Title = strings.TrimSpace(from.Name)
		to := &p.Categories[dst]
		to.Tasks = append(append(to.Tasks, sep), from.Tasks...)
		to.UpdatedAt = NowTimestamp()
	}
	return p.RemoveCategory(src)
}

func (p *Project) RenameCategory(index int, name string) error {
	if index < 0 || index >= len(p.Categories) {
		return errors.New("category index out of range")
	}
	if CategoryNameConflict(p.Categories, name, index) >= 0 {
		return ErrDuplicateCategoryName
	}
	p.Categories[index].Name = name
	now := NowTimestamp()
	p.Categories[index].UpdatedAt = now
	p.UpdatedAt = now
	return nil
}

func (p *Project) AddCategoryNamed(name string) (*Category, error) {
	if CategoryNameConflict(p.Categories, name, -1) >= 0 {
		return nil, ErrDuplicateCategoryName
	}
	cat, err := NewCategory(name)
	if err != nil {
		return nil, err
	}
	p.AddCategory(cat)
	return &p.Categories[len(p.Categories)-1], nil
}

func (p *Project) FindCategoryByID(id string) (int, error) {
	for i := range p.Categories {
		if p.Categories[i].ID == id {
			return i, nil
		}
	}
	return -1, ErrCategoryNotFound
}

func (c *Category) FindTaskByID(id string) (int, error) {
	for i := range c.Tasks {
		if c.Tasks[i].ID == id {
			return i, nil
		}
	}
	return -1, ErrTaskNotFound
}

const minTaskIDPrefix = 4

// FindTask matches a full ID, an ID prefix or a title; separators never match.
func (p *Project) FindTask(selector string) (catIdx, taskIdx int, err error) {
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return -1, -1, ErrTaskNotFound
	}
	needle := NormalizeName(selector)
	lower := strings.ToLower(selector)
	for c := range p.Categories {
		for t, task := range p.Categories[c].Tasks {
			if task.IsSeparator() {
				continue
			}
			if task.ID == selector ||
				(len(selector) >= minTaskIDPrefix && strings.HasPrefix(strings.ToLower(task.ID), lower)) ||
				NormalizeName(task.Title) == needle {
				return c, t, nil
			}
		}
	}
	return -1, -1, ErrTaskNotFound
}

func (p *Project) MoveCategory(idx, delta int) error {
	if idx < 0 || idx >= len(p.Categories) {
		return errors.New("category index out of range")
	}
	dst := idx + delta
	if dst < 0 || dst >= len(p.Categories) {
		return errors.New("destination index out of range")
	}
	p.Categories[idx], p.Categories[dst] = p.Categories[dst], p.Categories[idx]
	p.UpdatedAt = NowTimestamp()
	return nil
}

// ReverseCategories flips the order of the categories in place. It is its own
// inverse: invoking it twice restores the original order. Tasks within each
// category keep their order.
func (p *Project) ReverseCategories() {
	if len(p.Categories) < 2 {
		return
	}
	slices.Reverse(p.Categories)
	p.UpdatedAt = NowTimestamp()
}

func (c *Category) MoveTask(idx, delta int) error {
	if idx < 0 || idx >= len(c.Tasks) {
		return errors.New("task index out of range")
	}
	dst := idx + delta
	if dst < 0 || dst >= len(c.Tasks) {
		return errors.New("destination index out of range")
	}
	c.Tasks[idx], c.Tasks[dst] = c.Tasks[dst], c.Tasks[idx]
	c.UpdatedAt = NowTimestamp()
	return nil
}

func (p *Project) MoveTask(srcCatIdx, taskIdx, dstCatIdx int) error {
	if srcCatIdx < 0 || srcCatIdx >= len(p.Categories) {
		return errors.New("source category index out of range")
	}
	if dstCatIdx < 0 || dstCatIdx >= len(p.Categories) {
		return errors.New("destination category index out of range")
	}
	if srcCatIdx == dstCatIdx {
		return errors.New("source and destination categories are the same")
	}
	srcTasks := p.Categories[srcCatIdx].Tasks
	if taskIdx < 0 || taskIdx >= len(srcTasks) {
		return errors.New("task index out of range")
	}
	task := srcTasks[taskIdx]
	if err := p.Categories[srcCatIdx].RemoveTask(taskIdx); err != nil {
		return err
	}
	p.Categories[dstCatIdx].AddTask(task)
	p.UpdatedAt = NowTimestamp()
	return nil
}
