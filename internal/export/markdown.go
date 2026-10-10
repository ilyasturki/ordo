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

const descriptionIndent = "    "

func ExportCategoryMarkdown(cat domain.Category) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "## %s\n\n", cat.Name)
	for _, task := range cat.Tasks {
		if task.IsSeparator() {
			if label := strings.TrimSpace(task.Title); label != "" {
				fmt.Fprintf(&sb, "### %s\n", label)
			} else {
				sb.WriteString("---\n")
			}
			continue
		}
		marker := statusToMarker(task.Status)
		line := fmt.Sprintf("- [%s] %s", marker, task.Title)
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
				sb.WriteString(descriptionIndent)
				sb.WriteString(dline)
				sb.WriteByte('\n')
			}
		}
	}
	return sb.String()
}

func ExportMarkdown(project domain.Project, w io.Writer) error {
	if _, err := fmt.Fprintf(w, "# %s\n", project.Name); err != nil {
		return err
	}
	for _, cat := range project.Categories {
		if _, err := fmt.Fprintf(w, "\n%s", ExportCategoryMarkdown(cat)); err != nil {
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

		if m := taskLineRe.FindStringSubmatch(line); m != nil && currentCategory != nil {
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
				descBuf = append(descBuf, strings.TrimLeft(line, " \t"))
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
		var cat *domain.Category
		if i := domain.CategoryNameConflict(project.Categories, "", cd.name, -1); i >= 0 {
			cat = &project.Categories[i]
		} else {
			fresh, err := domain.NewCategory(cd.name)
			if err != nil {
				return domain.Project{}, err
			}
			project.Categories = append(project.Categories, fresh)
			cat = &project.Categories[len(project.Categories)-1]
		}
		for _, td := range cd.tasks {
			if td.kind == domain.KindSeparator {
				sep, err := domain.NewSeparator()
				if err != nil {
					return domain.Project{}, err
				}
				sep.Title = td.title
				cat.Tasks = append(cat.Tasks, sep)
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
			cat.Tasks = append(cat.Tasks, task)
		}
	}

	return project, nil
}

type categoryData struct {
	name  string
	tasks []taskData
}

type taskData struct {
	title       string
	status      string
	priority    string
	description string
	kind        string
}
