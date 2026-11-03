package app

import (
	"fmt"
	"strings"

	"github.com/ilyasturki/ordo/internal/app/selection"
	"github.com/ilyasturki/ordo/internal/operations"
	"github.com/ilyasturki/ordo/internal/ui"
)

type clashChoice int

const (
	clashRenameThis clashChoice = iota
	clashRenameOther
	clashMerge
)

type clashOption struct {
	key    string
	choice clashChoice
	label  string
}

// clashNames are the names the chooser offers: this one's with a free " (n)", and the other's.
func (m model) clashNames() (this, other string, ok bool) {
	src, _, ok := m.mergeOffer()
	if !ok {
		return "", "", false
	}
	p := m.project
	clash := operations.CategoryClash(&p, operations.Op{Action: "category.rename", Cat: p.Categories[src].ID, Title: m.ui.Edit.input.Value()})
	if clash == nil {
		return "", "", false
	}
	return clash.This, clash.Other, true
}

func (m model) clashOptions() []clashOption {
	this, other, ok := m.clashNames()
	if !ok {
		return nil
	}
	_, dst, _ := m.mergeOffer()
	taken := truncateText(m.project.Categories[dst].Name, 30)
	which := "this one"
	if m.ui.Edit.isAdding {
		which = "the new one"
	}
	return []clashOption{
		{"t", clashRenameThis, fmt.Sprintf("name %s “%s”", which, truncateText(this, 30))},
		{"o", clashRenameOther, fmt.Sprintf("rename the other to “%s”", truncateText(other, 30))},
		{"m", clashMerge, fmt.Sprintf("merge into “%s”", taken)},
	}
}

func (m *model) handleClashKey(key string) {
	options := m.clashOptions()
	switch key {
	case "esc", "n":
		m.ui.Edit.clash = ""
		return
	case "j", "down":
		m.ui.Edit.clashPick = min(m.ui.Edit.clashPick+1, len(options)-1)
		return
	case "k", "up":
		m.ui.Edit.clashPick = max(m.ui.Edit.clashPick-1, 0)
		return
	case "enter":
		if m.ui.Edit.clashPick < len(options) {
			m.resolveClash(options[m.ui.Edit.clashPick].choice)
		}
		return
	}
	for _, o := range options {
		if o.key == key {
			m.resolveClash(o.choice)
			return
		}
	}
}

func (m *model) resolveClash(choice clashChoice) {
	if choice == clashMerge {
		m.mergeEditedCategory()
		return
	}
	src, dst, ok := m.mergeOffer()
	this, other, named := m.clashNames()
	if !ok || !named {
		m.cancelEditing()
		return
	}
	edited, taken := m.project.Categories[src], m.project.Categories[dst]
	typed := strings.TrimSpace(m.ui.Edit.input.Value())
	if !m.ui.Edit.isAdding {
		m.recordHistory()
	}
	var err error
	if choice == clashRenameOther {
		if _, err = operations.RenameCategory(&m.project, taken.ID, other); err == nil {
			_, err = operations.RenameCategory(&m.project, edited.ID, typed)
		}
		m.ui.Screen.StatusMsg = fmt.Sprintf("Renamed %q to %q", taken.Name, other)
	} else {
		_, err = operations.RenameCategory(&m.project, edited.ID, this)
	}
	if err != nil {
		if !m.ui.Edit.isAdding {
			m.discardLastHistory()
		}
		m.ui.Screen.StatusMsg = fmt.Sprintf("Rename failed: %v", err)
		m.ui.Edit.clash = ""
		return
	}
	m.storeTaskUpdate()
	m.ui.Modes.ToNormal()
	m.ui.Edit.reset()
	m.rebuildPositions()
	m.selectItem(selection.FocusCategory, edited.ID)
}

func (m model) clashChooserView() string {
	_, dst, ok := m.mergeOffer()
	if !ok {
		return ""
	}
	lines := []string{ui.DialogTitleStyle.Render(fmt.Sprintf("“%s” is taken", truncateText(m.project.Categories[dst].Name, 30))), ""}
	options := m.clashOptions()
	for i, o := range options {
		line := "  " + o.key + "  " + o.label
		if i == m.ui.Edit.clashPick {
			line = ui.SelectedStyle.Render("> " + o.key + "  " + o.label)
		}
		lines = append(lines, line)
	}
	if m.ui.Edit.clashPick < len(options) && options[m.ui.Edit.clashPick].choice == clashMerge {
		lines = append(lines, "")
		for _, l := range m.mergeLines() {
			lines = append(lines, ui.MutedStyle.Render(l))
		}
	}
	lines = append(lines, "", ui.RenderHints([]ui.Hint{{Key: "enter", Label: "choose"}, {Key: "esc", Label: "keep editing"}}))
	return ui.HelpDialogStyle.Render(strings.Join(lines, "\n"))
}
