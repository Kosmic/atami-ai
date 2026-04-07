package templatefs

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultGitHubOwner   = "Kosmic"
	defaultGitHubRepo    = "atami-ai"
	defaultGitHubRef     = "main"
	defaultGitHubAPIBase = "https://api.github.com"
)

// ResolvedPaths contains the validated filesystem paths needed by the CLI.
type ResolvedPaths struct {
	RepoRoot    string
	TemplateDir string
	SkillsDir   string
}

type resolverOptions struct {
	lookupEnv   func(string) (string, bool)
	httpClient  *http.Client
	mkdirTemp   func(string, string) (string, error)
	remote      remoteConfig
	fetchRemote func(remoteConfig) (ResolvedPaths, error)
}

type remoteConfig struct {
	apiBaseURL string
	owner      string
	repo       string
	ref        string
	client     *http.Client
	mkdirTemp  func(string, string) (string, error)
}

// ResolvePaths locates and validates the atami-ai source needed by the CLI, preferring
// local development checkouts and falling back to the canonical GitHub repository.
func ResolvePaths(explicitTemplate string) (ResolvedPaths, error) {
	return resolvePaths(explicitTemplate, resolverOptions{
		lookupEnv: os.LookupEnv,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		mkdirTemp:   os.MkdirTemp,
		fetchRemote: fetchRemotePaths,
		remote: remoteConfig{
			apiBaseURL: defaultGitHubAPIBase,
			owner:      getenvOrDefault(os.LookupEnv, "ATAMI_GITHUB_OWNER", defaultGitHubOwner),
			repo:       getenvOrDefault(os.LookupEnv, "ATAMI_GITHUB_REPO", defaultGitHubRepo),
			ref:        getenvOrDefault(os.LookupEnv, "ATAMI_GITHUB_REF", defaultGitHubRef),
		},
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

	if opts.fetchRemote != nil {
		remoteOpts := opts.remote
		if remoteOpts.client == nil {
			remoteOpts.client = opts.httpClient
		}
		if remoteOpts.mkdirTemp == nil {
			remoteOpts.mkdirTemp = opts.mkdirTemp
		}

		resolved, err := opts.fetchRemote(remoteOpts)
		if err == nil {
			return resolved, nil
		}

		return ResolvedPaths{}, fmt.Errorf("could not find a local atami-ai repo and failed to fetch %s/%s@%s from GitHub: %w", remoteOpts.owner, remoteOpts.repo, remoteOpts.ref, err)
	}

	return ResolvedPaths{}, fmt.Errorf("could not resolve the atami-ai source; use --template-source, set ATAMI_AI_PATH, or enable the default GitHub fetch")
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

func fetchRemotePaths(cfg remoteConfig) (ResolvedPaths, error) {
	if cfg.apiBaseURL == "" {
		cfg.apiBaseURL = defaultGitHubAPIBase
	}
	if cfg.owner == "" {
		cfg.owner = defaultGitHubOwner
	}
	if cfg.repo == "" {
		cfg.repo = defaultGitHubRepo
	}
	if cfg.ref == "" {
		cfg.ref = defaultGitHubRef
	}
	if cfg.client == nil {
		cfg.client = &http.Client{Timeout: 30 * time.Second}
	}
	if cfg.mkdirTemp == nil {
		cfg.mkdirTemp = os.MkdirTemp
	}

	repoRoot, err := downloadGitHubTarball(cfg)
	if err != nil {
		return ResolvedPaths{}, err
	}

	resolved, err := newResolvedPaths(repoRoot)
	if err != nil {
		return ResolvedPaths{}, err
	}
	if err := validateResolvedPaths(resolved); err != nil {
		return ResolvedPaths{}, err
	}
	return resolved, nil
}

func downloadGitHubTarball(cfg remoteConfig) (string, error) {
	tempDir, err := cfg.mkdirTemp("", "atami-source-*")
	if err != nil {
		return "", fmt.Errorf("creating temporary directory for remote source: %w", err)
	}

	url := strings.TrimRight(cfg.apiBaseURL, "/") + "/repos/" + cfg.owner + "/" + cfg.repo + "/tarball/" + cfg.ref
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("creating GitHub archive request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "atami-cli")

	resp, err := cfg.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("downloading GitHub archive: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 4096))
		if readErr != nil {
			return "", fmt.Errorf("GitHub archive request returned %s", resp.Status)
		}
		return "", fmt.Errorf("GitHub archive request returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	gzipReader, err := gzip.NewReader(resp.Body)
	if err != nil {
		return "", fmt.Errorf("opening downloaded archive: %w", err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	var repoPrefix string

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("reading downloaded archive: %w", err)
		}

		name := strings.TrimPrefix(header.Name, "./")
		name = filepath.ToSlash(filepath.Clean(name))
		if name == "." || name == "" {
			continue
		}

		parts := strings.Split(name, "/")
		if len(parts) == 1 {
			repoPrefix = parts[0]
			continue
		}
		if repoPrefix == "" {
			repoPrefix = parts[0]
		}

		relativePath := filepath.Join(parts[1:]...)
		if relativePath == "." || relativePath == "" {
			continue
		}
		if strings.HasPrefix(relativePath, "..") {
			return "", fmt.Errorf("downloaded archive contains invalid path %q", header.Name)
		}

		targetPath := filepath.Join(tempDir, repoPrefix, relativePath)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode).Perm()); err != nil {
				return "", fmt.Errorf("creating directory %q from archive: %w", targetPath, err)
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
				return "", fmt.Errorf("creating parent directory for %q: %w", targetPath, err)
			}
			file, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(header.Mode).Perm())
			if err != nil {
				return "", fmt.Errorf("creating file %q from archive: %w", targetPath, err)
			}
			if _, err := io.Copy(file, tarReader); err != nil {
				_ = file.Close()
				return "", fmt.Errorf("writing file %q from archive: %w", targetPath, err)
			}
			if err := file.Close(); err != nil {
				return "", fmt.Errorf("closing file %q from archive: %w", targetPath, err)
			}
		}
	}

	if repoPrefix == "" {
		return "", fmt.Errorf("downloaded archive did not contain a repository root")
	}

	return filepath.Join(tempDir, repoPrefix), nil
}

func getenvOrDefault(lookup func(string) (string, bool), key string, fallback string) string {
	if value, ok := lookup(key); ok && value != "" {
		return value
	}
	return fallback
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
