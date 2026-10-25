package recent

import (
	"context"
	"io/fs"
	"iter"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Repos yields the directories that have a .git entry, at any depth below
// each root, in walk order. A root may be a symlink, but symlinks below it are
// not followed. It does not look inside a repo or a hidden directory, ignores
// roots and subtrees that it cannot read, and stops when ctx is done.
func Repos(ctx context.Context, roots []string) iter.Seq[string] {
	return func(yield func(string) bool) {
		stop := false
		for _, root := range roots {
			root = filepath.Clean(root)
			real, err := filepath.EvalSymlinks(root)
			if err != nil {
				continue
			}
			_ = filepath.WalkDir(real, func(p string, d fs.DirEntry, err error) error {
				if ctx.Err() != nil {
					return fs.SkipAll
				}
				if err != nil || !d.IsDir() {
					return nil
				}
				rel, _ := filepath.Rel(real, p)
				if rel != "." && strings.HasPrefix(d.Name(), ".") {
					return fs.SkipDir
				}
				if _, err := os.Lstat(filepath.Join(p, ".git")); err != nil {
					return nil
				}
				if !yield(filepath.Join(root, rel)) {
					stop = true
					return fs.SkipAll
				}
				return fs.SkipDir
			})
			if stop || ctx.Err() != nil {
				return
			}
		}
	}
}

// Candidates returns the recent directories that still exist, then the repos,
// each directory once.
func Candidates(recent, repos []string) []string {
	var dirs []string
	for _, d := range slices.Concat(recent, repos) {
		if slices.Contains(dirs, d) {
			continue
		}
		if fi, err := os.Stat(d); err == nil && fi.IsDir() {
			dirs = append(dirs, d)
		}
	}
	return dirs
}
