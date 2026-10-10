package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"phasionary/internal/data"
	"phasionary/internal/domain"
	"phasionary/internal/operations"
)

func newTasksCmd() *cobra.Command {
	var (
		status   string
		category string
		priority string
	)

	cmd := &cobra.Command{
		Use:     "tasks",
		Aliases: []string{"ts"},
		Short:   "List tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			if status != "" {
				if err := domain.ValidateStatus(status); err != nil {
					return err
				}
			}
			if priority != "" {
				if err := domain.ValidatePriority(priority); err != nil {
					return err
				}
			}

			return viewProject(projectSelector(nil), func(project domain.Project) error {
				lo, hi := 0, len(project.Categories)
				if category != "" {
					_, idx, err := resolveCategory(project, category)
					if err != nil {
						return err
					}
					lo, hi = idx, domain.BlockEnd(project.Categories, idx)
				}
				var tasks []TaskListItem
				for ci := lo; ci < hi; ci++ {
					cat := project.Categories[ci]
					for _, task := range cat.Tasks {
						if task.IsSeparator() {
							continue
						}
						if status != "" && task.Status != status {
							continue
						}
						if priority != "" && task.Priority != priority {
							continue
						}
						tasks = append(tasks, TaskListItem{
							ID:              task.ID,
							ParentID:        task.ParentID,
							Title:           task.Title,
							Status:          task.Status,
							Priority:        task.Priority,
							Category:        cliPath(project, ci),
							EstimateMinutes: task.EstimateMinutes,
							TagColor:        task.TagColor,
						})
					}
				}

				return writeTaskList(cmd.OutOrStdout(), tasks)
			})
		},
	}

	cmd.Flags().StringVarP(&status, "status", "s", "", "filter by status (todo, in_progress, completed, cancelled)")
	cmd.Flags().StringVarP(&category, "category", "C", "", "filter by category, its sub-categories included: name, parent/name or ID")
	cmd.Flags().StringVar(&priority, "priority", "", "filter by priority (critical, high, medium, low, trivial)")

	_ = cmd.RegisterFlagCompletionFunc("status", completeStatuses)
	_ = cmd.RegisterFlagCompletionFunc("category", completeCategories)
	_ = cmd.RegisterFlagCompletionFunc("priority", completePriorities)

	return cmd
}

const taskIdentifierHelp = "The <task> argument accepts a task title or ID (full or 4+ char prefix)."

func newTaskCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "task",
		Short: "Manage tasks",
		Long:  "Manage tasks within a project.\n\n" + taskIdentifierHelp,
	}

	cmd.AddCommand(newTaskShowCmd())
	cmd.AddCommand(newTaskAddCmd())
	cmd.AddCommand(newTaskEditCmd())
	cmd.AddCommand(newTaskDeleteCmd())
	cmd.AddCommand(newTaskStatusCmd())
	cmd.AddCommand(newTaskPriorityCmd())
	cmd.AddCommand(newTaskMoveCmd())

	return cmd
}

func newTaskShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "show <task>",
		Aliases:           []string{"t"},
		Short:             "Show task details",
		Long:              "Show task details.\n\n" + taskIdentifierHelp,
		Args:              exactArgs("task"),
		ValidArgsFunction: completeTasks,
		RunE: func(cmd *cobra.Command, args []string) error {
			return viewProject(projectSelector(nil), func(project domain.Project) error {
				ref, err := resolveTask(project, args[0])
				if err != nil {
					return fmt.Errorf("task %q not found", args[0])
				}
				return writeTaskDetail(cmd.OutOrStdout(), project, ref.CategoryIndex, ref.TaskIndex)
			})
		},
	}
	return cmd
}

func newTaskAddCmd() *cobra.Command {
	var (
		categoryName string
		parent       string
		priority     string
		estimate     string
		description  string
	)

	cmd := &cobra.Command{
		Use:     "add <title>",
		Aliases: []string{"ta"},
		Short:   "Add a task",
		Long:    "Add a task to a category, or with --parent a subtask of a top-level task.\n\n" + categoryIdentifierHelp,
		Args:    exactArgs("title"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(categoryName) == "" && strings.TrimSpace(parent) == "" {
				return errors.New("--category is required")
			}
			if strings.TrimSpace(args[0]) == "" {
				return errors.New("title is required")
			}

			minutes, err := operations.ParseEstimate(estimate)
			if err != nil {
				return err
			}

			var created domain.Task
			err = withProject(projectSelector(nil), func(_ *data.Store, project *domain.Project) error {
				catID, parentID, err := taskDestination(*project, categoryName, parent)
				if err != nil {
					return err
				}
				created, err = operations.CreateTask(project, catID, operations.TaskFields{
					Title:       args[0],
					Priority:    priority,
					Estimate:    minutes,
					Description: description,
					ParentID:    parentID,
				})
				if errors.Is(err, domain.ErrNestDepth) {
					return fmt.Errorf("task %q is a subtask: nesting goes one level deep", parent)
				}
				return err
			})
			if err != nil {
				return err
			}
			writeSuccess(cmd.OutOrStdout(), fmt.Sprintf("Created task: %s (%s)", created.Title, created.ID))
			return nil
		},
	}

	cmd.Flags().StringVarP(&categoryName, "category", "C", "", "category: name, parent/name or ID (required without --parent)")
	cmd.Flags().StringVar(&parent, "parent", "", "add it as a subtask of this task: title or ID")
	cmd.Flags().StringVar(&priority, "priority", "", "priority: critical|high|medium|low|trivial")
	cmd.Flags().StringVarP(&estimate, "estimate", "e", "", "time estimate: 30, 2h, 1.5h, 2h30m")
	cmd.Flags().StringVar(&description, "description", "", "description, may span several lines")

	_ = cmd.RegisterFlagCompletionFunc("category", completeCategories)
	_ = cmd.RegisterFlagCompletionFunc("parent", completeTasks)
	_ = cmd.RegisterFlagCompletionFunc("priority", completePriorities)

	return cmd
}

// taskDestination resolves where task add puts a task: the category -C names,
// or the parent task's, which -C then has to agree with.
func taskDestination(project domain.Project, categoryName, parent string) (catID, parentID string, err error) {
	if parent == "" {
		cat, _, err := resolveCategory(project, categoryName)
		if err != nil {
			return "", "", err
		}
		return cat.ID, "", nil
	}
	ref, err := resolveTask(project, parent)
	if err != nil {
		return "", "", fmt.Errorf("task %q not found", parent)
	}
	cat := project.Categories[ref.CategoryIndex]
	if strings.TrimSpace(categoryName) != "" {
		named, _, err := resolveCategory(project, categoryName)
		if err != nil {
			return "", "", err
		}
		if named.ID != cat.ID {
			return "", "", fmt.Errorf("task %q is in category %q, not %q", ref.Task.Title, cliPath(project, ref.CategoryIndex), categoryName)
		}
	}
	return cat.ID, ref.Task.ID, nil
}

func newTaskEditCmd() *cobra.Command {
	var (
		title       string
		priority    string
		estimate    string
		description string
	)

	cmd := &cobra.Command{
		Use:               "edit <task>",
		Aliases:           []string{"te"},
		Short:             "Edit task properties",
		Long:              "Edit task properties.\n\n" + taskIdentifierHelp,
		Args:              exactArgs("task"),
		ValidArgsFunction: completeTasks,
		RunE: func(cmd *cobra.Command, args []string) error {
			var updatedTitle string
			err := withProject(projectSelector(nil), func(_ *data.Store, project *domain.Project) error {
				ref, err := resolveTask(*project, args[0])
				if err != nil {
					return fmt.Errorf("task %q not found", args[0])
				}

				var u operations.TaskUpdate
				if title != "" {
					u.Title = &title
				}
				if priority != "" {
					u.Priority = &priority
				}
				if estimate != "" {
					minutes, err := operations.ParseEstimate(estimate)
					if err != nil {
						return err
					}
					u.Estimate = &minutes
				}
				if cmd.Flags().Changed("description") {
					u.Description = &description
				}
				updated, err := operations.UpdateTask(project, project.Categories[ref.CategoryIndex].ID, ref.Task.ID, u)
				if err != nil && !errors.Is(err, operations.ErrNoChange) {
					return err
				}
				updatedTitle = updated.Title
				return nil
			})
			if err != nil {
				return err
			}
			writeSuccess(cmd.OutOrStdout(), fmt.Sprintf("Updated task: %s", updatedTitle))
			return nil
		},
	}

	cmd.Flags().StringVarP(&title, "title", "t", "", "new title")
	cmd.Flags().StringVar(&priority, "priority", "", "priority: critical|high|medium|low|trivial")
	cmd.Flags().StringVarP(&estimate, "estimate", "e", "", "time estimate: 30, 2h, 1.5h, 2h30m")
	cmd.Flags().StringVar(&description, "description", "", `new description, "" clears it`)

	_ = cmd.RegisterFlagCompletionFunc("priority", completePriorities)

	return cmd
}

func newTaskDeleteCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:               "delete <task>",
		Aliases:           []string{"td"},
		Short:             "Delete a task",
		Long:              "Delete a task.\n\n" + taskIdentifierHelp,
		Args:              exactArgs("task"),
		ValidArgsFunction: completeTasks,
		RunE: func(cmd *cobra.Command, args []string) error {
			store, project, err := loadStoreAndProject(projectSelector(nil))
			if err != nil {
				return err
			}

			ref, err := resolveTask(project, args[0])
			if err != nil {
				return fmt.Errorf("task %q not found", args[0])
			}

			subtasks := domain.BlockEnd(project.Categories[ref.CategoryIndex].Tasks, ref.TaskIndex) - ref.TaskIndex - 1
			if !force {
				what := fmt.Sprintf("task %q", ref.Task.Title)
				if subtasks > 0 {
					what += fmt.Sprintf(" and its %d subtask%s", subtasks, plural(subtasks, "", "s"))
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Delete %s? [y/N]: ", what)
				var response string
				if _, err := fmt.Fscanln(cmd.InOrStdin(), &response); err != nil {
					return nil
				}
				if response != "y" && response != "Y" {
					fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
					return nil
				}
			}

			title := ref.Task.Title
			// Re-resolve under the project flock so a concurrent writer can't
			// shift indices between the prompt and the actual delete.
			_, err = store.WithProjectLocked(project.ID, func(p *domain.Project) error {
				r, err := resolveTask(*p, args[0])
				if err != nil {
					return fmt.Errorf("task %q not found", args[0])
				}
				cat := &p.Categories[r.CategoryIndex]
				if !force && domain.BlockEnd(cat.Tasks, r.TaskIndex)-r.TaskIndex-1 > subtasks {
					return fmt.Errorf("task %q gained subtasks since the prompt; re-run with -f to confirm", r.Task.Title)
				}
				cat.RemoveTaskBlock(r.TaskIndex)
				return nil
			})
			if err != nil {
				return err
			}

			writeSuccess(cmd.OutOrStdout(), fmt.Sprintf("Deleted task: %s", title))
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "skip confirmation prompt")

	return cmd
}

func newTaskStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "status <task> <status>",
		Aliases: []string{"tst"},
		Short:   "Update task status",
		Long:    "Update task status.\n\n" + taskIdentifierHelp,
		Args:    exactArgs("task", "status"),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return completeTasks(cmd, args, toComplete)
			}
			return completeStatuses(cmd, args, toComplete)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			selector := args[0]
			status := args[1]

			if err := domain.ValidateStatus(status); err != nil {
				return err
			}

			var title string
			err := withProject(projectSelector(nil), func(_ *data.Store, project *domain.Project) error {
				ref, err := resolveTask(*project, selector)
				if err != nil {
					return fmt.Errorf("task %q not found", selector)
				}
				task := &project.Categories[ref.CategoryIndex].Tasks[ref.TaskIndex]
				if err := task.SetStatus(status); err != nil {
					return err
				}
				title = task.Title
				return nil
			})
			if err != nil {
				return err
			}
			writeSuccess(cmd.OutOrStdout(), fmt.Sprintf("Updated task %s to %s", title, status))
			return nil
		},
	}
	return cmd
}

func newTaskPriorityCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "priority <task> <priority>",
		Aliases: []string{"tp"},
		Short:   "Update task priority",
		Long:    "Update task priority.\n\n" + taskIdentifierHelp,
		Args:    exactArgs("task", "priority"),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return completeTasks(cmd, args, toComplete)
			}
			return completePriorities(cmd, args, toComplete)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			selector := args[0]
			priority := args[1]

			if err := domain.ValidatePriority(priority); err != nil {
				return err
			}

			var title string
			err := withProject(projectSelector(nil), func(_ *data.Store, project *domain.Project) error {
				ref, err := resolveTask(*project, selector)
				if err != nil {
					return fmt.Errorf("task %q not found", selector)
				}
				task := &project.Categories[ref.CategoryIndex].Tasks[ref.TaskIndex]
				if err := task.SetPriority(priority); err != nil {
					return err
				}
				title = task.Title
				return nil
			})
			if err != nil {
				return err
			}
			writeSuccess(cmd.OutOrStdout(), fmt.Sprintf("Updated task %s priority to %s", title, priority))
			return nil
		},
	}
	return cmd
}

func newTaskMoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "move <task> <category>",
		Aliases: []string{"tm"},
		Short:   "Move task to different category",
		Long:    "Move task to a different category.\n\n" + taskIdentifierHelp + "\n" + categoryIdentifierHelp,
		Args:    exactArgs("task", "category"),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return completeTasks(cmd, args, toComplete)
			}
			return completeCategories(cmd, args, toComplete)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			selector := args[0]
			targetCategory := args[1]

			var title, destName string
			err := withProject(projectSelector(nil), func(_ *data.Store, project *domain.Project) error {
				ref, err := resolveTask(*project, selector)
				if err != nil {
					return fmt.Errorf("task %q not found", selector)
				}
				_, dstCatIdx, err := resolveCategory(*project, targetCategory)
				if err != nil {
					return err
				}
				if ref.CategoryIndex == dstCatIdx {
					return fmt.Errorf("task is already in category %q", targetCategory)
				}
				title = ref.Task.Title
				if err := project.MoveTask(ref.CategoryIndex, ref.TaskIndex, dstCatIdx); err != nil {
					return err
				}
				destName = project.Categories[dstCatIdx].Name
				return nil
			})
			if err != nil {
				return err
			}
			writeSuccess(cmd.OutOrStdout(), fmt.Sprintf("Moved task %s to %s", title, destName))
			return nil
		},
	}
	return cmd
}
