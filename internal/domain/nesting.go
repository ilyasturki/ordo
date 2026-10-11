package domain

import (
	"errors"
	"iter"
	"slices"
	"strings"
)

// A child sits right after its parent, among its siblings; one level deep, separators never nest.

var (
	ErrNestDepth     = errors.New("nesting goes one level deep")
	ErrNothingToNest = errors.New("nothing above to nest under")
	ErrNotNested     = errors.New("not nested")
	ErrSeparatorNest = errors.New("a separator can't nest")
	ErrNotContiguous = errors.New("not contiguous")
	ErrMixedLevels   = errors.New("mixes levels")
)

// CategoryPathSep separates parent and name in CLI category paths and markdown headings.
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

// BlockEnd is one past items[i]'s last child, or i+1 for a child or an item with none.
func BlockEnd[T any, P outline[T]](items []T, i int) int {
	id, parent := P(&items[i]).links()
	end := i + 1
	// Every top-level item's parent is "", so an ID-less item would adopt all that follow it.
	if parent != "" || id == "" {
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

// ParentIndex returns the index of items[i]'s parent, or -1 when it is top-level.
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

func BlockStart[T any, P outline[T]](items []T, i int) int {
	if p := ParentIndex[T, P](items, i); p >= 0 {
		return p
	}
	return i
}

// LastBlockStart expects items[lo:hi] to be a run of whole sibling blocks.
func LastBlockStart[T any, P outline[T]](items []T, lo, hi int) int {
	if _, parent := P(&items[lo]).links(); parent != "" {
		return hi - 1
	}
	return BlockStart[T, P](items, hi-1)
}

// CoverBlocks returns the run [lo, hi) idxs cover once each parent brings its children.
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

// SettleParents: all rows nest under parentID (true), or each keeps only a parent pasted with it.
func SettleParents[T any, P outline[T]](rows []T, parentID string) bool {
	pasted := make(map[string]bool, len(rows))
	nest := parentID != ""
	for i := range rows {
		row := P(&rows[i])
		if id, _ := row.links(); id != "" {
			pasted[id] = true
		}
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

// PickBlocks returns the picked items, each parent with its children, in the order items holds them.
func PickBlocks[T any, P outline[T]](items []T, picked map[string]bool) []T {
	var out []T
	for i := 0; i < len(items); {
		end := i + 1
		if id, _ := P(&items[i]).links(); picked[id] {
			end = BlockEnd[T, P](items, i)
			out = append(out, items[i:end]...)
		}
		i = end
	}
	return out
}

func HasChildren[T any, P outline[T]](items []T, i int) bool {
	return ChildCount[T, P](items, i) > 0
}

func ChildCount[T any, P outline[T]](items []T, i int) int {
	return BlockEnd[T, P](items, i) - i - 1
}

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

// unnest makes items[lo:hi] top-level after their parent's block and returns their new start.
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

// SiblingBlock returns the sibling block [lo, hi) next to items[i]'s own in direction dir.
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

// moveBlock swaps items[i]'s block with the next sibling block in dir; returns i's new index.
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

// normalizeOutline implements the repair docs/file-format.md promises; change them together.
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

func NormalizeNesting(p *Project) {
	p.Categories = normalizeOutline(p.Categories)
	for i := range p.Categories {
		p.Categories[i].Tasks = normalizeOutline(p.Categories[i].Tasks)
	}
}

// RequiredSchema is the lowest version holding the file, so older binaries keep opening what they can.
func RequiredSchema(p Project) int {
	schema := 1
	for _, c := range p.Categories {
		if c.ParentID != "" {
			schema = 2
		}
		for _, t := range c.Tasks {
			if t.ParentID != "" {
				schema = 2
			}
		}
	}
	return schema
}

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

// CategoryNameConflict is the one name rule, unique among siblings: a clash's index, or -1.
func CategoryNameConflict(cats []Category, parentID, name string, except int) int {
	normalized := NormalizeName(name)
	for i, cat := range cats {
		if i != except && cat.ParentID == parentID && NormalizeName(cat.Name) == normalized {
			return i
		}
	}
	return -1
}

// CategoryPath is the "parent / name" form the TUI shows.
func (p *Project) CategoryPath(i int) string {
	if parent := ParentIndex(p.Categories, i); parent >= 0 {
		return p.Categories[parent].Name + " " + CategoryPathSep + " " + p.Categories[i].Name
	}
	return p.Categories[i].Name
}

// SplitCategoryPath tries every separator: a name may hold one too ("ui/ux").
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

func (p *Project) TreeTaskCount(i int) int {
	n := 0
	for _, c := range p.Categories[i:BlockEnd(p.Categories, i)] {
		n += c.TaskCount()
	}
	return n
}

func (p *Project) TreeStatusCounts(i int) StatusCounts {
	return statusCounts(p.treeTasks(i))
}

func (p *Project) TreeAggregateStatus(i int) string {
	return aggregateStatus(p.treeTasks(i))
}

func (c *Category) ChildProgress(i int) (done, total int) {
	for _, t := range c.Tasks[i+1 : BlockEnd(c.Tasks, i)] {
		total++
		if t.Status == StatusCompleted || t.Status == StatusCancelled {
			done++
		}
	}
	return done, total
}

func (c *Category) MoveTaskBlock(i, dir int) (int, bool) {
	at, ok := moveBlock(c.Tasks, i, dir)
	if ok {
		c.UpdatedAt = NowTimestamp()
	}
	return at, ok
}

func (p *Project) MoveCategoryBlock(i, dir int) (int, bool) {
	at, ok := moveBlock(p.Categories, i, dir)
	if ok {
		p.UpdatedAt = NowTimestamp()
	}
	return at, ok
}

func (c *Category) RemoveTaskBlock(i int) {
	c.Tasks = slices.Delete(c.Tasks, i, BlockEnd(c.Tasks, i))
	c.UpdatedAt = NowTimestamp()
}

func (p *Project) RemoveCategoryBlock(i int) {
	p.Categories = slices.Delete(p.Categories, i, BlockEnd(p.Categories, i))
	p.UpdatedAt = NowTimestamp()
}

func (c *Category) ReleaseSubtasks(i int) {
	now := NowTimestamp()
	for j, end := i+1, BlockEnd(c.Tasks, i); j < end; j++ {
		c.Tasks[j].ParentID = ""
		c.Tasks[j].UpdatedAt = now
	}
}

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
