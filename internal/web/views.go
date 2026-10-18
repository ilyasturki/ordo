package web

import (
	"strings"

	"github.com/ilyasturki/ordo/internal/attachments"
	"github.com/ilyasturki/ordo/internal/domain"
)

const (
	filterOpen = "open"
	filterAll  = "all"
	filterDone = "done"
)

func validFilter(f string) bool {
	return f == filterOpen || f == filterAll || f == filterDone
}

func finished(status string) bool {
	return status == domain.StatusCompleted || status == domain.StatusCancelled
}

func passes(t domain.Task, filter string) bool {
	switch {
	case t.IsSeparator():
		return filter == filterAll
	case filter == filterOpen:
		return !finished(t.Status)
	case filter == filterDone:
		return finished(t.Status)
	}
	return true
}

type imageView struct {
	Name   string
	Target string
	Served bool
}

type rowView struct {
	domain.Task
	Children   []rowView
	ChildDone  int
	ChildTotal int
	Images     []imageView
	Note       bool
}

func (r rowView) Est() string { return domain.FormatEstimate(r.EstimateMinutes) }

type catView struct {
	ID         string
	Name       string
	ParentName string
	Sub        bool
	Done       int
	Total      int
	Est        string
	Folded     bool
	Rows       []rowView
	Hidden     int
}

func (c catView) Percent() int {
	if c.Total == 0 {
		return 0
	}
	return c.Done * 100 / c.Total
}

type countsView struct {
	Open, Doing, Done, Total int
}

func imagesOf(description string) []imageView {
	var out []imageView
	for _, ref := range attachments.Refs(description) {
		out = append(out, imageView{Name: ref.Name, Target: ref.Target, Served: ref.Internal()})
	}
	return out
}

func newRow(t domain.Task) rowView {
	return rowView{
		Task:   t,
		Images: imagesOf(t.Description),
		Note:   strings.TrimSpace(attachments.Strip(t.Description)) != "",
	}
}

// buildRows nests each category's rows, keeping a parent whose subtasks pass the filter.
func buildRows(cat domain.Category, filter string) ([]rowView, int) {
	var rows []rowView
	hidden := 0
	for i := 0; i < len(cat.Tasks); {
		end := domain.BlockEnd(cat.Tasks, i)
		row := newRow(cat.Tasks[i])
		row.ChildDone, row.ChildTotal = cat.ChildProgress(i)
		for _, child := range cat.Tasks[i+1 : end] {
			if passes(child, filter) {
				row.Children = append(row.Children, newRow(child))
			} else {
				hidden++
			}
		}
		if passes(row.Task, filter) || len(row.Children) > 0 {
			rows = append(rows, row)
		} else {
			hidden++
		}
		i = end
	}
	return rows, hidden
}

func buildCategories(p *domain.Project, filter string, folded map[string]bool) []catView {
	cats := make([]catView, 0, len(p.Categories))
	for i, c := range p.Categories {
		counts := p.TreeStatusCounts(i)
		view := catView{
			ID:     c.ID,
			Name:   c.Name,
			Sub:    c.ParentID != "",
			Done:   counts.Completed + counts.Cancelled,
			Total:  counts.Total(),
			Est:    domain.FormatEstimate(c.EstimateMinutes),
			Folded: folded[c.ID],
		}
		if parent := domain.ParentIndex(p.Categories, i); parent >= 0 {
			view.ParentName = p.Categories[parent].Name
		}
		view.Rows, view.Hidden = buildRows(c, filter)
		cats = append(cats, view)
	}
	return cats
}

func projectCounts(p *domain.Project) countsView {
	c := p.StatusCounts()
	return countsView{Open: c.Todo + c.InProgress, Doing: c.InProgress, Done: c.Completed + c.Cancelled, Total: c.Total()}
}

// splitDescription separates the trailing image-only lines from the text, when that split loses nothing.
func splitDescription(description string) (text string, images []string) {
	lines := strings.Split(description, "\n")
	cut := len(lines)
	for cut > 0 {
		line := strings.TrimSpace(lines[cut-1])
		if line == "" {
			cut--
			continue
		}
		pieces := attachments.Split(line)
		if len(pieces) != 1 || pieces[0].Ref == nil {
			break
		}
		cut--
	}
	for _, line := range lines[cut:] {
		if strings.TrimSpace(line) != "" {
			images = append(images, strings.TrimSpace(line))
		}
	}
	return strings.TrimRight(strings.Join(lines[:cut], "\n"), "\n"), images
}

func joinDescription(text string, images []string) string {
	text = strings.TrimRight(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	parts := make([]string, 0, len(images)+1)
	if strings.TrimSpace(text) != "" {
		parts = append(parts, text)
	}
	parts = append(parts, images...)
	return strings.Join(parts, "\n")
}

var statusLabels = map[string]string{
	domain.StatusTodo:       "To do",
	domain.StatusInProgress: "Doing",
	domain.StatusCompleted:  "Done",
	domain.StatusCancelled:  "Cancelled",
}

var priorityLabels = map[string]string{
	"":                      "None",
	domain.PriorityCritical: "Critical",
	domain.PriorityHigh:     "High",
	domain.PriorityMedium:   "Medium",
	domain.PriorityLow:      "Low",
	domain.PriorityTrivial:  "Trivial",
}

func tagName(color string) string {
	if color == "" {
		return "None"
	}
	hue, diamond := strings.CutSuffix(color, domain.TagDiamondSuffix)
	if hue == "" {
		return color
	}
	name := strings.ToUpper(hue[:1]) + hue[1:]
	if diamond {
		name += " diamond"
	}
	return name
}
