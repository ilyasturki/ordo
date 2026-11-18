package app

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/ilyasturki/ordo/internal/data"
	"github.com/ilyasturki/ordo/internal/domain"
	"github.com/ilyasturki/ordo/internal/merge"
	"github.com/ilyasturki/ordo/internal/ui"
)

const conflictValueLines = 3

// ConflictState is a merge waiting on picks; the copy already took the file as its version.
type ConflictState struct {
	result merge.Result
	theirs domain.Project
	picks  []merge.Pick
	kept   string
	pick   int
}

func (s ConflictState) current() merge.Collision { return s.result.Collisions[len(s.picks)] }

func (s ConflictState) left() int { return len(s.result.Collisions) - len(s.picks) }

func (m *model) openConflict(res merge.Result, theirs domain.Project) {
	kept, _ := m.deps.Store.WriteRecovery(m.project, "mine")
	m.deps.Store.WriteRecovery(theirs, "theirs")
	m.ui.Conflict = ConflictState{result: res, theirs: theirs, kept: kept}
	m.ui.Modes.ToConflict()
	m.checkSaves()
	m.ui.Screen.StatusMsg = ""
}

type conflictPick struct {
	pick merge.Pick
	all  bool
}

func (m model) conflictChoice() (ui.Choice, []conflictPick) {
	c := m.ui.Conflict
	col := c.current()
	options := []ui.ChoiceOption{{Label: "Mine", Key: "m"}, {Label: "Theirs", Key: "t"}}
	picks := []conflictPick{{pick: merge.PickMine}, {pick: merge.PickTheirs}}
	if col.CanCopy() {
		options = append(options, ui.ChoiceOption{Label: "Both", Key: "b", Detail: "theirs, then mine as a copy"})
		picks = append(picks, conflictPick{pick: merge.PickBoth})
	}
	if c.left() > 1 {
		options = append(options,
			ui.ChoiceOption{Label: "All mine", Key: "a", Aliases: []string{"M"}},
			ui.ChoiceOption{Label: "All theirs", Key: "h", Aliases: []string{"T"}})
		picks = append(picks, conflictPick{pick: merge.PickMine, all: true}, conflictPick{pick: merge.PickTheirs, all: true})
	}

	title := "Changed on disk: 1 conflict"
	if total := len(c.result.Collisions); total > 1 {
		title = fmt.Sprintf("Changed on disk: %d of %d conflicts", len(c.picks)+1, total)
	}
	width := m.dialogWidth() - 10
	body := []string{truncateText(col.Row, width) + ", " + col.Field, ""}
	for _, side := range []struct{ label, value string }{{"mine  ", col.Mine}, {"theirs", col.Theirs}} {
		for i, line := range conflictValue(side.value, width) {
			if i == 0 {
				body = append(body, "  "+ui.MutedStyle.Render(side.label)+"  "+line)
			} else {
				body = append(body, "          "+line)
			}
		}
	}
	choice := ui.Choice{Title: title, Body: body, Options: options, Selected: c.pick}
	choice.Clamp()
	return choice, picks
}

// Esc keeps theirs for every conflict left: the file stays as it is, and u brings mine back.
func (m model) handleConflictKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	choice, picks := m.conflictChoice()
	act, i := choice.Handle(msg.String())
	c := &m.ui.Conflict
	c.pick = choice.Selected
	var picked conflictPick
	switch act {
	case ui.ChoicePicked:
		picked = picks[i]
	case ui.ChoiceCancelled:
		picked = conflictPick{pick: merge.PickTheirs, all: true}
	default:
		return m, nil
	}
	c.picks = append(c.picks, picked.pick)
	for picked.all && c.left() > 0 {
		c.picks = append(c.picks, picked.pick)
	}
	if c.left() == 0 {
		m.settleConflict()
	}
	return m, nil
}

func opposite(picks []merge.Pick) []merge.Pick {
	out := make([]merge.Pick, len(picks))
	for i, pick := range picks {
		if pick == merge.PickTheirs {
			out[i] = merge.PickMine
		}
	}
	return out
}

// settleConflict leaves one undo step, the opposite picks, so a wrong key costs one `u`.
func (m *model) settleConflict() {
	c := m.ui.Conflict
	m.ui.Modes.Uncover()
	m.ui.Conflict = ConflictState{}
	merged, err := c.result.Resolve(c.picks)
	if err != nil {
		m.ui.Screen.StatusMsg = "Merge failed: " + err.Error()
		return
	}
	other, err := c.result.Resolve(opposite(c.picks))
	if err != nil {
		m.ui.Screen.StatusMsg = "Merge failed: " + err.Error()
		return
	}
	m.takeMerged(merged, c.theirs)
	entry := m.snapshotEntry()
	entry.project = other
	m.ui.History.undo = append(m.ui.History.undo, entry)
	m.ui.Screen.StatusMsg = "Changed on disk: merged, u takes the choice back"
}

// leaveConflict leaves the file as theirs, for ordo's exit to name the copy of mine.
func (m *model) leaveConflict() {
	err := error(data.ErrStaleProject)
	if kept := m.ui.Conflict.kept; kept != "" {
		err = fmt.Errorf("%w: a copy stays in %s", err, kept)
	}
	m.deps.Store.Refuse(m.project.ID, err)
}

func conflictValue(s string, width int) []string {
	lines := strings.Split(s, "\n")
	cut := len(lines) > conflictValueLines
	lines = lines[:min(len(lines), conflictValueLines)]
	for i, line := range lines {
		lines[i] = truncateText(line, width)
	}
	if cut {
		lines[len(lines)-1] += " ..."
	}
	return lines
}

func (m model) conflictView() string {
	c, _ := m.conflictChoice()
	return m.choiceView(c)
}
