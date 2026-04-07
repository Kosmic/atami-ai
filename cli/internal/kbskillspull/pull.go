package kbskillspull

import (
	"crypto/sha256"
	"encoding/hex"
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
	stateDirName         = ".atami"
	stateFileName        = "kb-skills-state.json"
)

// Options configures a targeted pull of canonical project-kb skill files.
type Options struct {
	TargetDir  string
	APIBaseURL string
	Owner      string
	Repo       string
	Ref        string
	HTTPClient *http.Client
	Force      bool
}

// Result describes the files processed by a skills pull.
type Result struct {
	TargetDir      string
	UpdatedFiles   []string
	UnchangedFiles []string
	SkippedFiles   []SkippedFile
}

// SkippedFile records a file that was not overwritten during pull.
type SkippedFile struct {
	Path   string
	Reason string
}

type contentEntry struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	DownloadURL string `json:"download_url"`
}

type syncState struct {
	Source stateSource              `json:"source"`
	Files  map[string]stateFileInfo `json:"files"`
}

type stateSource struct {
	Owner string `json:"owner"`
	Repo  string `json:"repo"`
	Ref   string `json:"ref"`
}

type stateFileInfo struct {
	SyncedSHA256 string `json:"synced_sha256"`
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

	statePath := filepath.Join(projectKBPath, stateDirName, stateFileName)
	state, err := loadState(statePath)
	if err != nil {
		return Result{}, err
	}

	entries, err := listRemoteSkillFiles(opts)
	if err != nil {
		return Result{}, err
	}

	result := Result{TargetDir: opts.TargetDir}
	for _, entry := range entries {
		targetPath := filepath.Join(targetSkillsDir, entry.Name)
		relativePath := filepath.ToSlash(filepath.Join(".project-kb", "skills", entry.Name))

		remoteContent, err := fetchFileContent(opts.HTTPClient, entry.DownloadURL)
		if err != nil {
			return Result{}, fmt.Errorf("downloading %s: %w", entry.Name, err)
		}
		remoteHash := sha256Hex(remoteContent)

		fileState, tracked := state.Files[entry.Name]
		localContent, localExists, err := readFileIfExists(targetPath)
		if err != nil {
			return Result{}, fmt.Errorf("reading local skill file %q: %w", targetPath, err)
		}

		switch {
		case !tracked:
			if err := writeContent(targetPath, remoteContent); err != nil {
				return Result{}, fmt.Errorf("writing %s: %w", entry.Name, err)
			}
			result.UpdatedFiles = append(result.UpdatedFiles, relativePath)
			state.Files[entry.Name] = stateFileInfo{SyncedSHA256: remoteHash}
		case !localExists:
			if opts.Force {
				if err := writeContent(targetPath, remoteContent); err != nil {
					return Result{}, fmt.Errorf("writing %s: %w", entry.Name, err)
				}
				result.UpdatedFiles = append(result.UpdatedFiles, relativePath)
				state.Files[entry.Name] = stateFileInfo{SyncedSHA256: remoteHash}
				continue
			}
			result.SkippedFiles = append(result.SkippedFiles, SkippedFile{
				Path:   relativePath,
				Reason: "local file is missing after a previous sync",
			})
		default:
			localHash := sha256Hex(localContent)

			switch {
			case localHash == remoteHash:
				result.UnchangedFiles = append(result.UnchangedFiles, relativePath)
				state.Files[entry.Name] = stateFileInfo{SyncedSHA256: remoteHash}
			case localHash == fileState.SyncedSHA256:
				if err := writeContent(targetPath, remoteContent); err != nil {
					return Result{}, fmt.Errorf("writing %s: %w", entry.Name, err)
				}
				result.UpdatedFiles = append(result.UpdatedFiles, relativePath)
				state.Files[entry.Name] = stateFileInfo{SyncedSHA256: remoteHash}
			case opts.Force:
				if err := writeContent(targetPath, remoteContent); err != nil {
					return Result{}, fmt.Errorf("writing %s: %w", entry.Name, err)
				}
				result.UpdatedFiles = append(result.UpdatedFiles, relativePath)
				state.Files[entry.Name] = stateFileInfo{SyncedSHA256: remoteHash}
			default:
				reason := "local edits detected"
				if remoteHash != fileState.SyncedSHA256 {
					reason = "local edits conflict with newer remote content"
				}
				result.SkippedFiles = append(result.SkippedFiles, SkippedFile{
					Path:   relativePath,
					Reason: reason,
				})
			}
		}
	}

	state.Source = stateSource{
		Owner: opts.Owner,
		Repo:  opts.Repo,
		Ref:   opts.Ref,
	}

	if err := writeState(statePath, state); err != nil {
		return Result{}, err
	}

	return result, nil
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

func fetchFileContent(client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating download request: %w", err)
	}
	req.Header.Set("User-Agent", "atami-cli")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("requesting remote file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 4096))
		if readErr != nil {
			return nil, fmt.Errorf("remote file request returned %s", resp.Status)
		}
		return nil, fmt.Errorf("remote file request returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading remote file response: %w", err)
	}
	return content, nil
}

func loadState(path string) (syncState, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return syncState{Files: map[string]stateFileInfo{}}, nil
		}
		return syncState{}, fmt.Errorf("reading kb skills state: %w", err)
	}

	var state syncState
	if err := json.Unmarshal(content, &state); err != nil {
		return syncState{}, fmt.Errorf("parsing kb skills state: %w", err)
	}
	if state.Files == nil {
		state.Files = map[string]stateFileInfo{}
	}
	return state, nil
}

func writeState(path string, state syncState) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating kb skills state directory: %w", err)
	}

	content, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding kb skills state: %w", err)
	}
	content = append(content, '\n')

	if err := os.WriteFile(path, content, 0o644); err != nil {
		return fmt.Errorf("writing kb skills state: %w", err)
	}
	return nil
}

func readFileIfExists(path string) ([]byte, bool, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return content, true, nil
}

func writeContent(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating parent directory for %q: %w", path, err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return fmt.Errorf("writing target file %q: %w", path, err)
	}
	return nil
}

func sha256Hex(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func getenvOrDefault(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
