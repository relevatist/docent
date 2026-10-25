package recent

import (
	"context"
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
	got := slices.Collect(Repos(t.Context(), []string{src, filepath.Join(src, "missing"), wt}))
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

func TestReposDeep(t *testing.T) {
	src := t.TempDir()
	tree(t, src, "a/b/c/d/e/.git/")
	if got, want := slices.Collect(Repos(t.Context(), []string{src})), []string{filepath.Join(src, "a/b/c/d/e")}; !slices.Equal(got, want) {
		t.Errorf("Repos = %q; want %q", got, want)
	}
}

func TestReposCancel(t *testing.T) {
	src := t.TempDir()
	tree(t, src, "a/.git/", "b/.git/", "c/.git/")
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	var got []string
	for d := range Repos(ctx, []string{src, src}) {
		got = append(got, d)
		cancel()
	}
	if want := []string{filepath.Join(src, "a")}; !slices.Equal(got, want) {
		t.Errorf("Repos after cancel = %q; want %q", got, want)
	}
}

func TestReposBreak(t *testing.T) {
	src := t.TempDir()
	tree(t, src, "a/.git/", "b/.git/")
	for range Repos(t.Context(), []string{src, src}) {
		break
	}
}

func TestCandidates(t *testing.T) {
	home := t.TempDir()
	tree(t, home, "wt/retry/app/", "src/app/")
	retry, app := filepath.Join(home, "wt/retry/app"), filepath.Join(home, "src/app")
	gone := filepath.Join(home, "gone")
	got := Candidates([]string{retry, gone, app, retry})
	if want := []string{retry, app}; !slices.Equal(got, want) {
		t.Errorf("Candidates = %q; want %q", got, want)
	}
}

func TestReposSymlinkRoot(t *testing.T) {
	real, dir := t.TempDir(), t.TempDir()
	tree(t, real, "app/.git/", "lib/")
	link := filepath.Join(dir, "src")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(real, "app"), filepath.Join(real, "lib", "app")); err != nil {
		t.Fatal(err)
	}
	if got, want := slices.Collect(Repos(t.Context(), []string{link})), []string{filepath.Join(link, "app")}; !slices.Equal(got, want) {
		t.Errorf("Repos = %q; want %q", got, want)
	}
}

func TestDirs(t *testing.T) {
	home := t.TempDir()
	tree(t, home,
		"src/app/.git/",
		"src/app/internal/",
		"src/app/node_modules/x/",
		".cache/x/",
		"Library/Caches/",
		"notes/Library/",
		"notes/readme",
	)
	got := slices.Collect(Dirs(t.Context(), home))
	want := []string{home}
	for _, d := range []string{"notes", "notes/Library", "src", "src/app", "src/app/internal"} {
		want = append(want, filepath.Join(home, d))
	}
	if !slices.Equal(got, want) {
		t.Errorf("Dirs = %q; want %q", got, want)
	}
}

func TestDirsCancel(t *testing.T) {
	home := t.TempDir()
	tree(t, home, "a/", "b/")
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	var got []string
	for d := range Dirs(ctx, home) {
		got = append(got, d)
		cancel()
	}
	if want := []string{home}; !slices.Equal(got, want) {
		t.Errorf("Dirs after cancel = %q; want %q", got, want)
	}
}
