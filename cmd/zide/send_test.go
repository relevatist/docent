package main

import (
	"encoding/json/v2"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/relevatist/zide/internal/zellij"
)

// sendFake runs in tab two; its harness pane runs harness ("" for a shell).
func sendFake(t *testing.T, cfg, harness string) *zellij.Fake {
	t.Helper()
	b, err := os.ReadFile("../../internal/panes/testdata/two_tabs.json")
	if err != nil {
		t.Fatal(err)
	}
	if harness != "" {
		var all []map[string]any
		if err := json.Unmarshal(b, &all); err != nil {
			t.Fatal(err)
		}
		for _, p := range all {
			if p["title"] == "harness" && p["tab_id"] == 1.0 {
				p["terminal_command"] = harness
			}
		}
		if b, err = json.Marshal(all); err != nil {
			t.Fatal(err)
		}
	}
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	if cfg != "" {
		if err := os.MkdirAll(filepath.Join(dir, "zide"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "zide", "config.toml"), []byte(cfg), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("ZELLIJ_PANE_ID", "3")
	return &zellij.Fake{Out: map[string]string{"action list-panes --all --json": string(b)}}
}

func TestSend(t *testing.T) {
	paste := func(s string) []string { return []string{"action", "paste", "--pane-id", "terminal_4", "--", s} }
	focus := []string{"action", "focus-pane-id", "terminal_4"}
	list := []string{"action", "list-panes", "--all", "--json"}
	tests := []struct {
		name, cfg, harness, stdin string
		args                      []string
		want                      [][]string
	}{
		{"ref claude", "", "", "", []string{"ref", "src/a.go:10-20"}, [][]string{list, paste("@src/a.go:10-20 "), focus}},
		{"ref single", "", "", "", []string{"ref", "src/a.go:10-10"}, [][]string{list, paste("@src/a.go:10 "), focus}},
		{"ref aider", `harness = "aider --yes"`, "", "", []string{"ref", "src/a.go:10-20"}, [][]string{list, paste("src/a.go:10-20 "), focus}},
		{"ref tab omp", `harness = "aider"`, "/opt/bin/omp --model x", "", []string{"ref", "a.go:1"}, [][]string{list, paste("@a.go:1 "), focus}},
		{"ref tab cat", `harness = "claude"`, "cat", "", []string{"ref", "a.go:1"}, [][]string{list, paste("a.go:1 "), focus}},
		{"ref tab kind", `harness = "aider"`, "env ZIDE_HARNESS_KIND=omp sandbox-exec -f omp.sb omp", "", []string{"ref", "a.go:1"}, [][]string{list, paste("@a.go:1 "), focus}},
		{"ref config alias", "harness = \"box\"\n[harnesses.box]\ncommand = \"sandbox-exec omp\"\nkind = \"omp\"\n", "", "", []string{"ref", "a.go:1"}, [][]string{list, paste("@a.go:1 "), focus}},
		{"ref no config harness", `harness = ""`, "omp", "", []string{"ref", "a.go:1"}, [][]string{list, paste("@a.go:1 "), focus}},
		{"ref no harness", `harness = ""`, "", "", []string{"ref", "a.go:1"}, [][]string{list, paste("a.go:1 "), focus}},
		{"ref no focus", "focus_on_send = false", "", "", []string{"ref", "a.go:1"}, [][]string{list, paste("@a.go:1 ")}},
		{"text", "", "", "x := 1\ny := 2\n", []string{"text"}, [][]string{list, paste("x := 1\ny := 2\n"), focus}},
		{"text empty", "", "", "", []string{"text"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := sendFake(t, tt.cfg, tt.harness)
			if err := doSend(f, strings.NewReader(tt.stdin), tt.args); err != nil {
				t.Fatal(err)
			}
			if !slices.EqualFunc(f.Calls, tt.want, slices.Equal) {
				t.Errorf("calls = %q; want %q", f.Calls, tt.want)
			}
			for _, c := range f.Calls {
				if slices.Contains(c, "send-keys") || slices.Contains(c, "Enter") {
					t.Errorf("call %q may submit the harness input", c)
				}
			}
		})
	}
}

func TestSendUsageErrors(t *testing.T) {
	for _, args := range [][]string{nil, {"ref"}, {"ref", "a", "b"}, {"text", "x"}, {"bogus"}, {"ref", "a.go:9-1"}} {
		f := sendFake(t, "", "")
		if err := doSend(f, strings.NewReader(""), args); err == nil {
			t.Errorf("send(%q) = nil; want error", args)
		}
		if len(f.Calls) != 0 {
			t.Errorf("send(%q) calls = %q; want none", args, f.Calls)
		}
	}
}
