package app

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"phasionary/internal/app/selection"
	"phasionary/internal/domain"
	"phasionary/internal/ui"
)

type moveTarget struct {
	categoryID string
	label      string
}

type MovePickerState struct {
	targets  []moveTarget
	selected int
	offset   int
	taskIDs  []string
	title    string
	// last outlives the picker, so a run of moves from the inbox keeps its target.
	last string
}

func (m *model) startMoveToVersion() {
	pos, ok := m.selectedPosition()
	if !ok || !visualTaskLevel(pos.Kind) {
		m.ui.Screen.StatusMsg = "Put the cursor on a task to move it"
		return
	}
	m.openMovePicker([]string{m.project.Categories[pos.CategoryIndex].Tasks[pos.TaskIndex].ID})
}

func (m *model) visualMoveToVersion() {
	if m.ui.Visual.Kind == selection.FocusCategory {
		m.ui.Screen.StatusMsg = "Select tasks to move them"
		return
	}
	rows := m.visualTaskRows(m.visualSelectedPositions())
	ids := make([]string, len(rows))
	for i, t := range rows {
		ids[i] = t.ID
	}
	m.exitVisualMode()
	m.openMovePicker(ids)
}

func (m *model) openMovePicker(ids []string) {
	if len(ids) == 0 {
		return
	}
	targets := m.moveTargets(m.sourceCategory(ids))
	if len(targets) == 0 {
		m.ui.Screen.StatusMsg = "No open version to move to: add one with A on a version"
		return
	}
	title := "Move to"
	if rows := m.movedRows(ids); len(rows) > 1 {
		title = "Move " + rowCountLabel(countTasks(rows), len(rows)-countTasks(rows)) + " to"
	}
	m.ui.MovePicker = MovePickerState{
		targets:  targets,
		selected: m.defaultMoveTarget(targets),
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

// moveTargets lists the inbox, then each open version followed by its sub-categories.
func (m *model) moveTargets(exclude string) []moveTarget {
	cats := m.project.Categories
	var out []moveTarget
	if i := m.project.InboxIndex(); i >= 0 && cats[i].ID != exclude {
		out = append(out, moveTarget{cats[i].ID, cats[i].Name})
	}
	for _, i := range m.project.VersionIndexes() {
		if cats[i].Shipped() {
			continue
		}
		listed := cats[i].ID != exclude
		if listed {
			out = append(out, moveTarget{cats[i].ID, cats[i].Name})
		}
		for j := i + 1; j < domain.BlockEnd(cats, i); j++ {
			if cats[j].ID == exclude {
				continue
			}
			label := m.project.CategoryPath(j)
			if listed {
				label = "  " + cats[j].Name
			}
			out = append(out, moveTarget{cats[j].ID, label})
		}
	}
	return out
}

// defaultMoveTarget is the last target picked, else the lowest open version: the next to ship.
func (m *model) defaultMoveTarget(targets []moveTarget) int {
	best, bestV := -1, domain.Semver{}
	for i, t := range targets {
		if t.categoryID == m.ui.MovePicker.last {
			return i
		}
		ci, err := m.project.FindCategoryByID(t.categoryID)
		if err != nil {
			continue
		}
		if v, ok := m.project.Categories[ci].Version(); ok && (best < 0 || v.Compare(bestV) < 0) {
			best, bestV = i, v
		}
	}
	return max(best, 0)
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

func countTasks(rows []domain.Task) int {
	n := 0
	for _, t := range rows {
		if !t.IsSeparator() {
			n++
		}
	}
	return n
}

func (m model) handleMovePickerKey(msg tea.KeyPressMsg) model {
	p := &m.ui.MovePicker
	switch msg.String() {
	case "q", "esc":
		m.ui.Modes.ToNormal()
	case "j", "down":
		p.selected = min(p.selected+1, len(p.targets)-1)
	case "k", "up":
		p.selected = max(p.selected-1, 0)
	case "g", "home":
		p.selected = 0
	case "G", "end":
		p.selected = len(p.targets) - 1
	case "enter":
		m.ui.Modes.ToNormal()
		if p.selected >= 0 && p.selected < len(p.targets) {
			m.moveToCategory(p.targets[p.selected].categoryID)
		}
		return m
	}
	p.ensureVisible(m.movePickerRows())
	return m
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
	m.ui.Screen.StatusMsg = strings.TrimSuffix(pastedRowsStatus(rows, true), "!") + " to " + m.project.CategoryPath(dst)
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

// movePickerRows leaves room for the dialog's border, padding, title and hints.
func (m model) movePickerRows() int {
	return max(3, m.ui.Screen.Height-12)
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
	width := safeWidth(m.ui.Screen.Width, 12)
	lines := []string{ui.DialogTitleStyle.Render(p.title), ""}
	end := min(len(p.targets), p.offset+m.movePickerRows())
	if p.offset > 0 {
		lines = append(lines, ui.MutedStyle.Render(fmt.Sprintf("  ↑ %d more", p.offset)))
	}
	for i := p.offset; i < end; i++ {
		line := ansi.Truncate(p.targets[i].label, width, "…")
		if i == p.selected {
			lines = append(lines, ui.SelectedStyle.Render("> "+line))
			continue
		}
		lines = append(lines, "  "+line)
	}
	if end < len(p.targets) {
		lines = append(lines, ui.MutedStyle.Render(fmt.Sprintf("  ↓ %d more", len(p.targets)-end)))
	}
	hints := []ui.Hint{{Key: "enter", Label: "move"}, {Key: "esc", Label: "cancel"}}
	lines = append(lines, "", ui.RenderHints(hints))
	return ui.HelpDialogStyle.Render(strings.Join(lines, "\n"))
}
