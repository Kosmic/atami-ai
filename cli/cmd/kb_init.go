package cmd

import (
	"fmt"
	"strings"

	"github.com/atami-ai/atami-ai/cli/internal/kbinit"
	"github.com/spf13/cobra"
)

func newKBInitCmd(opts Options) *cobra.Command {
	var (
		name           string
		description    string
		force          bool
		templateSource string
	)

	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialise .project-kb in the current directory",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			targetDir, err := opts.Getwd()
			if err != nil {
				return fmt.Errorf("determining current working directory: %w", err)
			}

			result, err := kbinit.Initialize(kbinit.Options{
				TargetDir:      targetDir,
				Name:           name,
				Description:    description,
				Force:          force,
				TemplateSource: templateSource,
				ResolvePaths:   opts.ResolvePaths,
			})
			if err != nil {
				return err
			}

			for _, warning := range result.Warnings {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Warning: %s\n", warning)
			}

			_, _ = fmt.Fprint(cmd.OutOrStdout(), formatInitSuccess(result))
			return nil
		},
	}

	initCmd.Flags().StringVar(&name, "name", "", "Project name to write into .project-kb/kb-config.yaml")
	initCmd.Flags().StringVar(&description, "description", "", "Project description to write into .project-kb/kb-config.yaml")
	initCmd.Flags().BoolVar(&force, "force", false, "Overwrite an existing .project-kb directory")
	initCmd.Flags().StringVar(&templateSource, "template-source", "", "Explicit path to the atami-ai project-kb template directory")

	return initCmd
}

func formatInitSuccess(result kbinit.Result) string {
	var builder strings.Builder

	_, _ = fmt.Fprintf(&builder, "✓ Initialised .project-kb in %s\n\n", result.TargetDir)
	builder.WriteString("Created:\n")
	for _, path := range result.CreatedPaths {
		_, _ = fmt.Fprintf(&builder, "  %s\n", path)
	}

	_, _ = fmt.Fprintf(&builder, "\nSynced %d skill files into .project-kb/skills/\n\n", len(result.SyncedSkillFiles))
	builder.WriteString(formatAgentsMDStatus(result))
	builder.WriteString("\n\nNext steps:\n")
	builder.WriteString("  1. Edit .project-kb/kb-config.yaml to set the project name and team members.\n")
	builder.WriteString("  2. Drop your first meeting notes into .project-kb/inbox/.\n")
	builder.WriteString("  3. Ask your coding agent to process the inbox.\n")

	return builder.String()
}

func formatAgentsMDStatus(result kbinit.Result) string {
	switch {
	case result.AgentsMD.FileCreated:
		return "Created AGENTS.md (added project-kb section)"
	case result.AgentsMD.SnippetAdded:
		return "Updated AGENTS.md (added project-kb section)"
	default:
		return "AGENTS.md already contained the project-kb section"
	}
}
