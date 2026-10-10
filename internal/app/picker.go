package app

import (
	"fmt"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"phasionary/internal/domain"
	"phasionary/internal/ui"
)

// pickerScrollOff keeps a small margin of context rows between the cursor and
// the top/bottom edges of the scrolled window, like vim's scrolloff. It's
// clamped to the window size, so it disengages near the list ends.
const pickerScrollOff = 1

func (m *model) openProjectPicker() {
	projects, err := m.deps.Store.ListProjects()
	if err != nil {
		m.ui.Screen.StatusMsg = fmt.Sprintf("Error loading projects: %v", err)
		return
	}

	order := m.deps.StateManager.GetProjectOrder()
	projects = orderProjects(projects, order)

	currentIdx := 0
	for i, p := range projects {
		if p.ID == m.project.ID {
			currentIdx = i
			break
		}
	}

	m.ui.Picker = ProjectPickerState{
		projects:     projects,
		selected:     currentIdx,
		onNew:        len(projects) == 0,
		scrollOffset: 0,
	}
	m.ui.Picker.ensureVisible(m.pickerVisibleCount())
	m.ui.Modes.ToProjectPicker()
}

func orderProjects(projects []domain.Project, order []string) []domain.Project {
	if len(order) == 0 {
		return projects
	}

	projectMap := make(map[string]domain.Project)
	for _, p := range projects {
		projectMap[p.ID] = p
	}

	var ordered []domain.Project
	seen := make(map[string]bool)
	for _, id := range order {
		if p, ok := projectMap[id]; ok {
			ordered = append(ordered, p)
			seen[id] = true
		}
	}

	var remaining []domain.Project
	for _, p := range projects {
		if !seen[p.ID] {
			remaining = append(remaining, p)
		}
	}
	sort.Slice(remaining, func(i, j int) bool {
		return strings.ToLower(remaining[i].Name) < strings.ToLower(remaining[j].Name)
	})

	return append(ordered, remaining...)
}

func (m model) handleProjectPickerKey(msg tea.KeyPressMsg) (model, tea.Cmd) {
	if m.ui.Picker.isAdding {
		return m.handlePickerAddKey(msg)
	}
	if m.ui.Picker.filtering {
		return m.handlePickerFilterKey(msg)
	}
	visible := m.pickerVisibleCount()
	switch msg.String() {
	case "/":
		if len(m.ui.Picker.projects) > 0 {
			m.ui.Picker.startFiltering()
		}
	case "a":
		m.ui.Picker.startAdding()
	case "j", "down":
		m.ui.Picker.moveSelection(1, visible)
	case "k", "up":
		m.ui.Picker.moveSelection(-1, visible)
	case "ctrl+d":
		m.ui.Picker.moveSelection(halfPage(visible), visible)
	case "ctrl+u":
		m.ui.Picker.moveSelection(-halfPage(visible), visible)
	case "ctrl+f", "pgdown":
		m.ui.Picker.moveSelection(visible, visible)
	case "ctrl+b", "pgup":
		m.ui.Picker.moveSelection(-visible, visible)
	case "g", "home":
		m.ui.Picker.jumpToFirst(visible)
	case "G", "end":
		m.ui.Picker.jumpToLast(visible)
	case "J":
		m.moveProjectDown()
	case "K":
		m.moveProjectUp()
	case "enter":
		if m.ui.Picker.isOnNewProject() {
			m.ui.Picker.startAdding()
		} else {
			m.selectProject()
		}
	case "d":
		m.initiateProjectDelete()
	case "esc", "q":
		if m.project.ID == "" {
			return m, tea.Quit
		}
		m.ui.Picker.reset()
		m.ui.Modes.ToNormal()
	}
	return m, nil
}

func (m *model) initiateProjectDelete() {
	if m.ui.Picker.isOnNewProject() {
		return
	}
	if len(m.ui.Picker.projects) <= 1 {
		m.ui.Screen.StatusMsg = "Cannot delete the only project"
		return
	}
	selectedProject := m.ui.Picker.projects[m.ui.Picker.selected]
	m.ui.ConfirmDelete = ConfirmDeleteState{
		Kind:      ConfirmDeleteProject,
		ProjectID: selectedProject.ID,
	}
	m.ui.Modes.ToConfirmDelete()
}

func (m *model) confirmDeleteProject() {
	deleteID := m.ui.ConfirmDelete.ProjectID
	if deleteID == "" {
		return
	}

	var deletedProjectName string
	for _, p := range m.ui.Picker.projects {
		if p.ID == deleteID {
			deletedProjectName = p.Name
			break
		}
	}

	if err := m.deps.Store.DeleteProject(deleteID); err != nil {
		m.ui.Screen.StatusMsg = fmt.Sprintf("Error deleting project: %v", err)
		m.ui.ConfirmDelete.reset()
		m.ui.Modes.ToProjectPicker()
		return
	}

	m.removeProjectFromOrder(deleteID)
	_ = m.deps.StateManager.DeleteFoldedCategories(deleteID)
	_ = m.deps.StateManager.DeleteCursor(deleteID)
	m.forgetPendingCursor(deleteID)

	if m.project.ID == deleteID {
		projects, err := m.deps.Store.ListProjects()
		if err != nil {
			m.ui.Screen.StatusMsg = fmt.Sprintf("Error loading projects: %v", err)
			m.ui.ConfirmDelete.reset()
			m.ui.Modes.ToProjectPicker()
			return
		}
		if len(projects) > 0 {
			m.project = projects[0]
			_ = m.deps.StateManager.SetProjectForDir(m.project.ID)
			m.ui.Filter = NewFilterState()
			m.ui.Fold = NewFoldStateFrom(m.deps.StateManager.GetFoldedCategories(m.project.ID))
			m.ui.History.Reset()
			m.rebuildPositions()
			m.applyStoredCursor()
			// Reset first: centering is a no-op on an empty project, which would
			// otherwise inherit the deleted project's scroll offset.
			m.ui.Screen.TopRow = 0
			m.centerOnSelected()
		}
	}

	projects, err := m.deps.Store.ListProjects()
	if err != nil {
		m.ui.Screen.StatusMsg = fmt.Sprintf("Error reloading projects: %v", err)
	} else {
		projects = orderProjects(projects, m.deps.StateManager.GetProjectOrder())
		m.ui.Picker.projects = projects
		if m.ui.Picker.selected >= len(projects) {
			m.ui.Picker.selected = len(projects) - 1
		}
		if m.ui.Picker.selected < 0 {
			m.ui.Picker.selected = 0
		}
		m.ui.Picker.ensureVisible(m.pickerVisibleCount())
	}

	m.ui.Screen.StatusMsg = fmt.Sprintf("Deleted project: %s", deletedProjectName)
	m.ui.ConfirmDelete.reset()
	m.ui.Modes.ToProjectPicker()
}

func (m model) handlePickerAddKey(msg tea.KeyPressMsg) (model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.createProjectFromPicker()
		return m, nil
	case "esc":
		m.ui.Picker.cancelAdding()
		return m, nil
	}
	var cmd tea.Cmd
	m.ui.Picker.input, cmd = m.ui.Picker.input.Update(msg)
	sanitizeInput(&m.ui.Picker.input)
	return m, cmd
}

// handlePickerFilterKey drives the type-to-filter sub-mode. Reorder (J/K) and
// delete (d) aren't handled here — their keys are filter text — so those actions
// stay operating on the full, unfiltered list.
func (m model) handlePickerFilterKey(msg tea.KeyPressMsg) (model, tea.Cmd) {
	visible := m.pickerVisibleCount()
	switch msg.String() {
	case "esc":
		m.ui.Picker.cancelFiltering(visible)
		return m, nil
	case "enter":
		if len(m.ui.Picker.projects) > 0 {
			m.selectProject()
		}
		return m, nil
	case "down", "ctrl+n":
		m.ui.Picker.moveWithinProjects(1, visible)
		return m, nil
	case "up", "ctrl+p":
		m.ui.Picker.moveWithinProjects(-1, visible)
		return m, nil
	}
	var cmd tea.Cmd
	m.ui.Picker.filter, cmd = m.ui.Picker.filter.Update(msg)
	sanitizeInput(&m.ui.Picker.filter)
	m.ui.Picker.query = m.ui.Picker.filter.Value()
	m.ui.Picker.applyFilter(visible)
	return m, cmd
}

func (m *model) createProjectFromPicker() {
	name := strings.TrimSpace(m.ui.Picker.input.Value())
	if name == "" {
		m.ui.Picker.cancelAdding()
		return
	}

	project, err := m.deps.Store.CreateProject(name)
	if err != nil {
		m.ui.Screen.StatusMsg = fmt.Sprintf("Error: %v", err)
		return
	}

	_ = m.deps.StateManager.SetProjectForDir(project.ID)
	order := m.deps.StateManager.GetProjectOrder()
	order = append(order, project.ID)
	_ = m.deps.StateManager.SetProjectOrder(order)

	// The project we are leaving keeps its cursor for the next time it is opened.
	m.saveCursorState()

	m.project = project
	m.ui.Filter = NewFilterState()
	m.ui.Fold = NewFoldState()
	m.ui.History.Reset()
	m.rebuildPositions()
	m.ui.Selection.SetSelected(findFirstTaskIndex(m.ui.Selection.Positions()))
	m.ui.Screen.TopRow = 0

	m.ensureVisible()
	m.ui.Screen.StatusMsg = fmt.Sprintf("Created project: %s", project.Name)
	m.ui.Picker.reset()
	m.ui.Modes.ToNormal()
}

func (m *model) selectProject() {
	if m.ui.Picker.selected < 0 || m.ui.Picker.selected >= len(m.ui.Picker.projects) {
		m.ui.Picker.reset()
		m.ui.Modes.ToNormal()
		return
	}

	selectedProject := m.ui.Picker.projects[m.ui.Picker.selected]
	if selectedProject.ID == m.project.ID {
		m.ui.Picker.reset()
		m.ui.Modes.ToNormal()
		return
	}

	project, err := m.deps.Store.LoadProject(selectedProject.ID)
	if err != nil {
		m.ui.Screen.StatusMsg = fmt.Sprintf("Error loading project: %v", err)
		m.ui.Picker.reset()
		m.ui.Modes.ToNormal()
		return
	}

	// Switching to an existing project is a transient, session-only view change:
	// only establish a directory link when none exists yet, never override one.
	// The link is changed deliberately via `project link`/`add`.
	linkDirIfUnset(m.deps.StateManager, project.ID)

	// Hand the outgoing project its cursor back before adopting the incoming
	// project's, so switching away and back returns to the same row.
	m.saveCursorState()

	m.project = project
	m.ui.Filter = NewFilterState()
	m.ui.Fold = NewFoldStateFrom(m.deps.StateManager.GetFoldedCategories(project.ID))
	m.ui.History.Reset()
	m.rebuildPositions()
	m.applyStoredCursor()
	// Reset first: centering is a no-op on an empty project, which would
	// otherwise inherit the previous project's scroll offset.
	m.ui.Screen.TopRow = 0
	// Center rather than ensureVisible — reopening a project carries no
	// information about where the view sat, so a restored row deep in the list
	// would otherwise arrive pinned to the bottom edge.
	m.centerOnSelected()

	m.ui.Screen.StatusMsg = fmt.Sprintf("Switched to: %s", project.Name)
	if notice := conflictNotice(m.deps.Store, project.ID); notice != "" {
		m.ui.Screen.StatusMsg = notice
	}
	m.ui.Picker.reset()
	m.ui.Modes.ToNormal()
}

// halfPage is the slot delta for a half-page jump, never less than one row.
func halfPage(visible int) int {
	if visible < 2 {
		return 1
	}
	return visible / 2
}

// moveSelection moves the cursor by delta over the flattened list (New Project
// then projects), so j/k and paging cross the New Project boundary uniformly.
func (p *ProjectPickerState) moveSelection(delta, visible int) {
	p.setVirtual(p.virtualIndex() + delta)
	p.ensureVisible(visible)
}

// moveWithinProjects steps the cursor over the projects only, never the pinned
// New Project row — used while filtering, where that row is hidden.
func (p *ProjectPickerState) moveWithinProjects(delta, visible int) {
	if len(p.projects) == 0 {
		return
	}
	p.onNew = false
	p.selected = max(0, min(p.selected+delta, len(p.projects)-1))
	p.ensureVisible(visible)
}

// applyFilter recomputes the displayed projects from the snapshot in allProjects
// and the current query (smartcase substring match on name), resetting the
// cursor to the first match. An empty query restores the full list.
func (p *ProjectPickerState) applyFilter(visible int) {
	if p.query == "" {
		p.projects = p.allProjects
	} else {
		filtered := make([]domain.Project, 0, len(p.allProjects))
		for _, pr := range p.allProjects {
			if ui.Contains(pr.Name, p.query) {
				filtered = append(filtered, pr)
			}
		}
		p.projects = filtered
	}
	p.onNew = false
	p.selected = 0
	p.scrollOffset = 0
	p.ensureVisible(visible)
}

// jumpToFirst/jumpToLast target the first/last project (not the pinned New
// Project row, which is reached by arrowing up past the top), mirroring g/G on
// the main list.
func (p *ProjectPickerState) jumpToFirst(visible int) {
	p.setVirtual(1)
	p.ensureVisible(visible)
}

func (p *ProjectPickerState) jumpToLast(visible int) {
	p.setVirtual(len(p.projects))
	p.ensureVisible(visible)
}

// ensureVisible scrolls the projects window so the selected project stays
// visible, keeping a pickerScrollOff margin of context rows from the edges
// where the list is long enough to allow it. New Project is pinned outside the
// window, so selecting it leaves the scroll position untouched (just clamped).
func (p *ProjectPickerState) ensureVisible(visible int) {
	if visible < 1 {
		visible = 1
	}
	n := len(p.projects)

	if !p.onNew {
		scrollOff := pickerScrollOff
		if maxOff := (visible - 1) / 2; scrollOff > maxOff {
			scrollOff = maxOff
		}

		if p.selected < p.scrollOffset+scrollOff {
			p.scrollOffset = p.selected - scrollOff
		}
		if p.selected > p.scrollOffset+visible-1-scrollOff {
			p.scrollOffset = p.selected - visible + 1 + scrollOff
		}
	}

	maxOffset := n - visible
	if maxOffset < 0 {
		maxOffset = 0
	}
	if p.scrollOffset > maxOffset {
		p.scrollOffset = maxOffset
	}
	if p.scrollOffset < 0 {
		p.scrollOffset = 0
	}
}

func (m *model) moveProjectDown() {
	if m.ui.Picker.isOnNewProject() || m.ui.Picker.selected >= len(m.ui.Picker.projects)-1 {
		return
	}
	idx := m.ui.Picker.selected
	m.ui.Picker.projects[idx], m.ui.Picker.projects[idx+1] =
		m.ui.Picker.projects[idx+1], m.ui.Picker.projects[idx]
	m.ui.Picker.moveSelection(1, m.pickerVisibleCount())
	m.saveProjectOrder()
}

func (m *model) moveProjectUp() {
	if m.ui.Picker.isOnNewProject() || m.ui.Picker.selected <= 0 {
		return
	}
	idx := m.ui.Picker.selected
	m.ui.Picker.projects[idx], m.ui.Picker.projects[idx-1] =
		m.ui.Picker.projects[idx-1], m.ui.Picker.projects[idx]
	m.ui.Picker.moveSelection(-1, m.pickerVisibleCount())
	m.saveProjectOrder()
}

func (m *model) saveProjectOrder() {
	order := make([]string, len(m.ui.Picker.projects))
	for i, p := range m.ui.Picker.projects {
		order[i] = p.ID
	}
	_ = m.deps.StateManager.SetProjectOrder(order)
}

func (m *model) removeProjectFromOrder(id string) {
	order := m.deps.StateManager.GetProjectOrder()
	var newOrder []string
	for _, pid := range order {
		if pid != id {
			newOrder = append(newOrder, pid)
		}
	}
	_ = m.deps.StateManager.SetProjectOrder(newOrder)
}
