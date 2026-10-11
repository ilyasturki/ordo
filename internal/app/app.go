package app

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/ilyasturki/ordo/internal/app/components"
	"github.com/ilyasturki/ordo/internal/app/modes"
	"github.com/ilyasturki/ordo/internal/app/selection"
	"github.com/ilyasturki/ordo/internal/clipboard"
	"github.com/ilyasturki/ordo/internal/config"
	"github.com/ilyasturki/ordo/internal/data"
	"github.com/ilyasturki/ordo/internal/domain"
	"github.com/ilyasturki/ordo/internal/export"
	"github.com/ilyasturki/ordo/internal/ui"
)

type clipboardResultMsg struct {
	err error
	// label, when set, names what was copied (e.g. "UUID: 550e..."); empty
	// falls back to a generic "Copied!".
	label string
	text  string
	// status, when set, replaces the "Copied …" report.
	status string
	ofCopy bool
	blurs  int
}

// clipboardWrite is what ordo last put on the system clipboard; fresh until the terminal loses focus.
type clipboardWrite struct {
	text  string
	fresh bool
}

// copyToClipboard sends text out as OSC 52 for the terminal to put on the
// clipboard, or through wl-copy (pbcopy on macOS) when copy_via asks for it.
//
// tmux passes an application's OSC 52 on only with `set-clipboard on`.
func (m *model) copyToClipboard(text, label string) tea.Cmd {
	return m.writeClipboard(clipboardResultMsg{text: text, label: label})
}

func (m *model) writeClipboard(result clipboardResultMsg) tea.Cmd {
	result.blurs = m.ui.Blurs
	result.ofCopy = !m.ui.Clipboard.empty() && result.text == m.ui.Clipboard.Text
	if result.ofCopy {
		m.ui.ClipboardWrites++
	}
	viaTerminal := tea.Sequence(tea.SetClipboard(result.text), func() tea.Msg { return result })
	if m.deps.CfgManager.Get().CopyVia != config.CopyViaWlCopy && !clipboard.Redirected() {
		return viaTerminal
	}
	return func() tea.Msg {
		err := clipboard.Write(result.text)
		if errors.Is(err, clipboard.ErrNoBackend) {
			return viaTerminal()
		}
		result.err = err
		return result
	}
}

// distrustClipboardWrites marks a moment another program may have replaced our last write.
func (m *model) distrustClipboardWrites() {
	m.ui.Blurs++
	m.ui.LastWrite.fresh = false
}

func (m *model) handleClipboardResult(msg clipboardResultMsg) {
	if msg.ofCopy {
		m.ui.ClipboardWrites = max(0, m.ui.ClipboardWrites-1)
	}
	// A clipboard that never got the copy can't vouch for it.
	if msg.err != nil && m.ui.Clipboard.Text == msg.text {
		m.ui.Clipboard.Written = false
	}
	if msg.err == nil {
		m.ui.LastWrite = clipboardWrite{text: msg.text, fresh: msg.blurs == m.ui.Blurs}
	}
	// A cut moved or cancelled before its write answered no longer waits on p.
	cutGone := msg.status != "" && !(m.ui.Clipboard.IsCut && m.ui.Clipboard.Text == msg.text)
	switch {
	case cutGone && msg.err == nil:
	case msg.err != nil && msg.status != "" && !cutGone:
		m.ui.Screen.StatusMsg = fmt.Sprintf("%s (copy failed: %v)", msg.status, msg.err)
	case msg.err != nil:
		m.ui.Screen.StatusMsg = fmt.Sprintf("Copy failed: %v", msg.err)
	case msg.status != "":
		m.ui.Screen.StatusMsg = msg.status
	case msg.label != "":
		m.ui.Screen.StatusMsg = "Copied " + msg.label
	default:
		m.ui.Screen.StatusMsg = "Copied!"
	}
}

// saveErrMsg reports a failed background save from the async saver. Successful
// writes send nothing; a failure surfaces here as a status message.
type saveErrMsg struct{ err error }

type model struct {
	project domain.Project
	ui      *UIState
	deps    *Dependencies
}

func (m model) Init() tea.Cmd {
	return m.listenSaveErrors()
}

// listenSaveErrors subscribes to the async saver's error channel so a failed
// background write becomes a visible status message. It resolves one delivery
// then must be re-issued (see the saveErrMsg case) to keep listening.
func (m model) listenSaveErrors() tea.Cmd {
	if m.deps.Saver == nil {
		return nil
	}
	results := m.deps.Saver.Results()
	return func() tea.Msg {
		err, ok := <-results
		if !ok {
			return nil
		}
		return saveErrMsg{err: err}
	}
}

// Update runs the event and then records where the cursor ended up. Doing it
// here rather than in each of the many handlers that can move the cursor means
// no new movement can be added without being remembered.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	next, cmd := m.update(msg)
	if updated, ok := next.(model); ok {
		updated.collapseLeftRow()
		updated.trackCursor()
		return updated, cmd
	}
	return next, cmd
}

func (m model) update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.ui.Screen.Width = msg.Width
		m.ui.Screen.Height = msg.Height
		// The first size message is the earliest point a restored cursor can be
		// centered; every later one is a real resize, where the user has the view
		// in front of them and only wants the cursor kept on screen.
		if m.ui.Screen.PendingCenter {
			m.ui.Screen.PendingCenter = false
			m.centerOnSelected()
		} else {
			m.ensureVisible()
		}
		if m.ui.Modes.IsProjectPicker() {
			m.ui.Picker.ensureVisible(m.pickerVisibleCount())
		}
		if m.ui.Modes.IsMovePicker() {
			m.ui.MovePicker.ensureVisible(m.movePickerRows())
		}
	case clipboardResultMsg:
		m.handleClipboardResult(msg)
	case clipboardLinesMsg:
		m.resolvePaste(msg)
	case editorClipboardMsg:
		return m, m.handleEditorClipboard(msg)
	case tea.PasteMsg:
		return m, m.handlePaste(msg.Content)
	case openURLResultMsg:
		m.handleOpenURLResult(msg)
	case taskCommandResultMsg:
		m.handleTaskCommandResult(msg)
	case editorFinishedMsg:
		m.handleEditorFinished(msg)
		return m, nil
	case tea.FocusMsg:
		m.ui.Screen.WindowFocused = true
		m.ui.FocusReported = true
	case tea.BlurMsg:
		m.ui.Screen.WindowFocused = false
		m.ui.FocusReported = true
		m.distrustClipboardWrites()
	case tea.MouseWheelMsg:
		m.handleMouseWheel(msg)
		return m, nil
	case saveErrMsg:
		if msg.err != nil {
			m.ui.Screen.StatusMsg = saveFailedMessage(msg.err)
		}
		return m, m.listenSaveErrors()
	case tea.KeyPressMsg:
		m.ui.Screen.StatusMsg = ""
		return m.handleKeyMsg(msg)
	default:
		return m.forwardToInput(msg)
	}
	return m, nil
}

// wheelTick accumulates a raw wheel event and reports whether enough have piled
// up (wheelScrollDivisor) to move a line, resetting the counter when it fires.
// This smooths out trackpads and hi-res wheels that emit several events per
// physical notch. A change of direction restarts the count.
func (m *model) wheelTick(button tea.MouseButton) bool {
	var dir int
	switch button {
	case tea.MouseWheelUp:
		dir = -1
	case tea.MouseWheelDown:
		dir = 1
	default:
		return false
	}
	if (dir > 0) != (m.ui.Screen.WheelAccum > 0) {
		m.ui.Screen.WheelAccum = 0
	}
	m.ui.Screen.WheelAccum += dir
	if m.ui.Screen.WheelAccum <= -wheelScrollDivisor || m.ui.Screen.WheelAccum >= wheelScrollDivisor {
		m.ui.Screen.WheelAccum = 0
		return true
	}
	return false
}

func (m *model) handleMouseWheel(msg tea.MouseWheelMsg) {
	if !m.wheelTick(msg.Button) {
		return
	}
	switch m.ui.Modes.Current() {
	case modes.ModeNormal:
		switch msg.Button {
		case tea.MouseWheelUp:
			m.scrollUp(wheelScrollStep)
		case tea.MouseWheelDown:
			m.scrollDown(wheelScrollStep)
		}
	case modes.ModeDescriptionEdit:
		// The textarea's view follows its cursor, so scrolling a long
		// description means nudging the cursor a line at a time.
		for i := 0; i < wheelScrollStep; i++ {
			switch msg.Button {
			case tea.MouseWheelUp:
				m.ui.DescriptionEdit.textarea.CursorUp()
			case tea.MouseWheelDown:
				m.ui.DescriptionEdit.textarea.CursorDown()
			}
		}
	}
}

// normalizeKey rewrites Shift+Backspace to a plain Backspace so it deletes a
// character wherever the user types. Terminals speaking the enhanced keyboard
// protocol deliver Shift+Backspace as a distinct key, but bubbles' text inputs
// bind delete-backward to "backspace"/"ctrl+h" only, so the modified form is
// otherwise silently ignored. Normal mode binds nothing to backspace, so this
// is a no-op outside the text-entry modes.
func normalizeKey(msg tea.KeyPressMsg) tea.KeyPressMsg {
	if msg.Code == tea.KeyBackspace && msg.Mod&tea.ModShift != 0 {
		msg.Mod &^= tea.ModShift
	}
	return msg
}

// The title and description editors read ctrl+v in their own key handlers.
func (m model) lineInputFocused() bool {
	switch m.ui.Modes.Current() {
	case modes.ModeSearch, modes.ModeCommandPrompt:
		return true
	case modes.ModeHelp:
		return m.ui.Help.Filtering
	case modes.ModeProjectPicker:
		return m.ui.Picker.isAdding || m.ui.Picker.filtering
	}
	return false
}

func (m model) handleKeyMsg(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	msg = normalizeKey(msg)
	if msg.String() == "ctrl+c" && !m.ui.Modes.IsExternalEdit() {
		if m.ui.Modes.IsDescriptionEdit() {
			m.finishDescriptionEdit()
		}
		return m, tea.Quit
	}
	if msg.String() == "ctrl+v" && m.lineInputFocused() {
		return m, readClipboardForEditor(m.ui.Modes.Current())
	}
	switch m.ui.Modes.Current() {
	case modes.ModeHelp:
		return m.handleHelpKey(msg)
	case modes.ModeConfirmDelete:
		return m.handleConfirmDeleteKey(msg), nil
	case modes.ModeOptions:
		return m.handleOptionsKey(msg), nil
	case modes.ModeProjectPicker:
		return m.handleProjectPickerKey(msg)
	case modes.ModeFilter:
		return m.handleFilterKey(msg), nil
	case modes.ModeInfo:
		return m.handleInfoKey(msg), nil
	case modes.ModeChangelog:
		return m.handleChangelogKey(msg), nil
	case modes.ModeEstimatePicker:
		return m.handleEstimatePickerKey(msg), nil
	case modes.ModeURLPicker:
		return m.handleURLPickerKey(msg)
	case modes.ModeYankPicker:
		return m.handleYankPickerKey(msg)
	case modes.ModeSearch:
		return m.handleSearchKey(msg)
	case modes.ModeVisual:
		return m.handleVisualKey(msg)
	case modes.ModeEdit:
		cmd := m.handleEditKey(msg)
		return m, cmd
	case modes.ModeDescriptionEdit:
		return m.handleDescriptionEditKey(msg)
	case modes.ModeTagEdit:
		return m.handleTagEditKey(msg)
	case modes.ModeCommandPrompt:
		return m.handleCommandPromptKey(msg)
	case modes.ModePasteChooser:
		return m.handlePasteChooserKey(msg)
	case modes.ModeMovePicker:
		return m.handleMovePickerKey(msg), nil
	case modes.ModeExternalEdit:
		return m, nil
	default:
		return m.handleNormalKey(msg)
	}
}

func (m model) handleHelpKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.ui.Help.Filtering {
		return m.handleHelpFilterKey(msg)
	}
	switch msg.String() {
	case "q", "esc", "?":
		m.ui.Modes.ToNormal()
		return m, nil
	case "/":
		cmd := m.startHelpFilter()
		return m, cmd
	case "a":
		m.toggleHelpExpanded()
	case "j", "down":
		m.moveHelpFocus(1)
	case "k", "up":
		m.moveHelpFocus(-1)
	case "h", "left":
		m.moveHelpColumn(-1)
	case "l", "right":
		m.moveHelpColumn(1)
	case "ctrl+d":
		m.moveHelpFocus(m.helpBodyHeight() / 2)
	case "ctrl+u":
		m.moveHelpFocus(-m.helpBodyHeight() / 2)
	case "g":
		m.moveHelpFocus(-len(m.helpBody().targets))
	case "G":
		m.moveHelpFocus(len(m.helpBody().targets))
	case "enter":
		return m.runFocusedHelpBinding()
	}
	return m, nil
}

// handleHelpFilterKey handles keys while the `/` filter is active: Esc clears the
// filter (Esc again closes the dialog), Enter runs the focused match, arrows and
// ctrl+d/u navigate, and everything else edits the query.
func (m model) handleHelpFilterKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.ui.Help.Filtering = false
		m.ui.Help.Filter = textinput.Model{}
		m.ui.Help.Focused = 0
		m.ui.Help.ScrollOffset = 0
		m.ensureHelpVisible()
		return m, nil
	case "enter":
		return m.runFocusedHelpBinding()
	case "down":
		m.moveHelpFocus(1)
		return m, nil
	case "up":
		m.moveHelpFocus(-1)
		return m, nil
	case "ctrl+d":
		m.moveHelpFocus(m.helpBodyHeight() / 2)
		return m, nil
	case "ctrl+u":
		m.moveHelpFocus(-m.helpBodyHeight() / 2)
		return m, nil
	}
	return m, m.updateHelpFilter(msg)
}

func (m *model) updateHelpFilter(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.ui.Help.Filter, cmd = m.ui.Help.Filter.Update(msg)
	sanitizeInput(&m.ui.Help.Filter)
	m.ui.Help.Focused = 0
	m.ui.Help.ScrollOffset = 0
	m.ensureHelpVisible()
	return cmd
}

// startHelpFilter enters the `/` filter sub-mode with a fresh, focused input.
func (m *model) startHelpFilter() tea.Cmd {
	ti := newTextInput()
	cmd := ti.Focus()
	m.ui.Help.Filter = ti
	m.ui.Help.Filtering = true
	m.ui.Help.Focused = 0
	m.ui.Help.ScrollOffset = 0
	m.ensureHelpVisible()
	return cmd
}

// runFocusedHelpBinding runs the shortcut under the cursor and closes the
// dialog. The cursor can land on reference-only rows, so they are checked here
// rather than being made unreachable.
func (m model) runFocusedHelpBinding() (tea.Model, tea.Cmd) {
	body := m.helpBody()
	idx := m.ui.Help.Focused
	if idx < 0 || idx >= len(body.targets) {
		return m, nil
	}
	entry := body.entryAt(body.targets[idx])
	if !entry.runnable {
		return m, nil
	}
	m.ui.Modes.ToNormal()
	if entry.command != nil {
		return m, m.startTaskCommand(*entry.command)
	}
	return m, normalBindings[entry.bindingIndex].action(&m)
}

func (m model) handleConfirmDeleteKey(msg tea.KeyPressMsg) model {
	switch msg.String() {
	case "y", "enter":
		switch m.ui.ConfirmDelete.Kind {
		case ConfirmDeleteProject:
			m.confirmDeleteProject()
		case ConfirmDeleteVisualRange:
			m.confirmDeleteVisualRange()
		default:
			m.confirmDeleteAction()
		}
	case "k":
		if pos, ok := m.selectedPosition(); ok && m.ui.ConfirmDelete.Kind == ConfirmDeleteSelection && m.deletedChildren(pos) > 0 {
			m.confirmDeleteKeepingChildren()
		}
	case "n", "esc":
		switch m.ui.ConfirmDelete.Kind {
		case ConfirmDeleteProject:
			m.ui.ConfirmDelete.reset()
			m.ui.Modes.ToProjectPicker()
		default:
			m.ui.ConfirmDelete.reset()
			m.ui.Modes.ToNormal()
		}
	}
	return m
}

func (m model) handleOptionsKey(msg tea.KeyPressMsg) model {
	switch msg.String() {
	case "q", "esc", "enter":
		m.ui.Modes.ToNormal()
		// Re-clamp scroll now that the shortcut bar (which is hidden while
		// Options was open) may have toggled visible/hidden — its row count
		// changes available content height.
		m.ensureVisible()
	case "j", "down":
		if m.ui.Options.selectedOption < len(optionSpecs)-1 {
			m.ui.Options.selectedOption++
		}
	case "k", "up":
		if m.ui.Options.selectedOption > 0 {
			m.ui.Options.selectedOption--
		}
	case "space", "tab", "l", "right":
		m.cycleSelectedOption(1)
	case "h", "left":
		m.cycleSelectedOption(-1)
	case "d":
		m.resetSelectedOption()
	}
	return m
}

func (m model) handleFilterKey(msg tea.KeyPressMsg) model {
	catCount := len(m.project.Categories)
	switch msg.String() {
	case "q", "f":
		m.ui.Filter.ResetToHub()
		m.ui.Modes.ToNormal()
		m.rebuildPositions()
		m.ensureVisible()
	case "esc":
		if m.ui.Filter.View() == FilterViewHub {
			m.ui.Modes.ToNormal()
			m.rebuildPositions()
			m.ensureVisible()
		} else {
			m.ui.Filter.SetView(FilterViewHub)
		}
	case "j", "down":
		m.ui.Filter.MoveDown(catCount)
	case "k", "up":
		m.ui.Filter.MoveUp()
	case "enter", "space":
		if m.ui.Filter.View() == FilterViewHub {
			m.openFilterHubSelection()
		} else {
			m.ui.Filter.ToggleSelected(m.project.Categories)
			m.rebuildPositions()
			m.ensureVisible()
		}
	}
	return m
}

func (m *model) openFilterHubSelection() {
	switch m.ui.Filter.HubSelected() {
	case FilterHubStatus:
		m.ui.Filter.SetView(FilterViewStatus)
	case FilterHubPriority:
		m.ui.Filter.SetView(FilterViewPriority)
	case FilterHubCategory:
		m.ui.Filter.SetView(FilterViewCategory)
	case FilterHubTag:
		m.ui.Filter.SetView(FilterViewTag)
	case FilterHubClearAll:
		if m.ui.Filter.HasActiveFilter() {
			m.ui.Filter.ClearAll()
			m.rebuildPositions()
			m.ensureVisible()
		}
	}
}

func (m model) handleInfoKey(msg tea.KeyPressMsg) model {
	if scrollDialogKey(msg.String(), &m.ui.Info.ScrollOffset, m.infoViewportHeight(), m.infoMaxScroll()) {
		m.ui.Modes.ToNormal()
	}
	return m
}

func (m model) handleChangelogKey(msg tea.KeyPressMsg) model {
	if scrollDialogKey(msg.String(), &m.ui.Changelog.ScrollOffset, m.changelogViewportHeight(), m.changelogMaxScroll()) {
		m.ui.Modes.ToNormal()
	}
	return m
}

// scrollDialogKey moves a read-only dialog's window and reports whether key closes it.
func scrollDialogKey(key string, offset *int, height, maxScroll int) bool {
	switch key {
	case "q", "esc":
		return true
	case "j", "down":
		*offset++
	case "k", "up":
		*offset--
	case "ctrl+d":
		*offset += height / 2
	case "ctrl+u":
		*offset -= height / 2
	case "g":
		*offset = 0
	case "G":
		*offset = maxScroll
	}
	*offset = min(max(*offset, 0), maxScroll)
	return false
}

func (m model) handleEstimatePickerKey(msg tea.KeyPressMsg) model {
	switch msg.String() {
	case "q", "esc":
		m.ui.Modes.ToNormal()
	case "j", "down":
		m.ui.EstimatePicker.MoveDown()
	case "k", "up":
		m.ui.EstimatePicker.MoveUp()
	case "enter":
		m.selectEstimate(m.ui.EstimatePicker.SelectedValue())
		m.ui.Modes.ToNormal()
	}
	return m
}

func (m model) handleURLPickerKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		m.ui.Modes.ToNormal()
		return m, nil
	case "j", "down":
		m.ui.URLPicker.MoveDown()
		return m, nil
	case "k", "up":
		m.ui.URLPicker.MoveUp()
		return m, nil
	case "enter":
		url := m.ui.URLPicker.SelectedURL()
		m.ui.Modes.ToNormal()
		if url == "" {
			return m, nil
		}
		return m, openURL(url)
	}
	return m, nil
}

func (m model) handleYankPickerKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		m.ui.Modes.ToNormal()
		return m, nil
	case "j", "down":
		m.ui.YankPicker.MoveDown()
		return m, nil
	case "k", "up":
		m.ui.YankPicker.MoveUp()
		return m, nil
	case "enter", "y":
		it, ok := m.ui.YankPicker.SelectedItem()
		m.ui.Modes.ToNormal()
		if !ok {
			return m, nil
		}
		return m, m.copyYankItem(it)
	}
	return m, nil
}

func (m model) handleNormalKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	cmd := m.dispatchNormalKey(msg.String())
	return m, cmd
}

// copyTextForPosition returns the plain text that `y` copies for the item at
// pos: the name for a project/category, the title for a separator, the title
// plus description (blank-line separated) for a task, and the body alone for a
// description row. Empty when there is nothing to copy.
func (m *model) copyTextForPosition(pos selection.Position) string {
	switch pos.Kind {
	case selection.FocusProject:
		return m.project.Name
	case selection.FocusCategory:
		return m.project.Categories[pos.CategoryIndex].Name
	case selection.FocusTask, selection.FocusSeparator:
		task := m.project.Categories[pos.CategoryIndex].Tasks[pos.TaskIndex]
		if pos.Kind == selection.FocusTask && task.Description != "" {
			return task.Title + "\n\n" + task.Description
		}
		return task.Title
	case selection.FocusDescription:
		return m.project.Categories[pos.CategoryIndex].Tasks[pos.TaskIndex].Description
	}
	return ""
}

func (m *model) copySelected() tea.Cmd {
	pos, ok := m.selectedPosition()
	if !ok {
		return nil
	}
	// A task copy also stashes the whole task for paste; other kinds (category,
	// description, separator) only put text on the system clipboard.
	text := m.copyTextForPosition(pos)
	if pos.Kind == selection.FocusTask {
		tasks, ids := m.taskBlock(pos)
		m.ui.Clipboard = ClipboardState{Tasks: tasks, TaskIDs: ids, Text: text, Written: true}
		m.ui.TagCopiedLast = false
	}
	return m.copyToClipboard(text, "")
}

func (m *model) copyCategoryContent() tea.Cmd {
	pos, ok := m.selectedPosition()
	if !ok || pos.Kind == selection.FocusProject {
		return nil
	}
	return m.copyToClipboard(export.CategoryMarkdown(m.project, pos.CategoryIndex), "")
}

func (m model) View() tea.View {
	v := tea.NewView(m.renderView())
	v.AltScreen = true
	// Mouse tracking is on so the wheel scrolls the viewport (see
	// handleMouseWheel). This means the terminal forwards drags to the app, so
	// selecting on-screen text is done with Shift+drag (the usual override).
	v.MouseMode = tea.MouseModeCellMotion
	v.ReportFocus = true
	return v
}

func (m model) renderView() string {
	if m.ui.Screen.Height == 0 {
		return ""
	}

	if m.ui.Picker.fullscreen && (m.ui.Modes.IsProjectPicker() || m.isConfirmingProjectDelete()) {
		if m.isConfirmingProjectDelete() {
			return components.NewModal(m.ui.Screen.Width, m.ui.Screen.Height).
				Render(m.projectPickerView(), m.confirmDeleteView())
		}
		return m.projectPickerView()
	}

	layout := m.buildLayout()
	viewport := NewViewport(layout, m.ui.Screen.Height, m.layoutConfig())
	viewport.ComputeVisibility(m.ui.Screen.TopRow)

	var lines []string

	if viewport.HasMoreAbove {
		lines = append(lines, withStatus(ui.MutedStyle.Render(scrollMoreAbove), m.statusText(), m.ui.Screen.Width))
	}

	// Only the first item drawn can be entered partway through; if none fit at
	// all, the partial below is that item and inherits the offset.
	top := viewport.RowOffset
	for i := viewport.VisibleStart; i < viewport.VisibleEnd; i++ {
		lines = append(lines, m.renderLayoutItemWindow(layout.Items[i], top, 0))
		top = 0
	}

	if viewport.HasMoreBelow && viewport.VisibleEnd < len(layout.Items) {
		if remaining := viewport.RemainingContentHeight(); remaining > 0 {
			if partial := m.renderLayoutItemWindow(layout.Items[viewport.VisibleEnd], top, remaining); partial != "" {
				lines = append(lines, partial)
			}
		}
	}

	if viewport.HasMoreBelow {
		lines = append(lines, ui.MutedStyle.Render(scrollMoreBelow))
	}

	content := strings.Join(lines, "\n")
	if bar := m.renderBottomBar(); bar != "" {
		// Push the bar to the bottom row even when content is shorter than the
		// screen. The viewport already reserved this row via FooterHeight, so
		// the gap calculation below accounts for the bar itself.
		rendered := strings.Count(content, "\n") + 1
		if content == "" {
			rendered = 0
		}
		gap := m.ui.Screen.Height - rendered - 1
		if gap > 0 {
			content += strings.Repeat("\n", gap)
		}
		if content != "" {
			content += "\n"
		}
		content += bar
	}
	modal := components.NewModal(m.ui.Screen.Width, m.ui.Screen.Height)
	if m.ui.Modes.IsEdit() && m.ui.Edit.mergeInto != "" {
		return modal.Render(content, m.confirmMergeView())
	}
	switch m.ui.Modes.Current() {
	case modes.ModeHelp:
		return modal.Render(content, m.helpView())
	case modes.ModeConfirmDelete:
		if m.isConfirmingProjectDelete() {
			return modal.Render(modal.Render(content, m.projectPickerView()), m.confirmDeleteView())
		}
		return modal.Render(content, m.confirmDeleteView())
	case modes.ModeOptions:
		return modal.Render(content, m.optionsView())
	case modes.ModeProjectPicker:
		return modal.Render(content, m.projectPickerView())
	case modes.ModeFilter:
		return modal.Render(content, m.filterView())
	case modes.ModeInfo:
		return modal.Render(content, m.infoView())
	case modes.ModeChangelog:
		return modal.Render(content, m.changelogView())
	case modes.ModeEstimatePicker:
		return modal.Render(content, m.estimatePickerView())
	case modes.ModeURLPicker:
		return modal.Render(content, m.urlPickerView())
	case modes.ModeYankPicker:
		return modal.Render(content, m.yankPickerView())
	case modes.ModeDescriptionEdit:
		return modal.Render(content, m.descriptionEditView())
	case modes.ModeTagEdit:
		return modal.Render(content, m.tagEditView())
	case modes.ModePasteChooser:
		return modal.Render(content, m.pasteChooserView())
	case modes.ModeMovePicker:
		return modal.Render(content, m.movePickerView())
	}
	return content
}

func (m model) isConfirmingProjectDelete() bool {
	return m.ui.Modes.IsConfirmDelete() && m.ui.ConfirmDelete.Kind == ConfirmDeleteProject
}

func (m model) renderLayoutItem(item LayoutItem) string {
	isCursor := item.PositionIndex >= 0 && item.PositionIndex == m.selected()
	inVisualRange := item.PositionIndex >= 0 && m.isInVisualRange(item.PositionIndex)
	// A row participates in the selection-style band if it is the cursor OR
	// inside the visual range. The cursor is rendered distinctly within the
	// range so the user can see which end will extend on j/k.
	isSelected := isCursor || inVisualRange
	visualMode := m.ui.Modes.IsVisual()
	focused := m.ui.Screen.WindowFocused

	switch item.Kind {
	case LayoutProject:
		if m.editorOpen() && isCursor {
			return m.renderEditProjectLine()
		}
		return renderProjectLine(m.project.Name, isSelected, focused, m.ui.Filter.HasActiveFilter(), visualMode, m.statusText(), m.ui.Screen.Width)

	case LayoutCategory:
		indent := m.rowIndent(item.CategoryIndex, -1)
		if m.editorOpen() && isCursor {
			return m.renderEditCategoryLine(indent)
		}
		cut := m.categoryRowCut(m.project.Categories[item.CategoryIndex])
		header := headerOf(&m.project, item.CategoryIndex, &m.ui.Fold, cut, m.deps.CfgManager.Get().ShowCategoryCounts)
		return renderCategoryLine(header, isSelected, m.ui.Screen.Width, focused, inVisualRange, isCursor, m.searchQuery(), m.searchMatchStyle(isCursor))

	case LayoutTask:
		task := m.project.Categories[item.CategoryIndex].Tasks[item.TaskIndex]
		if m.editorOpen() && isCursor {
			return m.renderEditTaskLine(task, m.rowIndent(item.CategoryIndex, item.TaskIndex))
		}
		cut := m.taskRowCut(task)
		return m.renderTaskLine(item.CategoryIndex, item.TaskIndex, isSelected, m.ui.Screen.Width, focused, inVisualRange, isCursor, cut)

	case LayoutSeparator:
		indent := m.rowIndent(item.CategoryIndex, item.TaskIndex)
		if m.editorOpen() && isCursor {
			return m.renderEditSeparatorLine(indent)
		}
		sep := m.project.Categories[item.CategoryIndex].Tasks[item.TaskIndex]
		return m.renderSeparatorLine(sep.Title, indent, isSelected, focused, visualMode, isCursor, m.isTaskCut(sep.ID), m.ui.Screen.Width, m.searchQuery(), m.searchMatchStyle(isCursor))

	case LayoutDescription:
		task := m.project.Categories[item.CategoryIndex].Tasks[item.TaskIndex]
		cut := m.taskRowCut(task)
		rows := m.rowLimits().descriptionRows(item.CategoryIndex, item.TaskIndex, task.ID)
		return m.renderTaskDescription(item.CategoryIndex, item.TaskIndex, rows, isSelected, m.ui.Screen.Width, focused, visualMode, isCursor, cut)

	case LayoutEmptyCategory:
		return ui.MutedStyle.Render(placeholderIndent(m.rowIndent(item.CategoryIndex, -1)) + "(no tasks)")

	case LayoutSpacing:
		return strings.Repeat("\n", item.Height-1)
	}

	return ""
}

func placeholderIndent(indent int) string {
	return strings.Repeat(" ", prefixWidth+categoryFoldWidth+indent)
}

// renderLayoutItemWindow draws the item's rows from top on; maxRows <= 0 means
// every row it has.
func (m model) renderLayoutItemWindow(item LayoutItem, top, maxRows int) string {
	rendered := strings.Split(m.renderLayoutItem(item), "\n")
	if top >= len(rendered) {
		return ""
	}
	rendered = rendered[top:]
	if maxRows > 0 {
		rendered = rendered[:min(maxRows, len(rendered))]
	}
	return strings.Join(rendered, "\n")
}

// linkDirIfUnset establishes the directory→project link only when the directory
// has none yet, so a session-only project switch (--project, picker select)
// never overrides an existing link. Deliberate relinking (`project link`/`add`)
// calls SetProjectForDir directly.
func linkDirIfUnset(sm data.StateRepository, projectID string) {
	if sm.GetProjectForDir() == "" {
		_ = sm.SetProjectForDir(projectID)
	}
}

func Run(dataDir, projectSelector, taskSelector string, cfgManager config.Reader, workingDir string, forcePicker bool) error {
	taskCommands, err := bindTaskCommands(cfgManager.Get().TaskCommands)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	store := data.NewStore(dataDir)
	if err := store.Ensure(); err != nil {
		return err
	}

	// Persist off the event loop so no keystroke blocks on fsync. Deferring
	// Close here means the final edit is flushed on exit regardless of which
	// quit path the program took — program.Run only returns once the event loop
	// has stopped, so no Enqueue can race this Close.
	saver := data.NewSaver(store)
	defer saver.Close()

	stateManager := data.NewStateManager(filepath.Dir(dataDir), workingDir)
	if err := stateManager.Load(); err != nil {
		return err
	}

	// Records the focused row as it moves, so the cursor is already on disk when
	// the process dies without running any exit code. Close flushes the last
	// interval, which covers every ordinary quit path.
	cursorSaver := data.NewCursorSaver(stateManager)
	defer cursorSaver.Close()

	projects, err := store.ListProjects()
	if err != nil {
		return err
	}

	var project domain.Project
	startMode := modes.ModeNormal

	if len(projects) == 0 {
		project, err = store.InitDefault()
		if err != nil {
			return err
		}
		_ = stateManager.SetProjectForDir(project.ID)
		projects = []domain.Project{project}
	} else if projectSelector != "" {
		project, err = store.LoadProject(projectSelector)
		if err != nil {
			if errors.Is(err, data.ErrProjectNotFound) {
				return fmt.Errorf("project %q not found", projectSelector)
			}
			return err
		}
		// --project opens a project for this session only: link the directory
		// when it has no link yet, but never override an existing one.
		linkDirIfUnset(stateManager, project.ID)
	} else if linkedID := stateManager.GetProjectForDir(); linkedID != "" {
		project, err = store.LoadProject(linkedID)
		if err != nil {
			if errors.Is(err, data.ErrProjectNotFound) {
				startMode = modes.ModeProjectPicker
			} else {
				return err
			}
		} else {
			_ = stateManager.SetProjectForDir(project.ID)
		}
	} else {
		startMode = modes.ModeProjectPicker
	}

	if forcePicker {
		startMode = modes.ModeProjectPicker
	}

	foldState := loadFoldState(stateManager, project)
	expandDescriptions := cfgManager.Get().ExpandDescriptionsByDefault
	positions := rebuildPositions(project.Categories, nil, &foldState, expandDescriptions, "")
	initialSelection := findFirstTaskIndex(positions)
	selMgr := selection.NewManager(positions, initialSelection)
	modeMachine := modes.NewMachine(startMode)

	deps := NewDependencies(store, cfgManager, stateManager)
	deps.Saver = saver
	deps.CursorSaver = cursorSaver
	deps.setTaskCommands(taskCommands)
	m := model{
		project: project,
		ui:      NewUIState(selMgr, modeMachine),
		deps:    deps,
	}
	m.ui.Fold = foldState
	m.ui.Screen.ExpandDescriptions = expandDescriptions
	if startMode == modes.ModeNormal {
		m.announceConflicts("")
	}
	// Reopen on the row this project was last left on. Runs after the fold state
	// is in place so a cursor inside a folded category resolves to that
	// category's header rather than to a row that isn't rendered. The viewport
	// can only be centered on it once the terminal size arrives.
	m.applyStoredCursor()
	m.ui.Screen.PendingCenter = true
	if taskSelector != "" {
		if startMode == modes.ModeProjectPicker {
			return errors.New("--task needs a project: pass --project")
		}
		if err := m.focusTask(taskSelector); err != nil {
			return err
		}
	}

	if startMode == modes.ModeProjectPicker {
		ordered := orderProjects(projects, stateManager.GetProjectOrder())
		selected := 0
		for i, p := range ordered {
			if p.ID == project.ID {
				selected = i
				break
			}
		}
		m.ui.Picker = ProjectPickerState{
			projects:   ordered,
			selected:   selected,
			fullscreen: true,
		}
		m.ui.Picker.ensureVisible(m.pickerVisibleCount())
	}

	_, err = tea.NewProgram(m).Run()
	return err
}

func findFirstTaskIndex(positions []selection.Position) int {
	for i, pos := range positions {
		if pos.Kind == selection.FocusTask {
			return i
		}
	}
	if len(positions) > 0 {
		return 0
	}
	return -1
}
