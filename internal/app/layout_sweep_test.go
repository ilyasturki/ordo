package app

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/ilyasturki/ordo/internal/app/selection"
	"github.com/ilyasturki/ordo/internal/config"
	"github.com/ilyasturki/ordo/internal/domain"
)

func sweepKey(k string) tea.KeyPressMsg {
	switch k {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	}
	if c, ok := strings.CutPrefix(k, "ctrl+"); ok {
		return tea.KeyPressMsg{Code: []rune(c)[0], Mod: tea.ModCtrl}
	}
	return tea.KeyPressMsg{Code: []rune(k)[0], Text: k}
}

func keys(parts ...any) []string {
	var out []string
	for _, p := range parts {
		switch p := p.(type) {
		case string:
			out = append(out, p)
		case []string:
			out = append(out, p...)
		}
	}
	return out
}

func typed(s string) []string {
	return strings.Split(s, "")
}

func asModel(m tea.Model) model {
	if p, ok := m.(*model); ok {
		return *p
	}
	return m.(model)
}

// sweepProject mixes wrapped titles, descriptions and a sub-category, and can
// end on an empty category.
func sweepProject(emptyLast bool) domain.Project {
	long := "a dictated title long enough to wrap over two rows on an eighty column screen, with a few more words"
	p := domain.Project{ID: "p1", Name: "P"}
	for c := range 6 {
		cat := domain.Category{ID: fmt.Sprintf("c%d", c), Name: fmt.Sprintf("Cat %d", c)}
		for i := range 3 + c%3 {
			t := domain.Task{ID: fmt.Sprintf("c%dt%d", c, i), Title: fmt.Sprintf("Task %d.%d", c, i), Status: domain.StatusTodo}
			if (c+i)%4 == 0 {
				t.Title += " " + long
			}
			if (c+i)%3 == 1 {
				t.Description = "one line of notes"
			}
			cat.Tasks = append(cat.Tasks, t)
		}
		p.Categories = append(p.Categories, cat)
		if c == 2 {
			p.Categories = append(p.Categories, domain.Category{ID: "sub", Name: "area", ParentID: "c2", Tasks: []domain.Task{
				{ID: "s1", Title: "Sub task", Status: domain.StatusTodo, Description: "sub notes"},
			}})
		}
	}
	last := &p.Categories[len(p.Categories)-1]
	last.Tasks[len(last.Tasks)-1].Description = "notes under the last task"
	if emptyLast {
		p.Categories = append(p.Categories, domain.Category{ID: "empty", Name: "Empty"})
	}
	return p
}

type sweepCase struct {
	name          string
	project       func() domain.Project
	width, height int
	bar, desc     bool
}

func (c sweepCase) String() string {
	return fmt.Sprintf("%s %dx%d bar=%v desc=%v", c.name, c.width, c.height, c.bar, c.desc)
}

func sweepCases() []sweepCase {
	projects := []struct {
		name    string
		project func() domain.Project
	}{
		{"40 tasks", longProject},
		{"mixed", func() domain.Project { return sweepProject(false) }},
		{"empty last", func() domain.Project { return sweepProject(true) }},
	}
	var out []sweepCase
	for _, p := range projects {
		for _, height := range []int{20, 24, 35} {
			for _, bar := range []bool{true, false} {
				for _, desc := range []bool{false, true} {
					out = append(out, sweepCase{p.name, p.project, 60, height, bar, desc})
				}
			}
		}
	}
	return out
}

func (c sweepCase) model(t *testing.T) model {
	cfg := config.DefaultConfig()
	cfg.ShowShortcutBar = c.bar
	m := newTestModel(t, c.project())
	m.deps.CfgManager = &stubConfigReader{cfg: cfg}
	withStubStateManager(m)
	m.ui.Screen.ExpandDescriptions = c.desc
	m.ui.Screen.Width = c.width
	m.rebuildPositions()
	next, _ := m.Update(tea.WindowSizeMsg{Width: c.width, Height: c.height})
	return asModel(next)
}

// cursorWhole reports whether the cursor's row is drawn whole whenever it fits
// the view, and with render, that the frame fits the screen.
func cursorWhole(t *testing.T, m model, where string, render bool) bool {
	t.Helper()
	if render {
		if got := strings.Count(m.View().Content, "\n") + 1; got > m.ui.Screen.Height {
			t.Errorf("%s: the frame has %d rows on a %d-row screen", where, got, m.ui.Screen.Height)
			return false
		}
	}
	if m.ui.Selection.IsEmpty() || m.selected() < 0 {
		return true
	}
	layout := m.buildLayout()
	v := NewViewport(layout, m.ui.Screen.Height, m.layoutConfig())
	v.ComputeVisibility(m.ui.Screen.TopRow)
	for i, item := range layout.Items {
		if item.PositionIndex != m.selected() || item.Height > v.availableHeight()-2 || shownWhole(v, i) {
			continue
		}
		t.Errorf("%s: the cursor's item %d (%d rows) is cut off: top %d shows items [%d,%d) from row %d of the first",
			where, i, item.Height, v.TopRow, v.VisibleStart, v.VisibleEnd, v.RowOffset)
		return false
	}
	return true
}

// Every key, from both ends of a project taller than the screen, leaves the
// cursor's row on screen.
func TestLayoutSweep_CursorStaysOnScreen(t *testing.T) {
	starts := [][]string{{"j"}, {"G"}, {"G", "k"}, {"G", "k", "k"}, {"G", "{"}, {"G", "{", "k"}, {"G", "{", "{", "j"}}
	scripts := [][]string{
		keys("A", typed("New"), "enter"),
		keys("A", typed("New"), "enter", "A", typed("More"), "enter"),
		keys("A", "X", "enter", "u", "ctrl+r", "u"),
		keys("a", typed("New"), "enter"),
		keys("G", "o", "N", "enter"),
		keys("a", "esc"),
		keys("-", "esc"),
		keys("enter", typed(" and some more words to wrap the title over two rows or more"), "enter"),
		keys("y", "p"),
		keys("y", "G", "p"),
		keys("x", "esc"),
		keys("x", "k", "p"),
		keys("G", "x", "esc", "u"),
		keys("J", "J", "K"),
		keys(">", "<"),
		keys("v", "k", "k", "esc"),
		keys("v", "j", "J", "esc"),
		keys("z", "a"),
		keys("z", "a", "j", "z", "a"),
		keys("z", "c", "G", "z", "o"),
		keys("z", "c", "G", "A", "X", "enter"),
		keys("z", "c", "G", "k", "y", "G", "p"),
		keys("z", "e", "j", "k", "z", "e"),
		keys("z", "d", "j", "j", "z", "d"),
		keys("z", "b"), keys("z", "z"), keys("z", "t"),
		keys("ctrl+d", "ctrl+d", "ctrl+d", "ctrl+u", "ctrl+u", "ctrl+u"),
		keys("{", "}", "}", "{"),
		keys("l", "l"), keys("h", "h"), keys(" ", " "), keys("t"), keys("s"),
		keys("T", "esc"), keys("E", "esc"), keys("g", "i", "esc"),
	}
	for _, c := range sweepCases() {
		for _, start := range starts {
			for _, script := range scripts {
				m := c.model(t)
				ks := keys("g", "g", start, script)
				for i, k := range ks {
					// The stub store can't rename the project.
					if p, ok := m.selectedPosition(); ok && k == "enter" && m.ui.Modes.IsNormal() && p.Kind == selection.FocusProject {
						continue
					}
					next, _ := m.Update(sweepKey(k))
					m = asModel(next)
					if !cursorWhole(t, m, fmt.Sprintf("%s keys %q", c, ks[:i+1]), i == len(ks)-1) {
						break
					}
				}
				if t.Failed() {
					return
				}
			}
		}
	}
}

func TestLayoutSweep_WalkEveryRow(t *testing.T) {
	for _, c := range sweepCases() {
		m := c.model(t)
		for _, k := range []string{"j", "k"} {
			for i := range m.ui.Selection.Count() + 1 {
				next, _ := m.Update(sweepKey(k))
				m = asModel(next)
				if !cursorWhole(t, m, fmt.Sprintf("%s %s #%d", c, k, i), true) {
					return
				}
			}
		}
	}
}
