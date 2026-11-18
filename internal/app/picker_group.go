package app

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/ilyasturki/ordo/internal/domain"
	"github.com/ilyasturki/ordo/internal/operations"
	"github.com/ilyasturki/ordo/internal/ui"
)

// OrderProjects is the picker's order: by group, then by recency.
func OrderProjects(projects []domain.Project, lastOpened map[string]string) []domain.Project {
	return orderProjects(projects, lastOpened)
}

// ProjectGroups are the groups projects use, each once.
func ProjectGroups(projects []domain.Project) []string {
	return projectGroups(projects)
}

func groupKey(p domain.Project) string {
	return domain.NormalizeName(p.Group)
}

// groupProjects gathers projects ordered by recency into their groups: each group where its most
// recent project stands, the projects without a group last.
func groupProjects(ordered []domain.Project) []domain.Project {
	lead := make(map[string]int)
	for i, p := range ordered {
		if _, ok := lead[groupKey(p)]; !ok {
			lead[groupKey(p)] = i
		}
	}
	lead[""] = len(ordered)
	out := slices.Clone(ordered)
	slices.SortStableFunc(out, func(a, b domain.Project) int {
		return lead[groupKey(a)] - lead[groupKey(b)]
	})
	return out
}

type pickerRow struct {
	project int // index into projects, -1 on a separator
	label   string
}

// pickerRows heads each group with a separator, the projects without one with a bare rule, and
// draws none while no project has a group.
func pickerRows(projects []domain.Project) []pickerRow {
	grouped := slices.ContainsFunc(projects, func(p domain.Project) bool { return groupKey(p) != "" })
	rows := make([]pickerRow, 0, len(projects))
	for i, p := range projects {
		if grouped && (i == 0 || groupKey(projects[i-1]) != groupKey(p)) {
			rows = append(rows, pickerRow{project: -1, label: p.Group})
		}
		rows = append(rows, pickerRow{project: i})
	}
	return rows
}

// projectGroups are the groups in use, the most recent first, each spelt as its most recent project has it.
func projectGroups(projects []domain.Project) []string {
	var out []string
	seen := make(map[string]bool)
	for _, p := range projects {
		if k := groupKey(p); k != "" && !seen[k] {
			seen[k] = true
			out = append(out, p.Group)
		}
	}
	return out
}

// startGroupChoice opens the choice of a group for the selected project, on its current one.
func (m *model) startGroupChoice() {
	p := &m.ui.Picker
	if p.onNew || p.selected >= len(p.projects) {
		return
	}
	p.choosingGroup = true
	p.groupFor = p.projects[p.selected].ID
	current := groupKey(p.projects[p.selected])
	groups := projectGroups(p.projects)
	p.groupPick = slices.IndexFunc(groups, func(g string) bool { return domain.NormalizeName(g) == current })
	if p.groupPick < 0 {
		p.groupPick = len(groups)
	}
}

func (p *ProjectPickerState) groupProject() (domain.Project, bool) {
	i := slices.IndexFunc(p.projects, func(pr domain.Project) bool { return pr.ID == p.groupFor })
	if i < 0 {
		return domain.Project{}, false
	}
	return p.projects[i], true
}

// groupChoice lists the groups in use, then "No group" and "New group…".
func (m model) groupChoice() (ui.Choice, []string) {
	p := m.ui.Picker
	project, _ := p.groupProject()
	groups := projectGroups(p.projects)
	c := ui.Choice{Title: "Group of " + truncateText(project.Name, 40), Selected: p.groupPick, Numbered: true}
	for _, g := range groups {
		opt := ui.ChoiceOption{Label: g}
		if domain.NormalizeName(g) == groupKey(project) {
			opt.Detail = "current"
		}
		c.Options = append(c.Options, opt)
	}
	none := ui.ChoiceOption{Label: "No group", Key: "n"}
	if groupKey(project) == "" {
		none.Detail = "current"
	}
	c.Options = append(c.Options, none, ui.ChoiceOption{Label: "New group…", Key: "w"})
	return c, groups
}

func (m model) handleGroupChoiceKey(msg tea.KeyPressMsg) (model, tea.Cmd) {
	c, groups := m.groupChoice()
	act, i := c.Handle(msg.String())
	p := &m.ui.Picker
	p.groupPick = c.Selected
	switch {
	case act == ui.ChoiceCancelled:
		p.cancelGrouping()
	case act != ui.ChoicePicked:
	case i < len(groups):
		m.setProjectGroup(p.groupFor, groups[i])
	case i == len(groups):
		m.setProjectGroup(p.groupFor, "")
	default:
		return m, m.startNewGroup()
	}
	return m, nil
}

func (m *model) startNewGroup() tea.Cmd {
	p := &m.ui.Picker
	p.choosingGroup = false
	p.grouping = true
	p.input = newTextInput()
	return p.input.Focus()
}

func (p *ProjectPickerState) cancelGrouping() {
	p.choosingGroup = false
	p.grouping = false
	p.groupFor = ""
	p.input = lineField{}
}

// A name field: esc saves like enter, and an empty name changes nothing.
func (m model) handlePickerGroupKey(msg tea.KeyPressMsg) (model, tea.Cmd) {
	switch msg.String() {
	case "enter", "esc":
		if name := strings.TrimSpace(m.ui.Picker.input.Value()); name != "" {
			m.setProjectGroup(m.ui.Picker.groupFor, name)
		} else {
			m.ui.Picker.cancelGrouping()
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.ui.Picker.input, cmd = m.ui.Picker.input.Update(msg)
	sanitizeInput(&m.ui.Picker.input)
	return m, cmd
}

func (m model) groupChoiceView() string {
	c, _ := m.groupChoice()
	return m.choiceView(c)
}

// setProjectGroup saves the group straight to the file, so the save keeps updated_at; the open
// project takes that file back, unless it holds edits a save must carry first.
func (m *model) setProjectGroup(id, value string) {
	m.ui.Picker.cancelGrouping()
	op := operations.Op{Action: "project.group", Value: value}
	var out operations.Outcome
	var err error
	if id == m.project.ID && m.holdsUnsaved() {
		if m.ui.Modes.IsConflict() {
			return
		}
		mine := cloneProject(m.project)
		if out, err = operations.Apply(&m.project, op); err == nil {
			m.ui.History.rebase(mine, m.project)
			m.storeTaskUpdate()
		}
	} else {
		var saved domain.Project
		saved, err = m.deps.Store.WithProjectLocked(id, func(p *domain.Project) error {
			var applyErr error
			out, applyErr = operations.Apply(p, op)
			return applyErr
		})
		if err == nil && id == m.project.ID {
			m.ui.History.rebase(m.project, saved)
			m.showProject(saved)
		}
	}
	switch {
	case errors.Is(err, operations.ErrNoChange):
		return
	case err != nil:
		msg, ok := operations.Explain(err)
		if !ok {
			msg = fmt.Sprintf("Error: %v", err)
		}
		m.ui.Screen.StatusMsg = msg
		return
	}
	m.ui.Screen.StatusMsg = out.Msg
	m.reloadPickerProjects(id)
}

func (m *model) reloadPickerProjects(id string) {
	projects, err := m.deps.Store.ListProjects()
	if err != nil {
		m.ui.Screen.StatusMsg = fmt.Sprintf("Error reloading projects: %v", err)
		return
	}
	for i := range projects {
		if projects[i].ID == m.project.ID {
			projects[i].Group = m.project.Group
		}
	}
	p := &m.ui.Picker
	p.projects = orderProjects(projects, m.deps.StateManager.GetLastOpened())
	if i := slices.IndexFunc(p.projects, func(pr domain.Project) bool { return pr.ID == id }); i >= 0 {
		p.onNew, p.selected = false, i
	}
	p.ensureVisible(m.pickerVisibleCount())
}

func (m model) renderPickerGroupLine() string {
	p := m.ui.Picker
	project, _ := p.groupProject()
	split := splitAtCursor(p.input.Value(), p.input.Position())
	return "  " + ui.MutedStyle.Render("New group of "+project.Name+": ") + split.left +
		ui.GetCursorStyle(m.ui.Screen.WindowFocused).Render(split.cursorCh) + split.right
}

func (m model) renderPickerSeparator(label string, width int) string {
	return m.renderSeparatorLine(domain.Task{Title: label, Kind: domain.KindSeparator}, 0, false, m.ui.Screen.WindowFocused, false, false, false, width, "", ui.SearchMatchStyle)
}
