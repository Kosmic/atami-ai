package kbinit

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/atami-ai/atami-ai/cli/internal/agentsmd"
	"github.com/atami-ai/atami-ai/cli/internal/templatefs"
	"gopkg.in/yaml.v3"
)

const (
	namePlaceholder        = "\"TODO: replace with project name\""
	descriptionPlaceholder = "\"TODO: brief description for LLM context\""
)

var createdSummaryPaths = []string{
	filepath.Join(".project-kb", "inbox", "archive") + string(os.PathSeparator),
	filepath.Join(".project-kb", "items", "index.md"),
	filepath.Join(".project-kb", "releases") + string(os.PathSeparator),
	filepath.Join(".project-kb", "outputs") + string(os.PathSeparator),
	filepath.Join(".project-kb", "skills") + string(os.PathSeparator),
	filepath.Join(".project-kb", "skills", "overrides") + string(os.PathSeparator),
	filepath.Join(".project-kb", "kb-config.yaml"),
}

// Options configures the project-kb initialisation flow.
type Options struct {
	TargetDir      string
	Name           string
	Description    string
	Force          bool
	TemplateSource string
	ResolvePaths   func(string) (templatefs.ResolvedPaths, error)
}

// Result describes what happened during initialisation.
type Result struct {
	TargetDir        string
	CreatedPaths     []string
	SyncedSkillFiles []string
	Warnings         []string
	AgentsMD         agentsmd.AppendResult
}

// Initialize creates .project-kb in the target directory and syncs the canonical files.
func Initialize(opts Options) (result Result, err error) {
	if opts.TargetDir == "" {
		return Result{}, fmt.Errorf("target directory is required")
	}
	if opts.ResolvePaths == nil {
		opts.ResolvePaths = templatefs.ResolvePaths
	}

	resolved, err := opts.ResolvePaths(opts.TemplateSource)
	if err != nil {
		return Result{}, fmt.Errorf("resolving template paths: %w", err)
	}
	defer func() {
		if resolved.Cleanup == nil {
			return
		}
		if cleanupErr := resolved.Cleanup(); cleanupErr != nil && err == nil {
			err = fmt.Errorf("cleaning up resolved template source: %w", cleanupErr)
		}
	}()

	targetProjectKB := filepath.Join(opts.TargetDir, ".project-kb")
	if err := prepareTargetDirectory(targetProjectKB, opts.Force); err != nil {
		return Result{}, err
	}

	if err := copyTree(filepath.Join(resolved.TemplateDir, "project-kb"), targetProjectKB); err != nil {
		return Result{}, fmt.Errorf("copying template directory: %w", err)
	}

	syncedSkills, err := copySkillFiles(resolved.SkillsDir, filepath.Join(targetProjectKB, "skills"))
	if err != nil {
		return Result{}, fmt.Errorf("copying canonical skill files: %w", err)
	}

	warnings, err := applyConfig(filepath.Join(targetProjectKB, "kb-config.yaml"), opts.Name, opts.Description)
	if err != nil {
		return Result{}, fmt.Errorf("updating kb-config.yaml: %w", err)
	}

	snippetBytes, err := os.ReadFile(filepath.Join(resolved.TemplateDir, "AGENTS.md.snippet"))
	if err != nil {
		return Result{}, fmt.Errorf("reading AGENTS.md snippet: %w", err)
	}

	appendResult, err := agentsmd.AppendSnippet(filepath.Join(opts.TargetDir, "AGENTS.md"), string(snippetBytes))
	if err != nil {
		return Result{}, fmt.Errorf("updating AGENTS.md: %w", err)
	}

	return Result{
		TargetDir:        opts.TargetDir,
		CreatedPaths:     collectCreatedPaths(opts.TargetDir),
		SyncedSkillFiles: syncedSkills,
		Warnings:         warnings,
		AgentsMD:         appendResult,
	}, nil
}

func prepareTargetDirectory(targetProjectKB string, force bool) error {
	info, err := os.Stat(targetProjectKB)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("checking for existing .project-kb directory: %w", err)
	}

	if !force {
		if info.IsDir() {
			return fmt.Errorf(".project-kb/ already exists in this directory. Use --force to overwrite.")
		}
		return fmt.Errorf(".project-kb already exists in this directory and is not a directory. Use --force to overwrite.")
	}

	if err := os.RemoveAll(targetProjectKB); err != nil {
		return fmt.Errorf("removing existing .project-kb directory: %w", err)
	}

	return nil
}

func copyTree(srcRoot string, dstRoot string) error {
	return filepath.WalkDir(srcRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("walking %q: %w", path, walkErr)
		}

		relPath, err := filepath.Rel(srcRoot, path)
		if err != nil {
			return fmt.Errorf("computing relative path for %q: %w", path, err)
		}

		targetPath := dstRoot
		if relPath != "." {
			targetPath = filepath.Join(dstRoot, relPath)
		}

		info, err := d.Info()
		if err != nil {
			return fmt.Errorf("reading file info for %q: %w", path, err)
		}

		switch {
		case d.IsDir():
			if err := os.MkdirAll(targetPath, info.Mode().Perm()); err != nil {
				return fmt.Errorf("creating directory %q: %w", targetPath, err)
			}
			if err := os.Chmod(targetPath, info.Mode().Perm()); err != nil {
				return fmt.Errorf("setting mode on directory %q: %w", targetPath, err)
			}
		case info.Mode().IsRegular():
			if err := copyFile(path, targetPath, info.Mode().Perm()); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported file type in template: %s", path)
		}

		return nil
	})
}

func copySkillFiles(skillsDir string, targetSkillsDir string) ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(skillsDir, "*.md"))
	if err != nil {
		return nil, fmt.Errorf("listing skill files: %w", err)
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("canonical skills directory contains no top-level .md files: %s", skillsDir)
	}

	sort.Strings(matches)

	var synced []string
	for _, path := range matches {
		info, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("statting skill file %q: %w", path, err)
		}
		if info.IsDir() {
			continue
		}

		targetPath := filepath.Join(targetSkillsDir, filepath.Base(path))
		if err := copyFile(path, targetPath, info.Mode().Perm()); err != nil {
			return nil, err
		}
		synced = append(synced, filepath.Base(path))
	}

	return synced, nil
}

func copyFile(srcPath string, dstPath string, mode fs.FileMode) error {
	sourceFile, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("opening source file %q: %w", srcPath, err)
	}
	defer sourceFile.Close()

	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return fmt.Errorf("creating parent directory for %q: %w", dstPath, err)
	}

	targetFile, err := os.OpenFile(dstPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return fmt.Errorf("opening target file %q: %w", dstPath, err)
	}

	if _, err := io.Copy(targetFile, sourceFile); err != nil {
		_ = targetFile.Close()
		return fmt.Errorf("copying %q to %q: %w", srcPath, dstPath, err)
	}

	if err := targetFile.Close(); err != nil {
		return fmt.Errorf("closing target file %q: %w", dstPath, err)
	}

	if err := os.Chmod(dstPath, mode); err != nil {
		return fmt.Errorf("setting mode on file %q: %w", dstPath, err)
	}

	return nil
}

func applyConfig(configPath string, name string, description string) ([]string, error) {
	content, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("reading %q: %w", configPath, err)
	}

	updated := string(content)
	var warnings []string
	changed := false

	if name != "" {
		replaced, didReplace := replacePlaceholder(updated, namePlaceholder, strconv.Quote(name))
		updated = replaced
		changed = changed || didReplace
		if !didReplace {
			warnings = append(warnings, "project name placeholder not found in .project-kb/kb-config.yaml")
		}
	}

	if description != "" {
		replaced, didReplace := replacePlaceholder(updated, descriptionPlaceholder, strconv.Quote(description))
		updated = replaced
		changed = changed || didReplace
		if !didReplace {
			warnings = append(warnings, "project description placeholder not found in .project-kb/kb-config.yaml")
		}
	}

	var parsed any
	if err := yaml.Unmarshal([]byte(updated), &parsed); err != nil {
		return nil, fmt.Errorf("validating resulting YAML: %w", err)
	}

	if changed {
		if err := os.WriteFile(configPath, []byte(updated), 0o644); err != nil {
			return nil, fmt.Errorf("writing %q: %w", configPath, err)
		}
	}

	return warnings, nil
}

func replacePlaceholder(content string, placeholder string, replacement string) (string, bool) {
	if !strings.Contains(content, placeholder) {
		return content, false
	}
	return strings.Replace(content, placeholder, replacement, 1), true
}

func collectCreatedPaths(targetDir string) []string {
	var created []string
	for _, relPath := range createdSummaryPaths {
		statPath := relPath
		if strings.HasSuffix(statPath, string(os.PathSeparator)) {
			statPath = strings.TrimSuffix(statPath, string(os.PathSeparator))
		}

		if _, err := os.Stat(filepath.Join(targetDir, statPath)); err == nil {
			created = append(created, filepath.ToSlash(relPath))
		}
	}
	return created
}
