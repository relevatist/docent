package tour

import (
	"bytes"
	"cmp"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// Hunk is one hunk of a unified diff. Lines keep their ' ', '+', '-' prefix.
type Hunk struct {
	File               string
	Deleted            bool
	OldStart, OldCount int
	NewStart, NewCount int
	Lines              []string
}

// Span is the working-tree line range the hunk touches. A pure deletion spans
// the lines on both sides of it.
func (h Hunk) Span() (start, end int) {
	if h.NewCount == 0 {
		return max(h.NewStart, 1), h.NewStart + 1
	}
	return h.NewStart, h.NewStart + h.NewCount - 1
}

func (h Hunk) ref(cwd string) string {
	f := relPath(cwd, h.File)
	if h.Deleted {
		return f + " (deleted)"
	}
	return f + ":" + lineRange(h.Span())
}

// ParseDiff parses `git diff` output made with a/ and b/ prefixes.
func ParseDiff(diff string) ([]Hunk, error) {
	var hunks []Hunk
	var file string
	var deleted, inHunk bool
	for line := range strings.Lines(diff) {
		line = strings.TrimSuffix(line, "\n")
		switch {
		case strings.HasPrefix(line, "diff "):
			file, deleted, inHunk = "", false, false
		case !inHunk && strings.HasPrefix(line, "--- "):
			if p, err := diffPath(line[4:], "a/"); err != nil {
				return nil, err
			} else if p != "" {
				file = p
			}
		case !inHunk && strings.HasPrefix(line, "+++ "):
			p, err := diffPath(line[4:], "b/")
			if err != nil {
				return nil, err
			}
			if p == "" {
				deleted = true
			} else {
				file = p
			}
		case strings.HasPrefix(line, "@@ "):
			h, err := hunkHeader(line)
			if err != nil {
				return nil, err
			}
			h.File, h.Deleted = file, deleted
			hunks = append(hunks, h)
			inHunk = true
		case inHunk && line != "" && strings.ContainsRune("+- ", rune(line[0])):
			h := &hunks[len(hunks)-1]
			h.Lines = append(h.Lines, line)
		}
	}
	return hunks, nil
}

// diffPath returns the path in a ---/+++ line, or "" for /dev/null.
func diffPath(s, prefix string) (string, error) {
	s = strings.TrimSuffix(s, "\t") // git adds a tab after names with spaces
	if s == "/dev/null" {
		return "", nil
	}
	if strings.HasPrefix(s, `"`) {
		u, err := strconv.Unquote(s)
		if err != nil {
			return "", fmt.Errorf("bad diff path %s", s)
		}
		s = u
	}
	return strings.TrimPrefix(s, prefix), nil
}

// hunkHeader parses "@@ -a[,b] +c[,d] @@".
func hunkHeader(line string) (Hunk, error) {
	var h Hunk
	f := strings.Fields(line)
	if len(f) < 4 || f[3] != "@@" || !strings.HasPrefix(f[1], "-") || !strings.HasPrefix(f[2], "+") {
		return h, fmt.Errorf("bad hunk header %q", line)
	}
	var err1, err2 error
	h.OldStart, h.OldCount, err1 = hunkRange(f[1][1:])
	h.NewStart, h.NewCount, err2 = hunkRange(f[2][1:])
	if err1 != nil || err2 != nil {
		return h, fmt.Errorf("bad hunk header %q", line)
	}
	return h, nil
}

func hunkRange(s string) (start, count int, err error) {
	a, b, ok := strings.Cut(s, ",")
	if start, err = strconv.Atoi(a); err != nil || !ok {
		return start, 1, err
	}
	count, err = strconv.Atoi(b)
	return start, count, err
}

func git(dir string, args ...string) ([]byte, error) {
	var stderr bytes.Buffer
	cmd := exec.Command("git", append([]string{"-C", dir, "-c", "core.quotePath=false"}, args...)...)
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}

// repoRoot returns the top of the repository at dir. Unlike --show-toplevel,
// it keeps symlinks in dir, so paths match those made from the same cwd.
func repoRoot(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	up, err := git(abs, "rev-parse", "--show-cdup")
	if err != nil {
		return "", err
	}
	return filepath.Join(abs, strings.TrimSpace(string(up))), nil
}

// GitDiff runs `git diff args` in dir and returns its hunks with absolute paths.
func GitDiff(dir string, args ...string) ([]Hunk, error) {
	top, err := repoRoot(dir)
	if err != nil {
		return nil, err
	}
	out, err := git(dir, append([]string{"diff", "--no-color", "--no-ext-diff", "--no-textconv", "--no-relative",
		"-U0", "--src-prefix=a/", "--dst-prefix=b/"}, args...)...)
	if err != nil {
		return nil, err
	}
	hunks, err := ParseDiff(string(out))
	for i := range hunks {
		hunks[i].File = filepath.Join(top, hunks[i].File)
	}
	return hunks, err
}

// Changes returns the hunks between base (default HEAD) and the working tree,
// staged and unstaged, plus one all-added hunk per untracked text file.
func Changes(dir, base string) ([]Hunk, error) {
	hunks, err := GitDiff(dir, "--end-of-options", cmp.Or(base, "HEAD"), "--")
	if err != nil {
		return nil, err
	}
	top, err := repoRoot(dir)
	if err != nil {
		return nil, err
	}
	out, err := git(dir, "ls-files", "--others", "--exclude-standard", "--full-name", "-z")
	if err != nil {
		return nil, err
	}
	for name := range strings.SplitSeq(strings.TrimSuffix(string(out), "\x00"), "\x00") {
		if name == "" {
			continue
		}
		path := filepath.Join(top, name)
		data, err := os.ReadFile(path)
		if err != nil || bytes.IndexByte(data, 0) >= 0 {
			continue // unreadable or binary
		}
		h := Hunk{File: path, NewStart: 1}
		for l := range strings.Lines(string(data)) {
			h.Lines = append(h.Lines, "+"+strings.TrimRight(l, "\r\n"))
		}
		if h.NewCount = len(h.Lines); h.NewCount > 0 {
			hunks = append(hunks, h)
		}
	}
	return hunks, nil
}
