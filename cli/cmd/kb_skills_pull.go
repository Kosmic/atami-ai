package cmd

import (
	"fmt"
	"strings"

	"github.com/atami-ai/atami-ai/cli/internal/kbskillspull"
	"github.com/spf13/cobra"
)

func newKBSkillsPullCmd(opts Options) *cobra.Command {
	return &cobra.Command{
		Use:   "pull",
		Short: "Sync canonical project-kb skill files into .project-kb/skills",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			targetDir, err := opts.Getwd()
			if err != nil {
				return fmt.Errorf("determining current working directory: %w", err)
			}

			result, err := opts.PullKBSkills(kbskillspull.Options{TargetDir: targetDir})
			if err != nil {
				return err
			}

			_, _ = fmt.Fprint(cmd.OutOrStdout(), formatKBSkillsPullSuccess(result))
			return nil
		},
	}
}

func formatKBSkillsPullSuccess(result kbskillspull.Result) string {
	var builder strings.Builder

	_, _ = fmt.Fprintf(&builder, "✓ Synced %d project-kb skill files into .project-kb/skills/ in %s\n", len(result.SyncedFiles), result.TargetDir)
	if len(result.SyncedFiles) > 0 {
		builder.WriteString("\nUpdated:\n")
		for _, path := range result.SyncedFiles {
			_, _ = fmt.Fprintf(&builder, "  %s\n", path)
		}
	}

	builder.WriteString("\nLeft untouched:\n")
	builder.WriteString("  .project-kb/skills/overrides/\n")

	return builder.String()
}
