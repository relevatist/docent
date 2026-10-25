package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/relevatist/zide/internal/tui"
	"github.com/relevatist/zide/internal/zellij"
)

const sandboxCfg = "harness = \"codex\"\n[harnesses.omp-sandbox]\ncommand = \"sandbox-exec -f omp.sb omp\"\nkind = \"omp\"\n"

// newFake is a session with tab app (id 4) open for dir and tab app-2 (id 7),
// where zide new runs as pane 5; harness is the app-2 harness pane command,
// or "" for no harness pane.
func newFake(t *testing.T, cfg, dir, harness string) *zellij.Fake {
	t.Helper()
	conf := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", conf)
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	writeTestFile(t, filepath.Join(conf, "zide", "config.toml"), cfg)
	t.Setenv("ZELLIJ", "0")
	t.Setenv("ZELLIJ_PANE_ID", "5")
	harnessPane := ""
	if harness != "" {
		harnessPane = fmt.Sprintf(`{"id":3,"title":"harness","tab_id":7,"tab_name":"app-2","terminal_command":%q},`, harness)
	}
	return &zellij.Fake{Out: map[string]string{
		"--version": "zellij 0.45.1",
		"action list-panes --all --json": fmt.Sprintf(`[
			{"id":0,"title":"editor","tab_id":4,"tab_name":"app","pane_cwd":%q,"terminal_command":"hx"},
			{"id":2,"title":"editor","tab_id":7,"tab_name":"app-2","pane_cwd":"/elsewhere","terminal_command":"hx"},
			%s
			{"id":5,"title":"zide new","tab_id":7,"tab_name":"app-2","is_floating":true}]`, dir, harnessPane),
	}}
}

// appDir returns a directory named app, and the same directory as list-panes
// shows it (symlinks resolved, as for /tmp on macOS).
func appDir(t *testing.T) (dir, cwd string) {
	t.Helper()
	dir = filepath.Join(t.TempDir(), "app")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	cwd, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	return dir, cwd
}

func pickReturns(p tui.Picked) func(tui.PickerInput) (tui.Picked, bool, error) {
	return func(tui.PickerInput) (tui.Picked, bool, error) { return p, true, nil }
}

func TestNewOutsideZellij(t *testing.T) {
	t.Setenv("ZELLIJ", "")
	f := &zellij.Fake{}
	var w bytes.Buffer
	pick := func(tui.PickerInput) (tui.Picked, bool, error) {
		t.Fatal("picker shown outside Zellij")
		return tui.Picked{}, false, nil
	}
	if code := newCmd(f, nil, &w, pick); code == 0 {
		t.Error("want non-zero exit")
	}
	if !strings.Contains(w.String(), "must run inside Zellij") || len(f.Calls) != 0 {
		t.Errorf("output %q, calls %q", w.String(), f.Calls)
	}
}

func TestNewDefaultHarness(t *testing.T) {
	for _, tt := range []struct{ harness, want string }{
		{"/usr/bin/env ZIDE_HARNESS_KIND=omp sandbox-exec -f omp.sb omp", "omp-sandbox"},
		{"omp --model x", "omp --model x"},
		{"", "codex"},
	} {
		dir, cwd := appDir(t)
		f := newFake(t, sandboxCfg, cwd, tt.harness)
		var in tui.PickerInput
		pick := func(got tui.PickerInput) (tui.Picked, bool, error) { in = got; return tui.Picked{}, false, nil }
		if code := newCmd(f, nil, os.Stderr, pick); code != 0 {
			t.Fatalf("exit %d", code)
		}
		if in.Harness != tt.want || !slices.Equal(in.Aliases, []string{"omp-sandbox"}) {
			t.Errorf("harness %q: input %+v; want harness %q", tt.harness, in, tt.want)
		}
		if tab, ok := in.IsOpen(dir); !ok || tab != "app" {
			t.Errorf("IsOpen(%s) = %q, %v; want app", dir, tab, ok)
		}
		if _, ok := in.IsOpen("/elsewhere/x"); ok {
			t.Error("IsOpen true for a dir that is not open")
		}
		if hasCall(f, "action", "new-tab") || len(f.Calls) > 2 {
			t.Errorf("calls after cancel: %q", f.Calls)
		}
	}
}

func lastNewTab(t *testing.T, f *zellij.Fake) []string {
	t.Helper()
	last := f.Calls[len(f.Calls)-1]
	if !slices.Equal(last[:2], []string{"action", "new-tab"}) {
		t.Fatalf("last call %q; want new-tab", last)
	}
	return last
}

func TestNewOpen(t *testing.T) {
	dir, _ := appDir(t)
	f := newFake(t, sandboxCfg, "/other", "claude")
	if code := newCmd(f, nil, os.Stderr, pickReturns(tui.Picked{Dir: dir, Harness: "omp-sandbox"})); code != 0 {
		t.Fatalf("exit %d", code)
	}
	tab := lastNewTab(t, f)
	if !strings.Contains(tab[3], `"-f" "omp.sb" "omp"`) || !slices.Equal(tab[4:], []string{"--name", "app", "--cwd", dir}) {
		t.Errorf("new-tab %q", tab)
	}
}

func TestNewSwitch(t *testing.T) {
	dir, cwd := appDir(t)
	f := newFake(t, "", cwd, "claude")
	if code := newCmd(f, nil, os.Stderr, pickReturns(tui.Picked{Dir: dir, Harness: "claude", Action: tui.Switch})); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if last := f.Calls[len(f.Calls)-1]; !slices.Equal(last, []string{"action", "go-to-tab-by-id", "4"}) {
		t.Errorf("last call %q; want go-to-tab-by-id 4", last)
	}
}

func TestNewAnother(t *testing.T) {
	dir, cwd := appDir(t)
	f := newFake(t, "", cwd, "claude")
	if code := newCmd(f, nil, os.Stderr, pickReturns(tui.Picked{Dir: dir, Harness: "claude", Action: tui.Another})); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if tab := lastNewTab(t, f); !slices.Equal(tab[4:6], []string{"--name", "app-3"}) {
		t.Errorf("new-tab %q; want --name app-3", tab)
	}
}

func TestNewHelp(t *testing.T) {
	var w bytes.Buffer
	if code := newCmd(&zellij.Fake{}, []string{"--help"}, &w, nil); code != 0 || !strings.Contains(w.String(), "Usage: zide new") {
		t.Errorf("exit %d: %q", code, w.String())
	}
	if code := newCmd(&zellij.Fake{}, []string{"x"}, &w, nil); code != 2 {
		t.Errorf("extra arg: exit %d; want 2", code)
	}
}

func TestNewKeepsKind(t *testing.T) {
	for _, tt := range []struct{ picked, want string }{
		{"wrap omp", `"/usr/bin/env" {` + "\n" + `                    args "ZIDE_HARNESS_KIND=omp" "wrap" "omp"`},
		{"wrap omp --x", `"wrap" {` + "\n" + `                    args "omp" "--x"`},
	} {
		dir, _ := appDir(t)
		f := newFake(t, sandboxCfg, "/other", "/usr/bin/env ZIDE_HARNESS_KIND=omp wrap omp")
		pick := func(in tui.PickerInput) (tui.Picked, bool, error) {
			if in.Harness != "wrap omp" {
				t.Errorf("picker harness %q; want wrap omp", in.Harness)
			}
			return tui.Picked{Dir: dir, Harness: tt.picked}, true, nil
		}
		if code := newCmd(f, nil, os.Stderr, pick); code != 0 {
			t.Fatalf("exit %d", code)
		}
		if tab := lastNewTab(t, f); !strings.Contains(tab[3], tt.want) {
			t.Errorf("picked %q: layout\n%s\nwant %q", tt.picked, tab[3], tt.want)
		}
	}
}

func TestNewRecentWarning(t *testing.T) {
	f := newFake(t, "", "/other", "claude")
	state := t.TempDir()
	t.Setenv("XDG_STATE_HOME", state)
	writeTestFile(t, filepath.Join(state, "zide", "recent.toml"), "dirs = [")
	var in tui.PickerInput
	pick := func(got tui.PickerInput) (tui.Picked, bool, error) { in = got; return tui.Picked{}, false, nil }
	var w bytes.Buffer
	if code := newCmd(f, nil, &w, pick); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(in.Warning, "recent directories: toml:") || w.Len() != 0 {
		t.Errorf("warning %q, output %q; want the warning in the picker only", in.Warning, w.String())
	}
}

type readCount struct{ n int }

func (r *readCount) Read(b []byte) (int, error) {
	r.n++
	return copy(b, "\n"), nil
}

func TestNewPause(t *testing.T) {
	for _, tt := range []struct {
		name          string
		zellij        string
		tty, readOnly bool
		pick          tui.Picked
		want          bool
	}{
		{"success", "0", true, false, tui.Picked{Harness: "cat"}, false},
		{"warning", "0", true, true, tui.Picked{Harness: "cat"}, true},
		{"error", "0", true, false, tui.Picked{Harness: " "}, true},
		{"outside Zellij", "", true, false, tui.Picked{}, false},
		{"no terminal", "0", false, false, tui.Picked{Harness: " "}, false},
	} {
		dir, _ := appDir(t)
		f := newFake(t, "", "/other", "claude")
		t.Setenv("ZELLIJ", tt.zellij)
		if tt.readOnly {
			notDir := filepath.Join(t.TempDir(), "file")
			writeTestFile(t, notDir, "")
			t.Setenv("XDG_STATE_HOME", notDir)
		}
		tt.pick.Dir = dir
		var r readCount
		var w bytes.Buffer
		newAndPause(f, nil, pickReturns(tt.pick), tt.tty, &r, &w)
		if paused := r.n > 0 && strings.HasSuffix(w.String(), "press enter to close"); paused != tt.want {
			t.Errorf("%s: reads %d, output %q; want pause %v", tt.name, r.n, w.String(), tt.want)
		}
	}
}
