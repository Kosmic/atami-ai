package templatefs

import (
	"fmt"
	"os"
	"path/filepath"
)

var defaultSearchRelativePaths = []string{
	"atami-ai",
	filepath.Join("code", "atami-ai"),
	filepath.Join("Code", "atami-ai"),
	filepath.Join("dev", "atami-ai"),
}

// ResolvedPaths contains the validated filesystem paths needed by the CLI.
type ResolvedPaths struct {
	RepoRoot    string
	TemplateDir string
	SkillsDir   string
}

type resolverOptions struct {
	lookupEnv      func(string) (string, bool)
	userHome       func() (string, error)
	searchRelPaths []string
}

// ResolvePaths locates and validates the local atami-ai repository paths needed by the CLI.
func ResolvePaths(explicitTemplate string) (ResolvedPaths, error) {
	return resolvePaths(explicitTemplate, resolverOptions{
		lookupEnv:      os.LookupEnv,
		userHome:       os.UserHomeDir,
		searchRelPaths: defaultSearchRelativePaths,
	})
}

func resolvePaths(explicitTemplate string, opts resolverOptions) (ResolvedPaths, error) {
	if explicitTemplate != "" {
		templateDir, err := filepath.Abs(explicitTemplate)
		if err != nil {
			return ResolvedPaths{}, fmt.Errorf("resolving explicit template path: %w", err)
		}

		resolved := ResolvedPaths{
			RepoRoot:    filepath.Dir(filepath.Dir(templateDir)),
			TemplateDir: templateDir,
			SkillsDir:   filepath.Join(filepath.Dir(filepath.Dir(templateDir)), "project-kb", "skills"),
		}

		if err := validateResolvedPaths(resolved); err != nil {
			return ResolvedPaths{}, err
		}

		return resolved, nil
	}

	if repoRoot, ok := opts.lookupEnv("ATAMI_AI_PATH"); ok && repoRoot != "" {
		resolved, err := newResolvedPaths(repoRoot)
		if err != nil {
			return ResolvedPaths{}, err
		}
		if err := validateResolvedPaths(resolved); err != nil {
			return ResolvedPaths{}, err
		}
		return resolved, nil
	}

	homeDir, err := opts.userHome()
	if err != nil {
		return ResolvedPaths{}, fmt.Errorf("determining home directory: %w", err)
	}

	for _, relPath := range opts.searchRelPaths {
		candidateRoot := filepath.Join(homeDir, relPath)
		info, err := os.Stat(candidateRoot)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return ResolvedPaths{}, fmt.Errorf("checking repo root %q: %w", candidateRoot, err)
		}
		if !info.IsDir() {
			return ResolvedPaths{}, fmt.Errorf("repo root is not a directory: %s", candidateRoot)
		}

		resolved, err := newResolvedPaths(candidateRoot)
		if err != nil {
			return ResolvedPaths{}, err
		}
		if err := validateResolvedPaths(resolved); err != nil {
			return ResolvedPaths{}, err
		}
		return resolved, nil
	}

	return ResolvedPaths{}, fmt.Errorf("could not find the atami-ai repo; set ATAMI_AI_PATH or clone the repo to ~/atami-ai, ~/code/atami-ai, ~/Code/atami-ai, or ~/dev/atami-ai")
}

func newResolvedPaths(repoRoot string) (ResolvedPaths, error) {
	absRepoRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return ResolvedPaths{}, fmt.Errorf("resolving repo root %q: %w", repoRoot, err)
	}

	return ResolvedPaths{
		RepoRoot:    absRepoRoot,
		TemplateDir: filepath.Join(absRepoRoot, "project-kb", "template"),
		SkillsDir:   filepath.Join(absRepoRoot, "project-kb", "skills"),
	}, nil
}

func validateResolvedPaths(paths ResolvedPaths) error {
	if err := requireDir(paths.TemplateDir, "template directory"); err != nil {
		return err
	}
	if err := requireDir(filepath.Join(paths.TemplateDir, "project-kb"), "template project-kb directory"); err != nil {
		return err
	}
	if err := requireFile(filepath.Join(paths.TemplateDir, "AGENTS.md.snippet"), "AGENTS.md snippet"); err != nil {
		return err
	}
	if err := requireDir(paths.SkillsDir, "canonical skills directory"); err != nil {
		return err
	}

	skillMatches, err := filepath.Glob(filepath.Join(paths.SkillsDir, "*.md"))
	if err != nil {
		return fmt.Errorf("listing skill files: %w", err)
	}
	if len(skillMatches) == 0 {
		return fmt.Errorf("canonical skills directory contains no top-level .md files: %s", paths.SkillsDir)
	}

	return nil
}

func requireDir(path string, label string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("required %s missing: %s", label, path)
		}
		return fmt.Errorf("checking %s %q: %w", label, path, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("required %s is not a directory: %s", label, path)
	}
	return nil
}

func requireFile(path string, label string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("required %s missing: %s", label, path)
		}
		return fmt.Errorf("checking %s %q: %w", label, path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("required %s is not a file: %s", label, path)
	}
	return nil
}
