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

func newCategoriesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "categories",
		Aliases: []string{"cs"},
		Short:   "List categories",
		RunE: func(cmd *cobra.Command, args []string) error {
			return viewProject(projectSelector(nil), func(project domain.Project) error {
				return writeCategories(cmd.OutOrStdout(), project)
			})
		},
	}
	return cmd
}

const categoryIdentifierHelp = "The <category> argument accepts a category name, a path parent/name for a sub-category, or an ID (full or 4+ char prefix)."

func newCategoryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "category",
		Short: "Manage categories",
		Long:  "Manage categories within a project.\n\n" + categoryIdentifierHelp,
	}

	cmd.AddCommand(newCategoryShowCmd())
	cmd.AddCommand(newCategoryAddCmd())
	cmd.AddCommand(newCategoryEditCmd())
	cmd.AddCommand(newCategoryDeleteCmd())
	cmd.AddCommand(newCategoryShipCmd(true))
	cmd.AddCommand(newCategoryShipCmd(false))

	return cmd
}

func newCategoryShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "show <category>",
		Aliases:           []string{"c"},
		Short:             "Show category details",
		Long:              "Show category details.\n\n" + categoryIdentifierHelp,
		Args:              exactArgs("category"),
		ValidArgsFunction: completeCategories,
		RunE: func(cmd *cobra.Command, args []string) error {
			return viewProject(projectSelector(nil), func(project domain.Project) error {
				_, idx, err := resolveCategory(project, args[0])
				if err != nil {
					return err
				}
				return writeCategoryDetail(cmd.OutOrStdout(), project, idx)
			})
		},
	}
	return cmd
}

func newCategoryAddCmd() *cobra.Command {
	var parent string

	cmd := &cobra.Command{
		Use:     "add <name>",
		Aliases: []string{"ca"},
		Short:   "Add a category",
		Long:    "Add a category, or with --parent a sub-category of a top-level one.\n\n" + categoryIdentifierHelp,
		Args:    exactArgs("name"),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			var created domain.Category
			err := withProject(projectSelector(nil), func(_ *data.Store, project *domain.Project) error {
				parentID := ""
				if parent != "" {
					p, _, err := resolveCategory(*project, parent)
					if err != nil {
						return err
					}
					if p.ParentID != "" {
						return fmt.Errorf("category %q is a sub-category: nesting goes one level deep", p.Name)
					}
					parentID = p.ID
				}
				cat, err := operations.CreateCategory(project, parentID, name)
				if err != nil {
					if errors.Is(err, domain.ErrDuplicateCategoryName) {
						return fmt.Errorf("category %q already exists", strings.TrimSpace(name))
					}
					return err
				}
				created = cat
				return nil
			})
			if err != nil {
				return err
			}
			writeSuccess(cmd.OutOrStdout(), fmt.Sprintf("Created category: %s (%s)", created.Name, created.ID))
			return nil
		},
	}

	cmd.Flags().StringVar(&parent, "parent", "", "add it under this top-level category")
	_ = cmd.RegisterFlagCompletionFunc("parent", completeParentCategories)

	return cmd
}

func newCategoryEditCmd() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:               "edit <category>",
		Aliases:           []string{"ce"},
		Short:             "Edit category (rename)",
		Long:              "Rename a category.\n\n" + categoryIdentifierHelp,
		Args:              exactArgs("category"),
		ValidArgsFunction: completeCategories,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}

			var renamed domain.Category
			err := withProject(projectSelector(nil), func(_ *data.Store, project *domain.Project) error {
				cat, _, err := resolveCategory(*project, args[0])
				if err != nil {
					return err
				}
				renamed, err = operations.RenameCategory(project, cat.ID, name)
				if errors.Is(err, domain.ErrDuplicateCategoryName) {
					return fmt.Errorf("category %q already exists", strings.TrimSpace(name))
				}
				return err
			})
			if err != nil {
				return err
			}
			writeSuccess(cmd.OutOrStdout(), fmt.Sprintf("Renamed category to: %s", renamed.Name))
			return nil
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "new category name")

	return cmd
}

func categoryLoad(project domain.Project, i int) (rows, tasks, subs int) {
	block := project.Categories[i:domain.BlockEnd(project.Categories, i)]
	for _, c := range block {
		rows += len(c.Tasks)
	}
	return rows, project.TreeTaskCount(i), len(block) - 1
}

func plural(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}

func newCategoryDeleteCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:               "delete <category>",
		Aliases:           []string{"cd"},
		Short:             "Delete a category",
		Long:              "Delete a category.\n\n" + categoryIdentifierHelp,
		Args:              exactArgs("category"),
		ValidArgsFunction: completeCategories,
		RunE: func(cmd *cobra.Command, args []string) error {
			store, project, err := loadStoreAndProject(projectSelector(nil))
			if err != nil {
				return err
			}

			cat, idx, err := resolveCategory(project, args[0])
			if err != nil {
				return err
			}

			rows, tasks, subs := categoryLoad(project, idx)
			if (rows > 0 || subs > 0) && !force {
				var parts []string
				if subs > 0 {
					parts = append(parts, fmt.Sprintf("%d sub-categor%s", subs, plural(subs, "y", "ies")))
				}
				parts = append(parts, fmt.Sprintf("%d tasks", tasks))
				if seps := rows - tasks; seps > 0 {
					parts = append(parts, fmt.Sprintf("%d separator%s", seps, plural(seps, "", "s")))
				}
				what := parts[len(parts)-1]
				if len(parts) > 1 {
					what = strings.Join(parts[:len(parts)-1], ", ") + " and " + what
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Category %q has %s. Delete anyway? [y/N]: ", cat.Name, what)
				var response string
				if _, err := fmt.Fscanln(cmd.InOrStdin(), &response); err != nil {
					return nil
				}
				if response != "y" && response != "Y" {
					fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
					return nil
				}
			}

			// Re-resolve under the project flock so a concurrent writer can't
			// shift indices between the prompt and the actual delete, and
			// re-check the task count so the destructive guard isn't bypassed
			// by tasks added between the unlocked load and the locked delete.
			var deletedName string
			_, err = store.WithProjectLocked(project.ID, func(p *domain.Project) error {
				lockedCat, idx, rerr := resolveCategory(*p, args[0])
				if rerr != nil {
					return rerr
				}
				lockedRows, _, lockedSubs := categoryLoad(*p, idx)
				if !force && (lockedRows > rows || lockedSubs > subs) {
					return fmt.Errorf("category %q gained tasks or sub-categories since the prompt; re-run with -f to confirm", lockedCat.Name)
				}
				deletedName = lockedCat.Name
				p.RemoveCategoryBlock(idx)
				return nil
			})
			if err != nil {
				return err
			}

			writeSuccess(cmd.OutOrStdout(), fmt.Sprintf("Deleted category: %s", deletedName))
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "skip confirmation prompt")

	return cmd
}
