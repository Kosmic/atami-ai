package kbinit

import (
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/atami-ai/atami-ai/cli/internal/templatefs"
	"gopkg.in/yaml.v3"
)

func TestInit_FreshDirectory(t *testing.T) {
	targetDir := t.TempDir()

	result, err := Initialize(Options{
		TargetDir:      targetDir,
		Name:           "Test Project",
		Description:    "Test description",
		TemplateSource: fixtureTemplateDir(t),
	})
	if err != nil {
		t.Fatalf("Initialize returned error: %v", err)
	}

	expectedPaths := []string{
		filepath.Join(targetDir, ".project-kb", "inbox", "archive"),
		filepath.Join(targetDir, ".project-kb", "items", "index.md"),
		filepath.Join(targetDir, ".project-kb", "releases"),
		filepath.Join(targetDir, ".project-kb", "outputs"),
		filepath.Join(targetDir, ".project-kb", "skills"),
		filepath.Join(targetDir, ".project-kb", "skills", "overrides"),
		filepath.Join(targetDir, ".project-kb", ".gitignore"),
		filepath.Join(targetDir, ".project-kb", "kb-config.yaml"),
		filepath.Join(targetDir, "AGENTS.md"),
	}
	for _, path := range expectedPaths {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected path %q to exist: %v", path, err)
		}
	}

	configBytes, err := os.ReadFile(filepath.Join(targetDir, ".project-kb", "kb-config.yaml"))
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	configText := string(configBytes)
	if strings.Contains(configText, "TODO: replace with project name") {
		t.Fatalf("config should not contain project name placeholder:\n%s", configText)
	}
	if strings.Contains(configText, "TODO: brief description for LLM context") {
		t.Fatalf("config should not contain description placeholder:\n%s", configText)
	}
	if !strings.Contains(configText, "\"Test Project\"") || !strings.Contains(configText, "\"Test description\"") {
		t.Fatalf("config missing expected values:\n%s", configText)
	}

	agentsBytes, err := os.ReadFile(filepath.Join(targetDir, "AGENTS.md"))
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if !strings.Contains(string(agentsBytes), "<!-- BEGIN atami project-kb -->") {
		t.Fatalf("AGENTS.md is missing snippet markers:\n%s", string(agentsBytes))
	}

	gitignoreBytes, err := os.ReadFile(filepath.Join(targetDir, ".project-kb", ".gitignore"))
	if err != nil {
		t.Fatalf("ReadFile returned error for .gitignore: %v", err)
	}
	if string(gitignoreBytes) != "outputs/kanban.html\n" {
		t.Fatalf("unexpected .gitignore content:\n%s", string(gitignoreBytes))
	}

	expectedSkills := map[string]string{
		"assets/kanban-board.template.html": "<!-- Test kanban-board template fixture -->\n",
		"generate-output.md":                "# Test generate-output skill\n",
		"kanban-board.md":                   "# Test kanban-board skill\n",
		"process-inbox.md":                  "# Test process-inbox skill\n",
		"release-notes.md":                  "# Test release-notes skill\n",
	}
	for name, expected := range expectedSkills {
		skillBytes, err := os.ReadFile(filepath.Join(targetDir, ".project-kb", "skills", name))
		if err != nil {
			t.Fatalf("ReadFile returned error for %s: %v", name, err)
		}
		if string(skillBytes) != expected {
			t.Fatalf("unexpected skill content for %s:\n%s", name, string(skillBytes))
		}
	}

	if len(result.SyncedSkillFiles) != 5 {
		t.Fatalf("unexpected synced skill count: %d", len(result.SyncedSkillFiles))
	}
	if !result.AgentsMD.FileCreated || !result.AgentsMD.SnippetAdded {
		t.Fatalf("unexpected AGENTS result: %+v", result.AgentsMD)
	}
}

func TestInit_ExistingProjectKbWithoutForce(t *testing.T) {
	targetDir := t.TempDir()
	sentinelPath := filepath.Join(targetDir, ".project-kb", "sentinel.txt")
	if err := os.MkdirAll(filepath.Dir(sentinelPath), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(sentinelPath, []byte("keep me"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	_, err := Initialize(Options{
		TargetDir:      targetDir,
		TemplateSource: fixtureTemplateDir(t),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), ".project-kb/ already exists") {
		t.Fatalf("unexpected error: %v", err)
	}
	if content, readErr := os.ReadFile(sentinelPath); readErr != nil || string(content) != "keep me" {
		t.Fatalf("sentinel should be unchanged, got content=%q err=%v", string(content), readErr)
	}
}

func TestInit_ExistingProjectKbWithForce(t *testing.T) {
	targetDir := t.TempDir()
	sentinelPath := filepath.Join(targetDir, ".project-kb", "sentinel.txt")
	if err := os.MkdirAll(filepath.Dir(sentinelPath), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(sentinelPath, []byte("delete me"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	_, err := Initialize(Options{
		TargetDir:      targetDir,
		TemplateSource: fixtureTemplateDir(t),
		Force:          true,
	})
	if err != nil {
		t.Fatalf("Initialize returned error: %v", err)
	}

	if _, err := os.Stat(sentinelPath); !os.IsNotExist(err) {
		t.Fatalf("expected sentinel to be removed, stat err=%v", err)
	}
}

func TestInit_ExistingAgentsMdWithoutSnippet(t *testing.T) {
	targetDir := t.TempDir()
	existing := "# Existing project rules\n\nDo not commit secrets.\n"
	if err := os.WriteFile(filepath.Join(targetDir, "AGENTS.md"), []byte(existing), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	_, err := Initialize(Options{
		TargetDir:      targetDir,
		TemplateSource: fixtureTemplateDir(t),
	})
	if err != nil {
		t.Fatalf("Initialize returned error: %v", err)
	}

	snippet := readFixtureSnippet(t)
	agentsBytes, err := os.ReadFile(filepath.Join(targetDir, "AGENTS.md"))
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}

	expected := existing + "\n" + snippet
	if string(agentsBytes) != expected {
		t.Fatalf("unexpected AGENTS.md content:\n%s", string(agentsBytes))
	}
}

func TestInit_ExistingAgentsMdWithSnippet(t *testing.T) {
	targetDir := t.TempDir()
	original := "# AGENTS.md\n\n" + readFixtureSnippet(t)
	agentsPath := filepath.Join(targetDir, "AGENTS.md")
	if err := os.WriteFile(agentsPath, []byte(original), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := Initialize(Options{
		TargetDir:      targetDir,
		TemplateSource: fixtureTemplateDir(t),
	})
	if err != nil {
		t.Fatalf("Initialize returned error: %v", err)
	}

	agentsBytes, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if string(agentsBytes) != original {
		t.Fatalf("AGENTS.md should be unchanged:\n%s", string(agentsBytes))
	}
	if result.AgentsMD.FileCreated || result.AgentsMD.SnippetAdded {
		t.Fatalf("unexpected AGENTS result: %+v", result.AgentsMD)
	}
}

func TestInit_NoNameOrDescription(t *testing.T) {
	targetDir := t.TempDir()

	_, err := Initialize(Options{
		TargetDir:      targetDir,
		TemplateSource: fixtureTemplateDir(t),
	})
	if err != nil {
		t.Fatalf("Initialize returned error: %v", err)
	}

	configBytes, err := os.ReadFile(filepath.Join(targetDir, ".project-kb", "kb-config.yaml"))
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	configText := string(configBytes)
	if !strings.Contains(configText, "TODO: replace with project name") {
		t.Fatalf("project name placeholder should remain:\n%s", configText)
	}
	if !strings.Contains(configText, "TODO: brief description for LLM context") {
		t.Fatalf("description placeholder should remain:\n%s", configText)
	}
}

// Re-initialisation with --force intentionally recreates .project-kb from scratch.
// Override preservation is not part of Phase 1 and will be covered by Phase 2 sync behaviour.

func TestInit_MissingTemplateFails(t *testing.T) {
	targetDir := t.TempDir()

	_, err := Initialize(Options{
		TargetDir:      targetDir,
		TemplateSource: filepath.Join(t.TempDir(), "missing-template"),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "required template directory missing") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInit_MalformedTemplateFails(t *testing.T) {
	targetDir := t.TempDir()
	repoRoot := t.TempDir()
	templateDir := filepath.Join(repoRoot, "project-kb", "template")
	if err := os.MkdirAll(filepath.Join(templateDir, "project-kb"), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(repoRoot, "project-kb", "skills"), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "project-kb", "skills", "process-inbox.md"), []byte("# skill\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	_, err := Initialize(Options{
		TargetDir:      targetDir,
		TemplateSource: templateDir,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "required AGENTS.md snippet missing") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInit_SpecialCharactersInNameAndDescription(t *testing.T) {
	targetDir := t.TempDir()
	name := "Project \"Delta\"\n#1"
	description := "Line one\nLine two with \"quotes\" and #hash"

	_, err := Initialize(Options{
		TargetDir:      targetDir,
		TemplateSource: fixtureTemplateDir(t),
		Name:           name,
		Description:    description,
	})
	if err != nil {
		t.Fatalf("Initialize returned error: %v", err)
	}

	configBytes, err := os.ReadFile(filepath.Join(targetDir, ".project-kb", "kb-config.yaml"))
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}

	var parsed struct {
		Project struct {
			Name        string `yaml:"name"`
			Description string `yaml:"description"`
		} `yaml:"project"`
	}
	if err := yaml.Unmarshal(configBytes, &parsed); err != nil {
		t.Fatalf("yaml.Unmarshal returned error: %v", err)
	}
	if parsed.Project.Name != name {
		t.Fatalf("unexpected name: got %q want %q", parsed.Project.Name, name)
	}
	if parsed.Project.Description != description {
		t.Fatalf("unexpected description: got %q want %q", parsed.Project.Description, description)
	}
}

func TestInit_CallsResolvedSourceCleanup(t *testing.T) {
	targetDir := t.TempDir()
	var cleanupCalls atomic.Int32

	_, err := Initialize(Options{
		TargetDir: targetDir,
		ResolvePaths: func(string) (templatefs.ResolvedPaths, error) {
			resolved := templatefs.ResolvedPaths{
				RepoRoot:    fixtureRepoRoot(t),
				TemplateDir: fixtureTemplateDir(t),
				SkillsDir:   filepath.Join(fixtureRepoRoot(t), "project-kb", "skills"),
				Cleanup: func() error {
					cleanupCalls.Add(1)
					return nil
				},
			}
			return resolved, nil
		},
	})
	if err != nil {
		t.Fatalf("Initialize returned error: %v", err)
	}
	if cleanupCalls.Load() != 1 {
		t.Fatalf("expected cleanup to be called once, got %d", cleanupCalls.Load())
	}
}

func fixtureRepoRoot(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %v", err)
	}

	repoRoot, err := filepath.Abs(filepath.Join(wd, "..", "..", "testdata", "fake-atami-ai"))
	if err != nil {
		t.Fatalf("Abs returned error: %v", err)
	}
	return repoRoot
}

func fixtureTemplateDir(t *testing.T) string {
	t.Helper()
	return filepath.Join(fixtureRepoRoot(t), "project-kb", "template")
}

func readFixtureSnippet(t *testing.T) string {
	t.Helper()

	content, err := os.ReadFile(filepath.Join(fixtureTemplateDir(t), "AGENTS.md.snippet"))
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	return string(content)
}
