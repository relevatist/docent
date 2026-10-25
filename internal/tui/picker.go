package tui

import (
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// filter returns the dirs whose display path (see tilde) holds the runes of q
// in order, ignoring case. When none match and q is a path to a directory, it
// returns that directory alone.
func filter(dirs []string, q, home string) []string {
	var out []string
	for _, d := range dirs {
		if subsequence(strings.ToLower(tilde(d, home)), strings.ToLower(q)) {
			out = append(out, d)
		}
	}
	if len(out) > 0 || q == "" {
		return out
	}
	p := q
	if p == "~" || strings.HasPrefix(p, "~/") {
		p = filepath.Join(home, p[1:])
	}
	if p, err := filepath.Abs(p); err == nil {
		if fi, err := os.Stat(p); err == nil && fi.IsDir() {
			return []string{p}
		}
	}
	return nil
}

func subsequence(s, q string) bool {
	for _, r := range q {
		i := strings.IndexRune(s, r)
		if i < 0 {
			return false
		}
		s = s[i+utf8.RuneLen(r):]
	}
	return true
}

// tilde writes a path under home as ~/...
func tilde(p, home string) string {
	if p == home {
		return "~"
	}
	if rest, ok := strings.CutPrefix(p, home+string(filepath.Separator)); ok && home != "" {
		return "~/" + rest
	}
	return p
}
