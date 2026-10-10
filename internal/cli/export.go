package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"phasionary/internal/domain"
	"phasionary/internal/export"
)

func newExportCmd() *cobra.Command {
	var (
		format string
		output string
	)

	cmd := &cobra.Command{
		Use:     "export",
		Aliases: []string{"x"},
		Short:   "Export project to markdown or JSON",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := storeFromViper()
			if err != nil {
				return err
			}
			project, err := store.LoadProject(projectSelector(nil))
			if err != nil {
				return err
			}

			var w io.Writer = cmd.OutOrStdout()
			if output != "" {
				f, err := os.Create(output)
				if err != nil {
					return fmt.Errorf("failed to create output file: %w", err)
				}
				defer f.Close()
				w = f
			}

			format = strings.ToLower(format)
			switch format {
			case "json":
				enc := json.NewEncoder(w)
				enc.SetIndent("", "  ")
				if err := enc.Encode(project); err != nil {
					return err
				}
			case "markdown", "md":
				if err := export.ExportMarkdown(project, w); err != nil {
					return err
				}
			default:
				return fmt.Errorf("unsupported format: %s (use json or markdown)", format)
			}

			if output != "" {
				writeSuccess(cmd.OutOrStdout(), fmt.Sprintf("Exported to %s", output))
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&format, "format", "f", "markdown", "output format: json or markdown")
	cmd.Flags().StringVarP(&output, "output", "o", "", "output file path (defaults to stdout)")

	_ = cmd.RegisterFlagCompletionFunc("format", completeExportFormats)

	return cmd
}

func newImportCmd() *cobra.Command {
	var (
		format string
		name   string
	)

	cmd := &cobra.Command{
		Use:     "import <file>",
		Aliases: []string{"im"},
		Short:   "Import project from markdown or JSON",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			inputPath := args[0]

			f, err := os.Open(inputPath)
			if err != nil {
				return fmt.Errorf("failed to open input file: %w", err)
			}
			defer f.Close()

			if format == "" {
				ext := strings.ToLower(filepath.Ext(inputPath))
				switch ext {
				case ".json":
					format = "json"
				case ".md", ".markdown":
					format = "markdown"
				default:
					return fmt.Errorf("cannot determine format from extension %q, use --format", ext)
				}
			}

			store, err := storeFromViper()
			if err != nil {
				return err
			}

			format = strings.ToLower(format)
			switch format {
			case "json":
				project, err := importJSON(f, name)
				if err != nil {
					return err
				}
				saved, err := store.ImportProject(project)
				if err != nil {
					return err
				}
				writeSuccess(cmd.OutOrStdout(), fmt.Sprintf("Imported project: %s (%s)", saved.Name, saved.ID))
			case "markdown", "md":
				project, err := export.ImportMarkdown(f, name)
				if err != nil {
					return err
				}
				saved, err := store.ImportProject(project)
				if err != nil {
					return err
				}
				writeSuccess(cmd.OutOrStdout(), fmt.Sprintf("Imported project: %s (%s)", saved.Name, saved.ID))
			default:
				return fmt.Errorf("unsupported format: %s (use json or markdown)", format)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&format, "format", "f", "", "input format: json or markdown (auto-detected from extension)")
	cmd.Flags().StringVarP(&name, "name", "n", "", "override project name")

	_ = cmd.RegisterFlagCompletionFunc("format", completeExportFormats)

	return cmd
}

func importJSON(r io.Reader, overrideName string) (domain.Project, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return domain.Project{}, err
	}

	var p domain.Project
	if err := json.Unmarshal(data, &p); err != nil {
		return domain.Project{}, fmt.Errorf("invalid JSON: %w", err)
	}
	if err := domain.CheckSchema(p); err != nil {
		return domain.Project{}, err
	}

	if overrideName != "" {
		p.Name = overrideName
	}
	for i := range p.Categories {
		p.Categories[i].Name = domain.UniqueCategoryName(p.Categories[:i], p.Categories[i].ParentID, p.Categories[i].Name)
	}

	return p, nil
}
