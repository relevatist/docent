package recent

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func setup(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", dir)
	return filepath.Join(dir, "docent", "recent.toml")
}

func TestPath(t *testing.T) {
	p := setup(t)
	if got, err := Path(); err != nil || got != p {
		t.Errorf("Path = %q, %v; want %q", got, err, p)
	}
	home := t.TempDir()
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("HOME", home)
	if got, err := Path(); err != nil || got != filepath.Join(home, ".local", "state", "docent", "recent.toml") {
		t.Errorf("Path = %q, %v", got, err)
	}
}

func TestLoadMissing(t *testing.T) {
	setup(t)
	if dirs, err := Load(); err != nil || len(dirs) != 0 {
		t.Errorf("Load = %q, %v; want empty", dirs, err)
	}
}

func TestAdd(t *testing.T) {
	setup(t)
	for _, d := range []string{"/b", "/a", "/b"} {
		if err := Add(d); err != nil {
			t.Fatal(err)
		}
	}
	if dirs, err := Load(); err != nil || !slices.Equal(dirs, []string{"/b", "/a"}) {
		t.Errorf("Load = %q, %v; want [/b /a]", dirs, err)
	}
}

func TestLimit(t *testing.T) {
	setup(t)
	var want []string
	for i := range 21 {
		d := fmt.Sprintf("/d%d", i)
		if err := Add(d); err != nil {
			t.Fatal(err)
		}
		want = append([]string{d}, want...)
	}
	if dirs, err := Load(); err != nil || !slices.Equal(dirs, want[:20]) {
		t.Errorf("Load = %q, %v; want %q", dirs, err, want[:20])
	}
}

func TestLoadKeepsMissingDirs(t *testing.T) {
	p := setup(t)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(`dirs = ["/no/such/dir", "/x"]`), 0o644); err != nil {
		t.Fatal(err)
	}
	if dirs, err := Load(); err != nil || !slices.Equal(dirs, []string{"/no/such/dir", "/x"}) {
		t.Errorf("Load = %q, %v", dirs, err)
	}
}

func TestAddRenames(t *testing.T) {
	p := setup(t)
	if err := Add("/a"); err != nil {
		t.Fatal(err)
	}
	old := filepath.Join(t.TempDir(), "old.toml")
	if err := os.Link(p, old); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(old)
	if err := Add("/b"); err != nil {
		t.Fatal(err)
	}
	if after, _ := os.ReadFile(old); string(after) != string(before) {
		t.Errorf("Add wrote the file in place; want a new file renamed over it")
	}
	entries, _ := os.ReadDir(filepath.Dir(p))
	if len(entries) != 1 {
		t.Errorf("state dir holds %d entries; want only recent.toml", len(entries))
	}
}
