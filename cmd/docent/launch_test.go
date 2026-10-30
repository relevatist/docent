package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/relevatist/docent/internal/recent"
	"github.com/relevatist/docent/internal/zellij"
)

func TestLaunchMissingDir(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if code := runLaunch([]string{"--harness", "cat", "/no/such/dir"}); code == 0 {
		t.Error("want non-zero exit")
	}
}

func TestLaunchTooManyArgs(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if code := runLaunch([]string{"a", "b"}); code != 2 {
		t.Errorf("got %d, want 2", code)
	}
}

// launchInZellij runs launchDir inside Zellij with config cfg and returns
// the layout passed to new-tab.
func launchInZellij(t *testing.T, cfg, harness string, w io.Writer) string {
	t.Helper()
	conf := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", conf)
	if err := os.MkdirAll(filepath.Join(conf, "docent"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(conf, "docent", "config.toml"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ZELLIJ", "0")
	f := &zellij.Fake{Out: map[string]string{"--version": "zellij 0.45.1"}}
	if err := launchDir(f, nil, w, t.TempDir(), "", harness, "", "", false); err != nil {
		t.Fatal(err)
	}
	last := f.Calls[len(f.Calls)-1]
	if !slices.Equal(last[:2], []string{"action", "new-tab"}) {
		t.Fatalf("last call %q; want new-tab", last)
	}
	return last[3]
}

func TestLaunchAlias(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	const cfg = "[harnesses.omp-sandbox]\ncommand = \"sandbox-exec -f omp.sb omp\"\nkind = \"omp\"\n"
	for _, kdl := range []string{
		launchInZellij(t, cfg, "omp-sandbox", io.Discard),
		launchInZellij(t, "harness = \"omp-sandbox\"\n"+cfg, "", io.Discard),
	} {
		if !strings.Contains(kdl, `"sandbox-exec"`) || !strings.Contains(kdl, `"-f" "omp.sb" "omp"`) {
			t.Errorf("layout does not run the alias command:\n%s", kdl)
		}
	}
}

func TestLaunchRecordsRecent(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	conf := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", conf)
	t.Setenv("ZELLIJ", "0")
	dir := t.TempDir()
	f := &zellij.Fake{Out: map[string]string{"--version": "zellij 0.45.1"}}
	var w bytes.Buffer
	if err := launchDir(f, nil, &w, dir, "", "cat", "", "", false); err != nil {
		t.Fatal(err)
	}
	if dirs, err := recent.Load(); err != nil || !slices.Equal(dirs, []string{dir}) {
		t.Errorf("recent = %q, %v; want [%s]", dirs, err, dir)
	}
	if w.Len() != 0 {
		t.Errorf("unexpected output %q", w.String())
	}
}

func TestLaunchRecentNotWritable(t *testing.T) {
	notDir := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(notDir, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_STATE_HOME", notDir)
	var w bytes.Buffer
	launchInZellij(t, "", "cat", &w)
	p, _ := recent.Path()
	if !strings.HasPrefix(w.String(), "docent: warning: ") || !strings.Contains(w.String(), p) {
		t.Errorf("got %q; want a warning that names %s", w.String(), p)
	}
}

func TestLaunchRecordsBeforeAttach(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("ZELLIJ", "")
	dir := t.TempDir()
	f := &zellij.Fake{Out: map[string]string{"--version": "zellij 0.45.1"}}
	attached := 0
	attach := func(string, ...string) error {
		attached++
		if dirs, err := recent.Load(); err != nil || !slices.Equal(dirs, []string{dir}) {
			t.Errorf("recent at attach = %q, %v; want [%s]", dirs, err, dir)
		}
		return nil
	}
	if err := launchDir(f, attach, io.Discard, dir, "", "cat", "", "", false); err != nil {
		t.Fatal(err)
	}
	if attached != 1 {
		t.Errorf("attach called %d times; want 1", attached)
	}
}

const testLayout = `layout {
    tab name={{.Name}} {
        pane name="harness" {{.Harness}}
        pane name="shell"
        pane name="editor" command="hx" // %s
    }
}
`

func TestLaunchLayout(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	conf := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", conf)
	t.Setenv("ZELLIJ", "0")
	writeTestFile(t, filepath.Join(conf, "docent", "config.toml"), `layout = "wide.kdl"`)
	wide := filepath.Join(conf, "docent", "wide.kdl")
	writeTestFile(t, wide, fmt.Sprintf(testLayout, "wide"))
	pair := filepath.Join(t.TempDir(), "pair.kdl")
	writeTestFile(t, pair, fmt.Sprintf(testLayout, "pair"))
	for _, tc := range []struct {
		layout  string
		builtin bool
		want    string
	}{
		{"", false, "// wide"},
		{pair, false, "// pair"},
		{"", true, "zellij:tab-bar"},
	} {
		f := &zellij.Fake{Out: map[string]string{"--version": "zellij 0.45.1"}}
		if err := launchDir(f, nil, io.Discard, t.TempDir(), "", "cat", "", tc.layout, tc.builtin); err != nil {
			t.Fatal(err)
		}
		if kdl := f.Calls[len(f.Calls)-1][3]; !strings.Contains(kdl, tc.want) {
			t.Errorf("layout %q, builtin %v: want %q in\n%s", tc.layout, tc.builtin, tc.want, kdl)
		}
	}
}

func TestLaunchLayoutMissing(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("ZELLIJ", "0")
	missing := filepath.Join(t.TempDir(), "no-such.kdl")
	f := &zellij.Fake{}
	err := launchDir(f, nil, io.Discard, t.TempDir(), "", "cat", "", missing, false)
	if err == nil || !strings.Contains(err.Error(), missing) || len(f.Calls) != 0 {
		t.Errorf("want error naming %s and no zellij calls, got %v, calls %q", missing, err, f.Calls)
	}
	if code := runLaunch([]string{"--layout", missing, t.TempDir()}); code != 1 {
		t.Errorf("runLaunch: exit %d; want 1", code)
	}
}

func TestLayoutFlag(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := t.TempDir()
	t.Chdir(dir)
	for p, want := range map[string]string{
		"":           "",
		"~/x.kdl":    filepath.Join(home, "x.kdl"),
		"./pair.kdl": filepath.Join(dir, "pair.kdl"),
		"/a/b.kdl":   "/a/b.kdl",
	} {
		if got, err := layoutFlag(p); err != nil || got != want {
			t.Errorf("layoutFlag(%q) = %q, %v; want %q", p, got, err, want)
		}
	}
}
