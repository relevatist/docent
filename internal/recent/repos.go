package recent

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const depth = 3

// Repos returns the directories that have a .git entry, down to 3 levels
// below each root. A root may be a symlink, but symlinks below it are not
// followed. It does not look inside a repo or a hidden directory, and ignores
// roots and subtrees that it cannot read.
func Repos(roots []string) []string {
	var found []string
	for _, root := range roots {
		root = filepath.Clean(root)
		real, err := filepath.EvalSymlinks(root)
		if err != nil {
			continue
		}
		_ = filepath.WalkDir(real, func(p string, d fs.DirEntry, err error) error {
			if err != nil || !d.IsDir() {
				return nil
			}
			rel, _ := filepath.Rel(real, p)
			if rel != "." && strings.HasPrefix(d.Name(), ".") {
				return fs.SkipDir
			}
			if _, err := os.Lstat(filepath.Join(p, ".git")); err == nil {
				found = append(found, filepath.Join(root, rel))
				return fs.SkipDir
			}
			if rel != "." && strings.Count(rel, string(filepath.Separator)) >= depth-1 {
				return fs.SkipDir
			}
			return nil
		})
	}
	return found
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
