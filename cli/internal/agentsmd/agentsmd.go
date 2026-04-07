package agentsmd

import (
	"fmt"
	"os"
	"strings"
)

const beginMarker = "<!-- BEGIN atami project-kb -->"

// AppendResult describes how AGENTS.md was changed.
type AppendResult struct {
	FileCreated  bool
	SnippetAdded bool
}

// AppendSnippet creates or updates AGENTS.md with the provided project-kb snippet.
func AppendSnippet(targetPath string, snippet string) (AppendResult, error) {
	content, err := os.ReadFile(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			initial := "# AGENTS.md\n\n" + snippet
			if writeErr := os.WriteFile(targetPath, []byte(initial), 0o644); writeErr != nil {
				return AppendResult{}, fmt.Errorf("writing new AGENTS.md: %w", writeErr)
			}
			return AppendResult{FileCreated: true, SnippetAdded: true}, nil
		}
		return AppendResult{}, fmt.Errorf("reading AGENTS.md: %w", err)
	}

	if strings.Contains(string(content), beginMarker) {
		return AppendResult{}, nil
	}

	separator := "\n\n"
	switch {
	case len(content) == 0:
		separator = ""
	case strings.HasSuffix(string(content), "\n\n"):
		separator = ""
	case strings.HasSuffix(string(content), "\n"):
		separator = "\n"
	}

	updated := string(content) + separator + snippet
	if err := os.WriteFile(targetPath, []byte(updated), 0o644); err != nil {
		return AppendResult{}, fmt.Errorf("writing AGENTS.md: %w", err)
	}

	return AppendResult{SnippetAdded: true}, nil
}
