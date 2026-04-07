package cmd

import (
	"fmt"
	"strings"

	"github.com/atami-ai/atami-ai/cli/internal/kbskillspull"
	"github.com/spf13/cobra"
)

func newKBSkillsPullCmd(opts Options) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Sync canonical project-kb skill files into .project-kb/skills",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			targetDir, err := opts.Getwd()
			if err != nil {
				return fmt.Errorf("determining current working directory: %w", err)
			}

			result, err := opts.PullKBSkills(kbskillspull.Options{
				TargetDir: targetDir,
				Force:     force,
			})
			if err != nil {
				return err
			}

			_, _ = fmt.Fprint(cmd.OutOrStdout(), formatKBSkillsPullSuccess(result))
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Overwrite locally edited synced skill files")
	return cmd
}

func formatKBSkillsPullSuccess(result kbskillspull.Result) string {
	var builder strings.Builder

	_, _ = fmt.Fprintf(
		&builder,
		"✓ Project-kb skills sync complete in %s\n\nUpdated: %d\nUnchanged: %d\nSkipped: %d\n",
		result.TargetDir,
		len(result.UpdatedFiles),
		len(result.UnchangedFiles),
		len(result.SkippedFiles),
	)

	if len(result.UpdatedFiles) > 0 {
		builder.WriteString("\nUpdated:\n")
		for _, path := range result.UpdatedFiles {
			_, _ = fmt.Fprintf(&builder, "  %s\n", path)
		}
	}

	if len(result.UnchangedFiles) > 0 {
		builder.WriteString("\nUnchanged:\n")
		for _, path := range result.UnchangedFiles {
			_, _ = fmt.Fprintf(&builder, "  %s\n", path)
		}
	}

	if len(result.SkippedFiles) > 0 {
		builder.WriteString("\nSkipped:\n")
		for _, skipped := range result.SkippedFiles {
			_, _ = fmt.Fprintf(&builder, "  %s (%s)\n", skipped.Path, skipped.Reason)
		}
		builder.WriteString("\nNext step:\n")
		builder.WriteString("  Re-run `atami kb skills pull --force` to overwrite skipped locally edited synced files.\n")
	}

	builder.WriteString("\nLeft untouched:\n")
	builder.WriteString("  .project-kb/skills/overrides/\n")

	return builder.String()
}
