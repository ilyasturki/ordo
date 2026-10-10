package app

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"phasionary/internal/app/modes"
	"phasionary/internal/app/selection"
	"phasionary/internal/domain"
	"phasionary/internal/ui"
)

// tagEditColors is the color picker order: the palette preceded by a leading ""
// "None (remove)" row, so removing a tag is just selecting the first row.
var tagEditColors = append([]string{""}, domain.TagColorCycle...)

type tagEditField int

const (
	tagFieldLabel tagEditField = iota
	tagFieldColor
)

// TagEditState backs the `T` tag editor: a color picker plus a label field.
// taskIDs holds every task the edit applies to — one for a single task, many
// for a visual-mode bulk edit — so it survives any position shift while open.
type TagEditState struct {
	input    textinput.Model
	taskIDs  []string
	colorIdx int
	field    tagEditField
}

// cycleTag advances the selected task's tag color one step through the palette.
func (m *model) cycleTag() {
	if !m.ui.Modes.CanPerformAction(modes.ActionChangeTag) {
		return
	}
	position, ok := m.selectedPosition()
	if !ok || position.Kind != selection.FocusTask {
		return
	}
	task := &m.project.Categories[position.CategoryIndex].Tasks[position.TaskIndex]
	m.recordHistory()
	if task.CycleTag() {
		m.storeTaskUpdate()
		return
	}
	m.discardLastHistory()
}

// startTagEdit opens the tag editor for the selected task, seeded with its
// current color and label.
func (m *model) startTagEdit() tea.Cmd {
	if !m.ui.Modes.CanPerformAction(modes.ActionChangeTag) {
		return nil
	}
	position, ok := m.selectedPosition()
	if !ok || position.Kind != selection.FocusTask {
		return nil
	}
	task := m.project.Categories[position.CategoryIndex].Tasks[position.TaskIndex]
	return m.beginTagEdit([]string{task.ID}, task.TagColor, task.TagLabel)
}

// beginTagEdit focuses a fresh editor over the given tasks and enters the modal.
// Callers must already be in (or have returned to) normal mode.
func (m *model) beginTagEdit(taskIDs []string, color, label string) tea.Cmd {
	if len(taskIDs) == 0 {
		return nil
	}
	ti := textinput.New()
	ti.Prompt = ""
	ti.SetValue(label)
	ti.SetCursor(len([]rune(label)))
	cmd := ti.Focus()
	m.ui.TagEdit = TagEditState{
		input:    ti,
		taskIDs:  taskIDs,
		colorIdx: tagColorIndex(color),
		field:    tagFieldLabel,
	}
	if !m.ui.Modes.ToTagEdit() {
		return nil
	}
	return cmd
}

func (m *model) handleTagEditKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	s := &m.ui.TagEdit
	switch msg.String() {
	case "esc":
		m.cancelTagEdit()
		return m, nil
	case "enter":
		m.finishTagEdit()
		return m, nil
	case "tab", "shift+tab":
		s.field = otherTagField(s.field)
		return m, nil
	}

	if s.field == tagFieldColor {
		switch msg.String() {
		case "down", "j", "right", "l":
			if s.colorIdx < len(tagEditColors)-1 {
				s.colorIdx++
			}
		case "up", "k", "left", "h":
			if s.colorIdx > 0 {
				s.colorIdx--
			} else {
				s.field = tagFieldLabel // step up past the first color into the label
			}
		}
		return m, nil
	}

	// Label field.
	if msg.String() == "down" {
		s.field = tagFieldColor
		return m, nil
	}
	return m, m.updateTagLabel(msg)
}

func (m *model) updateTagLabel(msg tea.Msg) tea.Cmd {
	s := &m.ui.TagEdit
	var cmd tea.Cmd
	s.input, cmd = s.input.Update(msg)
	sanitizeInput(&s.input)
	// A label needs a colored dot to ride on: typing one onto an untagged task
	// promotes the "None" selection to the first real color.
	if strings.TrimSpace(s.input.Value()) != "" && s.colorIdx == 0 {
		s.colorIdx = tagColorIndex(domain.TagGreen)
	}
	return cmd
}

func otherTagField(f tagEditField) tagEditField {
	if f == tagFieldLabel {
		return tagFieldColor
	}
	return tagFieldLabel
}

func (m *model) cancelTagEdit() {
	m.ui.TagEdit = TagEditState{}
	m.ui.Modes.ToNormal()
	m.ui.Screen.StatusMsg = ""
}

func (m *model) finishTagEdit() {
	s := m.ui.TagEdit
	color := tagEditColors[s.colorIdx]
	label := strings.TrimSpace(s.input.Value())
	if color == "" {
		label = "" // "None" removes the tag entirely
	}
	m.ui.TagEdit = TagEditState{}
	m.ui.Modes.ToNormal()
	if len(s.taskIDs) == 0 {
		return
	}

	m.recordHistory()
	changed := 0
	for _, id := range s.taskIDs {
		task := m.taskByID(id)
		if task == nil {
			continue
		}
		if task.TagColor == color && task.TagLabel == label {
			continue
		}
		_ = task.SetTagColor(color)
		if color != "" {
			task.SetTagLabel(label)
		}
		changed++
	}
	if changed == 0 {
		m.discardLastHistory()
		return
	}
	m.storeTaskUpdate()
	// A tag change can move a task in or out of an active tag filter, so rebuild
	// and keep the cursor on the first edited task.
	m.rebuildPositions()
	m.selectTaskByID(s.taskIDs[0])
	m.ensureVisible()
	if len(s.taskIDs) > 1 {
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

	labelMarker := "  "
	if s.field == tagFieldLabel {
		labelMarker = "> "
	}
	lines = append(lines, labelMarker+"Label  "+s.input.View(), "", "  Color")

	for i, color := range tagEditColors {
		marker := "  "
		if i == s.colorIdx {
			if s.field == tagFieldColor {
				marker = "> "
			} else {
				marker = "· "
			}
		}
		var swatch string
		if color == "" {
			swatch = ui.MutedStyle.Render("○ None (remove)")
		} else {
			style, _ := ui.TagDotStyle(color, false)
			swatch = style.Render(ui.TagDot) + " " + tagColorName(color)
		}
		lines = append(lines, marker+swatch)
	}

	lines = append(lines, "", ui.RenderHints([]ui.Hint{
		{Key: "tab", Label: "switch"},
		{Key: "enter", Label: "save"},
		{Key: "esc", Label: "cancel"},
	}))
	return m.dialogStyle().Render(strings.Join(lines, "\n"))
}

// copyTagFromSelected grabs the focused task's tag into the tag clipboard so it
// can be painted onto other tasks with p, and — when the tag carries a label —
// also copies that label to the system clipboard. Copying an untagged task
// stores an empty tag, so a subsequent paste clears the target.
func (m *model) copyTagFromSelected() tea.Cmd {
	pos, ok := m.selectedPosition()
	if !ok || pos.Kind != selection.FocusTask {
		m.ui.Screen.StatusMsg = "Copy tag: focus a task first"
		return nil
	}
	task := m.project.Categories[pos.CategoryIndex].Tasks[pos.TaskIndex]
	m.ui.TagClip = TagClipboard{Set: true, Color: task.TagColor, Label: task.TagLabel}
	m.ui.TagCopiedLast = true
	desc := tagClipDescription(m.ui.TagClip)
	// Only a label carries text worth exporting; a color-only or untagged tag has
	// nothing to put on the system clipboard, so report the copy synchronously.
	if task.TagLabel == "" {
		m.ui.Screen.StatusMsg = "Copied tag: " + desc
		return nil
	}
	return m.copyToClipboard(task.TagLabel, "tag: "+desc)
}

// paste routes the normal-mode `p`: a tag copied last wins, then the clipboard.
func (m *model) paste() tea.Cmd {
	if m.ui.TagCopiedLast && m.ui.TagClip.Set {
		m.pasteTagOntoSelected()
		return nil
	}
	return m.pasteFromClipboard()
}

// pasteTagOntoSelected applies the copied tag to the focused task, replacing its
// color and label without touching the title or status.
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
	task := &m.project.Categories[pos.CategoryIndex].Tasks[pos.TaskIndex]
	if task.TagColor == m.ui.TagClip.Color && task.TagLabel == m.ui.TagClip.Label {
		return
	}
	taskID := task.ID
	m.recordHistory()
	m.applyTagClip(task)
	m.storeTaskUpdate()
	// A tag change can move the task in or out of an active tag filter.
	m.rebuildPositions()
	m.selectTaskByID(taskID)
	m.ensureVisible()
	m.ui.Screen.StatusMsg = "Pasted tag: " + tagClipDescription(m.ui.TagClip)
}

// applyTagClip writes the copied tag onto task. SetTagColor("") already clears
// the label, so the label is only reapplied when a color is present.
func (m *model) applyTagClip(task *domain.Task) {
	_ = task.SetTagColor(m.ui.TagClip.Color)
	if m.ui.TagClip.Color != "" {
		task.SetTagLabel(m.ui.TagClip.Label)
	}
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
	changed := 0
	for _, pos := range selPositions {
		task := &m.project.Categories[pos.CategoryIndex].Tasks[pos.TaskIndex]
		if task.TagColor == m.ui.TagClip.Color && task.TagLabel == m.ui.TagClip.Label {
			continue
		}
		m.applyTagClip(task)
		changed++
	}
	m.exitVisualMode()
	if changed == 0 {
		m.discardLastHistory()
		return
	}
	m.storeTaskUpdate()
	m.rebuildPositions()
	m.ensureVisible()
	m.ui.Screen.StatusMsg = fmt.Sprintf("Pasted tag onto %d task(s)", changed)
}

// tagClipDescription is a short human label for a copied tag, e.g. "Cyan
// (urgent)" or "none".
func tagClipDescription(clip TagClipboard) string {
	if clip.Color == "" {
		return "none"
	}
	s := tagColorName(clip.Color)
	if clip.Label != "" {
		s += " (" + clip.Label + ")"
	}
	return s
}

// visualCycleTag sets every task in the visual selection to a single shared next
// color, stepped from the first selected task so repeated presses walk the whole
// block through the palette together. Stays in visual mode.
func (m *model) visualCycleTag() {
	if m.ui.Visual.Kind != selection.FocusTask {
		return
	}
	selPositions := visualTaskPositions(m.visualSelectedPositions())
	if len(selPositions) == 0 {
		return
	}
	first := m.project.Categories[selPositions[0].CategoryIndex].Tasks[selPositions[0].TaskIndex]
	next := domain.NextTagColor(first.TagColor)
	m.recordHistory()
	for _, pos := range selPositions {
		_ = m.project.Categories[pos.CategoryIndex].Tasks[pos.TaskIndex].SetTagColor(next)
	}
	m.storeTaskUpdate()
}

// visualStartTagEdit collects the selected task IDs, leaves visual mode (a modal
// can only open from normal mode), and opens the tag editor seeded from the
// first task's color for a shared edit across the whole selection.
func (m *model) visualStartTagEdit() tea.Cmd {
	if m.ui.Visual.Kind != selection.FocusTask {
		m.exitVisualMode()
		return nil
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
	return m.beginTagEdit(ids, color, "")
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

// tagColorName is the human-facing name of a tag color, used in the info,
// filter, and tag-editor views.
func tagColorName(color string) string {
	switch color {
	case domain.TagGreen:
		return "Green"
	case domain.TagBlue:
		return "Blue"
	case domain.TagMagenta:
		return "Magenta"
	case domain.TagCyan:
		return "Cyan"
	default:
		return color
	}
}
