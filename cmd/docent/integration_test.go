package main

import (
	"encoding/json/v2"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/relevatist/docent/assets"
	"github.com/relevatist/docent/internal/config"
	"github.com/relevatist/docent/internal/panes"
	"github.com/relevatist/docent/internal/tui"
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
// checks the built-in layout and the panes that docent new reads, runs
// `docent open` from the shell pane, opens a tab with a custom layout, and
// runs `docent new` from that tab.
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
	docent := filepath.Join(tmp, "docent")
	if out, err := exec.Command("go", "build", "-o", docent, ".").CombinedOutput(); err != nil {
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
	if err := os.MkdirAll(filepath.Join(cfg, "docent"), 0o755); err != nil {
		t.Fatal(err)
	}
	toml := "[harnesses.sandboxed]\ncommand = \"cat\"\nkind = \"omp\"\n"
	if err := os.WriteFile(filepath.Join(cfg, "docent", "config.toml"), []byte(toml), 0o644); err != nil {
		t.Fatal(err)
	}

	// A temp HOME keeps the user's Zellij config and cache out. The Zellij
	// config and the cache get what `docent install zellij` writes.
	home := filepath.Join(tmp, "home")
	plugin := filepath.Join(tmp, "docent.wasm")
	zcfg := filepath.Join(tmp, "zellij")
	zellijConfig, err := editMarked("show_startup_tips false\nshow_release_notes false\n", pluginNode(plugin), false)
	if err != nil {
		t.Fatal(err)
	}
	perms, ok := editPermissions("", plugin)
	if !ok {
		t.Fatal("editPermissions failed")
	}
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, ".cache"))
	for name, data := range map[string]string{
		plugin:                            string(assets.ZellijPlugin),
		filepath.Join(zcfg, "config.kdl"): zellijConfig,
		filepath.Join(zellijCacheDir(home), "permissions.kdl"): perms,
	} {
		if err := replaceFile(name, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Outside Zellij, so docent creates a session instead of a tab in ours.
	var env []string
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "ZELLIJ") && !strings.HasPrefix(kv, "SHELL=") && !strings.HasPrefix(kv, "HOME=") {
			env = append(env, kv)
		}
	}
	env = append(env, "SHELL=/bin/sh", "HOME="+home, "XDG_CACHE_HOME="+filepath.Join(home, ".cache"),
		"ZELLIJ_CONFIG_DIR="+zcfg, "XDG_STATE_HOME="+filepath.Join(tmp, "state"), "XDG_CONFIG_HOME="+cfg)
	session := fmt.Sprintf("docent-it-%d", os.Getpid())
	zj := func(args ...string) (string, error) {
		cmd := exec.Command("zellij", append([]string{"--session", session}, args...)...)
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		return string(out), err
	}

	// zellij needs a terminal client to spawn panes; script gives it one.
	launch := []string{docent, "--name", session, "--harness", "sandboxed", dir}
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
		Focused  bool   `json:"is_focused"`
		X        int    `json:"pane_x"`
		Y        int    `json:"pane_y"`
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
		for range 25 {
			ed, ok := running()
			if !ok {
				break
			}
			logErr(zj("action", "send-keys", "--pane-id", ed, "Esc"))
			logErr(zj("action", "write-chars", "--pane-id", ed, ":q!"))
			logErr(zj("action", "send-keys", "--pane-id", ed, "Enter"))
			time.Sleep(200 * time.Millisecond)
		}
		logErr(zj("kill-session", session))
		logErr("", client.Process.Kill())
		_ = client.Wait()
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
	byTitle := map[string]pane{}
	for _, p := range ps {
		byTitle[p.Title] = p
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
	if h, sh, ed := byTitle["harness"], byTitle["shell"], byTitle["editor"]; h.X >= sh.X || sh.X != ed.X || sh.Y >= ed.Y {
		t.Errorf("want harness left, shell top right, editor bottom right; got %+v", ps)
	}
	if !byTitle["harness"].Focused || byTitle["editor"].Focused || byTitle["shell"].Focused {
		t.Errorf("want focus on the harness only; got %+v", ps)
	}
	tabs, err := panes.Tabs(sessionRunner{session, env})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(tabs, func(tab panes.Tab) bool { return tab.Name == session && tab.Dir == realDir }) {
		t.Errorf("panes.Tabs = %+v; want tab %q with dir %q", tabs, session, realDir)
	}

	if out, err := zj("action", "write-chars", "--pane-id", ids["shell"], docent+" open target.txt:17\n"); err != nil {
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
	layout := filepath.Join(tmp, "my layouts", "old.kdl")
	if err := os.MkdirAll(filepath.Dir(layout), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(layout, []byte(fmt.Sprintf(testLayout, "custom")), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := zj("action", "write-chars", "--pane-id", ids["shell"], fmt.Sprintf("%s --name custom --harness cat --layout '%s' %s\n", docent, layout, dir)); err != nil {
		t.Fatalf("write-chars: %v\n%s", err, out)
	}
	var harness pane
	poll("custom layout tab", func() bool {
		ps, _ := listPanes()
		i := slices.IndexFunc(ps, func(p pane) bool { return p.TabName == "custom" && p.Title == "harness" })
		if i >= 0 {
			harness = ps[i]
		}
		return i >= 0
	})
	if want := "/usr/bin/env " + panes.KindVar + "=cat " + panes.LayoutVar + "=" + strings.ReplaceAll(layout, " ", "%20") + " cat"; harness.Command != want {
		t.Errorf("custom harness command = %q; want %q", harness.Command, want)
	}
	t.Setenv("ZELLIJ_PANE_ID", fmt.Sprint(harness.ID))
	if path, builtin := currentLayout(sessionRunner{session, env}, config.Config{}); path != layout || builtin {
		t.Errorf("currentLayout = %q, %v; want %q", path, builtin, layout)
	}

	// docent new from the custom tab opens a tab with the same layout marker.
	t.Setenv("ZELLIJ", "0")
	t.Setenv("XDG_CONFIG_HOME", cfg)
	t.Setenv("XDG_STATE_HOME", filepath.Join(tmp, "state"))
	pick := func(in tui.PickerInput) (tui.Picked, bool, error) {
		return tui.Picked{Dir: dir, Harness: in.Harness, Action: tui.Another}, true, nil
	}
	if err := newWorkstream(sessionRunner{session, env}, io.Discard, pick); err != nil {
		t.Fatal(err)
	}
	poll("docent new tab", func() bool {
		ps, _ := listPanes()
		i := slices.IndexFunc(ps, func(p pane) bool { return p.TabName == "ws-2" && p.Title == "harness" })
		if i >= 0 {
			harness = ps[i]
		}
		return i >= 0
	})
	if marker := " " + panes.LayoutVar + "=" + strings.ReplaceAll(layout, " ", "%20") + " "; !strings.Contains(harness.Command, marker) {
		t.Errorf("docent new harness command = %q; want %q", harness.Command, marker)
	}

	// A write from another tab keeps the focused pane of tab 1 scrolled.
	// `zellij action write` would scroll it to the bottom.
	if out, err := zj("action", "focus-pane-id", ids["shell"]); err != nil {
		t.Fatalf("focus-pane-id: %v\n%s", err, out)
	}
	if out, err := zj("action", "write-chars", "--pane-id", ids["shell"], "seq 300\n"); err != nil {
		t.Fatalf("write-chars: %v\n%s", err, out)
	}
	poll("seq output", func() bool { return strings.Contains(screen(ids["shell"]), "\n300\n") })
	if out, err := zj("action", "page-scroll-up", "--pane-id", ids["shell"]); err != nil {
		t.Fatalf("page-scroll-up: %v\n%s", err, out)
	}
	scrolled := screen(ids["shell"])
	if strings.Contains(scrolled, "\n300\n") {
		t.Fatalf("shell did not scroll:\n%s", scrolled)
	}

	// runIn runs docent in pane, as from a shell in that pane.
	runIn := func(pane, stdin string, args ...string) {
		t.Helper()
		cmd := exec.Command(docent, args...)
		cmd.Dir, cmd.Stdin = dir, strings.NewReader(stdin)
		cmd.Env = append(env, "ZELLIJ=0", "ZELLIJ_SESSION_NAME="+session, "ZELLIJ_PANE_ID="+strings.TrimPrefix(pane, "terminal_"))
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("docent %q: %v\n%s", args, err, out)
		}
	}
	runIn(fmt.Sprintf("terminal_%d", harness.ID), "", "reload")
	if got := screen(ids["shell"]); got != scrolled {
		t.Errorf("docent reload scrolled tab 1; shell screen:\n%s\nwant:\n%s", got, scrolled)
	}

	// With two clients there are two plugin instances; only one writes.
	attachArgs := []string{"-q", "/dev/null", "zellij", "attach", session}
	if runtime.GOOS == "linux" {
		attachArgs = []string{"-qfec", "zellij attach " + session, "/dev/null"}
	}
	// An open stdin keeps the client attached.
	stdin, stdinW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	client2 := exec.Command("script", attachArgs...)
	client2.Env, client2.Stdin = env, stdin
	if err := client2.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = client2.Process.Kill()
		_ = client2.Wait()
		_ = stdin.Close()
		_ = stdinW.Close()
	})
	poll("second client", func() bool {
		out, _ := zj("action", "list-clients")
		return len(strings.Split(strings.TrimSpace(out), "\n")) == 3
	})
	scrolled = screen(ids["shell"])
	runIn(ids["shell"], "", "open", "target.txt:3")
	poll("editor to show line 3", func() bool {
		for l := range strings.Lines(screen(ids["editor"])) {
			if strings.Contains(l, " 3:1") {
				return true
			}
		}
		return false
	})
	const marker = "pasted-once"
	runIn(ids["shell"], marker, "send", "text")
	poll("paste in the harness", func() bool { return strings.Contains(screen(ids["harness"]), marker) })
	time.Sleep(500 * time.Millisecond)
	if got := screen(ids["harness"]); strings.Count(got, marker) != 1 || strings.Contains(got, "200~") {
		t.Errorf("harness screen has %d pastes, or paste markers; want 1 paste and no markers:\n%s", strings.Count(got, marker), got)
	}

	// The plugin does not write after the deadline, nor without one.
	for _, args := range []string{"pane=" + ids["harness"] + ",deadline=1", "pane=" + ids["harness"]} {
		out, err := zj("pipe", "--name", "docent-write", "--args", args, "--", "not-written")
		if want := map[bool]string{true: "late", false: ""}[strings.Contains(args, "deadline")]; err != nil || out != want {
			t.Errorf("pipe --args %s = %q, %v; want %q", args, out, err, want)
		}
	}
	time.Sleep(500 * time.Millisecond)
	if got := screen(ids["harness"]); strings.Contains(got, "not-written") {
		t.Errorf("plugin wrote after the deadline or without one:\n%s", got)
	}
	if got := screen(ids["shell"]); got != scrolled {
		t.Errorf("writes with two clients scrolled tab 1; shell screen:\n%s\nwant:\n%s", got, scrolled)
	}
}
