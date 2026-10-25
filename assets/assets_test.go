package assets

import (
	"encoding/json"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/relevatist/docent/internal/tour"
)

func TestSkillFrontmatter(t *testing.T) {
	front, body, ok := strings.Cut(strings.TrimPrefix(Skill, "---\n"), "\n---\n")
	if !strings.HasPrefix(Skill, "---\n") || !ok {
		t.Fatal("SKILL.md has no frontmatter")
	}
	if !regexp.MustCompile(`(?m)^name: docent$`).MatchString(front) {
		t.Errorf("frontmatter has no `name: docent`:\n%s", front)
	}
	if !regexp.MustCompile(`(?m)^description: \S.*$`).MatchString(front) {
		t.Errorf("frontmatter has no description:\n%s", front)
	}
	for _, s := range []string{"$ZELLIJ", "docent open", "docent reload", "docent tour", "unsaved", "pseudocode", "call tree", "component tree", "file tree", "```diff", "```mermaid", "sequenceDiagram", "erDiagram"} {
		if !strings.Contains(body, s) {
			t.Errorf("SKILL.md body does not mention %q", s)
		}
	}
	for _, h := range []string{"claude", "codex", "omp", "oh-my-pi", "PostToolUse"} {
		if regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(h) + `\b`).MatchString(body) {
			t.Errorf("SKILL.md body mentions harness-specific %q", h)
		}
	}
	// The tour pane cannot show HTML, and docent diff is gone.
	for _, s := range []string{"html", "browser", "docent diff"} {
		if strings.Contains(strings.ToLower(body), s) {
			t.Errorf("SKILL.md body mentions %q", s)
		}
	}
}

func TestSkillTourExampleIsValidJSON(t *testing.T) {
	_, rest, ok := strings.Cut(Skill, "```json\n")
	if !ok {
		t.Fatal("SKILL.md has no json example")
	}
	example, _, _ := strings.Cut(rest, "\n```\n")
	if _, err := tour.Parse([]byte(example)); err != nil {
		t.Fatal(err)
	}
}

func TestClaudeHooks(t *testing.T) {
	var cfg struct {
		Hooks map[string][]struct {
			Matcher string `json:"matcher"`
			Hooks   []struct {
				Type    string `json:"type"`
				Command string `json:"command"`
			} `json:"hooks"`
		} `json:"hooks"`
	}
	data, err := os.ReadFile("claude-plugin/hooks/hooks.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	groups := cfg.Hooks["PostToolUse"]
	if len(groups) != 1 || len(groups[0].Hooks) != 1 {
		t.Fatalf("want one PostToolUse hook, got %+v", cfg)
	}
	m := regexp.MustCompile("^(" + groups[0].Matcher + ")$")
	for _, tool := range []string{"Edit", "Write", "MultiEdit", "NotebookEdit"} {
		if !m.MatchString(tool) {
			t.Errorf("matcher %q misses %s", groups[0].Matcher, tool)
		}
	}
	if m.MatchString("Read") || m.MatchString("Bash") {
		t.Errorf("matcher %q matches read-only tools", groups[0].Matcher)
	}
	if h := groups[0].Hooks[0]; h.Type != "command" || h.Command != "docent reload --quiet" {
		t.Errorf("hook = %+v", h)
	}
}

func TestClaudePluginFiles(t *testing.T) {
	for _, name := range []string{".claude-plugin/plugin.json", "hooks/hooks.json", "skills/docent/SKILL.md"} {
		if _, err := fs.Stat(ClaudePlugin, name); err != nil {
			t.Error(err)
		}
	}
}

func TestZellijKeys(t *testing.T) {
	if _, err := exec.LookPath("zellij"); err != nil || testing.Short() {
		t.Skip("needs zellij")
	}
	dir := t.TempDir()
	data, err := os.ReadFile("zellij/keys.kdl")
	if err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(dir, "config.kdl")
	if err := os.WriteFile(cfg, data, 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command("zellij", "--config", cfg, "--config-dir", dir, "setup", "--check").CombinedOutput()
	if err != nil || !strings.Contains(string(out), "[CONFIG FILE]: Well defined.") {
		t.Errorf("zellij setup --check: %v\n%s", err, out)
	}
}
