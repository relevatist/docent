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
// below each root. It does not look inside a repo or a hidden directory, and
// ignores roots and subtrees that it cannot read.
func Repos(roots []string) []string {
	var found []string
	for _, root := range roots {
		root = filepath.Clean(root)
		_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil || !d.IsDir() {
				return nil
			}
			if p != root && strings.HasPrefix(d.Name(), ".") {
				return fs.SkipDir
			}
			if _, err := os.Lstat(filepath.Join(p, ".git")); err == nil {
				found = append(found, p)
				return fs.SkipDir
			}
			if rel, _ := filepath.Rel(root, p); rel != "." && strings.Count(rel, string(filepath.Separator)) >= depth-1 {
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
