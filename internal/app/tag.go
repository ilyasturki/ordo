package app

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/ilyasturki/ordo/internal/app/modes"
	"github.com/ilyasturki/ordo/internal/app/selection"
	"github.com/ilyasturki/ordo/internal/domain"
	"github.com/ilyasturki/ordo/internal/operations"
	"github.com/ilyasturki/ordo/internal/ui"
)

// tagEditColors is the color picker order: the palette preceded by a leading ""
// "None (remove)" row, so removing a tag is just selecting the first row.
var tagEditColors = append([]string{""}, domain.TagColorCycle...)

// TagEditState backs the `T` tag picker. taskIDs holds every task the edit
// applies to — one for a single task, many for a visual-mode bulk edit — so it
// survives any position shift while open. batchIDs, when set, are the tasks of
// the category that share the edited task's mark; batch applies the pick to all
// of them.
type TagEditState struct {
	taskIDs    []string
	colorIdx   int
	batchIDs   []string
	batchColor string
	batchCat   string
	batch      bool
}

// cycleTag gives an untagged task the category's first free mark and steps a
// tagged one through the palette.
func (m *model) cycleTag() {
	if !m.ui.Modes.CanPerformAction(modes.ActionChangeTag) {
		return
	}
	position, ok := m.selectedPosition()
	if !ok || position.Kind != selection.FocusTask {
		return
	}
	id := m.project.Categories[position.CategoryIndex].Tasks[position.TaskIndex].ID
	m.recordHistory()
	if _, err := operations.CycleTags(&m.project, []string{id}); m.applied(err) {
		m.storeTaskUpdate()
		m.holdFilteredOut([]string{id})
	}
}

// startTagEdit opens the tag picker for the selected task, seeded with its
// current color, and offers the batch of tasks sharing its mark.
func (m *model) startTagEdit() {
	if !m.ui.Modes.CanPerformAction(modes.ActionChangeTag) {
		return
	}
	position, ok := m.selectedPosition()
	if !ok || position.Kind != selection.FocusTask {
		return
	}
	cat := m.project.Categories[position.CategoryIndex]
	task := cat.Tasks[position.TaskIndex]
	m.beginTagEdit([]string{task.ID}, task.TagColor)
	if task.TagColor == "" {
		return
	}
	var batch []string
	for _, t := range cat.Tasks {
		if !t.IsSeparator() && t.TagColor == task.TagColor {
			batch = append(batch, t.ID)
		}
	}
	if len(batch) > 1 {
		m.ui.TagEdit.batchIDs = batch
		m.ui.TagEdit.batchColor = task.TagColor
		m.ui.TagEdit.batchCat = cat.Name
	}
}

// beginTagEdit opens the picker over the given tasks. Callers must already be
// in (or have returned to) normal mode.
func (m *model) beginTagEdit(taskIDs []string, color string) {
	if len(taskIDs) == 0 {
		return
	}
	m.ui.TagEdit = TagEditState{
		taskIDs:  taskIDs,
		colorIdx: tagColorIndex(color),
	}
	m.ui.Modes.ToTagEdit()
}

func (m *model) handleTagEditKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	s := &m.ui.TagEdit
	switch msg.String() {
	case "esc", "q":
		m.cancelTagEdit()
	case "enter":
		m.finishTagEdit()
	case "down", "j":
		if s.colorIdx < len(tagEditColors)-1 {
			s.colorIdx++
		}
	case "up", "k":
		if s.colorIdx > 0 {
			s.colorIdx--
		}
	case "space":
		s.batch = len(s.batchIDs) > 0 && !s.batch
	}
	return m, nil
}

func (m *model) cancelTagEdit() {
	m.ui.TagEdit = TagEditState{}
	m.ui.Modes.ToNormal()
	m.ui.Screen.StatusMsg = ""
}

func (m *model) finishTagEdit() {
	s := m.ui.TagEdit
	color := tagEditColors[s.colorIdx]
	m.ui.TagEdit = TagEditState{}
	m.ui.Modes.ToNormal()
	if len(s.taskIDs) == 0 {
		return
	}
	ids := s.taskIDs
	if s.batch {
		ids = s.batchIDs
	}

	m.recordHistory()
	changed, err := operations.SetTags(&m.project, ids, color)
	if !m.applied(err) {
		return
	}
	m.storeTaskUpdate()
	m.holdFilteredOut(ids)
	m.selectTaskByID(s.taskIDs[0])
	m.ensureVisible()
	if len(ids) > 1 {
		m.ui.Screen.StatusMsg = fmt.Sprintf("Tagged %d task(s)", changed)
	}
}

func (m model) tagEditView() string {
	s := m.ui.TagEdit
	title := "Edit Tag"
	if n := len(s.taskIDs); n > 1 {
		title = fmt.Sprintf("Edit Tag · %d tasks", n)
	}
	lines := []string{ui.DialogTitleStyle.Render(title), ""}

	for i, color := range tagEditColors {
		marker := "  "
		if i == s.colorIdx {
			marker = "> "
		}
		var swatch string
		if color == "" {
			swatch = ui.MutedStyle.Render("○ None (remove)")
		} else {
			style, _ := ui.TagDotStyle(color, false)
			swatch = style.Render(ui.TagGlyph(color)) + " " + tagColorName(color)
		}
		lines = append(lines, marker+swatch)
	}

	hints := []ui.Hint{{Key: "enter", Label: "save"}, {Key: "esc", Label: "cancel"}}
	if n := len(s.batchIDs); n > 0 {
		box := "[ ]"
		if s.batch {
			box = "[x]"
		}
		style, _ := ui.TagDotStyle(s.batchColor, false)
		lines = append(lines, "", fmt.Sprintf("  %s all %d tasks marked %s in %s",
			box, n, style.Render(ui.TagGlyph(s.batchColor)), truncateText(s.batchCat, 30)))
		hints = append([]ui.Hint{{Key: "space", Label: "whole batch"}}, hints...)
	}
	lines = append(lines, "", ui.RenderHints(hints))
	return m.dialogStyle().Render(strings.Join(lines, "\n"))
}

// copyTagFromSelected grabs the focused task's tag into the tag clipboard so it
// can be painted onto other tasks with p. Copying an untagged task stores an
// empty tag, so a subsequent paste clears the target.
func (m *model) copyTagFromSelected() {
	pos, ok := m.selectedPosition()
	if !ok || pos.Kind != selection.FocusTask {
		m.ui.Screen.StatusMsg = "Copy tag: focus a task first"
		return
	}
	task := m.project.Categories[pos.CategoryIndex].Tasks[pos.TaskIndex]
	m.ui.TagClip = TagClipboard{Set: true, Color: task.TagColor}
	m.ui.TagCopiedLast = true
	m.ui.Screen.StatusMsg = "Copied tag: " + tagClipDescription(m.ui.TagClip)
}

// paste routes the normal-mode `p`: a tag copied last wins, then the clipboard.
func (m *model) paste() tea.Cmd {
	if m.onHiddenRow() {
		m.ui.Screen.StatusMsg = "Paste onto a task or a category"
		return nil
	}
	if m.ui.TagCopiedLast && m.ui.TagClip.Set {
		m.pasteTagOntoSelected()
		return nil
	}
	return m.pasteFromClipboard()
}

// pasteTagOntoSelected applies the copied tag to the focused task without
// touching the title or status.
func (m *model) pasteTagOntoSelected() {
	if !m.ui.TagClip.Set {
		m.ui.Screen.StatusMsg = "No tag copied (use gt on a task first)"
		return
	}
	pos, ok := m.selectedPosition()
	if !ok || pos.Kind != selection.FocusTask {
		m.ui.Screen.StatusMsg = "Paste tag: focus a task first"
		return
	}
	taskID := m.project.Categories[pos.CategoryIndex].Tasks[pos.TaskIndex].ID
	m.recordHistory()
	if _, err := operations.SetTags(&m.project, []string{taskID}, m.ui.TagClip.Color); !m.applied(err) {
		return
	}
	m.storeTaskUpdate()
	m.holdFilteredOut([]string{taskID})
	m.ui.Screen.StatusMsg = "Pasted tag: " + tagClipDescription(m.ui.TagClip)
}

// visualTaskPositions keeps only real task rows from a visual selection,
// dropping any separators caught in the range so tag actions — which have no
// meaning on a divider — apply to tasks alone. Description rows round up to
// their parent task (deduped), so a task touched via both its row and its
// description is tagged exactly once.
func visualTaskPositions(positions []selection.Position) []selection.Position {
	var out []selection.Position
	for _, p := range visualRoundUpPositions(positions) {
		if p.Kind == selection.FocusTask {
			out = append(out, p)
		}
	}
	return out
}

// visualPasteTag paints the copied tag onto every task in the visual selection.
func (m *model) visualPasteTag() {
	if !m.ui.TagClip.Set {
		m.ui.Screen.StatusMsg = "No tag copied (use gt on a task first)"
		m.exitVisualMode()
		return
	}
	if m.ui.Visual.Kind != selection.FocusTask {
		m.exitVisualMode()
		return
	}
	selPositions := visualTaskPositions(m.visualSelectedPositions())
	if len(selPositions) == 0 {
		m.exitVisualMode()
		return
	}
	m.recordHistory()
	ids := positionIDs(m.project, selPositions)
	changed, err := operations.SetTags(&m.project, ids, m.ui.TagClip.Color)
	m.exitVisualMode()
	if !m.applied(err) {
		return
	}
	m.storeTaskUpdate()
	m.holdFilteredOut(ids)
	m.ui.Screen.StatusMsg = fmt.Sprintf("Pasted tag onto %d task(s)", changed)
}

// tagClipDescription is a short human label for a copied tag, e.g. "Cyan" or
// "none".
func tagClipDescription(clip TagClipboard) string {
	if clip.Color == "" {
		return "none"
	}
	return tagColorName(clip.Color)
}

// visualCycleTag gives the whole selection one shared mark, stepped from the
// first selected task so repeated presses walk the block through the palette
// together. When that task is untagged the block gets the first mark no open
// task outside it uses in its categories. Stays in visual mode.
func (m *model) visualCycleTag() {
	if m.ui.Visual.Kind != selection.FocusTask {
		return
	}
	selPositions := visualTaskPositions(m.visualSelectedPositions())
	if len(selPositions) == 0 {
		return
	}
	m.recordHistory()
	ids := positionIDs(m.project, selPositions)
	if _, err := operations.CycleTags(&m.project, ids); m.applied(err) {
		m.storeTaskUpdate()
		m.holdFilteredOut(ids)
	}
}

func positionIDs(p domain.Project, positions []selection.Position) []string {
	ids := make([]string, len(positions))
	for i, pos := range positions {
		ids[i] = p.Categories[pos.CategoryIndex].Tasks[pos.TaskIndex].ID
	}
	return ids
}

// visualStartTagEdit collects the selected task IDs, leaves visual mode (a modal
// can only open from normal mode), and opens the tag picker seeded from the
// first task's color for a shared edit across the whole selection.
func (m *model) visualStartTagEdit() {
	if m.ui.Visual.Kind != selection.FocusTask {
		m.exitVisualMode()
		return
	}
	selPositions := visualTaskPositions(m.visualSelectedPositions())
	ids := make([]string, 0, len(selPositions))
	color := ""
	if len(selPositions) > 0 {
		color = m.project.Categories[selPositions[0].CategoryIndex].Tasks[selPositions[0].TaskIndex].TagColor
	}
	for _, pos := range selPositions {
		ids = append(ids, m.project.Categories[pos.CategoryIndex].Tasks[pos.TaskIndex].ID)
	}
	m.exitVisualMode()
	m.beginTagEdit(ids, color)
}

// taskByID returns a pointer to the stored task with the given ID, or nil.
func (m *model) taskByID(id string) *domain.Task {
	ci, ti := m.findTaskByID(id)
	if ci < 0 {
		return nil
	}
	return &m.project.Categories[ci].Tasks[ti]
}

// tagColorIndex is the position of color within tagEditColors (0 = None).
func tagColorIndex(color string) int {
	for i, c := range tagEditColors {
		if c == color {
			return i
		}
	}
	return 0
}

// tagColorName is the human-facing name of a tag mark, used in the info,
// filter, and tag-picker views.
func tagColorName(color string) string {
	hue, diamond := strings.CutSuffix(color, domain.TagDiamondSuffix)
	var name string
	switch hue {
	case domain.TagGreen:
		name = "Green"
	case domain.TagBlue:
		name = "Blue"
	case domain.TagMagenta:
		name = "Magenta"
	case domain.TagCyan:
		name = "Cyan"
	default:
		return color
	}
	if diamond {
		name += " diamond"
	}
	return name
}
