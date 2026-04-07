package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/atami-ai/atami-ai/cli/internal/kbskillspull"
	"github.com/atami-ai/atami-ai/cli/internal/templatefs"
	"github.com/spf13/cobra"
)

const version = "0.1.0-dev"

// Options configures command execution and is primarily used by tests.
type Options struct {
	Stdout       io.Writer
	Stderr       io.Writer
	Getwd        func() (string, error)
	ResolvePaths func(string) (templatefs.ResolvedPaths, error)
	PullKBSkills func(kbskillspull.Options) (kbskillspull.Result, error)
}

// Execute runs the CLI with the provided arguments and returns the process exit code.
func Execute(args []string, opts Options) int {
	rootCmd := NewRootCmd(opts)
	rootCmd.SetArgs(args)

	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintf(rootCmd.ErrOrStderr(), "Error: %v\n", err)
		return 1
	}

	return 0
}

// NewRootCmd builds the top-level Cobra command tree.
func NewRootCmd(opts Options) *cobra.Command {
	opts = withDefaults(opts)

	rootCmd := &cobra.Command{
		Use:           "atami",
		Short:         "Atami project-kb tooling",
		SilenceErrors: true,
		SilenceUsage:  true,
		Version:       version,
		Args:          cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	rootCmd.SetOut(opts.Stdout)
	rootCmd.SetErr(opts.Stderr)
	rootCmd.SetVersionTemplate("{{printf \"%s\\n\" .Version}}")
	rootCmd.AddCommand(newKbCmd(opts))

	return rootCmd
}

func withDefaults(opts Options) Options {
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	if opts.Stderr == nil {
		opts.Stderr = os.Stderr
	}
	if opts.Getwd == nil {
		opts.Getwd = os.Getwd
	}
	if opts.ResolvePaths == nil {
		opts.ResolvePaths = templatefs.ResolvePaths
	}
	if opts.PullKBSkills == nil {
		opts.PullKBSkills = kbskillspull.Pull
	}
	return opts
}
