package recent

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// tree makes each path under root: a name ending in / is a directory, others are files.
func tree(t *testing.T, root string, paths ...string) {
	t.Helper()
	for _, p := range paths {
		dir, isDir := strings.CutSuffix(p, "/")
		if !isDir {
			dir = filepath.Dir(p)
		}
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
		if !isDir {
			if err := os.WriteFile(filepath.Join(root, p), nil, 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestRepos(t *testing.T) {
	src, wt := t.TempDir(), t.TempDir()
	tree(t, src,
		"app/.git/",
		"app/vendor/x/.git/",
		"lib/.git/",
		"a/b/c/d/.git/",
		"a/b/c/.git/",
		".cache/x/.git/",
		"notes/readme",
	)
	tree(t, wt, "retry/app/.git")
	got := Repos([]string{src, filepath.Join(src, "missing"), wt})
	want := []string{
		filepath.Join(src, "a/b/c"),
		filepath.Join(src, "app"),
		filepath.Join(src, "lib"),
		filepath.Join(wt, "retry/app"),
	}
	if !slices.Equal(got, want) {
		t.Errorf("Repos = %q; want %q", got, want)
	}
}

func TestReposTooDeep(t *testing.T) {
	src := t.TempDir()
	tree(t, src, "a/b/c/d/.git/")
	if got := Repos([]string{src}); len(got) != 0 {
		t.Errorf("Repos = %q; want none", got)
	}
}

func TestCandidates(t *testing.T) {
	home := t.TempDir()
	tree(t, home, "wt/retry/app/", "src/app/", "src/lib/")
	retry, app, lib := filepath.Join(home, "wt/retry/app"), filepath.Join(home, "src/app"), filepath.Join(home, "src/lib")
	gone := filepath.Join(home, "gone")
	got := Candidates([]string{retry, gone, app, retry}, []string{app, lib})
	if want := []string{retry, app, lib}; !slices.Equal(got, want) {
		t.Errorf("Candidates = %q; want %q", got, want)
	}
}
