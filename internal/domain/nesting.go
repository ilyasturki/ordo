package domain

import (
	"errors"
	"iter"
	"slices"
	"strings"
)

// Categories and tasks nest the same way, one level deep. Each kind stays in
// one flat slice (Project.Categories, Category.Tasks) in display order, and an
// item with a ParentID is a child: it sits right after its parent, among the
// parent's other children. A child has no children, and a separator never
// nests. Positions therefore stay (category index, task index), and depth is
// read off ParentID.

var (
	ErrNestDepth     = errors.New("nesting goes one level deep")
	ErrNothingToNest = errors.New("nothing above to nest under")
	ErrNotNested     = errors.New("not nested")
	ErrSeparatorNest = errors.New("a separator can't nest")
)

// CategoryPathSep joins a sub-category's name to its parent's in a category
// path, the form the CLI resolves and markdown headings carry.
const CategoryPathSep = "/"

type outline[T any] interface {
	*T
	links() (id, parent string)
	setParent(id string)
	nests() bool
}

func (c *Category) links() (string, string) { return c.ID, c.ParentID }
func (c *Category) setParent(id string)     { c.ParentID = id }
func (c *Category) nests() bool             { return true }

func (t *Task) links() (string, string) { return t.ID, t.ParentID }
func (t *Task) setParent(id string)     { t.ParentID = id }
func (t *Task) nests() bool             { return !t.IsSeparator() }

// BlockEnd returns one past the last child of items[i]. A child, or an item
// without children, is a block of its own.
func BlockEnd[T any, P outline[T]](items []T, i int) int {
	id, parent := P(&items[i]).links()
	end := i + 1
	if parent != "" {
		return end
	}
	for end < len(items) {
		if _, p := P(&items[end]).links(); p != id {
			break
		}
		end++
	}
	return end
}

// ParentIndex returns the index of items[i]'s parent, or -1 when it is
// top-level.
func ParentIndex[T any, P outline[T]](items []T, i int) int {
	_, parent := P(&items[i]).links()
	if parent == "" {
		return -1
	}
	for j := i - 1; j >= 0; j-- {
		if id, _ := P(&items[j]).links(); id == parent {
			return j
		}
	}
	return -1
}

// BlockStart returns the index of the top-level item whose block holds
// items[i].
func BlockStart[T any, P outline[T]](items []T, i int) int {
	if p := ParentIndex[T, P](items, i); p >= 0 {
		return p
	}
	return i
}

// HasChildren reports whether items[i] is a parent.
func HasChildren[T any, P outline[T]](items []T, i int) bool {
	return BlockEnd[T, P](items, i) > i+1
}

// indentTarget returns the index of the item items[i] would nest under.
func indentTarget[T any, P outline[T]](items []T, i int) (int, error) {
	if i <= 0 || i >= len(items) {
		return -1, ErrNothingToNest
	}
	item := P(&items[i])
	if !item.nests() {
		return -1, ErrSeparatorNest
	}
	if _, parent := item.links(); parent != "" || HasChildren[T, P](items, i) {
		return -1, ErrNestDepth
	}
	head := BlockStart[T, P](items, i-1)
	if !P(&items[head]).nests() {
		return -1, ErrNothingToNest
	}
	return head, nil
}

// outdent makes the child items[i] top-level, right after its parent's block,
// and returns its new index.
func outdent[T any, P outline[T]](items []T, i int) (int, error) {
	parent := ParentIndex[T, P](items, i)
	if parent < 0 {
		return i, ErrNotNested
	}
	end := BlockEnd[T, P](items, parent)
	P(&items[i]).setParent("")
	moved := items[i]
	copy(items[i:end-1], items[i+1:end])
	items[end-1] = moved
	return end - 1, nil
}

// SiblingBlock returns the block next to items[i]'s own among its siblings in
// direction dir, as [lo, hi), or false at the edge of the siblings.
func SiblingBlock[T any, P outline[T]](items []T, i, dir int) (int, int, bool) {
	_, parent := P(&items[i]).links()
	if dir > 0 {
		next := BlockEnd[T, P](items, i)
		if next >= len(items) {
			return 0, 0, false
		}
		if _, p := P(&items[next]).links(); p != parent {
			return 0, 0, false
		}
		return next, BlockEnd[T, P](items, next), true
	}
	if i == 0 {
		return 0, 0, false
	}
	prev := i - 1
	if parent == "" {
		prev = BlockStart[T, P](items, prev)
	} else if _, p := P(&items[prev]).links(); p != parent {
		return 0, 0, false
	}
	return prev, i, true
}

// moveBlock swaps items[i]'s block with the sibling block in direction dir and
// returns the item's new index, or false at the edge of its siblings.
func moveBlock[T any, P outline[T]](items []T, i, dir int) (int, bool) {
	lo, hi, ok := SiblingBlock[T, P](items, i, dir)
	if !ok {
		return i, false
	}
	end := BlockEnd[T, P](items, i)
	if dir > 0 {
		rotateBlocks(items, i, lo, hi)
		return i + hi - lo, true
	}
	rotateBlocks(items, lo, i, end)
	return lo, true
}

// rotateBlocks swaps the adjacent ranges [a, b) and [b, c).
func rotateBlocks[T any](items []T, a, b, c int) {
	slices.Reverse(items[a:b])
	slices.Reverse(items[b:c])
	slices.Reverse(items[a:c])
}

// normalizeOutline repairs what a hand-edited or foreign file may get wrong. A
// child whose parent is missing, is itself a child, or is a separator becomes
// top-level, and so does a separator with a parent. Every remaining child is
// then regrouped right after its parent, keeping the order of both.
func normalizeOutline[T any, P outline[T]](items []T) []T {
	first := make(map[string]int, len(items))
	for i := range items {
		id, _ := P(&items[i]).links()
		if _, dup := first[id]; !dup {
			first[id] = i
		}
	}
	parentOf := make([]int, len(items))
	for i := range items {
		parentOf[i] = -1
		item := P(&items[i])
		_, parent := item.links()
		if parent == "" {
			continue
		}
		if j, ok := first[parent]; ok && j != i && item.nests() && P(&items[j]).nests() {
			parentOf[i] = j
		}
	}
	valid := slices.Clone(parentOf)
	for i, j := range parentOf {
		if j >= 0 && parentOf[j] >= 0 {
			valid[i] = -1
		}
	}

	children := make(map[int][]int)
	for i, j := range valid {
		if j >= 0 {
			children[j] = append(children[j], i)
		}
	}
	out := make([]T, 0, len(items))
	for i := range items {
		if valid[i] >= 0 {
			continue
		}
		out = append(out, items[i])
		P(&out[len(out)-1]).setParent("")
		for _, c := range children[i] {
			out = append(out, items[c])
		}
	}
	return out
}

// NormalizeNesting restores the nesting rules on a decoded project; see
// normalizeOutline.
func NormalizeNesting(p *Project) {
	p.Categories = normalizeOutline(p.Categories)
	for i := range p.Categories {
		p.Categories[i].Tasks = normalizeOutline(p.Categories[i].Tasks)
	}
}

// RequiredSchema is the lowest schema version that holds p: a file gains
// version 2 with its first nested item and drops back to 1 without one, so
// an older phasionary keeps opening every file it can read whole.
func RequiredSchema(p Project) int {
	for _, c := range p.Categories {
		if c.ParentID != "" {
			return 2
		}
		for _, t := range c.Tasks {
			if t.ParentID != "" {
				return 2
			}
		}
	}
	return 1
}

// IndentTask nests the task at i under the task above it, as its last subtask.
func (c *Category) IndentTask(i int) error {
	parent, err := indentTarget(c.Tasks, i)
	if err != nil {
		return err
	}
	c.Tasks[i].ParentID = c.Tasks[parent].ID
	c.Tasks[i].UpdatedAt = NowTimestamp()
	c.UpdatedAt = NowTimestamp()
	return nil
}

// OutdentTask makes the subtask at i top-level, right after its parent's
// block, and returns its new index.
func (c *Category) OutdentTask(i int) (int, error) {
	at, err := outdent(c.Tasks, i)
	if err != nil {
		return i, err
	}
	c.Tasks[at].UpdatedAt = NowTimestamp()
	c.UpdatedAt = NowTimestamp()
	return at, nil
}

// IndentCategory nests the category at i under the top-level category above
// it, as its last sub-category.
func (p *Project) IndentCategory(i int) error {
	parent, err := indentTarget(p.Categories, i)
	if err != nil {
		return err
	}
	parentID := p.Categories[parent].ID
	if CategoryNameConflict(p.Categories, parentID, p.Categories[i].Name, i) >= 0 {
		return ErrDuplicateCategoryName
	}
	p.Categories[i].ParentID = parentID
	p.Categories[i].UpdatedAt = NowTimestamp()
	p.UpdatedAt = NowTimestamp()
	return nil
}

// OutdentCategory makes the sub-category at i top-level, right after its
// parent's block, and returns its new index.
func (p *Project) OutdentCategory(i int) (int, error) {
	if p.Categories[i].ParentID == "" {
		return i, ErrNotNested
	}
	if CategoryNameConflict(p.Categories, "", p.Categories[i].Name, i) >= 0 {
		return i, ErrDuplicateCategoryName
	}
	at, err := outdent(p.Categories, i)
	if err != nil {
		return i, err
	}
	p.Categories[at].UpdatedAt = NowTimestamp()
	p.UpdatedAt = NowTimestamp()
	return at, nil
}

// CategoryNameConflict is the one category uniqueness rule: names are unique
// among siblings, compared trimmed and case-insensitively. It returns the
// index of the category under parentID, other than except, that already
// carries name, or -1. Pass except -1 to check a new name.
func CategoryNameConflict(cats []Category, parentID, name string, except int) int {
	normalized := NormalizeName(name)
	for i, cat := range cats {
		if i != except && cat.ParentID == parentID && NormalizeName(cat.Name) == normalized {
			return i
		}
	}
	return -1
}

// CategoryPath names the category at i the way the CLI resolves it: its own
// name, behind its parent's for a sub-category.
func (p *Project) CategoryPath(i int) string {
	if parent := ParentIndex(p.Categories, i); parent >= 0 {
		return p.Categories[parent].Name + " " + CategoryPathSep + " " + p.Categories[i].Name
	}
	return p.Categories[i].Name
}

// SplitCategoryPath lists the ways path splits into a parent name and a
// sub-category name, trimmed, one per separator in it.
func SplitCategoryPath(path string) [][2]string {
	var out [][2]string
	for i := 0; i < len(path); i++ {
		if strings.HasPrefix(path[i:], CategoryPathSep) {
			parent := strings.TrimSpace(path[:i])
			child := strings.TrimSpace(path[i+len(CategoryPathSep):])
			if parent != "" && child != "" {
				out = append(out, [2]string{parent, child})
			}
		}
	}
	return out
}

// treeTasks yields the tasks of the category at i and of its sub-categories.
func (p *Project) treeTasks(i int) iter.Seq[Task] {
	return func(yield func(Task) bool) {
		for c := i; c < BlockEnd(p.Categories, i); c++ {
			for _, t := range p.Categories[c].Tasks {
				if !yield(t) {
					return
				}
			}
		}
	}
}

// TreeStatusCounts counts the tasks of the category at i together with those
// of its sub-categories.
func (p *Project) TreeStatusCounts(i int) StatusCounts {
	return statusCounts(p.treeTasks(i))
}

// TreeAggregateStatus is AggregateStatus over the category at i and its
// sub-categories.
func (p *Project) TreeAggregateStatus(i int) string {
	return aggregateStatus(p.treeTasks(i))
}

// ChildProgress reports how many of the task at i's subtasks are finished
// (completed or cancelled), out of how many it has.
func (c *Category) ChildProgress(i int) (done, total int) {
	for _, t := range c.Tasks[i+1 : BlockEnd(c.Tasks, i)] {
		total++
		if t.Status == StatusCompleted || t.Status == StatusCancelled {
			done++
		}
	}
	return done, total
}

// MoveTaskBlock swaps the task at i, with its subtasks, and the sibling block
// next to it in direction dir. It returns the task's new index, or false at
// the edge of its siblings.
func (c *Category) MoveTaskBlock(i, dir int) (int, bool) {
	at, ok := moveBlock(c.Tasks, i, dir)
	if ok {
		c.UpdatedAt = NowTimestamp()
	}
	return at, ok
}

// MoveCategoryBlock swaps the category at i, with its sub-categories, and the
// sibling block next to it in direction dir. It returns the category's new
// index, or false at the edge of its siblings.
func (p *Project) MoveCategoryBlock(i, dir int) (int, bool) {
	at, ok := moveBlock(p.Categories, i, dir)
	if ok {
		p.UpdatedAt = NowTimestamp()
	}
	return at, ok
}

// RemoveTaskBlock removes the task at i together with its subtasks.
func (c *Category) RemoveTaskBlock(i int) {
	c.Tasks = slices.Delete(c.Tasks, i, BlockEnd(c.Tasks, i))
	c.UpdatedAt = NowTimestamp()
}

// RemoveCategoryBlock removes the category at i together with its
// sub-categories.
func (p *Project) RemoveCategoryBlock(i int) {
	p.Categories = slices.Delete(p.Categories, i, BlockEnd(p.Categories, i))
	p.UpdatedAt = NowTimestamp()
}

// ReleaseSubtasks makes the subtasks of the task at i top-level where they
// stand, so the task can go without them.
func (c *Category) ReleaseSubtasks(i int) {
	now := NowTimestamp()
	for j, end := i+1, BlockEnd(c.Tasks, i); j < end; j++ {
		c.Tasks[j].ParentID = ""
		c.Tasks[j].UpdatedAt = now
	}
}

// ReleaseSubCategories makes the sub-categories of the category at i
// top-level where they stand, so the category can go without them. It refuses
// when one would take the name of another top-level category.
func (p *Project) ReleaseSubCategories(i int) error {
	end := BlockEnd(p.Categories, i)
	for j := i + 1; j < end; j++ {
		if CategoryNameConflict(p.Categories, "", p.Categories[j].Name, i) >= 0 {
			return ErrDuplicateCategoryName
		}
	}
	now := NowTimestamp()
	for j := i + 1; j < end; j++ {
		p.Categories[j].ParentID = ""
		p.Categories[j].UpdatedAt = now
	}
	return nil
}
