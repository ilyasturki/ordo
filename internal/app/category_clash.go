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
		{"t", clashRenameThis, fmt.Sprintf("Name %s “%s”", which, truncateText(this, 30))},
		{"o", clashRenameOther, fmt.Sprintf("Rename the other to “%s”", truncateText(other, 30))},
		{"m", clashMerge, fmt.Sprintf("Merge into “%s”", taken)},
	}
}

func (m model) clashChoice() (ui.Choice, []clashOption) {
	_, dst, ok := m.mergeOffer()
	if !ok {
		return ui.Choice{}, nil
	}
	options := m.clashOptions()
	c := ui.Choice{
		Title:    fmt.Sprintf("“%s” is taken", truncateText(m.project.Categories[dst].Name, 30)),
		Selected: m.ui.Edit.clashPick,
	}
	for _, o := range options {
		c.Options = append(c.Options, ui.ChoiceOption{Label: o.label, Key: o.key})
	}
	if c.Selected < len(options) && options[c.Selected].choice == clashMerge {
		c.Note = m.mergeLines()
	}
	return c, options
}

func (m *model) handleClashKey(key string) {
	if key == "n" {
		key = "esc"
	}
	c, options := m.clashChoice()
	act, i := c.Handle(key)
	m.ui.Edit.clashPick = c.Selected
	switch act {
	case ui.ChoiceCancelled:
		m.ui.Edit.clash = ""
	case ui.ChoicePicked:
		m.resolveClash(options[i].choice)
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
	c, _ := m.clashChoice()
	if c.Title == "" {
		return ""
	}
	return m.choiceView(c)
}
