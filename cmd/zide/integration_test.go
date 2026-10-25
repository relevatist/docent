package main

import (
	"encoding/json/v2"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/relevatist/zide/internal/panes"
)

// sessionRunner runs zellij against one session with env.
type sessionRunner struct {
	session string
	env     []string
}

func (r sessionRunner) Run(args ...string) ([]byte, error) {
	cmd := exec.Command("zellij", append([]string{"--session", r.session}, args...)...)
	cmd.Env = r.env
	return cmd.Output()
}

// TestIntegrationOpen starts a real workstream session with a harness alias,
// checks the panes that zide new reads, and runs `zide open` from the shell pane.
func TestIntegrationOpen(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}
	for _, tool := range []string{"zellij", "hx", "script"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("no %s on PATH", tool)
		}
	}
	tmp := t.TempDir()
	zide := filepath.Join(tmp, "zide")
	if out, err := exec.Command("go", "build", "-o", zide, ".").CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	dir := filepath.Join(tmp, "ws")
	var lines strings.Builder
	for i := range 30 {
		fmt.Fprintf(&lines, "line %d\n", i+1)
	}
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "target.txt"), []byte(lines.String()), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := filepath.Join(tmp, "config")
	if err := os.MkdirAll(filepath.Join(cfg, "zide"), 0o755); err != nil {
		t.Fatal(err)
	}
	toml := "[harnesses.sandboxed]\ncommand = \"cat\"\nkind = \"omp\"\n"
	if err := os.WriteFile(filepath.Join(cfg, "zide", "config.toml"), []byte(toml), 0o644); err != nil {
		t.Fatal(err)
	}

	// Outside Zellij, so zide creates a session instead of a tab in ours.
	var env []string
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "ZELLIJ") && !strings.HasPrefix(kv, "SHELL=") {
			env = append(env, kv)
		}
	}
	env = append(env, "SHELL=/bin/sh", "XDG_STATE_HOME="+filepath.Join(tmp, "state"), "XDG_CONFIG_HOME="+cfg)
	session := fmt.Sprintf("zide-it-%d", os.Getpid())
	zj := func(args ...string) (string, error) {
		cmd := exec.Command("zellij", append([]string{"--session", session}, args...)...)
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		return string(out), err
	}

	// zellij needs a terminal client to spawn panes; script gives it one.
	launch := []string{zide, "--name", session, "--harness", "sandboxed", dir}
	scriptArgs := append([]string{"-q", "/dev/null"}, launch...)
	if runtime.GOOS == "linux" {
		// ponytail: joined without quoting; temp paths have no spaces.
		scriptArgs = []string{"-qfec", strings.Join(launch, " "), "/dev/null"}
	}
	client := exec.Command("script", scriptArgs...)
	client.Dir, client.Env = dir, env
	if err := client.Start(); err != nil {
		t.Fatal(err)
	}
	ids := map[string]string{}
	type pane struct {
		ID       int    `json:"id"`
		IsPlugin bool   `json:"is_plugin"`
		Exited   bool   `json:"exited"`
		Title    string `json:"title"`
		TabName  string `json:"tab_name"`
		Command  string `json:"terminal_command"`
		Cwd      string `json:"pane_cwd"`
	}
	listPanes := func() ([]pane, error) {
		out, err := zj("action", "list-panes", "--all", "--json")
		if err != nil {
			return nil, err
		}
		var ps []pane
		return ps, json.Unmarshal([]byte(out), &ps)
	}
	t.Cleanup(func() {
		// Quit Helix first: hx left behind on a closed pty spins at 100% CPU.
		running := func() (string, bool) {
			ps, _ := listPanes()
			i := slices.IndexFunc(ps, func(p pane) bool { return p.Title == "editor" && !p.IsPlugin && !p.Exited })
			if i < 0 {
				return "", false
			}
			return fmt.Sprintf("terminal_%d", ps[i].ID), true
		}
		logErr := func(_ string, err error) {
			if err != nil {
				t.Log("cleanup:", err)
			}
		}
		if ed, ok := running(); ok {
			logErr(zj("action", "send-keys", "--pane-id", ed, "Esc"))
			logErr(zj("action", "write-chars", "--pane-id", ed, ":q!"))
			logErr(zj("action", "send-keys", "--pane-id", ed, "Enter"))
			for range 25 {
				if _, ok := running(); !ok {
					break
				}
				time.Sleep(200 * time.Millisecond)
			}
		}
		logErr(zj("kill-session", session))
		logErr(zj("delete-session", session, "--force"))
		logErr("", client.Process.Kill())
		logErr("", client.Wait())
	})

	screen := func(pane string) string {
		out, _ := zj("action", "dump-screen", "--pane-id", pane)
		return out
	}
	poll := func(what string, f func() bool) {
		t.Helper()
		for deadline := time.Now().Add(20 * time.Second); time.Now().Before(deadline); time.Sleep(200 * time.Millisecond) {
			if f() {
				return
			}
		}
		t.Fatalf("timed out waiting for %s; editor screen:\n%s", what, screen(ids["editor"]))
	}
	poll("editor and shell panes", func() bool {
		ps, err := listPanes()
		if err != nil {
			return false
		}
		for _, p := range ps {
			if !p.IsPlugin {
				ids[p.Title] = fmt.Sprintf("terminal_%d", p.ID)
			}
		}
		return ids["editor"] != "" && ids["shell"] != ""
	})
	poll("helix to start", func() bool { return strings.Contains(screen(ids["editor"]), "NOR") })
	poll("shell prompt", func() bool { return strings.TrimSpace(screen(ids["shell"])) != "" })

	realDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	if ids["harness"] == "" {
		t.Error("no harness pane")
	}
	ps, err := listPanes()
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range ps {
		switch p.Title {
		case "editor":
			if p.TabName != session || p.Cwd != realDir {
				t.Errorf("editor pane: tab %q, cwd %q; want %q, %q", p.TabName, p.Cwd, session, realDir)
			}
		case "harness":
			if !strings.HasPrefix(p.Command, "/usr/bin/env "+panes.KindVar+"=omp ") {
				t.Errorf("harness command = %q; want the %s=omp wrapper", p.Command, panes.KindVar)
			}
		}
	}
	tabs, err := panes.Tabs(sessionRunner{session, env})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(tabs, func(tab panes.Tab) bool { return tab.Name == session && tab.Dir == realDir }) {
		t.Errorf("panes.Tabs = %+v; want tab %q with dir %q", tabs, session, realDir)
	}

	if out, err := zj("action", "write-chars", "--pane-id", ids["shell"], zide+" open target.txt:17\n"); err != nil {
		t.Fatalf("write-chars: %v\n%s", err, out)
	}
	poll("editor to show target.txt at line 17", func() bool {
		for l := range strings.Lines(screen(ids["editor"])) {
			if strings.Contains(l, "target.txt") && strings.Contains(l, "17:1") {
				return true
			}
		}
		return false
	})
}
