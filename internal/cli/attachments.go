package cli

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ilyasturki/ordo/internal/attachments"
	"github.com/ilyasturki/ordo/internal/config"
	"github.com/ilyasturki/ordo/internal/data"
	"github.com/ilyasturki/ordo/internal/domain"
	"github.com/ilyasturki/ordo/internal/operations"
)

type PruneItem struct {
	File     string `json:"file"`
	Size     int64  `json:"size"`
	Modified string `json:"modified"`
}

type PruneOutput struct {
	Removed bool        `json:"removed"`
	Files   []PruneItem `json:"files"`
}

func newAttachmentsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "attachments",
		Short: "Manage the images tasks carry",
	}
	cmd.AddCommand(newAttachmentsPruneCmd())
	cmd.AddCommand(newAttachmentsPathCmd())
	return cmd
}

func attachmentStore() (*attachments.Store, error) {
	dataDir, err := config.ResolveDataDir(viper.GetString("data"))
	if err != nil {
		return nil, err
	}
	return attachments.NewStore(dataDir), nil
}

func newTaskAttachCmd() *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "attach <task> <file|->",
		Short: "Attach an image to a task",
		Long: "Copy a png, jpeg, gif or webp image into projects/attachments/ and reference it at the end of\n" +
			"the task's description. - reads the image from stdin:\n\n" +
			"  wl-paste -t image/png | ordo task attach <task> -\n\n" + taskIdentifierHelp,
		Args: exactArgs("task", "file"),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return completeTasks(cmd, args, toComplete)
			}
			return nil, cobra.ShellCompDirectiveDefault
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := attachmentStore()
			if err != nil {
				return err
			}
			// Read before taking the project lock: stdin can take its time.
			image, defaultName, err := readAttachment(cmd.InOrStdin(), args[1])
			if err != nil {
				return err
			}
			if strings.TrimSpace(name) == "" {
				name = defaultName
			}
			var title, path string
			err = withProject(projectSelector(nil), func(_ *data.Store, project *domain.Project) error {
				ref, err := resolveTask(*project, args[0])
				if err != nil {
					return fmt.Errorf("task %q not found", args[0])
				}
				target, err := store.Add(image)
				if err != nil {
					return err
				}
				description := strings.TrimRight(ref.Task.Description, "\n")
				if description != "" {
					description += "\n"
				}
				description += attachments.Line(name, target)
				if _, err := operations.UpdateTask(project, project.Categories[ref.CategoryIndex].ID, ref.Task.ID,
					operations.TaskUpdate{Description: &description}); err != nil {
					return err
				}
				title = ref.Task.Title
				path, _ = store.Path(target)
				return nil
			})
			if err != nil {
				return err
			}
			writeSuccess(cmd.OutOrStdout(), fmt.Sprintf("Attached %s to %s: %s", name, title, path))
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "the image's name in the description (default: the file's name)")
	_ = cmd.RegisterFlagCompletionFunc("name", cobra.NoFileCompletions)
	return cmd
}

func readAttachment(stdin io.Reader, arg string) (data []byte, name string, err error) {
	if arg != "-" {
		data, err = attachments.ReadImage(arg)
		return data, filepath.Base(arg), err
	}
	data, err = io.ReadAll(io.LimitReader(stdin, attachments.MaxSize+1))
	if err != nil {
		return nil, "", err
	}
	if len(data) > attachments.MaxSize {
		return nil, "", attachments.ErrTooLarge
	}
	if attachments.Sniff(data) == "" {
		return nil, "", attachments.ErrNotImage
	}
	return data, attachments.ScreenshotName(time.Now()), nil
}

func newAttachmentsPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the directory the images are stored in",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := attachmentStore()
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), store.Dir())
			return nil
		},
	}
}

func newAttachmentsPruneCmd() *cobra.Command {
	var yes bool
	days := int(attachments.PruneAge / (24 * time.Hour))
	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Remove images no project references any more",
		Long: fmt.Sprintf("List the images in projects/attachments/ that no project file names, conflict copies\n"+
			"included, and that are older than %d days; --yes removes them.\n\n"+
			"Removing an image from a task never deletes its file: undo, a cut waiting to be pasted or\n"+
			"another device can bring the reference back. Syncthing carries a delete to every device,\n"+
			"so prune leaves a file younger than %d days alone, in case another device has just\n"+
			"referenced it in an edit this one hasn't received.", days, days),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := attachmentStore()
			if err != nil {
				return err
			}
			orphans, err := store.Unreferenced(time.Now())
			if err != nil {
				return err
			}
			if yes {
				if err := store.Remove(orphans); err != nil {
					return err
				}
			}
			return writePrune(cmd.OutOrStdout(), orphans, yes, days)
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "remove the images instead of listing them")
	return cmd
}

func writePrune(w io.Writer, orphans []attachments.Orphan, removed bool, days int) error {
	items := make([]PruneItem, len(orphans))
	var total int64
	for i, o := range orphans {
		items[i] = PruneItem{File: o.Name, Size: o.Size, Modified: o.ModTime.UTC().Format(time.RFC3339)}
		total += o.Size
	}
	if getOutputFormat() == FormatJSON {
		return writeJSON(w, PruneOutput{Removed: removed, Files: items})
	}
	if len(orphans) == 0 {
		writeSuccess(w, fmt.Sprintf("No unreferenced image older than %d days", days))
		return nil
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, o := range orphans {
		fmt.Fprintf(tw, "%s\t%s\t%s\n", o.Name, attachments.FormatSize(o.Size), o.ModTime.Local().Format(time.DateOnly))
	}
	if err := tw.Flush(); err != nil {
		return err
	}
	summary := fmt.Sprintf("%d %s, %s", len(orphans), plural(len(orphans), "image", "images"), attachments.FormatSize(total))
	if removed {
		writeSuccess(w, "Removed "+summary)
	} else {
		writeSuccess(w, "Would remove "+summary+": run again with --yes to remove them")
	}
	return nil
}
