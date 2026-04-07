package kbskillspull

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	defaultGitHubAPIBase = "https://api.github.com"
	defaultGitHubOwner   = "Kosmic"
	defaultGitHubRepo    = "atami-ai"
	defaultGitHubRef     = "main"
	skillsAPIPath        = "project-kb/skills"
)

// Options configures a targeted pull of canonical project-kb skill files.
type Options struct {
	TargetDir  string
	APIBaseURL string
	Owner      string
	Repo       string
	Ref        string
	HTTPClient *http.Client
}

// Result describes the files synced by a skills pull.
type Result struct {
	TargetDir   string
	SyncedFiles []string
}

type contentEntry struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	DownloadURL string `json:"download_url"`
}

// Pull refreshes the canonical project-kb skill files in .project-kb/skills.
func Pull(opts Options) (Result, error) {
	if opts.TargetDir == "" {
		return Result{}, fmt.Errorf("target directory is required")
	}

	opts = withDefaults(opts)

	projectKBPath := filepath.Join(opts.TargetDir, ".project-kb")
	projectKBInfo, err := os.Stat(projectKBPath)
	if err != nil {
		if os.IsNotExist(err) {
			return Result{}, fmt.Errorf(".project-kb/ does not exist in this directory. Run `atami kb init` first.")
		}
		return Result{}, fmt.Errorf("checking .project-kb directory: %w", err)
	}
	if !projectKBInfo.IsDir() {
		return Result{}, fmt.Errorf(".project-kb exists in this directory but is not a directory")
	}

	targetSkillsDir := filepath.Join(projectKBPath, "skills")
	if err := os.MkdirAll(targetSkillsDir, 0o755); err != nil {
		return Result{}, fmt.Errorf("creating .project-kb/skills directory: %w", err)
	}

	entries, err := listRemoteSkillFiles(opts)
	if err != nil {
		return Result{}, err
	}

	var synced []string
	for _, entry := range entries {
		targetPath := filepath.Join(targetSkillsDir, entry.Name)
		if err := downloadFile(opts.HTTPClient, entry.DownloadURL, targetPath); err != nil {
			return Result{}, fmt.Errorf("downloading %s: %w", entry.Name, err)
		}
		synced = append(synced, filepath.ToSlash(filepath.Join(".project-kb", "skills", entry.Name)))
	}

	return Result{
		TargetDir:   opts.TargetDir,
		SyncedFiles: synced,
	}, nil
}

func withDefaults(opts Options) Options {
	if opts.APIBaseURL == "" {
		opts.APIBaseURL = getenvOrDefault("ATAMI_GITHUB_API_BASE", defaultGitHubAPIBase)
	}
	if opts.Owner == "" {
		opts.Owner = getenvOrDefault("ATAMI_GITHUB_OWNER", defaultGitHubOwner)
	}
	if opts.Repo == "" {
		opts.Repo = getenvOrDefault("ATAMI_GITHUB_REPO", defaultGitHubRepo)
	}
	if opts.Ref == "" {
		opts.Ref = getenvOrDefault("ATAMI_GITHUB_REF", defaultGitHubRef)
	}
	if opts.HTTPClient == nil {
		opts.HTTPClient = &http.Client{Timeout: 30 * time.Second}
	}
	return opts
}

func listRemoteSkillFiles(opts Options) ([]contentEntry, error) {
	url := strings.TrimRight(opts.APIBaseURL, "/") + "/repos/" + opts.Owner + "/" + opts.Repo + "/contents/" + skillsAPIPath + "?ref=" + opts.Ref
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating GitHub contents request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "atami-cli")

	resp, err := opts.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("requesting GitHub contents listing: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 4096))
		if readErr != nil {
			return nil, fmt.Errorf("GitHub contents request returned %s", resp.Status)
		}
		return nil, fmt.Errorf("GitHub contents request returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var entries []contentEntry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, fmt.Errorf("decoding GitHub contents response: %w", err)
	}

	filtered := make([]contentEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.Type != "file" {
			continue
		}
		if filepath.Ext(entry.Name) != ".md" {
			continue
		}
		if entry.DownloadURL == "" {
			return nil, fmt.Errorf("GitHub contents entry for %s is missing a download_url", entry.Name)
		}
		filtered = append(filtered, entry)
	}

	if len(filtered) == 0 {
		return nil, fmt.Errorf("no top-level .md files found in %s/%s %s at %s", opts.Owner, opts.Repo, skillsAPIPath, opts.Ref)
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Name < filtered[j].Name
	})

	return filtered, nil
}

func downloadFile(client *http.Client, url string, targetPath string) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("creating download request: %w", err)
	}
	req.Header.Set("User-Agent", "atami-cli")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("requesting remote file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 4096))
		if readErr != nil {
			return fmt.Errorf("remote file request returned %s", resp.Status)
		}
		return fmt.Errorf("remote file request returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return fmt.Errorf("creating parent directory for %q: %w", targetPath, err)
	}

	file, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("opening target file %q: %w", targetPath, err)
	}

	if _, err := io.Copy(file, resp.Body); err != nil {
		_ = file.Close()
		return fmt.Errorf("writing target file %q: %w", targetPath, err)
	}

	if err := file.Close(); err != nil {
		return fmt.Errorf("closing target file %q: %w", targetPath, err)
	}

	return nil
}

func getenvOrDefault(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
