package kbskillspull

import (
	"encoding/json"
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

	server := newSkillsServer(t, map[string]string{
		"process-inbox.md":   "# downloaded process-inbox.md\n",
		"generate-output.md": "# downloaded generate-output.md\n",
		"release-notes.md":   "# downloaded release-notes.md\n",
	})
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

	state := readStateFile(t, filepath.Join(targetDir, ".project-kb", stateDirName, stateFileName))
	if len(state.Files) != 3 {
		t.Fatalf("unexpected state file count: %d", len(state.Files))
	}
	if len(result.UpdatedFiles) != 3 {
		t.Fatalf("unexpected updated file count: %d", len(result.UpdatedFiles))
	}
}

func TestPull_TamperedLocalFileIsSkipped(t *testing.T) {
	targetDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(targetDir, ".project-kb", "skills"), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}

	server := newSkillsServer(t, map[string]string{
		"process-inbox.md": "# canonical process-inbox.md\n",
	})
	defer server.Close()

	if _, err := Pull(Options{
		TargetDir:  targetDir,
		APIBaseURL: server.URL,
		Owner:      defaultGitHubOwner,
		Repo:       defaultGitHubRepo,
		Ref:        defaultGitHubRef,
		HTTPClient: server.Client(),
	}); err != nil {
		t.Fatalf("initial Pull returned error: %v", err)
	}

	localPath := filepath.Join(targetDir, ".project-kb", "skills", "process-inbox.md")
	if err := os.WriteFile(localPath, []byte("# local edit\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := Pull(Options{
		TargetDir:  targetDir,
		APIBaseURL: server.URL,
		Owner:      defaultGitHubOwner,
		Repo:       defaultGitHubRepo,
		Ref:        defaultGitHubRef,
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("second Pull returned error: %v", err)
	}

	content, err := os.ReadFile(localPath)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if string(content) != "# local edit\n" {
		t.Fatalf("expected local edit to be preserved, got %q", string(content))
	}
	if len(result.SkippedFiles) != 1 {
		t.Fatalf("expected one skipped file, got %+v", result.SkippedFiles)
	}
	if result.SkippedFiles[0].Reason != "local edits detected" {
		t.Fatalf("unexpected skip reason: %+v", result.SkippedFiles[0])
	}
}

func TestPull_ForceOverwritesTamperedFile(t *testing.T) {
	targetDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(targetDir, ".project-kb", "skills"), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}

	server := newSkillsServer(t, map[string]string{
		"process-inbox.md": "# canonical process-inbox.md\n",
	})
	defer server.Close()

	if _, err := Pull(Options{
		TargetDir:  targetDir,
		APIBaseURL: server.URL,
		Owner:      defaultGitHubOwner,
		Repo:       defaultGitHubRepo,
		Ref:        defaultGitHubRef,
		HTTPClient: server.Client(),
	}); err != nil {
		t.Fatalf("initial Pull returned error: %v", err)
	}

	localPath := filepath.Join(targetDir, ".project-kb", "skills", "process-inbox.md")
	if err := os.WriteFile(localPath, []byte("# local edit\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := Pull(Options{
		TargetDir:  targetDir,
		APIBaseURL: server.URL,
		Owner:      defaultGitHubOwner,
		Repo:       defaultGitHubRepo,
		Ref:        defaultGitHubRef,
		HTTPClient: server.Client(),
		Force:      true,
	})
	if err != nil {
		t.Fatalf("force Pull returned error: %v", err)
	}

	content, err := os.ReadFile(localPath)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if string(content) != "# canonical process-inbox.md\n" {
		t.Fatalf("expected canonical content after force, got %q", string(content))
	}
	if len(result.UpdatedFiles) != 1 || len(result.SkippedFiles) != 0 {
		t.Fatalf("unexpected result: %+v", result)
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

	server := newSkillsServer(t, map[string]string{
		"process-inbox.md": "# canonical process-inbox.md\n",
	})
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

func newSkillsServer(t *testing.T, files map[string]string) *httptest.Server {
	t.Helper()

	var baseURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/Kosmic/atami-ai/contents/project-kb/skills":
			w.Header().Set("Content-Type", "application/json")
			var entries []string
			for name := range files {
				entries = append(entries, `{"name":"`+name+`","type":"file","download_url":"`+baseURL+`/downloads/`+name+`"}`)
			}
			entries = append(entries, `{"name":"README.txt","type":"file","download_url":"`+baseURL+`/downloads/README.txt"}`)
			entries = append(entries, `{"name":"nested","type":"dir","download_url":""}`)
			_, _ = w.Write([]byte("[" + strings.Join(entries, ",") + "]"))
		case "/downloads/README.txt":
			_, _ = w.Write([]byte("ignore me\n"))
		default:
			const prefix = "/downloads/"
			if !strings.HasPrefix(r.URL.Path, prefix) {
				http.NotFound(w, r)
				return
			}
			name := strings.TrimPrefix(r.URL.Path, prefix)
			content, ok := files[name]
			if !ok {
				http.NotFound(w, r)
				return
			}
			_, _ = w.Write([]byte(content))
		}
	}))
	baseURL = server.URL
	return server
}

func readStateFile(t *testing.T, path string) syncState {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}

	var state syncState
	if err := json.Unmarshal(content, &state); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}
	return state
}
