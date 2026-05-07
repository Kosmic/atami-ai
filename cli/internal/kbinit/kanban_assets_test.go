package kbinit

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestKanbanTemplateBuildsTypeColumnsFromEffectiveStatus(t *testing.T) {
	template := readRepoFile(t, "project-kb", "skills", "kanban-board.template.html")

	if strings.Contains(template, "const activeItems = ITEMS.filter(i => i.status !== 'done' && i.status !== 'dropped');") {
		t.Fatal("type columns still use source status only")
	}
	if !strings.Contains(template, "const activeItems = ITEMS.filter(i => {\n      const eff = effectiveItem(i);") {
		t.Fatal("type columns do not inspect effective item state")
	}
	if !strings.Contains(template, "return s !== 'done' && s !== 'dropped';") {
		t.Fatal("type columns do not filter by effective active status")
	}
}

func TestKanbanSkillReplacesOutputWhenMarkersAreMissing(t *testing.T) {
	skill := readRepoFile(t, "project-kb", "skills", "kanban-board.md")

	if !strings.Contains(skill, "or exists but does not contain both `KANBAN_VARS_START` and `KANBAN_ITEMS_START` markers") {
		t.Fatal("skill does not tell agents to replace stale generated boards")
	}
}

func TestKanbanTemplateClipboardFallbackDoesNotAssumeClipboardAPI(t *testing.T) {
	template := readRepoFile(t, "project-kb", "skills", "kanban-board.template.html")

	if !strings.Contains(template, "function copyText(text, successMessage)") {
		t.Fatal("missing shared clipboard helper")
	}
	if !strings.Contains(template, "navigator.clipboard && typeof navigator.clipboard.writeText === 'function'") {
		t.Fatal("clipboard helper does not guard navigator.clipboard")
	}
	if strings.Contains(template, "navigator.clipboard.writeText(item.source).then") {
		t.Fatal("source copy button still calls navigator.clipboard directly")
	}
	if !strings.Contains(template, "copyText(text, 'Copied to clipboard") {
		t.Fatal("instructions copy does not use shared clipboard helper")
	}
}

func TestKanbanTemplateEscapesMarkdownLinkHrefAttributes(t *testing.T) {
	template := readRepoFile(t, "project-kb", "skills", "kanban-board.template.html")

	if !strings.Contains(template, "function escAttr(s)") {
		t.Fatal("missing attribute escaping helper")
	}
	if !strings.Contains(template, "'<a href=\"' + escAttr(url) + '\"'") {
		t.Fatal("markdown links do not use attribute escaping for href")
	}
}

func readRepoFile(t *testing.T, parts ...string) string {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", ".."))
	path := filepath.Join(append([]string{repoRoot}, parts...)...)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) returned error: %v", path, err)
	}
	return string(content)
}
