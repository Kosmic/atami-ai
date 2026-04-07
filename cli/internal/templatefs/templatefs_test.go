package templatefs

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const fixtureSnippet = "<!-- BEGIN atami project-kb -->\nTest snippet\n<!-- END atami project-kb -->\n"

func TestResolve_ExplicitPath(t *testing.T) {
	templateDir := fixtureTemplateDir(t)

	resolved, err := ResolvePaths(templateDir)
	if err != nil {
		t.Fatalf("ResolvePaths returned error: %v", err)
	}

	wantRepoRoot := fixtureRepoRoot(t)
	if resolved.RepoRoot != wantRepoRoot {
		t.Fatalf("unexpected repo root: got %q want %q", resolved.RepoRoot, wantRepoRoot)
	}
	if resolved.TemplateDir != templateDir {
		t.Fatalf("unexpected template dir: got %q want %q", resolved.TemplateDir, templateDir)
	}
	if resolved.SkillsDir != filepath.Join(wantRepoRoot, "project-kb", "skills") {
		t.Fatalf("unexpected skills dir: %q", resolved.SkillsDir)
	}
}

func TestResolve_ExplicitPathDoesNotExist(t *testing.T) {
	_, err := ResolvePaths(filepath.Join(t.TempDir(), "missing"))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "required template directory missing") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolve_EnvVar(t *testing.T) {
	repoRoot := fixtureRepoRoot(t)
	t.Setenv("ATAMI_AI_PATH", repoRoot)

	resolved, err := ResolvePaths("")
	if err != nil {
		t.Fatalf("ResolvePaths returned error: %v", err)
	}

	if resolved.RepoRoot != repoRoot {
		t.Fatalf("unexpected repo root: got %q want %q", resolved.RepoRoot, repoRoot)
	}
}

func TestResolve_NoneFound(t *testing.T) {
	tempHome := t.TempDir()

	_, err := resolvePaths("", resolverOptions{
		lookupEnv:      func(string) (string, bool) { return "", false },
		userHome:       func() (string, error) { return tempHome, nil },
		searchRelPaths: defaultSearchRelativePaths,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "could not find the atami-ai repo") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolve_MissingSkillsDirFails(t *testing.T) {
	repoRoot := t.TempDir()
	templateDir := filepath.Join(repoRoot, "project-kb", "template")
	if err := os.MkdirAll(filepath.Join(templateDir, "project-kb"), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(templateDir, "AGENTS.md.snippet"), []byte(fixtureSnippet), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	_, err := ResolvePaths(templateDir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "required canonical skills directory missing") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolve_UppercaseCodePath(t *testing.T) {
	tempHome := t.TempDir()
	repoRoot := filepath.Join(tempHome, "Code", "atami-ai")
	copyFixtureRepo(t, repoRoot)

	resolved, err := resolvePaths("", resolverOptions{
		lookupEnv:      func(string) (string, bool) { return "", false },
		userHome:       func() (string, error) { return tempHome, nil },
		searchRelPaths: []string{filepath.Join("Code", "atami-ai")},
	})
	if err != nil {
		t.Fatalf("resolvePaths returned error: %v", err)
	}
	if resolved.RepoRoot != repoRoot {
		t.Fatalf("unexpected repo root: got %q want %q", resolved.RepoRoot, repoRoot)
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

func copyFixtureRepo(t *testing.T, targetRoot string) {
	t.Helper()

	sourceRoot := fixtureRepoRoot(t)
	if err := filepath.Walk(sourceRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(sourceRoot, path)
		if err != nil {
			return err
		}

		targetPath := filepath.Join(targetRoot, relPath)
		if info.IsDir() {
			return os.MkdirAll(targetPath, info.Mode().Perm())
		}

		sourceFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer sourceFile.Close()

		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return err
		}

		targetFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode().Perm())
		if err != nil {
			return err
		}
		if _, err := io.Copy(targetFile, sourceFile); err != nil {
			_ = targetFile.Close()
			return err
		}
		return targetFile.Close()
	}); err != nil {
		t.Fatalf("copyFixtureRepo returned error: %v", err)
	}
}
