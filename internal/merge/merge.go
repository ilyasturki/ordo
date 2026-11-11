package merge

import (
	"maps"
	"slices"
	"strconv"

	"github.com/ilyasturki/ordo/internal/domain"
)

// Collision is one field both copies changed to different values. Task or Cat names the row,
// neither for the project's own name.
type Collision struct {
	Task, Cat    string
	Row, Field   string
	Mine, Theirs string
	take         func(p *domain.Project)
	copyRow      domain.Task
}

func (c Collision) CanCopy() bool { return c.Task != "" }

type Pick int

const (
	PickTheirs Pick = iota
	PickMine
	PickBoth
)

// Result holds theirs at every collision until Resolve says otherwise.
type Result struct {
	Project    domain.Project
	Collisions []Collision
}

type field[T any] struct {
	label string
	value func(T) string
	take  func(dst *T, src T)
}

func minutes(n int) string {
	if n == 0 {
		return "none"
	}
	return strconv.Itoa(n) + " min"
}

func orNone(s string) string {
	if s == "" {
		return "none"
	}
	return s
}

var taskFields = []field[domain.Task]{
	{"title", func(t domain.Task) string { return t.Title }, func(d *domain.Task, s domain.Task) { d.Title = s.Title }},
	{"details", func(t domain.Task) string { return t.Description }, func(d *domain.Task, s domain.Task) { d.Description = s.Description }},
	{"status", func(t domain.Task) string { return t.Status }, func(d *domain.Task, s domain.Task) {
		d.Status, d.CompletionDate = s.Status, s.CompletionDate
	}},
	{"priority", func(t domain.Task) string {
		if t.Priority == domain.PriorityMedium {
			return "medium"
		}
		return t.Priority
	}, func(d *domain.Task, s domain.Task) { d.Priority = s.Priority }},
	{"estimate", func(t domain.Task) string { return minutes(t.EstimateMinutes) }, func(d *domain.Task, s domain.Task) { d.EstimateMinutes = s.EstimateMinutes }},
	{"mark", func(t domain.Task) string { return orNone(t.TagColor) }, func(d *domain.Task, s domain.Task) { d.TagColor = s.TagColor }},
}

var categoryFields = []field[domain.Category]{
	{"name", func(c domain.Category) string { return c.Name }, func(d *domain.Category, s domain.Category) { d.Name = s.Name }},
	{"estimate", func(c domain.Category) string { return minutes(c.EstimateMinutes) }, func(d *domain.Category, s domain.Category) { d.EstimateMinutes = s.EstimateMinutes }},
	{"pin", func(c domain.Category) string { return strconv.FormatBool(c.Pinned) }, func(d *domain.Category, s domain.Category) { d.Pinned = s.Pinned }},
}

var projectFields = []field[domain.Project]{
	{"name", func(p domain.Project) string { return p.Name }, func(d *domain.Project, s domain.Project) { d.Name = s.Name }},
	{"group", func(p domain.Project) string { return orNone(p.Group) }, func(d *domain.Project, s domain.Project) { d.Group = s.Group }},
}

func mergeFields[T any](fields []field[T], base, mine, theirs T) (T, []field[T]) {
	out := theirs
	var clashes []field[T]
	for _, f := range fields {
		b, m, t := f.value(base), f.value(mine), f.value(theirs)
		switch {
		case m == t || m == b:
		case t == b:
			f.take(&out, mine)
		default:
			clashes = append(clashes, f)
		}
	}
	return out, clashes
}

func unchanged[T any](fields []field[T], a, b T) bool {
	return !slices.ContainsFunc(fields, func(f field[T]) bool { return f.value(a) != f.value(b) })
}

// pick settles a value that never collides: theirs wins a tie.
func pick(base, mine, theirs string) string {
	if theirs == base {
		return mine
	}
	return theirs
}

type placedTask struct {
	task domain.Task
	cat  string
}

func tasksOf(p domain.Project) map[string]placedTask {
	out := make(map[string]placedTask)
	for _, c := range p.Categories {
		for _, t := range c.Tasks {
			out[t.ID] = placedTask{t, c.ID}
		}
	}
	return out
}

func categoriesOf(p domain.Project) map[string]domain.Category {
	out := make(map[string]domain.Category, len(p.Categories))
	for _, c := range p.Categories {
		out[c.ID] = c
	}
	return out
}

func idsOf[T any](items []T, id func(T) string) []string {
	out := make([]string, len(items))
	for i, item := range items {
		out[i] = id(item)
	}
	return out
}

func taskID(t domain.Task) string         { return t.ID }
func categoryID(c domain.Category) string { return c.ID }

// Three keeps a row one side deleted and the other changed: a delete can be redone, an edit can't.
func Three(base, mine, theirs domain.Project) Result {
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
	for _, id := range mergeOrder(
		idsOf(base.Categories, categoryID), idsOf(mine.Categories, categoryID), idsOf(theirs.Categories, categoryID), keepCats,
	) {
		cat := cats[id]
		cat.Tasks = placeTasks(id, base, mine, theirs, tasks)
		merged.Categories = append(merged.Categories, cat)
	}
	domain.NormalizeNesting(&merged)
	res.Project = merged
	return res
}

func mergeTasks(base, mine, theirs domain.Project) (map[string]placedTask, []Collision) {
	b, m, t := tasksOf(base), tasksOf(mine), tasksOf(theirs)
	out := make(map[string]placedTask, len(t))
	var clashes []Collision
	for _, id := range slices.Sorted(maps.Keys(m)) {
		mt, inTheirs := t[id]
		if !inTheirs {
			if old, inBase := b[id]; !inBase || !unchanged(taskFields, old.task, m[id].task) {
				out[id] = m[id]
			}
			continue
		}
		old, inBase := b[id]
		if !inBase {
			old = placedTask{mt.task, mt.cat}
		}
		mine := m[id]
		task, fields := mergeFields(taskFields, old.task, mine.task, mt.task)
		task.ParentID = pick(old.task.ParentID, mine.task.ParentID, mt.task.ParentID)
		task.UpdatedAt = max(mine.task.UpdatedAt, mt.task.UpdatedAt)
		out[id] = placedTask{task, pick(old.cat, mine.cat, mt.cat)}
		for _, f := range fields {
			clashes = append(clashes, Collision{
				Task: id, Row: orTitle(mt.task), Field: f.label,
				Mine: f.value(mine.task), Theirs: f.value(mt.task),
				take:    takeTask(id, f, mine.task),
				copyRow: mine.task,
			})
		}
	}
	for id, row := range t {
		if _, inMine := m[id]; inMine {
			continue
		}
		if old, inBase := b[id]; !inBase || !unchanged(taskFields, old.task, row.task) {
			out[id] = row
		}
	}
	return out, clashes
}

func orTitle(t domain.Task) string {
	if t.Title == "" {
		return "Separator"
	}
	return t.Title
}

func takeTask(id string, f field[domain.Task], mine domain.Task) func(p *domain.Project) {
	return func(p *domain.Project) {
		for ci := range p.Categories {
			for ti := range p.Categories[ci].Tasks {
				if p.Categories[ci].Tasks[ti].ID == id {
					f.take(&p.Categories[ci].Tasks[ti], mine)
				}
			}
		}
	}
}

func takeCategory(id string, f field[domain.Category], mine domain.Category) func(p *domain.Project) {
	return func(p *domain.Project) {
		for ci := range p.Categories {
			if p.Categories[ci].ID == id {
				f.take(&p.Categories[ci], mine)
			}
		}
	}
}

func mergeCategories(base, mine, theirs domain.Project, tasks map[string]placedTask) (map[string]domain.Category, []Collision) {
	b, m, t := categoriesOf(base), categoriesOf(mine), categoriesOf(theirs)
	occupied := make(map[string]bool)
	for _, row := range tasks {
		occupied[row.cat] = true
	}
	out := make(map[string]domain.Category, len(t))
	var clashes []Collision
	for _, id := range slices.Sorted(maps.Keys(m)) {
		cat := m[id]
		tc, inTheirs := t[id]
		old, inBase := b[id]
		if !inTheirs {
			if !inBase || occupied[id] || !unchanged(categoryFields, old, cat) {
				out[id] = cat
			}
			continue
		}
		if !inBase {
			old = tc
		}
		merged, fields := mergeFields(categoryFields, old, cat, tc)
		merged.ParentID = pick(old.ParentID, cat.ParentID, tc.ParentID)
		merged.UpdatedAt = max(cat.UpdatedAt, tc.UpdatedAt)
		out[id] = merged
		for _, f := range fields {
			clashes = append(clashes, Collision{
				Cat: id, Row: tc.Name, Field: f.label, Mine: f.value(cat), Theirs: f.value(tc),
				take: takeCategory(id, f, cat),
			})
		}
	}
	for id, cat := range t {
		if _, inMine := m[id]; inMine {
			continue
		}
		if old, inBase := b[id]; !inBase || occupied[id] || !unchanged(categoryFields, old, cat) {
			out[id] = cat
		}
	}
	return out, clashes
}

// foldTwinCategories keeps names unique among siblings when both sides added the same one.
func foldTwinCategories(cats map[string]domain.Category, base, mine, theirs domain.Project, tasks map[string]placedTask) {
	b, t := categoriesOf(base), categoriesOf(theirs)
	for _, added := range mine.Categories {
		_, inBase := b[added.ID]
		_, inTheirs := t[added.ID]
		if inBase || inTheirs {
			continue
		}
		i := slices.IndexFunc(theirs.Categories, func(c domain.Category) bool {
			_, kept := cats[c.ID]
			return kept && c.ParentID == added.ParentID && domain.NormalizeName(c.Name) == domain.NormalizeName(added.Name)
		})
		if i < 0 {
			continue
		}
		twin := theirs.Categories[i].ID
		for id, row := range tasks {
			if row.cat == added.ID {
				row.cat = twin
				tasks[id] = row
			}
		}
		for id, cat := range cats {
			if cat.ParentID == added.ID {
				cat.ParentID = twin
				cats[id] = cat
			}
		}
		delete(cats, added.ID)
	}
}

func placeTasks(cat string, base, mine, theirs domain.Project, tasks map[string]placedTask) []domain.Task {
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
	order := mergeOrder(rowsIn(base), mineRows, theirRows, keep)
	out := make([]domain.Task, 0, len(order))
	for _, id := range order {
		out = append(out, tasks[id].task)
	}
	return out
}

func before(list []string, id string) string {
	if i := slices.Index(list, id); i > 0 {
		return list[i-1]
	}
	return ""
}

// mergeOrder reads moves among the rows all three hold, or a delete would pass for its neighbour moving.
func mergeOrder(base, mine, theirs []string, keep map[string]bool) []string {
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
			was := before(sharedBase, id)
			if before(sharedMine, id) == was || before(sharedTheirs, id) != was {
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

func (r Result) Resolve(picks []Pick) (domain.Project, error) {
	p := Clone(r.Project)
	copied := make(map[string]bool)
	for i, c := range r.Collisions {
		switch picks[i] {
		case PickMine:
			c.take(&p)
		case PickBoth:
			if !c.CanCopy() {
				c.take(&p)
				continue
			}
			if copied[c.Task] {
				continue
			}
			copied[c.Task] = true
			if err := insertCopy(&p, c.Task, c.copyRow); err != nil {
				return domain.Project{}, err
			}
		}
	}
	domain.NormalizeNesting(&p)
	return p, nil
}

func insertCopy(p *domain.Project, id string, mine domain.Task) error {
	newID, err := domain.NewID()
	if err != nil {
		return err
	}
	for ci := range p.Categories {
		tasks := p.Categories[ci].Tasks
		ti := slices.IndexFunc(tasks, func(t domain.Task) bool { return t.ID == id })
		if ti < 0 {
			continue
		}
		row := mine
		row.ID = newID
		row.ParentID = tasks[ti].ParentID
		p.Categories[ci].Tasks = slices.Insert(tasks, domain.BlockEnd(tasks, ti), row)
		return nil
	}
	return nil
}

func Clone(p domain.Project) domain.Project {
	out := p
	out.Categories = slices.Clone(p.Categories)
	for i := range out.Categories {
		out.Categories[i].Tasks = slices.Clone(out.Categories[i].Tasks)
	}
	return out
}

// Same ignores save stamps.
func Same(a, b domain.Project) bool {
	if a.Name != b.Name || a.Group != b.Group {
		return false
	}
	return slices.EqualFunc(a.Categories, b.Categories, func(x, y domain.Category) bool {
		xt, yt := x.Tasks, y.Tasks
		x.Tasks, y.Tasks = nil, nil
		x.UpdatedAt, y.UpdatedAt = "", ""
		return sameCategory(x, y) && slices.EqualFunc(xt, yt, func(s, t domain.Task) bool {
			s.UpdatedAt, t.UpdatedAt = "", ""
			return s == t
		})
	})
}

func sameCategory(a, b domain.Category) bool {
	return a.ID == b.ID && a.ParentID == b.ParentID && a.Name == b.Name &&
		a.EstimateMinutes == b.EstimateMinutes && a.Pinned == b.Pinned && a.CreatedAt == b.CreatedAt
}

// Holds compares whole rows, stamps included: two writers can reach one value alone, not one row.
func Holds(base, wrote, theirs domain.Project) bool {
	b, t := tasksOf(base), tasksOf(theirs)
	for id, row := range tasksOf(wrote) {
		old, inBase := b[id]
		if inBase && old.task == row.task {
			continue
		}
		if held, ok := t[id]; ok && held.task == row.task {
			return true
		}
	}
	bc, tc := categoriesOf(base), categoriesOf(theirs)
	for id, cat := range categoriesOf(wrote) {
		old, inBase := bc[id]
		if inBase && sameCategory(old, cat) && old.UpdatedAt == cat.UpdatedAt {
			continue
		}
		if held, ok := tc[id]; ok && sameCategory(held, cat) && held.UpdatedAt == cat.UpdatedAt {
			return true
		}
	}
	return false
}
