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
	ErrNotContiguous = errors.New("not contiguous")
	ErrMixedLevels   = errors.New("mixes levels")
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

// LastBlockStart returns the index of the first item of the last block in the
// run of sibling blocks items[lo:hi].
func LastBlockStart[T any, P outline[T]](items []T, lo, hi int) int {
	if _, parent := P(&items[lo]).links(); parent != "" {
		return hi - 1
	}
	return BlockStart[T, P](items, hi-1)
}

// CoverBlocks returns the run items[lo:hi] that the items at idxs cover once
// each parent brings its children along. It fails with ErrNotContiguous when
// the run has a gap, and with ErrMixedLevels unless it is made of whole sibling
// blocks: top-level blocks, or children of one parent.
func CoverBlocks[T any, P outline[T]](items []T, idxs []int) (int, int, error) {
	if len(idxs) == 0 {
		return 0, 0, ErrNotContiguous
	}
	covered := make(map[int]bool)
	lo, hi := len(items), 0
	for _, i := range idxs {
		end := BlockEnd[T, P](items, i)
		for j := i; j < end; j++ {
			covered[j] = true
		}
		lo, hi = min(lo, i), max(hi, end)
	}
	if len(covered) != hi-lo {
		return lo, hi, ErrNotContiguous
	}
	_, parent := P(&items[lo]).links()
	if parent == "" {
		if BlockEnd[T, P](items, BlockStart[T, P](items, hi-1)) != hi {
			return lo, hi, ErrMixedLevels
		}
		return lo, hi, nil
	}
	for i := lo; i < hi; i++ {
		if _, p := P(&items[i]).links(); p != parent {
			return lo, hi, ErrMixedLevels
		}
	}
	return lo, hi, nil
}

// SettleParents decides the parents of rows pasted among the children of
// parentID, or at the top level for "". They all nest under parentID when
// every one can be a child there; otherwise each keeps a parent pasted along
// with it and drops any other. Reports whether they nest under parentID.
func SettleParents[T any, P outline[T]](rows []T, parentID string) bool {
	pasted := make(map[string]bool, len(rows))
	nest := parentID != ""
	for i := range rows {
		row := P(&rows[i])
		id, _ := row.links()
		pasted[id] = true
		nest = nest && row.nests()
	}
	for i := range rows {
		if _, parent := P(&rows[i]).links(); pasted[parent] {
			nest = false
		}
	}
	for i := range rows {
		row := P(&rows[i])
		_, parent := row.links()
		switch {
		case nest:
			row.setParent(parentID)
		case !pasted[parent]:
			row.setParent("")
		}
	}
	return nest
}

// HasChildren reports whether items[i] is a parent.
func HasChildren[T any, P outline[T]](items []T, i int) bool {
	return BlockEnd[T, P](items, i) > i+1
}

// nestTarget returns the index of the top-level item that the run
// items[lo:hi] would nest under: the one above it. Every item of the run must
// be a top-level item without children.
func nestTarget[T any, P outline[T]](items []T, lo, hi int) (int, error) {
	if lo <= 0 || hi > len(items) || lo >= hi {
		return -1, ErrNothingToNest
	}
	for i := lo; i < hi; i++ {
		item := P(&items[i])
		if !item.nests() {
			return -1, ErrSeparatorNest
		}
		if _, parent := item.links(); parent != "" || HasChildren[T, P](items, i) {
			return -1, ErrNestDepth
		}
	}
	head := BlockStart[T, P](items, lo-1)
	if !P(&items[head]).nests() {
		return -1, ErrNothingToNest
	}
	return head, nil
}

// unnest makes the run items[lo:hi], children of one parent, top-level right
// after the parent's block, in their order, and returns the run's new start.
func unnest[T any, P outline[T]](items []T, lo, hi int) (int, error) {
	parent := ParentIndex[T, P](items, lo)
	if parent < 0 {
		return lo, ErrNotNested
	}
	parentID, _ := P(&items[parent]).links()
	for i := lo; i < hi; i++ {
		if _, p := P(&items[i]).links(); p != parentID {
			return lo, ErrMixedLevels
		}
	}
	end := BlockEnd[T, P](items, parent)
	for i := lo; i < hi; i++ {
		P(&items[i]).setParent("")
	}
	SwapRanges(items, lo, hi, end)
	return lo + end - hi, nil
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
		SwapRanges(items, i, lo, hi)
		return i + hi - lo, true
	}
	SwapRanges(items, lo, i, end)
	return lo, true
}

// SwapRanges swaps the adjacent ranges [a, b) and [b, c) of items.
func SwapRanges[T any](items []T, a, b, c int) {
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

// IndentTasks nests the top-level tasks lo to hi-1, none of them a parent,
// under the task above them, as its last subtasks.
func (c *Category) IndentTasks(lo, hi int) error {
	parent, err := nestTarget(c.Tasks, lo, hi)
	if err != nil {
		return err
	}
	now := NowTimestamp()
	for i := lo; i < hi; i++ {
		c.Tasks[i].ParentID = c.Tasks[parent].ID
		c.Tasks[i].UpdatedAt = now
	}
	c.UpdatedAt = now
	return nil
}

// IndentTask nests the task at i under the task above it, as its last subtask.
func (c *Category) IndentTask(i int) error {
	return c.IndentTasks(i, i+1)
}

// OutdentTasks makes the subtasks lo to hi-1 of one parent top-level, right
// after the parent's block, and returns where the first of them lands.
func (c *Category) OutdentTasks(lo, hi int) (int, error) {
	at, err := unnest(c.Tasks, lo, hi)
	if err != nil {
		return lo, err
	}
	now := NowTimestamp()
	for i := at; i < at+hi-lo; i++ {
		c.Tasks[i].UpdatedAt = now
	}
	c.UpdatedAt = now
	return at, nil
}

// OutdentTask makes the subtask at i top-level, right after its parent's
// block, and returns its new index.
func (c *Category) OutdentTask(i int) (int, error) {
	return c.OutdentTasks(i, i+1)
}

// IndentCategories nests the top-level categories lo to hi-1, none of them a
// parent, under the top-level category above them, as its last
// sub-categories. It refuses a name the parent's sub-categories carry.
func (p *Project) IndentCategories(lo, hi int) error {
	parent, err := nestTarget(p.Categories, lo, hi)
	if err != nil {
		return err
	}
	parentID := p.Categories[parent].ID
	for i := lo; i < hi; i++ {
		if CategoryNameConflict(p.Categories, parentID, p.Categories[i].Name, i) >= 0 {
			return ErrDuplicateCategoryName
		}
	}
	now := NowTimestamp()
	for i := lo; i < hi; i++ {
		p.Categories[i].ParentID = parentID
		p.Categories[i].UpdatedAt = now
	}
	p.UpdatedAt = now
	return nil
}

// IndentCategory nests the category at i under the top-level category above
// it, as its last sub-category.
func (p *Project) IndentCategory(i int) error {
	return p.IndentCategories(i, i+1)
}

// OutdentCategories makes the sub-categories lo to hi-1 of one parent
// top-level, right after the parent's block, and returns where the first of
// them lands. It refuses a name another top-level category carries.
func (p *Project) OutdentCategories(lo, hi int) (int, error) {
	if p.Categories[lo].ParentID == "" {
		return lo, ErrNotNested
	}
	for i := lo; i < hi; i++ {
		if CategoryNameConflict(p.Categories, "", p.Categories[i].Name, i) >= 0 {
			return lo, ErrDuplicateCategoryName
		}
	}
	at, err := unnest(p.Categories, lo, hi)
	if err != nil {
		return lo, err
	}
	now := NowTimestamp()
	for i := at; i < at+hi-lo; i++ {
		p.Categories[i].UpdatedAt = now
	}
	p.UpdatedAt = now
	return at, nil
}

// OutdentCategory makes the sub-category at i top-level, right after its
// parent's block, and returns its new index.
func (p *Project) OutdentCategory(i int) (int, error) {
	return p.OutdentCategories(i, i+1)
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
