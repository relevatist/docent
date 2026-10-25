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
		for _, root := range roots {
			ok := walk(ctx, root, func(p, rel string, d fs.DirEntry) (bool, error) {
				if rel != "." && strings.HasPrefix(d.Name(), ".") {
					return false, fs.SkipDir
				}
				if _, err := os.Lstat(filepath.Join(p, ".git")); err != nil {
					return false, nil
				}
				return true, fs.SkipDir
			}, yield)
			if !ok {
				return
			}
		}
	}
}

// Dirs yields root and each directory below it, in walk order, like Repos
// but for every directory. It skips hidden directories, node_modules, and
// Library just below root (the macOS ~/Library).
func Dirs(ctx context.Context, root string) iter.Seq[string] {
	return func(yield func(string) bool) {
		walk(ctx, root, func(p, rel string, d fs.DirEntry) (bool, error) {
			if rel != "." && (strings.HasPrefix(d.Name(), ".") || d.Name() == "node_modules" || rel == "Library") {
				return false, fs.SkipDir
			}
			return true, nil
		}, yield)
	}
}

// walk walks the directories below root. visit says if a directory is a hit,
// and may return fs.SkipDir. walk yields each hit joined onto root as given,
// and returns false when yield or ctx stopped it.
func walk(ctx context.Context, root string, visit func(p, rel string, d fs.DirEntry) (bool, error), yield func(string) bool) bool {
	root = filepath.Clean(root)
	real, err := filepath.EvalSymlinks(root)
	if err != nil {
		return ctx.Err() == nil
	}
	stop := false
	_ = filepath.WalkDir(real, func(p string, d fs.DirEntry, err error) error {
		if ctx.Err() != nil {
			return fs.SkipAll
		}
		if err != nil || !d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(real, p)
		hit, err := visit(p, rel, d)
		if hit && !yield(filepath.Join(root, rel)) {
			stop = true
			return fs.SkipAll
		}
		return err
	})
	return !stop && ctx.Err() == nil
}

// Candidates returns the recent directories that still exist, each once.
func Candidates(recent []string) []string {
	var dirs []string
	for _, d := range recent {
		if slices.Contains(dirs, d) {
			continue
		}
		if fi, err := os.Stat(d); err == nil && fi.IsDir() {
			dirs = append(dirs, d)
		}
	}
	return dirs
}
