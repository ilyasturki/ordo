package app

import (
	"errors"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"phasionary/internal/app/selection"
	"phasionary/internal/domain"
	"phasionary/internal/export"
)

// visualTaskLevel reports whether kind is a row that belongs to the unified
// task-level visual domain — a real task, a separator divider, or an expanded
// description row. Task-level selections cover all three, so any of them
// caught in the range (or chosen as an endpoint) highlights and travels with
// the block.
func visualTaskLevel(kind selection.FocusKind) bool {
	return kind == selection.FocusTask || kind == selection.FocusSeparator || kind == selection.FocusDescription
}

// matchesVisualDomain reports whether a row of kind k belongs to the same
// visual-selection domain as an anchor whose normalized kind is visualKind
// (either FocusCategory or FocusTask). Category anchors select only category
// rows; task-level anchors select the whole task+separator+description domain.
func matchesVisualDomain(k, visualKind selection.FocusKind) bool {
	if visualKind == selection.FocusCategory {
		return k == selection.FocusCategory
	}
	return visualTaskLevel(k)
}

// visualRoundUpPositions maps each description row in a visual selection to
// its parent task and drops the duplicate that appears when a task and its
// description are both in range — structural operations (delete, cut, move,
// tags) act on whole tasks, so the pair counts once. A description whose
// parent sits above the range still pulls that task in ("round up").
// Separators pass through untouched.
func visualRoundUpPositions(positions []selection.Position) []selection.Position {
	out := make([]selection.Position, 0, len(positions))
	seen := make(map[[2]int]struct{}, len(positions))
	for _, p := range positions {
		if p.Kind == selection.FocusDescription {
			p.Kind = selection.FocusTask
		}
		key := [2]int{p.CategoryIndex, p.TaskIndex}
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, p)
	}
	return out
}

// countRealTasks returns how many of the positions are real tasks, ignoring any
// separators swept into a task-level range — so status counts report the number
// of tasks the user acted on rather than the raw row span.
func countRealTasks(positions []selection.Position) int {
	n := 0
	for _, p := range positions {
		if p.Kind == selection.FocusTask {
			n++
		}
	}
	return n
}

// rowCountLabel phrases a mixed task/separator batch for a status message. It
// counts real tasks, only falling back to a separator count when the batch was
// dividers alone, so a separator caught in a range never inflates "N task(s)".
func rowCountLabel(realTasks, separators int) string {
	if realTasks > 0 || separators == 0 {
		return fmt.Sprintf("%d task(s)", realTasks)
	}
	return fmt.Sprintf("%d separator(s)", separators)
}

func (m *model) enterVisualMode() {
	if !m.ui.Modes.IsNormal() {
		return
	}
	pos, ok := m.selectedPosition()
	if !ok || pos.Kind == selection.FocusProject {
		m.ui.Screen.StatusMsg = "Visual mode doesn't work on the project row"
		return
	}
	if !m.ui.Modes.ToVisual() {
		return
	}
	cat := m.project.Categories[pos.CategoryIndex]
	// Normalize a separator anchor to the task-level kind so the whole selection
	// (tasks and separators alike) shares one navigable domain.
	kind := selection.FocusCategory
	if visualTaskLevel(pos.Kind) {
		kind = selection.FocusTask
	}
	anchor := VisualState{
		Active:           true,
		Kind:             kind,
		AnchorCategoryID: cat.ID,
	}
	if visualTaskLevel(pos.Kind) {
		anchor.AnchorTaskID = cat.Tasks[pos.TaskIndex].ID
		anchor.AnchorOnDescription = pos.Kind == selection.FocusDescription
	}
	m.ui.Visual = anchor
}

func (m *model) exitVisualMode() {
	m.ui.Visual = VisualState{}
	m.ui.Modes.ToNormal()
}

// visualSwap exchanges the anchor and cursor positions, leaving the covered
// range unchanged. Useful for extending the selection from the other end.
func (m *model) visualSwap() {
	if !m.ui.Visual.Active {
		return
	}
	anchorIdx := m.resolveVisualAnchor()
	if anchorIdx < 0 {
		return
	}
	cursorIdx := m.ui.Selection.Selected()
	if anchorIdx == cursorIdx {
		return
	}
	positions := m.ui.Selection.Positions()
	if cursorIdx < 0 || cursorIdx >= len(positions) {
		return
	}
	curPos := positions[cursorIdx]
	if curPos.CategoryIndex < 0 || curPos.CategoryIndex >= len(m.project.Categories) {
		return
	}
	cat := m.project.Categories[curPos.CategoryIndex]
	m.ui.Visual.AnchorCategoryID = cat.ID
	m.ui.Visual.AnchorTaskID = ""
	m.ui.Visual.AnchorOnDescription = false
	if visualTaskLevel(curPos.Kind) {
		if curPos.TaskIndex < 0 || curPos.TaskIndex >= len(cat.Tasks) {
			return
		}
		m.ui.Visual.AnchorTaskID = cat.Tasks[curPos.TaskIndex].ID
		m.ui.Visual.AnchorOnDescription = curPos.Kind == selection.FocusDescription
	}
	m.ui.Selection.MoveTo(anchorIdx)
	m.ensureVisible()
}

func (m *model) visualMoveCursor(delta int) {
	if !m.ui.Visual.Active {
		return
	}
	positions := m.ui.Selection.Positions()
	if len(positions) == 0 {
		return
	}
	cur := m.ui.Selection.Selected()
	step := 1
	if delta < 0 {
		step = -1
	}
	for i := 0; i < abs(delta); i++ {
		next := nextVisualPosition(positions, cur, step, m.ui.Visual.Kind)
		if next == cur {
			break
		}
		cur = next
	}
	m.ui.Selection.MoveTo(cur)
	m.ensureVisible()
}

func nextVisualPosition(positions []selection.Position, from, step int, kind selection.FocusKind) int {
	i := from + step
	for i >= 0 && i < len(positions) {
		if matchesVisualDomain(positions[i].Kind, kind) {
			return i
		}
		i += step
	}
	return from
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func (m model) visualRange() (lo, hi int, active bool) {
	if !m.ui.Visual.Active {
		return 0, 0, false
	}
	a := m.resolveVisualAnchor()
	if a < 0 {
		return 0, 0, false
	}
	b := m.ui.Selection.Selected()
	if a > b {
		a, b = b, a
	}
	return a, b, true
}

// resolveVisualAnchor finds the current position index of the visual anchor by
// its stable ID. Returns -1 if the anchor item no longer exists in the
// position list (e.g., it was deleted or filtered out). Task-level anchors
// resolve by task/separator ID alone — IDs are unique project-wide, so the
// anchor keeps resolving even after its row moves to another category.
func (m model) resolveVisualAnchor() int {
	positions := m.ui.Selection.Positions()
	for i, p := range positions {
		if !matchesVisualDomain(p.Kind, m.ui.Visual.Kind) {
			continue
		}
		if p.CategoryIndex < 0 || p.CategoryIndex >= len(m.project.Categories) {
			continue
		}
		cat := m.project.Categories[p.CategoryIndex]
		if m.ui.Visual.Kind == selection.FocusCategory {
			if cat.ID == m.ui.Visual.AnchorCategoryID {
				return i
			}
			continue
		}
		// A task row and its description row share the task's ID, so the
		// description flag disambiguates which of the two the anchor sits on.
		if (p.Kind == selection.FocusDescription) != m.ui.Visual.AnchorOnDescription {
			continue
		}
		if p.TaskIndex < 0 || p.TaskIndex >= len(cat.Tasks) {
			continue
		}
		if cat.Tasks[p.TaskIndex].ID == m.ui.Visual.AnchorTaskID {
			return i
		}
	}
	return -1
}

func (m model) visualSelectedPositions() []selection.Position {
	lo, hi, active := m.visualRange()
	if !active {
		return nil
	}
	positions := m.ui.Selection.Positions()
	var result []selection.Position
	for i := lo; i <= hi && i < len(positions); i++ {
		if matchesVisualDomain(positions[i].Kind, m.ui.Visual.Kind) {
			result = append(result, positions[i])
		}
	}
	return result
}

func (m model) isInVisualRange(index int) bool {
	lo, hi, active := m.visualRange()
	if !active {
		return false
	}
	if index < lo || index > hi {
		return false
	}
	positions := m.ui.Selection.Positions()
	if index < 0 || index >= len(positions) {
		return false
	}
	return matchesVisualDomain(positions[index].Kind, m.ui.Visual.Kind)
}

func (m *model) visualCopyBullets() tea.Cmd {
	selPositions := m.visualSelectedPositions()
	if len(selPositions) == 0 {
		m.exitVisualMode()
		return nil
	}
	text := m.visualMarkdownText(selPositions, false)
	m.stashVisualClipboard(selPositions, false, text)
	m.exitVisualMode()
	return m.copyToClipboard(text, "")
}

func (m *model) visualCopyChecklist() tea.Cmd {
	selPositions := m.visualSelectedPositions()
	if len(selPositions) == 0 {
		m.exitVisualMode()
		return nil
	}
	text := m.visualMarkdownText(selPositions, true)
	m.stashVisualClipboard(selPositions, false, text)
	m.exitVisualMode()
	return m.copyToClipboard(text, "")
}

// visualMarkdownText renders the selection as a markdown list — a bullet per
// task ("- [ ]"/"- [x]" instead when checkboxes is set), with a selected
// description row's body indented under its parent's item. Separators carry no
// task content, so they are omitted from the copied text (the internal
// clipboard still keeps them for a faithful paste).
func (m model) visualMarkdownText(selPositions []selection.Position, checkboxes bool) string {
	if m.ui.Visual.Kind == selection.FocusCategory {
		return m.visualCategoryText(selPositions, checkboxes)
	}
	picked := make(map[string]bool)
	described := make(map[string]bool)
	for _, pos := range selPositions {
		id := m.project.Categories[pos.CategoryIndex].Tasks[pos.TaskIndex].ID
		switch pos.Kind {
		case selection.FocusTask:
			picked[id] = true
		case selection.FocusDescription:
			described[id] = true
		}
	}
	var lines []string
	for _, task := range m.visualTaskRows(selPositions) {
		if task.IsSeparator() {
			continue
		}
		indent := ""
		if task.ParentID != "" && picked[task.ParentID] {
			indent = "  "
		}
		if picked[task.ID] || indent != "" {
			lines = append(lines, indent+markdownItem(task, checkboxes))
		}
		if described[task.ID] {
			body := indent + "  "
			lines = append(lines, body+strings.ReplaceAll(task.Description, "\n", "\n"+body))
		}
	}
	return strings.Join(lines, "\n")
}

func markdownItem(task domain.Task, checkboxes bool) string {
	if !checkboxes {
		return "- " + task.Title
	}
	mark := " "
	if task.Status == domain.StatusCompleted {
		mark = "x"
	}
	return fmt.Sprintf("- [%s] %s", mark, task.Title)
}

func (m model) visualCategoryText(selPositions []selection.Position, checkboxes bool) string {
	cats := m.visualCategoryRows(selPositions)
	picked := make(map[string]bool, len(cats))
	for _, c := range cats {
		picked[c.ID] = true
	}
	lines := make([]string, 0, len(cats))
	for _, c := range cats {
		i, _ := m.project.FindCategoryByID(c.ID)
		switch {
		case checkboxes:
			lines = append(lines, "## "+export.CategoryHeading(m.project, i))
		case c.ParentID != "" && picked[c.ParentID]:
			lines = append(lines, "  - "+c.Name)
		default:
			lines = append(lines, "- "+c.Name)
		}
	}
	return strings.Join(lines, "\n")
}

func (m model) visualTaskRows(selPositions []selection.Position) []domain.Task {
	seen := make(map[string]bool)
	var rows []domain.Task
	for _, pos := range visualRoundUpPositions(selPositions) {
		tasks := m.project.Categories[pos.CategoryIndex].Tasks
		for _, t := range tasks[pos.TaskIndex:domain.BlockEnd(tasks, pos.TaskIndex)] {
			if !seen[t.ID] {
				seen[t.ID] = true
				rows = append(rows, t)
			}
		}
	}
	return rows
}

func (m model) visualCategoryRows(selPositions []selection.Position) []domain.Category {
	cats := m.project.Categories
	seen := make(map[string]bool)
	var rows []domain.Category
	for _, pos := range selPositions {
		for _, c := range cats[pos.CategoryIndex:domain.BlockEnd(cats, pos.CategoryIndex)] {
			if !seen[c.ID] {
				seen[c.ID] = true
				rows = append(rows, c)
			}
		}
	}
	return rows
}

func (m *model) stashVisualClipboard(selPositions []selection.Position, isCut bool, text string) {
	if len(selPositions) == 0 {
		return
	}
	m.ui.TagCopiedLast = false
	switch m.ui.Visual.Kind {
	case selection.FocusCategory:
		cats := m.visualCategoryRows(selPositions)
		ids := make([]string, len(cats))
		for i, c := range cats {
			ids[i] = c.ID
		}
		// Cloned: each category's Tasks would otherwise alias the project's backing array.
		m.ui.Clipboard = cloneClipboard(ClipboardState{Categories: cats, CategoryIDs: ids, IsCut: isCut, Text: text, Written: true})
	default:
		// Task-level: stash the tasks and any interleaved separators so a paste
		// reproduces the block, dividers and all.
		tasks := m.visualTaskRows(selPositions)
		ids := make([]string, len(tasks))
		for i, t := range tasks {
			ids[i] = t.ID
		}
		m.ui.Clipboard = ClipboardState{Tasks: tasks, TaskIDs: ids, IsCut: isCut, Text: text, Written: true}
	}
}

// visualCycleStatus sets every task in the visual selection to a single shared
// next status, stepped from the first selected task so repeated presses walk
// the whole block through the cycle together (reverse walks it backwards).
// Separators carry no status and are skipped; description rows round up to
// their parent task. Stays in visual mode.
func (m *model) visualCycleStatus(reverse bool) {
	if m.ui.Visual.Kind != selection.FocusTask {
		return
	}
	selPositions := visualTaskPositions(m.visualSelectedPositions())
	if len(selPositions) == 0 {
		return
	}
	first := m.project.Categories[selPositions[0].CategoryIndex].Tasks[selPositions[0].TaskIndex]
	next := domain.NextStatus(first.Status, reverse)
	m.recordHistory()
	changed := 0
	for _, pos := range selPositions {
		task := &m.project.Categories[pos.CategoryIndex].Tasks[pos.TaskIndex]
		if task.Status == next {
			continue
		}
		_ = task.SetStatus(next)
		changed++
	}
	if changed == 0 {
		m.discardLastHistory()
		return
	}
	m.storeTaskUpdate()
	m.ui.Screen.StatusMsg = fmt.Sprintf("Set %d task(s) to %s", changed, formatStatusLabel(next))
}

func (m *model) visualDelete() {
	selPositions := m.visualSelectedPositions()
	if len(selPositions) == 0 {
		m.exitVisualMode()
		return
	}
	state := ConfirmDeleteState{Kind: ConfirmDeleteVisualRange}
	switch m.ui.Visual.Kind {
	case selection.FocusTask:
		for _, t := range m.visualTaskRows(selPositions) {
			state.TaskIDs = append(state.TaskIDs, t.ID)
		}
	case selection.FocusCategory:
		for _, c := range m.visualCategoryRows(selPositions) {
			state.CategoryIDs = append(state.CategoryIDs, c.ID)
		}
	}
	m.exitVisualMode()
	m.ui.ConfirmDelete = state
	m.ui.Modes.ToConfirmDelete()
}

func (m *model) confirmDeleteVisualRange() {
	taskIDs := m.ui.ConfirmDelete.TaskIDs
	catIDs := m.ui.ConfirmDelete.CategoryIDs
	m.ui.ConfirmDelete.reset()
	m.ui.Modes.ToNormal()

	m.recordHistory()
	// Track real tasks and separators separately so the message reflects tasks,
	// while `removed` still gates whether the edit is worth keeping in history.
	deletedTasks, deletedSeps, removed := 0, 0, 0
	for _, id := range taskIDs {
		isSep := false
		if t := m.taskByID(id); t != nil {
			isSep = t.IsSeparator()
		}
		if m.removeTaskByIDIfExists(id) {
			removed++
			if isSep {
				deletedSeps++
			} else {
				deletedTasks++
			}
		}
	}
	deletedCats := 0
	if len(catIDs) > 0 {
		idSet := make(map[string]struct{}, len(catIDs))
		for _, id := range catIDs {
			idSet[id] = struct{}{}
		}
		kept := m.project.Categories[:0]
		for _, c := range m.project.Categories {
			if _, drop := idSet[c.ID]; drop {
				deletedCats++
				removed++
				continue
			}
			kept = append(kept, c)
		}
		m.project.Categories = kept
		m.project.UpdatedAt = domain.NowTimestamp()
	}

	if removed == 0 {
		m.discardLastHistory()
		return
	}
	m.rebuildAndClamp()
	m.storeTaskUpdate()
	if len(catIDs) > 0 {
		m.ui.Screen.StatusMsg = fmt.Sprintf("Deleted %d categor%s", deletedCats, plural(deletedCats, "y", "ies"))
	} else {
		m.ui.Screen.StatusMsg = "Deleted " + rowCountLabel(deletedTasks, deletedSeps)
	}
}

func (m *model) removeTaskByIDIfExists(id string) bool {
	for catIdx := range m.project.Categories {
		for taskIdx, task := range m.project.Categories[catIdx].Tasks {
			if task.ID == id {
				_ = m.project.Categories[catIdx].RemoveTask(taskIdx)
				return true
			}
		}
	}
	return false
}

// visualMoveDown/visualMoveUp shift the visual selection block one row through
// the shared engine in move.go, mirroring the single-row J/K behavior.
func (m *model) visualMoveDown() {
	m.visualShift(+1)
}

func (m *model) visualMoveUp() {
	m.visualShift(-1)
}

func (m *model) visualShift(dir int) {
	lo, hi, catIdx, ok := m.visualRun()
	if !ok {
		return
	}
	var reason string
	if m.ui.Visual.Kind == selection.FocusCategory {
		reason = m.shiftCategories(lo, hi, dir)
	} else {
		reason = m.shiftTasks(catIdx, lo, hi, dir)
	}
	if reason != "" {
		m.ui.Screen.StatusMsg = reason
	}
}

// visualRun returns the selection's sibling blocks as [lo, hi); refusals go to the status line.
func (m *model) visualRun() (lo, hi, catIdx int, ok bool) {
	selPositions := visualRoundUpPositions(m.visualSelectedPositions())
	if len(selPositions) == 0 {
		return 0, 0, 0, false
	}
	catIdx = selPositions[0].CategoryIndex
	idxs := make([]int, 0, len(selPositions))
	for _, p := range selPositions {
		if m.ui.Visual.Kind == selection.FocusCategory {
			idxs = append(idxs, p.CategoryIndex)
			continue
		}
		if p.CategoryIndex != catIdx {
			m.ui.Screen.StatusMsg = "Can't: selection spans multiple categories"
			return 0, 0, 0, false
		}
		idxs = append(idxs, p.TaskIndex)
	}
	var err error
	if m.ui.Visual.Kind == selection.FocusCategory {
		lo, hi, err = domain.CoverBlocks(m.project.Categories, idxs)
	} else {
		lo, hi, err = domain.CoverBlocks(m.project.Categories[catIdx].Tasks, idxs)
	}
	switch {
	case errors.Is(err, domain.ErrNotContiguous):
		m.ui.Screen.StatusMsg = "Can't: selection isn't contiguous (rows hidden by filter)"
		return 0, 0, 0, false
	case err != nil:
		m.ui.Screen.StatusMsg = "Can't: the selection mixes levels"
		return 0, 0, 0, false
	}
	return lo, hi, catIdx, true
}

func (m *model) selectTaskLevelRow(id string) {
	if id == "" {
		return
	}
	m.ui.Selection.SelectByPredicate(func(p selection.Position) bool {
		if !visualTaskLevel(p.Kind) || p.Kind == selection.FocusDescription {
			return false
		}
		return m.project.Categories[p.CategoryIndex].Tasks[p.TaskIndex].ID == id
	})
}

func (m *model) visualCut() tea.Cmd {
	raw := m.visualSelectedPositions()
	selPositions := visualRoundUpPositions(raw)
	if len(selPositions) == 0 {
		m.exitVisualMode()
		return nil
	}
	text := m.visualMarkdownText(raw, false)
	m.stashVisualClipboard(selPositions, true, text)
	var what string
	if m.ui.Visual.Kind == selection.FocusCategory {
		n := len(m.ui.Clipboard.Categories)
		what = fmt.Sprintf("%d categor%s", n, plural(n, "y", "ies"))
	} else {
		rows := m.ui.Clipboard.Tasks
		tasks := 0
		for _, t := range rows {
			if !t.IsSeparator() {
				tasks++
			}
		}
		what = rowCountLabel(tasks, len(rows)-tasks)
	}
	status := "Cut: " + what + ", p moves them here, esc cancels"
	m.ui.Screen.StatusMsg = status
	m.exitVisualMode()
	return m.writeClipboard(clipboardResultMsg{text: text, status: status})
}

func plural(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}

func halfPageStep(availHeight int) int {
	step := availHeight / 2
	if step < 1 {
		return 1
	}
	return step
}

func (m model) handleVisualKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "v", "q":
		m.exitVisualMode()
		return m, nil
	case "j", "down":
		m.visualMoveCursor(1)
		return m, nil
	case "k", "up":
		m.visualMoveCursor(-1)
		return m, nil
	case "J":
		m.visualMoveDown()
		return m, nil
	case ">":
		m.visualNest(true)
		return m, nil
	case "<":
		m.visualNest(false)
		return m, nil
	case "K":
		m.visualMoveUp()
		return m, nil
	case "ctrl+d":
		m.visualMoveCursor(halfPageStep(m.availableHeight()))
		return m, nil
	case "ctrl+u":
		m.visualMoveCursor(-halfPageStep(m.availableHeight()))
		return m, nil
	case "ctrl+f":
		m.visualMoveCursor(m.availableHeight())
		return m, nil
	case "ctrl+b":
		m.visualMoveCursor(-m.availableHeight())
		return m, nil
	case "G":
		m.visualMoveCursor(len(m.ui.Selection.Positions()))
		return m, nil
	case "g":
		m.visualMoveCursor(-len(m.ui.Selection.Positions()))
		return m, nil
	case "o", "O":
		m.visualSwap()
		return m, nil
	case "y":
		return m, m.visualCopyBullets()
	case "Y":
		return m, m.visualCopyChecklist()
	case "x":
		return m, m.visualCut()
	case "d":
		m.visualDelete()
		return m, nil
	case "space":
		m.visualCycleStatus(false)
		return m, nil
	case "shift+space":
		m.visualCycleStatus(true)
		return m, nil
	case "t":
		m.visualCycleTag()
		return m, nil
	case "T":
		m.visualStartTagEdit()
		return m, nil
	case "p":
		m.visualPasteTag()
		return m, nil
	}
	return m, nil
}
