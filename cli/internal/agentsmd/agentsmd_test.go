package agentsmd

import (
	"os"
	"path/filepath"
	"testing"
)

const testSnippet = "<!-- BEGIN atami project-kb -->\nTest snippet\n<!-- END atami project-kb -->\n"

func TestAppendSnippet_NewFile(t *testing.T) {
	targetPath := filepath.Join(t.TempDir(), "AGENTS.md")

	result, err := AppendSnippet(targetPath, testSnippet)
	if err != nil {
		t.Fatalf("AppendSnippet returned error: %v", err)
	}

	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}

	expected := "# AGENTS.md\n\n" + testSnippet
	if string(content) != expected {
		t.Fatalf("unexpected AGENTS.md content:\n%s", string(content))
	}
	if !result.FileCreated || !result.SnippetAdded {
		t.Fatalf("unexpected append result: %+v", result)
	}
}

func TestAppendSnippet_ExistingFileWithoutSnippet(t *testing.T) {
	targetPath := filepath.Join(t.TempDir(), "AGENTS.md")
	original := "# Existing project rules\n\nDo not commit secrets.\n"
	if err := os.WriteFile(targetPath, []byte(original), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := AppendSnippet(targetPath, testSnippet)
	if err != nil {
		t.Fatalf("AppendSnippet returned error: %v", err)
	}

	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}

	expected := original + "\n" + testSnippet
	if string(content) != expected {
		t.Fatalf("unexpected AGENTS.md content:\n%s", string(content))
	}
	if result.FileCreated || !result.SnippetAdded {
		t.Fatalf("unexpected append result: %+v", result)
	}
}

func TestAppendSnippet_ExistingFileWithSnippet(t *testing.T) {
	targetPath := filepath.Join(t.TempDir(), "AGENTS.md")
	original := "# AGENTS.md\n\n" + testSnippet
	if err := os.WriteFile(targetPath, []byte(original), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := AppendSnippet(targetPath, testSnippet)
	if err != nil {
		t.Fatalf("AppendSnippet returned error: %v", err)
	}

	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}

	if string(content) != original {
		t.Fatalf("AGENTS.md should be unchanged:\n%s", string(content))
	}
	if result.FileCreated || result.SnippetAdded {
		t.Fatalf("unexpected append result: %+v", result)
	}
}
