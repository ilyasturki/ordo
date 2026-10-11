package cli

import (
	"errors"
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/ilyasturki/phasionary/internal/data"
	"github.com/ilyasturki/phasionary/internal/domain"
	"github.com/ilyasturki/phasionary/internal/operations"
)

type VersionListItem struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	ReleasedAt string `json:"released_at,omitempty"`
	TaskCount  int    `json:"task_count"`
	Todo       int    `json:"todo"`
	InProgress int    `json:"in_progress"`
	Completed  int    `json:"completed"`
	Cancelled  int    `json:"cancelled"`
}

type VersionsOutput struct {
	Versions []VersionListItem `json:"versions"`
}

func newVersionsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "versions",
		Short: "List the project's versions with their progress and ship dates",
		Long: "List the categories named after a version (vX.Y.Z, a label may follow) in project order,\n" +
			"with their task counts and the date each was shipped.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return viewProject(projectSelector(nil), func(project domain.Project) error {
				return writeVersions(cmd.OutOrStdout(), project)
			})
		},
	}
}

func versionItems(project domain.Project) []VersionListItem {
	items := []VersionListItem{}
	for _, i := range project.VersionIndexes() {
		c := project.Categories[i]
		counts := project.TreeStatusCounts(i)
		items = append(items, VersionListItem{
			ID:         c.ID,
			Name:       c.Name,
			ReleasedAt: c.ReleasedAt,
			TaskCount:  counts.Total(),
			Todo:       counts.Todo,
			InProgress: counts.InProgress,
			Completed:  counts.Completed,
			Cancelled:  counts.Cancelled,
		})
	}
	return items
}

func writeVersions(w io.Writer, project domain.Project) error {
	items := versionItems(project)
	if getOutputFormat() == FormatJSON {
		return writeJSON(w, VersionsOutput{Versions: items})
	}
	if len(items) == 0 {
		if !isQuiet() {
			fmt.Fprintln(w, "No versions found. A version is a category named vX.Y.Z.")
		}
		return nil
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if isLong() {
		fmt.Fprintln(tw, "NAME\tID\tDONE\tIN PROGRESS\tSHIPPED")
	} else {
		fmt.Fprintln(tw, "NAME\tDONE\tIN PROGRESS\tSHIPPED")
	}
	for _, v := range items {
		done := fmt.Sprintf("%d/%d", v.Completed+v.Cancelled, v.TaskCount)
		shipped := "-"
		if v.ReleasedAt != "" {
			shipped = shipDay(v.ReleasedAt)
			if open := v.Todo + v.InProgress; open > 0 {
				shipped += fmt.Sprintf(" (%d open)", open)
			}
		}
		if isLong() {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%d\t%s\n", v.Name, v.ID, done, v.InProgress, shipped)
		} else {
			fmt.Fprintf(tw, "%s\t%s\t%d\t%s\n", v.Name, done, v.InProgress, shipped)
		}
	}
	return tw.Flush()
}

func shipDay(ts string) string {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return ts
	}
	return t.Local().Format(time.DateOnly)
}

func newCategoryShipCmd(ship bool) *cobra.Command {
	use, short, verb := "ship <category>", "Mark a version shipped, stamping today's date", "Shipped"
	change := operations.ShipVersion
	if !ship {
		use, short, verb = "unship <category>", "Reopen a shipped version, clearing its ship date", "Reopened"
		change = operations.UnshipVersion
	}
	return &cobra.Command{
		Use:               use,
		Short:             short,
		Long:              short + ".\n\n" + categoryIdentifierHelp,
		Args:              exactArgs("category"),
		ValidArgsFunction: completeCategories,
		RunE: func(cmd *cobra.Command, args []string) error {
			var changed domain.Category
			err := withProject(projectSelector(nil), func(_ *data.Store, project *domain.Project) error {
				cat, _, err := resolveCategory(*project, args[0])
				if err != nil {
					return err
				}
				changed, err = change(project, cat.ID)
				switch {
				case errors.Is(err, domain.ErrNotVersion):
					return fmt.Errorf("category %q is not a version: its name must start with vX.Y.Z", cat.Name)
				case errors.Is(err, domain.ErrAlreadyShipped):
					return fmt.Errorf("version %q already shipped on %s", cat.Name, shipDay(cat.ReleasedAt))
				case errors.Is(err, domain.ErrNotShipped):
					return fmt.Errorf("version %q is not shipped", cat.Name)
				}
				return err
			})
			if err != nil {
				return err
			}
			writeSuccess(cmd.OutOrStdout(), fmt.Sprintf("%s version: %s", verb, changed.Name))
			return nil
		},
	}
}
