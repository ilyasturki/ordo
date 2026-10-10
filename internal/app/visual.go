package app

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"phasionary/internal/app/selection"
	"phasionary/internal/domain"
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
	lines := make([]string, 0, len(selPositions))
	for _, pos := range selPositions {
		switch pos.Kind {
		case selection.FocusTask:
			task := m.project.Categories[pos.CategoryIndex].Tasks[pos.TaskIndex]
			if checkboxes {
				mark := " "
				if task.Status == domain.StatusCompleted {
					mark = "x"
				}
				lines = append(lines, fmt.Sprintf("- [%s] %s", mark, task.Title))
			} else {
				lines = append(lines, "- "+task.Title)
			}
		case selection.FocusDescription:
			desc := m.project.Categories[pos.CategoryIndex].Tasks[pos.TaskIndex].Description
			lines = append(lines, "  "+strings.ReplaceAll(desc, "\n", "\n  "))
		case selection.FocusCategory:
			name := m.project.Categories[pos.CategoryIndex].Name
			if checkboxes {
				lines = append(lines, "## "+name)
			} else {
				lines = append(lines, "- "+name)
			}
		}
	}
	return strings.Join(lines, "\n")
}

func (m *model) stashVisualClipboard(selPositions []selection.Position, isCut bool, text string) {
	if len(selPositions) == 0 {
		return
	}
	m.ui.TagCopiedLast = false
	switch m.ui.Visual.Kind {
	case selection.FocusCategory:
		cats := make([]domain.Category, 0, len(selPositions))
		ids := make([]string, 0, len(selPositions))
		for _, pos := range selPositions {
			c := m.project.Categories[pos.CategoryIndex]
			cats = append(cats, c)
			ids = append(ids, c.ID)
		}
		// Cloned: each category's Tasks would otherwise share the project's
		// backing array, and an edit after the copy would rewrite the copy.
		m.ui.Clipboard = cloneClipboard(ClipboardState{Categories: cats, CategoryIDs: ids, IsCut: isCut, Text: text, Written: true})
	default:
		// Task-level: stash the tasks and any interleaved separators so a paste
		// reproduces the block, dividers and all. Description rows round up to
		// their parent task (deduped) — a task is stashed once even when both
		// its row and its description are in range.
		selPositions = visualRoundUpPositions(selPositions)
		tasks := make([]domain.Task, 0, len(selPositions))
		ids := make([]string, 0, len(selPositions))
		for _, pos := range selPositions {
			t := m.project.Categories[pos.CategoryIndex].Tasks[pos.TaskIndex]
			tasks = append(tasks, t)
			ids = append(ids, t.ID)
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
		for _, pos := range visualRoundUpPositions(selPositions) {
			state.TaskIDs = append(state.TaskIDs, m.project.Categories[pos.CategoryIndex].Tasks[pos.TaskIndex].ID)
		}
	case selection.FocusCategory:
		for _, pos := range selPositions {
			state.CategoryIDs = append(state.CategoryIDs, m.project.Categories[pos.CategoryIndex].ID)
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
	selPositions := m.visualSelectedPositions()
	if len(selPositions) == 0 {
		return
	}
	var reason string
	switch m.ui.Visual.Kind {
	case selection.FocusCategory:
		reason = m.shiftCategories(selPositions[0].CategoryIndex, selPositions[len(selPositions)-1].CategoryIndex, dir)
	case selection.FocusTask:
		reason = m.visualShiftTaskBlock(visualRoundUpPositions(selPositions), dir)
	}
	if reason != "" {
		m.ui.Screen.StatusMsg = reason
	}
}

func (m *model) visualShiftTaskBlock(selPositions []selection.Position, dir int) string {
	catIdx := selPositions[0].CategoryIndex
	for _, p := range selPositions[1:] {
		if p.CategoryIndex != catIdx {
			return "Can't move: selection spans multiple categories"
		}
	}
	taskLo := selPositions[0].TaskIndex
	taskHi := selPositions[len(selPositions)-1].TaskIndex
	if taskHi-taskLo+1 != len(selPositions) {
		return "Can't move: selection isn't contiguous (rows hidden by filter)"
	}
	return m.shiftTasks(catIdx, taskLo, taskHi, dir)
}

// visualCursorRowID captures the cursor's task ID and whether the cursor sits
// on the task's description row, so the cursor can re-attach to the same row
// after a structural change re-indexes the position list.
func (m *model) visualCursorRowID() (string, bool) {
	cur, ok := m.ui.Selection.SelectedPosition()
	if !ok || !visualTaskLevel(cur.Kind) {
		return "", false
	}
	if cur.CategoryIndex < 0 || cur.CategoryIndex >= len(m.project.Categories) {
		return "", false
	}
	cat := m.project.Categories[cur.CategoryIndex]
	if cur.TaskIndex < 0 || cur.TaskIndex >= len(cat.Tasks) {
		return "", false
	}
	return cat.Tasks[cur.TaskIndex].ID, cur.Kind == selection.FocusDescription
}

// selectTaskLevelRow moves the selection to the task-level row with the given
// task ID — the description row when onDescription is set, the task row
// otherwise. No-op when id is empty or the row no longer exists.
func (m *model) selectTaskLevelRow(id string, onDescription bool) {
	if id == "" {
		return
	}
	m.ui.Selection.SelectByPredicate(func(p selection.Position) bool {
		if !visualTaskLevel(p.Kind) || (p.Kind == selection.FocusDescription) != onDescription {
			return false
		}
		return m.project.Categories[p.CategoryIndex].Tasks[p.TaskIndex].ID == id
	})
}

// visualCut writes the range as visual y does and marks it for a move.
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
		n := len(selPositions)
		what = fmt.Sprintf("%d categor%s", n, plural(n, "y", "ies"))
	} else {
		tasks := countRealTasks(selPositions)
		what = rowCountLabel(tasks, len(selPositions)-tasks)
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
		return m, m.visualStartTagEdit()
	case "p":
		m.visualPasteTag()
		return m, nil
	}
	return m, nil
}
