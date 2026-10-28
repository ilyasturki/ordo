package operations

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/ilyasturki/ordo/internal/domain"
)

// Op is one edit addressed by IDs: what the web app's forms build, what the phone queues
// while offline and what `ordo serve` replays. docs/sync.md is its wire contract.
type Op struct {
	Action string   `json:"action"`
	IDs    []string `json:"ids,omitempty"`
	// New is the ID of the row an add creates, minted by the client.
	New    string `json:"new,omitempty"`
	Cat    string `json:"cat,omitempty"`
	Parent string `json:"parent,omitempty"`
	// Before is the row a placed run lands above; empty lands it last.
	Before string `json:"before,omitempty"`
	Value  string `json:"value,omitempty"`
	Title  string `json:"title,omitempty"`
	Text   string `json:"text,omitempty"`
	// Base is the text a title or description edit was typed over.
	Base    string   `json:"base,omitempty"`
	Keep    bool     `json:"keep,omitempty"`
	Reverse bool     `json:"reverse,omitempty"`
	Up      bool     `json:"up,omitempty"`
	Lines   []string `json:"lines,omitempty"`
}

type Outcome struct {
	Msg string
	// Row is the row an add created or a move took along first.
	Row string
	// Saved tells a text field that the file now holds what it sent.
	Saved bool
}

// Conflict refuses a text edit typed over text the file no longer holds.
type Conflict struct{ Theirs string }

func (c *Conflict) Error() string { return "Changed on another device: reload to see it" }

var ErrUnknownAction = errors.New("unknown action")

type applier func(p *domain.Project, op Op) (Outcome, error)

var appliers = map[string]applier{
	"task.add":         addTask,
	"task.insert":      insertRow(""),
	"separator.add":    insertRow(domain.KindSeparator),
	"task.title":       editTitle,
	"task.description": editDescription,
	"task.attach":      attachLines,
	"task.delete":      deleteRows,
	"status.cycle":     cycleStatus,
	"status.set":       setStatus,
	"priority.set":     setPriority,
	"tag.set":          setTag,
	"estimate.set":     setEstimate,
	"task.nest":        nestRows(true),
	"task.unnest":      nestRows(false),
	"task.shift":       shiftRows,
	"task.move":        moveRows,
	"task.place":       placeRows,
}

var gestures = map[string]bool{"status.cycle": true, "task.shift": true, "task.nest": true, "task.unnest": true}

// Replayable reports whether an op can wait in a queue and land later as it is. A gesture
// steps from the state it was made on, so it travels as the op Perform returns instead.
func Replayable(action string) bool {
	_, ok := appliers[action]
	return ok && !gestures[action]
}

// Apply runs op on p, and leaves p as it was on any error, ErrNoChange included.
func Apply(p *domain.Project, op Op) (Outcome, error) {
	fn, ok := appliers[op.Action]
	if !ok {
		return Outcome{}, fmt.Errorf("%w: %s", ErrUnknownAction, op.Action)
	}
	before := Clone(*p)
	out, err := fn(p, op)
	if err == nil && sameContent(before, *p) {
		err = ErrNoChange
	}
	if err != nil {
		*p = before
	}
	return out, err
}

// Perform applies op and returns the replayable op that reproduces its result on a file that
// may have moved on since: an add with the ID it minted, a cycle as the status it landed on,
// a nest, un-nest or shift as the place its rows landed.
func Perform(p *domain.Project, op Op) (Outcome, Op, error) {
	switch op.Action {
	case "task.add", "task.insert", "separator.add":
		if op.New == "" {
			id, err := domain.NewID()
			if err != nil {
				return Outcome{}, Op{}, err
			}
			op.New = id
		}
	}
	out, err := Apply(p, op)
	if err != nil {
		return out, Op{}, err
	}
	switch op.Action {
	case "status.cycle":
		return out, Op{Action: "status.set", IDs: op.IDs, Value: realTasks(p, op.IDs)[0].Status}, nil
	case "task.nest", "task.unnest", "task.shift":
		ci, lo, hi, err := TaskRun(p, op.IDs)
		if err != nil {
			return out, Op{}, err
		}
		cat := p.Categories[ci]
		place := Op{Action: "task.place", IDs: op.IDs, Cat: cat.ID, Parent: cat.Tasks[lo].ParentID}
		if hi < len(cat.Tasks) {
			place.Before = cat.Tasks[hi].ID
		}
		return out, place, nil
	}
	return out, op, nil
}

// Clone copies p deep enough that editing the copy leaves p untouched.
func Clone(p domain.Project) domain.Project {
	p.Categories = slices.Clone(p.Categories)
	for i := range p.Categories {
		p.Categories[i].Tasks = slices.Clone(p.Categories[i].Tasks)
	}
	return p
}

func sameContent(a, b domain.Project) bool {
	return reflect.DeepEqual(unstamped(a), unstamped(b))
}

func unstamped(p domain.Project) domain.Project {
	p = Clone(p)
	p.UpdatedAt, p.Schema = "", 0
	for i := range p.Categories {
		c := &p.Categories[i]
		c.UpdatedAt = ""
		for j := range c.Tasks {
			c.Tasks[j].UpdatedAt = ""
		}
	}
	return p
}

// present lets an edit to rows deleted elsewhere read as gone rather than as a no-op.
func present(p *domain.Project, ids []string) error {
	if len(ids) == 0 {
		return ErrNoChange
	}
	for _, id := range ids {
		if _, _, err := LocateTask(p, id); err == nil {
			return nil
		}
	}
	return domain.ErrTaskNotFound
}

func oneTask(p *domain.Project, ids []string) (*domain.Task, string, error) {
	if len(ids) == 0 {
		return nil, "", ErrNoChange
	}
	ci, ti, err := LocateTask(p, ids[0])
	if err != nil {
		return nil, "", err
	}
	return &p.Categories[ci].Tasks[ti], p.Categories[ci].ID, nil
}

// addTask falls back to a top-level row when the parent is gone, so an add queued offline survives it.
func addTask(p *domain.Project, op Op) (Outcome, error) {
	cat := op.Cat
	f := TaskFields{Title: op.Title, ParentID: op.Parent, ID: op.New}
	if f.ParentID != "" {
		if ci, _, err := LocateTask(p, f.ParentID); err == nil {
			cat = p.Categories[ci].ID
		} else {
			f.ParentID = ""
		}
	}
	t, err := CreateTask(p, cat, f)
	if err != nil {
		return Outcome{Row: op.New}, err
	}
	return Outcome{Msg: "Added " + quote(t.Title), Row: t.ID}, nil
}

// insertRow puts the row below the block of op.IDs[0], or first in op.Cat without one or once it is gone.
func insertRow(kind string) applier {
	return func(p *domain.Project, op Op) (Outcome, error) {
		f := TaskFields{Kind: kind, Title: op.Title, ID: op.New}
		var (
			t   domain.Task
			err error
		)
		if _, cat, lookup := oneTask(p, op.IDs); lookup == nil {
			t, err = CreateTaskAfter(p, cat, f, op.IDs[0])
		} else {
			t, err = CreateTaskAtTop(p, op.Cat, f)
		}
		if err != nil {
			return Outcome{Row: op.New}, err
		}
		if kind == domain.KindSeparator {
			return Outcome{Msg: "Added a separator", Row: t.ID}, nil
		}
		return Outcome{Row: t.ID}, nil
	}
}

func editTitle(p *domain.Project, op Op) (Outcome, error) {
	t, cat, err := oneTask(p, op.IDs)
	if err != nil {
		return Outcome{}, err
	}
	title := strings.TrimSpace(op.Title)
	if title == t.Title {
		return Outcome{Saved: true}, ErrNoChange
	}
	if t.Title != op.Base {
		return Outcome{}, &Conflict{Theirs: t.Title}
	}
	if _, err := UpdateTask(p, cat, t.ID, TaskUpdate{Title: &title}); err != nil {
		return Outcome{}, err
	}
	return Outcome{Saved: true}, nil
}

func editDescription(p *domain.Project, op Op) (Outcome, error) {
	t, cat, err := oneTask(p, op.IDs)
	if err != nil {
		return Outcome{}, err
	}
	desc := trimText(op.Text)
	if desc == strings.TrimRight(t.Description, "\n") {
		return Outcome{Saved: true}, ErrNoChange
	}
	if strings.TrimRight(t.Description, "\n") != trimText(op.Base) {
		return Outcome{}, &Conflict{Theirs: t.Description}
	}
	if _, err := UpdateTask(p, cat, t.ID, TaskUpdate{Description: &desc}); err != nil {
		return Outcome{}, err
	}
	return Outcome{Saved: true}, nil
}

func trimText(s string) string {
	return strings.TrimRight(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
}

// attachLines appends the image lines the description doesn't hold yet.
func attachLines(p *domain.Project, op Op) (Outcome, error) {
	tasks := realTasks(p, op.IDs[:min(1, len(op.IDs))])
	if len(tasks) == 0 {
		return Outcome{}, domain.ErrTaskNotFound
	}
	have := make(map[string]bool)
	for _, line := range strings.Split(tasks[0].Description, "\n") {
		have[strings.TrimSpace(line)] = true
	}
	var add []string
	for _, line := range op.Lines {
		line = strings.TrimSpace(line)
		if err := domain.ValidateLine(line); err != nil {
			return Outcome{}, err
		}
		if line != "" && !have[line] {
			have[line] = true
			add = append(add, line)
		}
	}
	if len(add) == 0 {
		return Outcome{}, ErrNoChange
	}
	if _, err := AppendDescription(p, tasks[0].ID, strings.Join(add, "\n")); err != nil {
		return Outcome{}, err
	}
	return Outcome{Msg: plural(len(add), "Image attached", "%d images attached")}, nil
}

func deleteRows(p *domain.Project, op Op) (Outcome, error) {
	removed, err := DeleteTasks(p, op.IDs, op.Keep)
	if err != nil {
		return Outcome{}, err
	}
	tasks, seps := 0, 0
	for _, t := range removed {
		if t.IsSeparator() {
			seps++
		} else {
			tasks++
		}
	}
	if tasks == 0 {
		return Outcome{Msg: plural(seps, "Separator deleted", "%d separators deleted")}, nil
	}
	if len(removed) == 1 {
		return Outcome{Msg: "Deleted " + quote(removed[0].Title)}, nil
	}
	return Outcome{Msg: plural(tasks, "1 task deleted", "%d tasks deleted")}, nil
}

func statusOutcome(status string, n int) Outcome {
	if n == 1 {
		return Outcome{Msg: "Marked " + strings.ToLower(StatusLabels[status])}
	}
	return Outcome{Msg: fmt.Sprintf("%d tasks marked %s", n, strings.ToLower(StatusLabels[status]))}
}

func cycleStatus(p *domain.Project, op Op) (Outcome, error) {
	if err := present(p, op.IDs); err != nil {
		return Outcome{}, err
	}
	next, n, err := CycleStatus(p, op.IDs, op.Reverse)
	if err != nil {
		return Outcome{}, err
	}
	return statusOutcome(next, n), nil
}

func setStatus(p *domain.Project, op Op) (Outcome, error) {
	if err := present(p, op.IDs); err != nil {
		return Outcome{}, err
	}
	n, err := SetStatuses(p, op.IDs, op.Value)
	if err != nil {
		return Outcome{}, err
	}
	return statusOutcome(op.Value, n), nil
}

func onMany(what string, n int) string {
	if n > 1 {
		return what + fmt.Sprintf(" on %d tasks", n)
	}
	return what
}

func setPriority(p *domain.Project, op Op) (Outcome, error) {
	if err := present(p, op.IDs); err != nil {
		return Outcome{}, err
	}
	n, err := SetPriorities(p, op.IDs, op.Value)
	if err != nil {
		return Outcome{}, err
	}
	what := "Priority " + strings.ToLower(PriorityLabels[op.Value])
	if op.Value == "" {
		what = "Priority cleared"
	}
	return Outcome{Msg: onMany(what, n)}, nil
}

func setTag(p *domain.Project, op Op) (Outcome, error) {
	if err := present(p, op.IDs); err != nil {
		return Outcome{}, err
	}
	n, err := SetTags(p, op.IDs, op.Value)
	if err != nil {
		return Outcome{}, err
	}
	what := "Tagged " + strings.ToLower(TagName(op.Value))
	if op.Value == "" {
		what = "Tag removed"
	}
	return Outcome{Msg: onMany(what, n)}, nil
}

func setEstimate(p *domain.Project, op Op) (Outcome, error) {
	if err := present(p, op.IDs); err != nil {
		return Outcome{}, err
	}
	minutes, err := strconv.Atoi(op.Value)
	if err != nil {
		return Outcome{}, ErrNegativeEstimate
	}
	n, err := SetEstimates(p, op.IDs, minutes)
	if err != nil {
		return Outcome{}, err
	}
	what := "Estimate " + domain.FormatEstimate(minutes)
	if minutes == 0 {
		what = "Estimate cleared"
	}
	return Outcome{Msg: onMany(what, n)}, nil
}

func nestRows(in bool) applier {
	return func(p *domain.Project, op Op) (Outcome, error) {
		if len(op.IDs) == 0 {
			return Outcome{}, ErrNoChange
		}
		parent, err := NestTasks(p, op.IDs, in, AllShown)
		if err != nil {
			return Outcome{}, err
		}
		if !in {
			return Outcome{Msg: "Un-nested", Row: op.IDs[0]}, nil
		}
		t, _, _ := oneTask(p, []string{parent})
		return Outcome{Msg: "Nested under " + quote(t.Title), Row: op.IDs[0]}, nil
	}
}

func shiftRows(p *domain.Project, op Op) (Outcome, error) {
	if len(op.IDs) == 0 {
		return Outcome{}, ErrNoChange
	}
	dir := 1
	if op.Up {
		dir = -1
	}
	if err := ShiftTasks(p, op.IDs, dir, AllShown); err != nil {
		return Outcome{}, err
	}
	return Outcome{Row: op.IDs[0]}, nil
}

func moveRows(p *domain.Project, op Op) (Outcome, error) {
	if err := present(p, op.IDs); err != nil {
		return Outcome{}, err
	}
	dst, err := p.FindCategoryByID(op.Cat)
	if err != nil {
		return Outcome{}, refuse("That category is gone")
	}
	if ci, _, err := LocateTask(p, op.IDs[0]); err == nil && ci == dst && len(op.IDs) == 1 {
		return Outcome{}, ErrNoChange
	}
	rows, err := MoveRows(p, op.IDs, p.Categories[dst].ID, len(p.Categories[dst].Tasks), "")
	if err != nil {
		return Outcome{}, err
	}
	what := "Moved"
	if len(rows) > 1 {
		what = fmt.Sprintf("Moved %d rows", len(rows))
	}
	return Outcome{Msg: what + " to " + p.CategoryPath(dst), Row: rows[0].ID}, nil
}

// placeRows drops the rows above op.Before, or last in their list without it. A parent that is
// gone leaves them top-level, so a drag queued offline survives it.
func placeRows(p *domain.Project, op Op) (Outcome, error) {
	if err := present(p, op.IDs); err != nil {
		return Outcome{}, err
	}
	ci, err := p.FindCategoryByID(op.Cat)
	if err != nil {
		return Outcome{}, refuse("That category is gone")
	}
	tasks := p.Categories[ci].Tasks
	parent, index := op.Parent, len(tasks)
	if pi := slices.IndexFunc(tasks, func(t domain.Task) bool { return t.ID == parent }); parent != "" && pi >= 0 {
		index = domain.BlockEnd(tasks, pi)
	} else {
		parent = ""
	}
	if op.Before != "" {
		if bi := slices.IndexFunc(tasks, func(t domain.Task) bool { return t.ID == op.Before }); bi >= 0 {
			index = bi
		}
	}
	rows, err := MoveRows(p, op.IDs, p.Categories[ci].ID, index, parent)
	if err != nil {
		return Outcome{}, err
	}
	if t, _, err := oneTask(p, []string{rows[0].ParentID}); err == nil && rows[0].ParentID != "" {
		return Outcome{Msg: "Nested under " + quote(t.Title), Row: rows[0].ID}, nil
	}
	return Outcome{Msg: "Moved", Row: rows[0].ID}, nil
}

func quote(s string) string {
	r := []rune(s)
	if len(r) > 40 {
		s = string(r[:39]) + "…"
	}
	return "“" + s + "”"
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return fmt.Sprintf(many, n)
}
