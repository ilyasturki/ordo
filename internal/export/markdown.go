package export

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strings"

	"phasionary/internal/domain"
)

var (
	projectHeaderRe  = regexp.MustCompile(`^#\s+(.+)$`)
	categoryHeaderRe = regexp.MustCompile(`^##\s+(.+)$`)
	taskLineRe       = regexp.MustCompile(`^-\s+\[([ x\-~])\]\s+(.+)$`)
	// 1-3 spaces, less than a description's 4, so subtask and description lines never mix.
	subtaskLineRe    = regexp.MustCompile(`^ {1,3}-\s+\[([ x\-~])\]\s+(.+)$`)
	prioritySuffixRe = regexp.MustCompile(`\s+\((critical|high|medium|low|trivial)\)\s*$`)
	// A labeled separator is a level-3 heading (### label); a bare one is a
	// thematic break (--- or longer). Neither collides with the project (#) or
	// category (##) headers, which both require whitespace after their hashes.
	separatorLabelRe = regexp.MustCompile(`^###\s+(.+)$`)
	separatorBreakRe = regexp.MustCompile(`^-{3,}\s*$`)
)

func statusToMarker(status string) string {
	switch status {
	case domain.StatusCompleted:
		return "x"
	case domain.StatusCancelled:
		return "-"
	case domain.StatusInProgress:
		return "~"
	default:
		return " "
	}
}

func markerToStatus(marker string) string {
	switch marker {
	case "x":
		return domain.StatusCompleted
	case "-":
		return domain.StatusCancelled
	case "~":
		return domain.StatusInProgress
	default:
		return domain.StatusTodo
	}
}

const (
	descriptionIndent = "    "
	subtaskIndent     = "  "
)

func CategoryMarkdown(project domain.Project, i int) string {
	var sb strings.Builder
	for c := i; c < domain.BlockEnd(project.Categories, i); c++ {
		if c > i {
			sb.WriteByte('\n')
		}
		writeCategory(&sb, project.CategoryPath(c), project.Categories[c])
	}
	return sb.String()
}

func writeCategory(sb *strings.Builder, heading string, cat domain.Category) {
	fmt.Fprintf(sb, "## %s\n\n", heading)
	for _, task := range cat.Tasks {
		if task.IsSeparator() {
			if label := strings.TrimSpace(task.Title); label != "" {
				fmt.Fprintf(sb, "### %s\n", label)
			} else {
				sb.WriteString("---\n")
			}
			continue
		}
		indent := ""
		if task.ParentID != "" {
			indent = subtaskIndent
		}
		marker := statusToMarker(task.Status)
		line := fmt.Sprintf("%s- [%s] %s", indent, marker, task.Title)
		if task.Priority != "" {
			line += fmt.Sprintf(" (%s)", task.Priority)
		}
		sb.WriteString(line)
		sb.WriteByte('\n')
		if task.Description != "" {
			for _, dline := range strings.Split(task.Description, "\n") {
				if dline == "" {
					sb.WriteByte('\n')
					continue
				}
				sb.WriteString(indent + descriptionIndent)
				sb.WriteString(dline)
				sb.WriteByte('\n')
			}
		}
	}
}

func ExportMarkdown(project domain.Project, w io.Writer) error {
	if _, err := fmt.Fprintf(w, "# %s\n", project.Name); err != nil {
		return err
	}
	for i := 0; i < len(project.Categories); i = domain.BlockEnd(project.Categories, i) {
		if _, err := fmt.Fprintf(w, "\n%s", CategoryMarkdown(project, i)); err != nil {
			return err
		}
	}
	return nil
}

func ImportMarkdown(r io.Reader, projectName string) (domain.Project, error) {
	scanner := bufio.NewScanner(r)

	var parsedName string
	var categories []categoryData
	var currentCategory *categoryData

	// Blank lines between description chunks are buffered so trailing blanks
	// (which aren't part of the description) don't accidentally extend it.
	var currentTask *taskData
	var descBuf []string
	var pendingBlanks int

	flushDescription := func() {
		if currentTask == nil {
			return
		}
		if len(descBuf) > 0 {
			currentTask.description = strings.Join(descBuf, "\n")
		}
		currentTask = nil
		descBuf = nil
		pendingBlanks = 0
	}

	for scanner.Scan() {
		line := scanner.Text()

		if m := projectHeaderRe.FindStringSubmatch(line); m != nil {
			flushDescription()
			parsedName = strings.TrimSpace(m[1])
			continue
		}

		if m := categoryHeaderRe.FindStringSubmatch(line); m != nil {
			flushDescription()
			if currentCategory != nil {
				categories = append(categories, *currentCategory)
			}
			currentCategory = &categoryData{name: strings.TrimSpace(m[1])}
			continue
		}

		if m := separatorLabelRe.FindStringSubmatch(line); m != nil && currentCategory != nil {
			flushDescription()
			currentCategory.tasks = append(currentCategory.tasks, taskData{
				kind:  domain.KindSeparator,
				title: strings.TrimSpace(m[1]),
			})
			continue
		}

		if separatorBreakRe.MatchString(line) && currentCategory != nil {
			flushDescription()
			currentCategory.tasks = append(currentCategory.tasks, taskData{kind: domain.KindSeparator})
			continue
		}

		m := taskLineRe.FindStringSubmatch(line)
		sub := false
		if m == nil && currentCategory != nil {
			m = subtaskLineRe.FindStringSubmatch(line)
			sub = currentCategory.hasParent()
		}
		if m != nil && currentCategory != nil {
			flushDescription()
			marker := m[1]
			title := strings.TrimSpace(m[2])
			var priority string
			if pm := prioritySuffixRe.FindStringSubmatch(title); pm != nil {
				priority = pm[1]
				title = strings.TrimSpace(prioritySuffixRe.ReplaceAllString(title, ""))
			}
			currentCategory.tasks = append(currentCategory.tasks, taskData{
				title:    title,
				status:   markerToStatus(marker),
				priority: priority,
				subtask:  sub,
			})
			currentTask = &currentCategory.tasks[len(currentCategory.tasks)-1]
			continue
		}

		if currentTask != nil {
			if strings.TrimSpace(line) == "" {
				pendingBlanks++
				continue
			}
			if strings.HasPrefix(line, "  ") || strings.HasPrefix(line, "\t") {
				for i := 0; i < pendingBlanks; i++ {
					descBuf = append(descBuf, "")
				}
				pendingBlanks = 0
				descBuf = append(descBuf, dedentDescription(line, currentTask.subtask))
				continue
			}
			flushDescription()
		}
	}
	flushDescription()
	if err := scanner.Err(); err != nil {
		return domain.Project{}, err
	}
	if currentCategory != nil {
		categories = append(categories, *currentCategory)
	}

	name := projectName
	if name == "" {
		name = parsedName
	}
	if name == "" {
		name = "Imported Project"
	}

	project, err := domain.NewProject(name)
	if err != nil {
		return domain.Project{}, err
	}

	for _, cd := range categories {
		// A repeated heading continues the category it names.
		parentID, name := headingParent(project, cd.name)
		var cat *domain.Category
		if i := domain.CategoryNameConflict(project.Categories, parentID, name, -1); i >= 0 {
			cat = &project.Categories[i]
		} else {
			fresh, err := project.AddCategoryUnder(parentID, name)
			if err != nil {
				return domain.Project{}, err
			}
			cat = fresh
		}
		parentTask := ""
		for _, td := range cd.tasks {
			if td.kind == domain.KindSeparator {
				sep, err := domain.NewSeparator()
				if err != nil {
					return domain.Project{}, err
				}
				sep.Title = td.title
				cat.Tasks = append(cat.Tasks, sep)
				parentTask = ""
				continue
			}
			task, err := domain.NewTask(td.title)
			if err != nil {
				return domain.Project{}, err
			}
			if err := task.SetStatus(td.status); err != nil {
				return domain.Project{}, err
			}
			if td.priority != "" {
				if err := task.SetPriority(td.priority); err != nil {
					return domain.Project{}, err
				}
			}
			task.Description = td.description
			if td.subtask && parentTask != "" {
				task.ParentID = parentTask
			} else {
				parentTask = task.ID
			}
			cat.Tasks = append(cat.Tasks, task)
		}
	}

	return project, nil
}

func headingParent(project domain.Project, heading string) (parentID, name string) {
	for _, split := range domain.SplitCategoryHeading(heading) {
		if i := domain.CategoryNameConflict(project.Categories, "", split[0], -1); i >= 0 {
			return project.Categories[i].ID, split[1]
		}
	}
	return "", heading
}

func dedentDescription(line string, subtask bool) string {
	indent := descriptionIndent
	if subtask {
		indent = subtaskIndent + descriptionIndent
	}
	if rest, ok := strings.CutPrefix(line, indent); ok {
		return rest
	}
	return strings.TrimLeft(line, " \t")
}

type categoryData struct {
	name  string
	tasks []taskData
}

func (c *categoryData) hasParent() bool {
	return len(c.tasks) > 0 && c.tasks[len(c.tasks)-1].kind != domain.KindSeparator
}

type taskData struct {
	title       string
	status      string
	priority    string
	description string
	kind        string
	subtask     bool
}
