package cli

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/ilyasturki/phasionary/internal/domain"
)

func newCompletionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish]",
		Short: "Generate shell completion script",
		Long: `Generate shell completion script for phasionary.

Bash:
  $ source <(phasionary completion bash)
  # Persist: phasionary completion bash > /etc/bash_completion.d/phasionary

Zsh:
  $ phasionary completion zsh > "${fpath[1]}/_phasionary"

Fish:
  $ phasionary completion fish > ~/.config/fish/completions/phasionary.fish
`,
		ValidArgs:             []string{"bash", "zsh", "fish"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				return cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				return cmd.Root().GenFishCompletion(os.Stdout, true)
			}
			return nil
		},
	}
	return cmd
}

func completeProjects(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	store, err := storeFromViper()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	projects, err := store.ListProjects()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	var completions []string
	for _, p := range projects {
		completions = append(completions, p.Name)
	}
	return completions, cobra.ShellCompDirectiveNoFileComp
}

func completeTasks(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return completeTaskIDs(false)
}

func completeParentTasks(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return completeTaskIDs(true)
}

func completeTaskIDs(topLevel bool) ([]string, cobra.ShellCompDirective) {
	project, err := completionProject()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	var completions []string
	for _, cat := range project.Categories {
		for _, task := range cat.Tasks {
			if task.IsSeparator() || (topLevel && task.ParentID != "") {
				continue
			}
			completions = append(completions, task.ID)
		}
	}
	return completions, cobra.ShellCompDirectiveNoFileComp
}

func completeCategories(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return completeCategoryPaths(false)
}

func completeParentCategories(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return completeCategoryPaths(true)
}

func completeCategoryPaths(topLevel bool) ([]string, cobra.ShellCompDirective) {
	project, err := completionProject()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	var completions []string
	for i, cat := range project.Categories {
		if topLevel && cat.ParentID != "" {
			continue
		}
		completions = append(completions, cliPath(project, i))
	}
	return completions, cobra.ShellCompDirectiveNoFileComp
}

func completionProject() (domain.Project, error) {
	store, err := storeFromViper()
	if err != nil {
		return domain.Project{}, err
	}
	return store.LoadProject(projectSelector(nil))
}

func completeStatuses(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{
		domain.StatusTodo,
		domain.StatusInProgress,
		domain.StatusCompleted,
		domain.StatusCancelled,
	}, cobra.ShellCompDirectiveNoFileComp
}

func completePriorities(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return append([]string{}, domain.PriorityOrder...), cobra.ShellCompDirectiveNoFileComp
}

func completeExportFormats(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{"json", "markdown"}, cobra.ShellCompDirectiveNoFileComp
}
