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
				return writeCategories(cmd.OutOrStdout(), project.Categories)
			})
		},
	}
	return cmd
}

const categoryIdentifierHelp = "The <category> argument accepts a category name or ID (full or 4+ char prefix)."

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
				cat, _, err := resolveCategory(project, args[0])
				if err != nil {
					return err
				}
				return writeCategoryDetail(cmd.OutOrStdout(), *cat)
			})
		},
	}
	return cmd
}

func newCategoryAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "add <name>",
		Aliases: []string{"ca"},
		Short:   "Add a category",
		Args:    exactArgs("name"),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			var created domain.Category
			err := withProject(projectSelector(nil), func(_ *data.Store, project *domain.Project) error {
				cat, err := operations.CreateCategory(project, name)
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

			cat, _, err := resolveCategory(project, args[0])
			if err != nil {
				return err
			}

			if len(cat.Tasks) > 0 && !force {
				fmt.Fprintf(cmd.OutOrStdout(), "Category %q has %d tasks. Delete anyway? [y/N]: ", cat.Name, len(cat.Tasks))
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
				if !force && len(lockedCat.Tasks) > len(cat.Tasks) {
					return fmt.Errorf("category %q gained %d task(s) since the prompt; re-run with -f to confirm", lockedCat.Name, len(lockedCat.Tasks)-len(cat.Tasks))
				}
				deletedName = lockedCat.Name
				return p.RemoveCategory(idx)
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
