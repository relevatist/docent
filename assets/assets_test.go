package assets

import (
	"encoding/json"
	"io/fs"
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestSkillFrontmatter(t *testing.T) {
	front, body, ok := strings.Cut(strings.TrimPrefix(Skill, "---\n"), "\n---\n")
	if !strings.HasPrefix(Skill, "---\n") || !ok {
		t.Fatal("SKILL.md has no frontmatter")
	}
	if !regexp.MustCompile(`(?m)^name: zide$`).MatchString(front) {
		t.Errorf("frontmatter has no `name: zide`:\n%s", front)
	}
	if !regexp.MustCompile(`(?m)^description: \S.*$`).MatchString(front) {
		t.Errorf("frontmatter has no description:\n%s", front)
	}
	for _, s := range []string{"$ZELLIJ", "zide open", "zide reload", "zide tour", "zide diff", "unsaved"} {
		if !strings.Contains(body, s) {
			t.Errorf("SKILL.md body does not mention %q", s)
		}
	}
	for _, h := range []string{"claude", "codex", "omp", "oh-my-pi", "PostToolUse"} {
		if strings.Contains(strings.ToLower(body), strings.ToLower(h)) {
			t.Errorf("SKILL.md body mentions harness-specific %q", h)
		}
	}
}

func TestSkillTourExampleIsValidJSON(t *testing.T) {
	_, rest, ok := strings.Cut(Skill, "```json\n")
	if !ok {
		t.Fatal("SKILL.md has no json example")
	}
	example, _, _ := strings.Cut(rest, "```")
	var tour struct {
		Steps []struct {
			File  string `json:"file"`
			Lines []int  `json:"lines"`
		} `json:"steps"`
	}
	if err := json.Unmarshal([]byte(example), &tour); err != nil {
		t.Fatal(err)
	}
	if len(tour.Steps) == 0 || tour.Steps[0].File == "" {
		t.Errorf("example has no steps: %+v", tour)
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
	if h := groups[0].Hooks[0]; h.Type != "command" || h.Command != "zide reload --quiet" {
		t.Errorf("hook = %+v", h)
	}
}

func TestClaudePluginFiles(t *testing.T) {
	for _, name := range []string{".claude-plugin/plugin.json", "hooks/hooks.json", "skills/zide/SKILL.md"} {
		if _, err := fs.Stat(ClaudePlugin, name); err != nil {
			t.Error(err)
		}
	}
}
