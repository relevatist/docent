package main

import (
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/relevatist/zide/internal/zellij"
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
	if err := os.MkdirAll(filepath.Join(conf, "zide"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(conf, "zide", "config.toml"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ZELLIJ", "0")
	f := &zellij.Fake{Out: map[string]string{"--version": "zellij 0.45.1"}}
	if err := launchDir(f, w, t.TempDir(), "", harness); err != nil {
		t.Fatal(err)
	}
	last := f.Calls[len(f.Calls)-1]
	if !slices.Equal(last[:2], []string{"action", "new-tab"}) {
		t.Fatalf("last call %q; want new-tab", last)
	}
	return last[3]
}

func TestLaunchAlias(t *testing.T) {
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
