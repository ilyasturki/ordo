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

// pickerRow is one line of the picker's list: a project, or the separator heading a group.
type pickerRow struct {
	project int // index into projects, -1 on a separator
	label   string
}

// pickerRows heads each group with a separator, the projects without one with a bare rule, and
// draws none while no project has a group.
func pickerRows(projects []domain.Project) []pickerRow {
	grouped := slices.ContainsFunc(projects, func(p domain.Project) bool { return p.Group != "" })
	rows := make([]pickerRow, 0, len(projects))
	for i, p := range projects {
		if grouped && (i == 0 || groupKey(projects[i-1]) != groupKey(p)) {
			rows = append(rows, pickerRow{project: -1, label: p.Group})
		}
		rows = append(rows, pickerRow{project: i})
	}
	return rows
}

func (p *ProjectPickerState) rowOf(project int) int {
	for r, row := range pickerRows(p.projects) {
		if row.project == project {
			return r
		}
	}
	return 0
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

func (m *model) startGrouping() {
	p := &m.ui.Picker
	if p.onNew || p.selected >= len(p.projects) {
		return
	}
	p.grouping = true
	p.groupFor = p.projects[p.selected].ID
	p.input = newTextInput()
	p.input.SetValue(p.projects[p.selected].Group)
	p.input.CursorEnd()
	p.input.Focus()
	p.groupCycling = false
}

func (p *ProjectPickerState) cancelGrouping() {
	p.grouping = false
	p.groupFor = ""
	p.input = lineField{}
}

func (p *ProjectPickerState) groupsLike(stem string) []string {
	var out []string
	for _, g := range projectGroups(p.projects) {
		if strings.HasPrefix(domain.NormalizeName(g), domain.NormalizeName(stem)) {
			out = append(out, g)
		}
	}
	return out
}

// completeGroup fills in a group that starts like what was typed, every group when it already
// names one; tab again cycles on.
func (p *ProjectPickerState) completeGroup() {
	value := p.input.Value()
	if !p.groupCycling {
		p.groupCycling, p.groupStem, p.groupNext = true, value, 0
		if slices.Contains(p.groupsLike(""), strings.TrimSpace(value)) {
			p.groupStem = ""
		}
		if i := slices.Index(p.groupsLike(p.groupStem), strings.TrimSpace(value)); i >= 0 {
			p.groupNext = i + 1
		}
	}
	matches := p.groupsLike(p.groupStem)
	if len(matches) == 0 {
		return
	}
	p.input.SetValue(matches[p.groupNext%len(matches)])
	p.input.CursorEnd()
	p.groupNext++
}

// groupHint is the rest of the group tab would fill in, drawn after the cursor.
func (p *ProjectPickerState) groupHint() string {
	value := p.input.Value()
	if p.groupCycling || p.input.Position() < len([]rune(value)) {
		return ""
	}
	for _, g := range p.groupsLike(value) {
		if len([]rune(g)) > len([]rune(value)) {
			return string([]rune(g)[len([]rune(value)):])
		}
	}
	return ""
}

func (m model) handlePickerGroupKey(msg tea.KeyPressMsg) (model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.setProjectGroup(m.ui.Picker.groupFor, m.ui.Picker.input.Value())
		return m, nil
	case "esc":
		m.ui.Picker.cancelGrouping()
		return m, nil
	case "tab":
		m.ui.Picker.completeGroup()
		return m, nil
	}
	var cmd tea.Cmd
	before := m.ui.Picker.input.Value()
	m.ui.Picker.input, cmd = m.ui.Picker.input.Update(msg)
	sanitizeInput(&m.ui.Picker.input)
	if m.ui.Picker.input.Value() != before {
		m.ui.Picker.groupCycling = false
	}
	return m, cmd
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
		if out, err = operations.Apply(&m.project, op); err == nil {
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

// reloadPickerProjects lists the projects again in order, the cursor on id.
func (m *model) reloadPickerProjects(id string) {
	projects, err := m.deps.Store.ListProjects()
	if err != nil {
		m.ui.Screen.StatusMsg = fmt.Sprintf("Error reloading projects: %v", err)
		return
	}
	p := &m.ui.Picker
	projects = orderProjects(projects, m.deps.StateManager.GetLastOpened())
	if p.filtering {
		p.allProjects = projects
		p.applyFilter(m.pickerVisibleCount())
	} else {
		p.projects = projects
	}
	if i := slices.IndexFunc(p.projects, func(pr domain.Project) bool { return pr.ID == id }); i >= 0 {
		p.onNew, p.selected = false, i
	}
	p.ensureVisible(m.pickerVisibleCount())
}

func (m model) renderPickerGroupLine() string {
	p := m.ui.Picker
	name := ""
	if i := slices.IndexFunc(p.projects, func(pr domain.Project) bool { return pr.ID == p.groupFor }); i >= 0 {
		name = p.projects[i].Name
	}
	split := splitAtCursor(p.input.Value(), p.input.Position())
	hint := p.groupHint()
	cursorCh := split.cursorCh
	if hint != "" && split.cursorCh == " " && split.right == "" {
		r := []rune(hint)
		cursorCh, hint = string(r[0]), string(r[1:])
	}
	return "  " + ui.MutedStyle.Render("Group of "+name+": ") + split.left +
		ui.GetCursorStyle(m.ui.Screen.WindowFocused).Render(cursorCh) + split.right + ui.MutedStyle.Render(hint)
}

func (m model) renderPickerSeparator(label string, width int) string {
	return m.renderSeparatorLine(label, 0, false, m.ui.Screen.WindowFocused, false, false, false, width, "", ui.SearchMatchStyle)
}
