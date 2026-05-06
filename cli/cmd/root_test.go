package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atami-ai/atami-ai/cli/internal/kbskillspull"
)

func TestRoot_VersionFlag(t *testing.T) {
	stdout, stderr, exitCode := executeCLI(t, t.TempDir(), "--version")

	if exitCode != 0 {
		t.Fatalf("unexpected exit code: %d stderr=%q", exitCode, stderr)
	}
	if stdout != "0.1.0-dev\n" {
		t.Fatalf("unexpected stdout: %q", stdout)
	}
	if stderr != "" {
		t.Fatalf("unexpected stderr: %q", stderr)
	}
}

func TestRoot_Help(t *testing.T) {
	stdout, stderr, exitCode := executeCLI(t, t.TempDir())

	if exitCode != 0 {
		t.Fatalf("unexpected exit code: %d stderr=%q", exitCode, stderr)
	}
	if !strings.Contains(stdout, "Atami project-kb tooling") {
		t.Fatalf("help output missing short description:\n%s", stdout)
	}
	if !strings.Contains(stdout, "kb") {
		t.Fatalf("help output missing kb command:\n%s", stdout)
	}
}

func TestKb_Help(t *testing.T) {
	stdout, stderr, exitCode := executeCLI(t, t.TempDir(), "kb")

	if exitCode != 0 {
		t.Fatalf("unexpected exit code: %d stderr=%q", exitCode, stderr)
	}
	if !strings.Contains(stdout, "Manage project knowledge base scaffolding") {
		t.Fatalf("kb help missing description:\n%s", stdout)
	}
	if !strings.Contains(stdout, "init") {
		t.Fatalf("kb help missing init command:\n%s", stdout)
	}
	if !strings.Contains(stdout, "skills") {
		t.Fatalf("kb help missing skills command:\n%s", stdout)
	}
}

func TestKbSkills_Help(t *testing.T) {
	stdout, stderr, exitCode := executeCLI(t, t.TempDir(), "kb", "skills")

	if exitCode != 0 {
		t.Fatalf("unexpected exit code: %d stderr=%q", exitCode, stderr)
	}
	if !strings.Contains(stdout, "Manage project-kb skill syncing") {
		t.Fatalf("kb skills help missing description:\n%s", stdout)
	}
	if !strings.Contains(stdout, "pull") {
		t.Fatalf("kb skills help missing pull command:\n%s", stdout)
	}
}

func TestKbSkillsPull_Help(t *testing.T) {
	stdout, stderr, exitCode := executeCLI(t, t.TempDir(), "kb", "skills", "pull", "--help")

	if exitCode != 0 {
		t.Fatalf("unexpected exit code: %d stderr=%q", exitCode, stderr)
	}
	if !strings.Contains(stdout, "Sync canonical project-kb skill files") {
		t.Fatalf("kb skills pull help missing description:\n%s", stdout)
	}
	if !strings.Contains(stdout, "--force") {
		t.Fatalf("kb skills pull help missing --force flag:\n%s", stdout)
	}
}

func TestKbSkillsPull_UserErrorFormatting(t *testing.T) {
	stdout, stderr, exitCode := executeCLIWithOptions(t, t.TempDir(), Options{
		PullKBSkills: func(kbskillspull.Options) (kbskillspull.Result, error) {
			return kbskillspull.Result{}, fmt.Errorf(".project-kb/ does not exist in this directory. Run `atami kb init` first.")
		},
	}, "kb", "skills", "pull")

	if exitCode == 0 {
		t.Fatal("expected non-zero exit code")
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.HasPrefix(stderr, "Error: ") {
		t.Fatalf("stderr should start with Error:, got %q", stderr)
	}
}

func TestKbSkillsPull_EndToEndHappyPath(t *testing.T) {
	cwd := t.TempDir()
	stdout, stderr, exitCode := executeCLIWithOptions(t, cwd, Options{
		PullKBSkills: func(kbskillspull.Options) (kbskillspull.Result, error) {
			return kbskillspull.Result{
				TargetDir: cwd,
				UpdatedFiles: []string{
					".project-kb/skills/generate-output.md",
					".project-kb/skills/process-inbox.md",
				},
				UnchangedFiles: []string{
					".project-kb/skills/release-notes.md",
				},
				SkippedFiles: []kbskillspull.SkippedFile{
					{
						Path:   ".project-kb/skills/custom.md",
						Reason: "local edits detected",
					},
				},
			}, nil
		},
	}, "kb", "skills", "pull")

	if exitCode != 0 {
		t.Fatalf("unexpected exit code: %d stderr=%q", exitCode, stderr)
	}
	if stderr != "" {
		t.Fatalf("unexpected stderr: %q", stderr)
	}

	expected := fmt.Sprintf(
		"✓ Project-kb skills sync complete in %s\n\nUpdated: 2\nUnchanged: 1\nSkipped: 1\n\nUpdated:\n  .project-kb/skills/generate-output.md\n  .project-kb/skills/process-inbox.md\n\nUnchanged:\n  .project-kb/skills/release-notes.md\n\nSkipped:\n  .project-kb/skills/custom.md (local edits detected)\n\nNext step:\n  Re-run `atami kb skills pull --force` to overwrite skipped locally edited synced files.\n\nLeft untouched:\n  .project-kb/skills/overrides/\n",
		cwd,
	)
	if stdout != expected {
		t.Fatalf("unexpected stdout:\n%s", stdout)
	}
}

func TestKbInit_Help(t *testing.T) {
	stdout, stderr, exitCode := executeCLI(t, t.TempDir(), "kb", "init", "--help")

	if exitCode != 0 {
		t.Fatalf("unexpected exit code: %d stderr=%q", exitCode, stderr)
	}
	for _, flag := range []string{"--name", "--description", "--force", "--template-source"} {
		if !strings.Contains(stdout, flag) {
			t.Fatalf("init help missing flag %s:\n%s", flag, stdout)
		}
	}
}

func TestKbInit_UserErrorFormatting(t *testing.T) {
	cwd := t.TempDir()
	if err := os.MkdirAll(filepath.Join(cwd, ".project-kb"), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}

	stdout, stderr, exitCode := executeCLI(t, cwd, "kb", "init", "--template-source", fixtureTemplateDir(t))

	if exitCode == 0 {
		t.Fatal("expected non-zero exit code")
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.HasPrefix(stderr, "Error: ") {
		t.Fatalf("stderr should start with Error:, got %q", stderr)
	}
}

func TestKbInit_EndToEndHappyPath(t *testing.T) {
	cwd := t.TempDir()

	stdout, stderr, exitCode := executeCLI(
		t,
		cwd,
		"kb",
		"init",
		"--template-source", fixtureTemplateDir(t),
		"--name", "CLI Test Project",
		"--description", "CLI test description",
	)

	if exitCode != 0 {
		t.Fatalf("unexpected exit code: %d stderr=%q", exitCode, stderr)
	}
	if stderr != "" {
		t.Fatalf("unexpected stderr: %q", stderr)
	}

	expected := fmt.Sprintf(
		"✓ Initialised .project-kb in %s\n\nCreated:\n  .project-kb/inbox/archive/\n  .project-kb/items/index.md\n  .project-kb/releases/\n  .project-kb/outputs/\n  .project-kb/skills/\n  .project-kb/skills/overrides/\n  .project-kb/kb-config.yaml\n\nSynced 4 skill files into .project-kb/skills/\n\nCreated AGENTS.md (added project-kb section)\n\nNext steps:\n  1. Edit .project-kb/kb-config.yaml to set the project name and team members.\n  2. Drop your first unstructured meeting notes/bug/task into .project-kb/inbox/.\n  3. Ask your coding agent to process the inbox.\n",
		cwd,
	)
	if stdout != expected {
		t.Fatalf("unexpected stdout:\n%s", stdout)
	}
}

func executeCLI(t *testing.T, cwd string, args ...string) (string, string, int) {
	t.Helper()
	return executeCLIWithOptions(t, cwd, Options{}, args...)
}

func executeCLIWithOptions(t *testing.T, cwd string, opts Options, args ...string) (string, string, int) {
	t.Helper()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	opts.Stdout = &stdout
	opts.Stderr = &stderr
	opts.Getwd = func() (string, error) {
		return cwd, nil
	}

	exitCode := Execute(args, opts)

	return stdout.String(), stderr.String(), exitCode
}

func fixtureTemplateDir(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %v", err)
	}

	templateDir, err := filepath.Abs(filepath.Join(wd, "..", "testdata", "fake-atami-ai", "project-kb", "template"))
	if err != nil {
		t.Fatalf("Abs returned error: %v", err)
	}
	return templateDir
}
