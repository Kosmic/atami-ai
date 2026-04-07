package kbskillspull

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPull_FreshSkillsDirectory(t *testing.T) {
	targetDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(targetDir, ".project-kb", "skills", "overrides"), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}

	server := newSkillsServer(t)
	defer server.Close()

	result, err := Pull(Options{
		TargetDir:  targetDir,
		APIBaseURL: server.URL,
		Owner:      defaultGitHubOwner,
		Repo:       defaultGitHubRepo,
		Ref:        defaultGitHubRef,
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("Pull returned error: %v", err)
	}

	for _, name := range []string{"generate-output.md", "process-inbox.md", "release-notes.md"} {
		content, err := os.ReadFile(filepath.Join(targetDir, ".project-kb", "skills", name))
		if err != nil {
			t.Fatalf("ReadFile returned error for %s: %v", name, err)
		}
		if !strings.Contains(string(content), name) {
			t.Fatalf("unexpected content for %s: %q", name, string(content))
		}
	}

	if len(result.SyncedFiles) != 3 {
		t.Fatalf("unexpected synced file count: %d", len(result.SyncedFiles))
	}
}

func TestPull_OverwritesExistingCanonicalSkillFiles(t *testing.T) {
	targetDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(targetDir, ".project-kb", "skills"), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	existingPath := filepath.Join(targetDir, ".project-kb", "skills", "process-inbox.md")
	if err := os.WriteFile(existingPath, []byte("old content"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	server := newSkillsServer(t)
	defer server.Close()

	if _, err := Pull(Options{
		TargetDir:  targetDir,
		APIBaseURL: server.URL,
		Owner:      defaultGitHubOwner,
		Repo:       defaultGitHubRepo,
		Ref:        defaultGitHubRef,
		HTTPClient: server.Client(),
	}); err != nil {
		t.Fatalf("Pull returned error: %v", err)
	}

	content, err := os.ReadFile(existingPath)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if string(content) == "old content" {
		t.Fatalf("expected file to be overwritten, got %q", string(content))
	}
}

func TestPull_LeavesOverridesUntouched(t *testing.T) {
	targetDir := t.TempDir()
	overridesDir := filepath.Join(targetDir, ".project-kb", "skills", "overrides")
	if err := os.MkdirAll(overridesDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	overridePath := filepath.Join(overridesDir, "process-inbox.md")
	if err := os.WriteFile(overridePath, []byte("override"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	server := newSkillsServer(t)
	defer server.Close()

	if _, err := Pull(Options{
		TargetDir:  targetDir,
		APIBaseURL: server.URL,
		Owner:      defaultGitHubOwner,
		Repo:       defaultGitHubRepo,
		Ref:        defaultGitHubRef,
		HTTPClient: server.Client(),
	}); err != nil {
		t.Fatalf("Pull returned error: %v", err)
	}

	content, err := os.ReadFile(overridePath)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if string(content) != "override" {
		t.Fatalf("override content changed unexpectedly: %q", string(content))
	}
}

func TestPull_MissingProjectKBFails(t *testing.T) {
	_, err := Pull(Options{TargetDir: t.TempDir()})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "Run `atami kb init` first") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPull_RemoteListingFailure(t *testing.T) {
	targetDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(targetDir, ".project-kb", "skills"), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err := Pull(Options{
		TargetDir:  targetDir,
		APIBaseURL: server.URL,
		Owner:      defaultGitHubOwner,
		Repo:       defaultGitHubRepo,
		Ref:        defaultGitHubRef,
		HTTPClient: server.Client(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "GitHub contents request returned 500 Internal Server Error") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func newSkillsServer(t *testing.T) *httptest.Server {
	t.Helper()

	var baseURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/Kosmic/atami-ai/contents/project-kb/skills":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[
				{"name":"process-inbox.md","type":"file","download_url":"` + baseURL + `/downloads/process-inbox.md"},
				{"name":"generate-output.md","type":"file","download_url":"` + baseURL + `/downloads/generate-output.md"},
				{"name":"release-notes.md","type":"file","download_url":"` + baseURL + `/downloads/release-notes.md"},
				{"name":"README.txt","type":"file","download_url":"` + baseURL + `/downloads/README.txt"},
				{"name":"nested","type":"dir","download_url":""}
			]`))
		case "/downloads/process-inbox.md":
			_, _ = w.Write([]byte("# downloaded process-inbox.md\n"))
		case "/downloads/generate-output.md":
			_, _ = w.Write([]byte("# downloaded generate-output.md\n"))
		case "/downloads/release-notes.md":
			_, _ = w.Write([]byte("# downloaded release-notes.md\n"))
		case "/downloads/README.txt":
			_, _ = w.Write([]byte("ignore me\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	baseURL = server.URL
	return server
}
