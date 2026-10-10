package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/spf13/viper"

	"phasionary/internal/app"
	"phasionary/internal/domain"
)

type OutputFormat int

const (
	FormatHuman OutputFormat = iota
	FormatJSON
)

func getOutputFormat() OutputFormat {
	if viper.GetBool("json") {
		return FormatJSON
	}
	return FormatHuman
}

func isQuiet() bool {
	return viper.GetBool("quiet")
}

func isLong() bool {
	return viper.GetBool("long")
}

type ProjectListItem struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

type ProjectsOutput struct {
	Projects []ProjectListItem `json:"projects"`
}

func writeProjects(w io.Writer, projects []domain.Project) error {
	if getOutputFormat() == FormatJSON {
		output := ProjectsOutput{
			Projects: make([]ProjectListItem, 0, len(projects)),
		}
		for _, p := range projects {
			output.Projects = append(output.Projects, ProjectListItem{
				ID:        p.ID,
				Name:      p.Name,
				CreatedAt: p.CreatedAt,
			})
		}
		return writeJSON(w, output)
	}

	if len(projects) == 0 {
		if !isQuiet() {
			fmt.Fprintln(w, "No projects found.")
		}
		return nil
	}

	if !isLong() {
		for _, p := range projects {
			fmt.Fprintln(w, p.Name)
		}
		return nil
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "NAME\tID\tCREATED")
	for _, p := range projects {
		fmt.Fprintf(tw, "%s\t%s\t%s\n", p.Name, p.ID, app.FormatDate(p.CreatedAt))
	}
	return tw.Flush()
}

type CategoryListItem struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	TaskCount       int    `json:"task_count"`
	EstimateMinutes int    `json:"estimate_minutes,omitempty"`
}

type CategoriesOutput struct {
	Categories []CategoryListItem `json:"categories"`
}

func writeCategories(w io.Writer, categories []domain.Category) error {
	if getOutputFormat() == FormatJSON {
		output := CategoriesOutput{
			Categories: make([]CategoryListItem, 0, len(categories)),
		}
		for _, c := range categories {
			output.Categories = append(output.Categories, CategoryListItem{
				ID:              c.ID,
				Name:            c.Name,
				TaskCount:       c.TaskCount(),
				EstimateMinutes: c.EstimateMinutes,
			})
		}
		return writeJSON(w, output)
	}

	if len(categories) == 0 {
		if !isQuiet() {
			fmt.Fprintln(w, "No categories found.")
		}
		return nil
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if isLong() {
		fmt.Fprintln(tw, "NAME\tID\tSTATUS\tTASKS\tESTIMATE")
		for _, c := range categories {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%d\t%s\n", c.Name, c.ID, formatAggregateStatus(c.AggregateStatus()), c.TaskCount(), formatEstimateCell(c.EstimateMinutes))
		}
	} else {
		fmt.Fprintln(tw, "NAME\tSTATUS\tTASKS\tESTIMATE")
		for _, c := range categories {
			fmt.Fprintf(tw, "%s\t%s\t%d\t%s\n", c.Name, formatAggregateStatus(c.AggregateStatus()), c.TaskCount(), formatEstimateCell(c.EstimateMinutes))
		}
	}
	return tw.Flush()
}

func formatAggregateStatus(status string) string {
	if status == "" {
		return "-"
	}
	return status
}

func formatEstimateCell(minutes int) string {
	if minutes <= 0 {
		return "-"
	}
	return formatDuration(minutes)
}

type TaskListItem struct {
	ID              string `json:"id"`
	Title           string `json:"title"`
	Status          string `json:"status"`
	Priority        string `json:"priority,omitempty"`
	Category        string `json:"category"`
	EstimateMinutes int    `json:"estimate_minutes,omitempty"`
	TagColor        string `json:"tag_color,omitempty"`
}

type TasksOutput struct {
	Tasks []TaskListItem `json:"tasks"`
}

func writeTaskList(w io.Writer, tasks []TaskListItem) error {
	if getOutputFormat() == FormatJSON {
		output := TasksOutput{Tasks: tasks}
		return writeJSON(w, output)
	}

	if len(tasks) == 0 {
		if !isQuiet() {
			fmt.Fprintln(w, "No tasks found.")
		}
		return nil
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if isLong() {
		fmt.Fprintln(tw, "ID\tCATEGORY\tSTATUS\tPRIORITY\tESTIMATE\tTITLE")
		for _, t := range tasks {
			priority := t.Priority
			if priority == "" {
				priority = "-"
			}
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n", t.ID, t.Category, t.Status, priority, formatEstimateCell(t.EstimateMinutes), t.Title)
		}
	} else {
		fmt.Fprintln(tw, "CATEGORY\tSTATUS\tPRIORITY\tTITLE")
		for _, t := range tasks {
			priority := t.Priority
			if priority == "" {
				priority = "-"
			}
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", t.Category, t.Status, priority, t.Title)
		}
	}
	return tw.Flush()
}

type TaskDetailOutput struct {
	Task TaskDetail `json:"task"`
}

type TaskDetail struct {
	ID              string `json:"id"`
	Title           string `json:"title"`
	Status          string `json:"status"`
	Priority        string `json:"priority,omitempty"`
	Category        string `json:"category"`
	EstimateMinutes int    `json:"estimate_minutes,omitempty"`
	TagColor        string `json:"tag_color,omitempty"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
	CompletionDate  string `json:"completion_date,omitempty"`
	Description     string `json:"description,omitempty"`
}

func writeTaskDetail(w io.Writer, task domain.Task, categoryName string) error {
	detail := TaskDetail{
		ID:              task.ID,
		Title:           task.Title,
		Status:          task.Status,
		Priority:        task.Priority,
		Category:        categoryName,
		EstimateMinutes: task.EstimateMinutes,
		TagColor:        task.TagColor,
		CreatedAt:       task.CreatedAt,
		UpdatedAt:       task.UpdatedAt,
		CompletionDate:  task.CompletionDate,
		Description:     task.Description,
	}

	if getOutputFormat() == FormatJSON {
		output := TaskDetailOutput{Task: detail}
		return writeJSON(w, output)
	}

	fmt.Fprintf(w, "Title:    %s\n", detail.Title)
	fmt.Fprintf(w, "ID:       %s\n", detail.ID)
	fmt.Fprintf(w, "Category: %s\n", detail.Category)
	fmt.Fprintf(w, "Status:   %s\n", detail.Status)
	if detail.Priority != "" {
		fmt.Fprintf(w, "Priority: %s\n", detail.Priority)
	}
	if detail.TagColor != "" {
		fmt.Fprintf(w, "Tag:      %s\n", detail.TagColor)
	}
	if detail.EstimateMinutes > 0 {
		fmt.Fprintf(w, "Estimate: %s\n", formatDuration(detail.EstimateMinutes))
	}
	fmt.Fprintf(w, "Created:  %s\n", detail.CreatedAt)
	fmt.Fprintf(w, "Updated:  %s\n", detail.UpdatedAt)
	if detail.CompletionDate != "" {
		fmt.Fprintf(w, "Completed: %s\n", detail.CompletionDate)
	}
	if detail.Description != "" {
		fmt.Fprintf(w, "\n%s\n", detail.Description)
	}
	return nil
}

type ProjectDetailOutput struct {
	Project ProjectDetail `json:"project"`
}

type ProjectDetail struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	CreatedAt     string   `json:"created_at"`
	UpdatedAt     string   `json:"updated_at"`
	CategoryCount int      `json:"category_count"`
	TaskCount     int      `json:"task_count"`
	Categories    []string `json:"categories"`
}

func writeProjectDetail(w io.Writer, project domain.Project) error {
	taskCount := 0
	categories := make([]string, 0, len(project.Categories))
	for _, c := range project.Categories {
		taskCount += c.TaskCount()
		categories = append(categories, c.Name)
	}

	detail := ProjectDetail{
		ID:            project.ID,
		Name:          project.Name,
		CreatedAt:     project.CreatedAt,
		UpdatedAt:     project.UpdatedAt,
		CategoryCount: len(project.Categories),
		TaskCount:     taskCount,
		Categories:    categories,
	}

	if getOutputFormat() == FormatJSON {
		output := ProjectDetailOutput{Project: detail}
		return writeJSON(w, output)
	}

	fmt.Fprintf(w, "Name:       %s\n", detail.Name)
	fmt.Fprintf(w, "Categories: %d\n", detail.CategoryCount)
	fmt.Fprintf(w, "Tasks:      %d\n", detail.TaskCount)
	fmt.Fprintf(w, "Created:    %s\n", app.FormatDate(detail.CreatedAt))
	fmt.Fprintf(w, "Updated:    %s\n", app.FormatDate(detail.UpdatedAt))
	if len(categories) > 0 {
		fmt.Fprintf(w, "\nCategories: %s\n", strings.Join(categories, ", "))
	}
	return nil
}

type CategoryDetailOutput struct {
	Category CategoryDetail `json:"category"`
}

type CategoryDetail struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
	TaskCount       int    `json:"task_count"`
	EstimateMinutes int    `json:"estimate_minutes,omitempty"`
}

func writeCategoryDetail(w io.Writer, cat domain.Category) error {
	detail := CategoryDetail{
		ID:              cat.ID,
		Name:            cat.Name,
		CreatedAt:       cat.CreatedAt,
		UpdatedAt:       cat.UpdatedAt,
		TaskCount:       cat.TaskCount(),
		EstimateMinutes: cat.EstimateMinutes,
	}

	if getOutputFormat() == FormatJSON {
		output := CategoryDetailOutput{Category: detail}
		return writeJSON(w, output)
	}

	fmt.Fprintf(w, "Name:    %s\n", detail.Name)
	fmt.Fprintf(w, "ID:      %s\n", detail.ID)
	fmt.Fprintf(w, "Tasks:   %d\n", detail.TaskCount)
	if detail.EstimateMinutes > 0 {
		fmt.Fprintf(w, "Estimate: %s\n", formatDuration(detail.EstimateMinutes))
	}
	fmt.Fprintf(w, "Created: %s\n", detail.CreatedAt)
	if detail.UpdatedAt != "" {
		fmt.Fprintf(w, "Updated: %s\n", detail.UpdatedAt)
	}
	return nil
}

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func formatDuration(minutes int) string {
	if minutes < 60 {
		return fmt.Sprintf("%dm", minutes)
	}
	hours := minutes / 60
	mins := minutes % 60
	if mins == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dh%dm", hours, mins)
}

func writeSuccess(w io.Writer, message string) {
	if !isQuiet() {
		fmt.Fprintln(w, message)
	}
}
