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

func (m model) handleConflictKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	c := &m.ui.Conflict
	switch msg.String() {
	case "m":
		c.picks = append(c.picks, merge.PickMine)
	case "t":
		c.picks = append(c.picks, merge.PickTheirs)
	case "b":
		if c.current().CanCopy() {
			c.picks = append(c.picks, merge.PickBoth)
		}
	case "M", "T":
		pick := merge.PickTheirs
		if msg.String() == "M" {
			pick = merge.PickMine
		}
		for c.left() > 0 {
			c.picks = append(c.picks, pick)
		}
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
	c := m.ui.Conflict
	col := c.current()
	title := "Changed on disk: 1 conflict"
	if total := len(c.result.Collisions); total > 1 {
		title = fmt.Sprintf("Changed on disk: %d of %d conflicts", len(c.picks)+1, total)
	}
	width := max(20, min(60, m.ui.Screen.Width-20))
	lines := []string{title, "", truncateText(col.Row, width) + ", " + col.Field, ""}
	for _, side := range []struct{ label, value string }{{"mine  ", col.Mine}, {"theirs", col.Theirs}} {
		for i, line := range conflictValue(side.value, width) {
			if i == 0 {
				lines = append(lines, "  "+side.label+"  "+line)
			} else {
				lines = append(lines, "          "+line)
			}
		}
	}
	hints := []ui.Hint{{Key: "m", Label: "mine"}, {Key: "t", Label: "theirs"}}
	if col.CanCopy() {
		hints = append(hints, ui.Hint{Key: "b", Label: "both"})
	}
	lines = append(lines, "", ui.RenderHints(hints))
	if c.left() > 1 {
		lines = append(lines, ui.RenderHints([]ui.Hint{{Key: "M", Label: "all mine"}, {Key: "T", Label: "all theirs"}}))
	}
	if col.CanCopy() {
		lines = append(lines, "", "both keeps theirs and adds mine as a copy just below it")
	}
	return ui.HelpDialogStyle.Render(strings.Join(lines, "\n"))
}
