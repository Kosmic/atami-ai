package templatefs

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
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
	_, err := resolvePaths("", resolverOptions{
		lookupEnv:   func(string) (string, bool) { return "", false },
		fetchRemote: nil,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "could not resolve the atami-ai source") {
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

func TestResolve_RemoteFallback(t *testing.T) {
	server := newTarballServer(t, fixtureRepoRoot(t))
	defer server.Close()

	resolved, err := resolvePaths("", resolverOptions{
		lookupEnv:   func(string) (string, bool) { return "", false },
		httpClient:  server.Client(),
		mkdirTemp:   os.MkdirTemp,
		fetchRemote: fetchRemotePaths,
		remote: remoteConfig{
			apiBaseURL: server.URL,
			owner:      defaultGitHubOwner,
			repo:       defaultGitHubRepo,
			ref:        defaultGitHubRef,
			client:     server.Client(),
			mkdirTemp:  os.MkdirTemp,
		},
	})
	if err != nil {
		t.Fatalf("resolvePaths returned error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(resolved.TemplateDir, "project-kb", "kb-config.yaml")); err != nil {
		t.Fatalf("expected remote template content to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(resolved.SkillsDir, "process-inbox.md")); err != nil {
		t.Fatalf("expected remote skills content to exist: %v", err)
	}
}

func TestResolve_RemoteFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err := resolvePaths("", resolverOptions{
		lookupEnv:   func(string) (string, bool) { return "", false },
		httpClient:  server.Client(),
		mkdirTemp:   os.MkdirTemp,
		fetchRemote: fetchRemotePaths,
		remote: remoteConfig{
			apiBaseURL: server.URL,
			owner:      defaultGitHubOwner,
			repo:       defaultGitHubRepo,
			ref:        defaultGitHubRef,
			client:     server.Client(),
			mkdirTemp:  os.MkdirTemp,
		},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to fetch") {
		t.Fatalf("unexpected error: %v", err)
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

func newTarballServer(t *testing.T, sourceRoot string) *httptest.Server {
	t.Helper()

	archive := tarballFromDir(t, sourceRoot, "atami-ai-atami-ai-main")
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/Kosmic/atami-ai/tarball/main" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/gzip")
		_, _ = w.Write(archive)
	}))
}

func tarballFromDir(t *testing.T, sourceRoot string, prefix string) []byte {
	t.Helper()

	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)
	tarWriter := tar.NewWriter(gzipWriter)

	if err := filepath.Walk(sourceRoot, func(currentPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(sourceRoot, currentPath)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = path.Join(prefix, filepath.ToSlash(relPath))
		if info.IsDir() {
			header.Name += "/"
		}

		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		file, err := os.Open(currentPath)
		if err != nil {
			return err
		}
		_, err = io.Copy(tarWriter, file)
		closeErr := file.Close()
		if err != nil {
			return err
		}
		return closeErr
	}); err != nil {
		t.Fatalf("tarballFromDir returned error: %v", err)
	}

	if err := tarWriter.Close(); err != nil {
		t.Fatalf("closing tar writer: %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("closing gzip writer: %v", err)
	}

	return buffer.Bytes()
}
