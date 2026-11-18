package app

import (
	"fmt"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/ilyasturki/ordo/internal/app/selection"
	"github.com/ilyasturki/ordo/internal/domain"
	"github.com/ilyasturki/ordo/internal/ui"
)

const noMoveTarget = "No other category to move to"

type moveTarget struct {
	categoryID string
	label      string
	path       string
}

type MovePickerState struct {
	all       []moveTarget
	targets   []moveTarget
	selected  int
	offset    int
	taskIDs   []string
	title     string
	filtering bool
	filter    lineField
	query     string
	// last outlives the picker, so a run of moves from one category keeps its target.
	last string
}

func (m *model) startMove() {
	pos, ok := m.selectedPosition()
	if !ok || !visualTaskLevel(pos.Kind) {
		m.ui.Screen.StatusMsg = "Put the cursor on a task to move it"
		return
	}
	m.openMovePicker([]string{m.project.Categories[pos.CategoryIndex].Tasks[pos.TaskIndex].ID})
}

func (m *model) visualMove() {
	if m.ui.Visual.Kind == selection.FocusCategory {
		m.ui.Screen.StatusMsg = "Select tasks to move them"
		return
	}
	rows := m.visualTaskRows(m.visualSelectedPositions())
	ids := make([]string, len(rows))
	for i, t := range rows {
		ids[i] = t.ID
	}
	if len(ids) == 0 || len(m.moveTargets(m.sourceCategory(ids))) == 0 {
		m.ui.Screen.StatusMsg = noMoveTarget
		return
	}
	m.exitVisualMode()
	m.openMovePicker(ids)
}

func (m *model) openMovePicker(ids []string) {
	if len(ids) == 0 {
		return
	}
	source := m.sourceCategory(ids)
	targets := m.moveTargets(source)
	if len(targets) == 0 {
		m.ui.Screen.StatusMsg = noMoveTarget
		return
	}
	title := "Move to"
	if rows := m.movedRows(ids); len(rows) > 1 {
		title = "Move " + rowsLabel(rows) + " to"
	}
	m.ui.MovePicker = MovePickerState{
		all:      targets,
		targets:  targets,
		selected: m.defaultMoveTarget(targets, ids),
		taskIDs:  ids,
		title:    title,
		last:     m.ui.MovePicker.last,
	}
	m.ui.MovePicker.ensureVisible(m.movePickerRows())
	m.ui.Modes.ToMovePicker()
}

// sourceCategory is the one category every row comes from, or "" when they span several.
func (m *model) sourceCategory(ids []string) string {
	source := ""
	for _, id := range ids {
		ci, _ := m.findTaskByID(id)
		if ci < 0 {
			continue
		}
		cid := m.project.Categories[ci].ID
		if source != "" && source != cid {
			return ""
		}
		source = cid
	}
	return source
}

// moveTargets lists every category but the source in project order; a sub-category whose parent is the source goes by its path.
func (m *model) moveTargets(exclude string) []moveTarget {
	cats := m.project.Categories
	var out []moveTarget
	for i, c := range cats {
		if c.ID == exclude {
			continue
		}
		path := m.project.CategoryPath(i)
		label := c.Name
		if c.ParentID != "" {
			label = strings.Repeat(" ", nestIndent) + c.Name
			if c.ParentID == exclude {
				label = path
			}
		}
		out = append(out, moveTarget{categoryID: c.ID, label: label, path: path})
	}
	return out
}

// defaultMoveTarget is the last target picked, else the category right after the rows' own, else the one before it.
func (m *model) defaultMoveTarget(targets []moveTarget, ids []string) int {
	for i, t := range targets {
		if t.categoryID == m.ui.MovePicker.last {
			return i
		}
	}
	from, _ := m.findTaskByID(ids[0])
	for i, t := range targets {
		if ci, err := m.project.FindCategoryByID(t.categoryID); err == nil && ci > from {
			return i
		}
	}
	return len(targets) - 1
}

func (m *model) movedRows(ids []string) []domain.Task {
	picked := make(map[string]bool, len(ids))
	for _, id := range ids {
		picked[id] = true
	}
	var rows []domain.Task
	for _, cat := range m.project.Categories {
		rows = append(rows, domain.PickBlocks(cat.Tasks, picked)...)
	}
	return rows
}

func (m *model) anyRowShown(ids []string) bool {
	for _, p := range m.positions() {
		if taggable(p.Kind) && slices.Contains(ids, p.ID) {
			return true
		}
	}
	return false
}

func rowIDsOf(rows []domain.Task) []string {
	ids := make([]string, len(rows))
	for i, t := range rows {
		ids[i] = t.ID
	}
	return ids
}

func countTasks(rows []domain.Task) int {
	n := 0
	for _, t := range rows {
		if t.IsTask() {
			n++
		}
	}
	return n
}

func (m model) handleMovePickerKey(msg tea.KeyPressMsg) (model, tea.Cmd) {
	p := &m.ui.MovePicker
	if p.filtering {
		return m.handleMovePickerFilterKey(msg)
	}
	if msg.String() == "/" {
		p.startFiltering()
		p.ensureVisible(m.movePickerRows())
		return m, nil
	}
	c := m.moveChoice()
	act, _ := c.Handle(msg.String())
	p.selected = c.Selected
	switch act {
	case ui.ChoiceCancelled:
		m.ui.Modes.ToNormal()
	case ui.ChoicePicked:
		m.pickMoveTarget()
		return m, nil
	}
	p.ensureVisible(m.movePickerRows())
	return m, nil
}

// moveChoice numbers the targets while no filter is typed: digits go to the filter.
func (m model) moveChoice() ui.Choice {
	p := m.ui.MovePicker
	c := ui.Choice{Title: p.title, Selected: p.selected, Numbered: !p.filtering}
	for _, t := range p.targets {
		opt := ui.ChoiceOption{Label: t.label}
		if p.query != "" {
			opt.Label, opt.Match = t.path, p.query
		}
		c.Options = append(c.Options, opt)
	}
	return c
}

func (m model) handleMovePickerFilterKey(msg tea.KeyPressMsg) (model, tea.Cmd) {
	p := &m.ui.MovePicker
	switch msg.String() {
	case "esc":
		p.cancelFiltering()
	case "enter":
		m.pickMoveTarget()
		return m, nil
	case "down", "ctrl+n":
		p.selected = max(0, min(p.selected+1, len(p.targets)-1))
	case "up", "ctrl+p":
		p.selected = max(p.selected-1, 0)
	default:
		var cmd tea.Cmd
		p.filter, cmd = p.filter.Update(msg)
		sanitizeInput(&p.filter)
		p.query = p.filter.Value()
		p.applyFilter()
		p.ensureVisible(m.movePickerRows())
		return m, cmd
	}
	p.ensureVisible(m.movePickerRows())
	return m, nil
}

func (m *model) pickMoveTarget() {
	p := m.ui.MovePicker
	if p.selected < 0 || p.selected >= len(p.targets) {
		return
	}
	m.ui.Modes.ToNormal()
	m.moveToCategory(p.targets[p.selected].categoryID)
}

func (p *MovePickerState) startFiltering() {
	p.filtering = true
	p.query = ""
	p.filter = newTextInput()
	p.filter.Focus()
}

// cancelFiltering restores the whole list and keeps the highlighted target.
func (p *MovePickerState) cancelFiltering() {
	keep := ""
	if p.selected >= 0 && p.selected < len(p.targets) {
		keep = p.targets[p.selected].categoryID
	}
	p.filtering = false
	p.filter = lineField{}
	p.query = ""
	p.targets = p.all
	for i, t := range p.targets {
		if t.categoryID == keep {
			p.selected = i
		}
	}
}

func (p *MovePickerState) applyFilter() {
	p.targets = p.all
	if p.query != "" {
		p.targets = nil
		for _, t := range p.all {
			if ui.Contains(t.path, p.query) {
				p.targets = append(p.targets, t)
			}
		}
	}
	p.selected, p.offset = 0, 0
}

// moveToCategory leaves the cursor where the rows were, so a run of moves can triage a list.
func (m *model) moveToCategory(categoryID string) {
	dst, err := m.project.FindCategoryByID(categoryID)
	if err != nil {
		m.ui.Screen.StatusMsg = "That category is gone"
		return
	}
	ids := m.ui.MovePicker.taskIDs
	moving := make(map[string]bool)
	for _, t := range m.movedRows(ids) {
		moving[t.ID] = true
	}
	stay := m.stayAnchor(moving)
	m.recordHistory()
	rows := m.moveRows(ids, dst, len(m.project.Categories[dst].Tasks), "")
	if len(rows) == 0 {
		m.discardLastHistory()
		m.ui.Screen.StatusMsg = "Nothing to move: the rows are gone"
		return
	}
	m.ui.MovePicker.last = categoryID
	m.rebuildPositions()
	m.selectItem(stay.Kind, stay.ID)
	m.ensureVisible()
	m.storeTaskUpdate()
	status := strings.TrimSuffix(pastedRowsStatus(rows, true), "!") + " to " + m.project.CategoryPath(dst)
	if m.ui.Fold.IsFolded(categoryID) {
		status += " (folded)"
	} else if !m.anyRowShown(rowIDsOf(rows)) {
		status += " (hidden by the filter)"
	}
	m.ui.Screen.StatusMsg = status
}

// stayAnchor is the next row of the cursor's category the move leaves behind, else the row above.
func (m *model) stayAnchor(moving map[string]bool) selection.Position {
	positions := m.ui.Selection.Positions()
	cur := m.selected()
	if cur < 0 || cur >= len(positions) {
		return selection.Position{Kind: selection.FocusProject}
	}
	gone := func(p selection.Position) bool {
		return p.Kind == selection.FocusDescription || (visualTaskLevel(p.Kind) && moving[p.ID])
	}
	for i := cur + 1; i < len(positions); i++ {
		p := positions[i]
		if !visualTaskLevel(p.Kind) || p.CategoryIndex != positions[cur].CategoryIndex {
			break
		}
		if !gone(p) {
			return p
		}
	}
	for i := cur - 1; i >= 0; i-- {
		if !gone(positions[i]) {
			return positions[i]
		}
	}
	return selection.Position{Kind: selection.FocusProject}
}

// movePickerRows leaves room for the dialog's border, padding, title, filter prompt and the ↑↓ more rows.
func (m model) movePickerRows() int {
	rows := m.ui.Screen.Height - 10
	if m.ui.MovePicker.filtering {
		rows--
	}
	return max(3, rows)
}

func (p *MovePickerState) ensureVisible(rows int) {
	if p.selected < p.offset {
		p.offset = p.selected
	}
	if p.selected >= p.offset+rows {
		p.offset = p.selected - rows + 1
	}
	p.offset = max(0, min(p.offset, len(p.targets)-rows))
}

func (m model) movePickerView() string {
	p := m.ui.MovePicker
	width := m.dialogWidth()
	c := m.moveChoice()
	lines := c.Head(width)
	if p.filtering {
		lines = append(lines, m.filterPromptRow(p.filter, p.query, len(p.targets), width))
	}
	end := min(len(p.targets), p.offset+m.movePickerRows())
	if p.offset > 0 {
		lines = append(lines, ui.MutedStyle.Render(fmt.Sprintf("  ↑ %d more", p.offset)))
	}
	for i := p.offset; i < end; i++ {
		lines = append(lines, c.Row(i, width, m.ui.Screen.WindowFocused))
	}
	if end < len(p.targets) {
		lines = append(lines, ui.MutedStyle.Render(fmt.Sprintf("  ↓ %d more", len(p.targets)-end)))
	}
	return m.dialogStyle().Render(strings.Join(lines, "\n"))
}
