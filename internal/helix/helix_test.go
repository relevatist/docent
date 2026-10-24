package helix

import (
	"slices"
	"strings"
	"testing"
	"unicode"

	"github.com/relevatist/zide/internal/zellij"
)

func esc(p string) []string   { return []string{"action", "send-keys", "--pane-id", p, "Esc", "Esc"} }
func enter(p string) []string { return []string{"action", "send-keys", "--pane-id", p, "Enter"} }
func chars(p, s string) []string {
	return []string{"action", "write-chars", "--pane-id", p, "--", s}
}

func check(t *testing.T, got, want [][]string) {
	t.Helper()
	if !slices.EqualFunc(got, want, slices.Equal) {
		t.Fatalf("calls:\n got %q\nwant %q", got, want)
	}
}

func TestOpen(t *testing.T) {
	tests := []struct {
		name, path string
		line       int
		want       [][]string
	}{
		{"with line", "/abs/a.go", 42, [][]string{
			esc("terminal_3"), chars("terminal_3", ":open '/abs/a.go'"), enter("terminal_3"),
			chars("terminal_3", ":goto 42"), enter("terminal_3"),
		}},
		{"no line", "/abs/a.go", 0, [][]string{
			esc("terminal_3"), chars("terminal_3", ":open '/abs/a.go'"), enter("terminal_3"),
		}},
		{"spaces", "/abs/my dir/a b.go", 0, [][]string{
			esc("terminal_3"), chars("terminal_3", ":open '/abs/my dir/a b.go'"), enter("terminal_3"),
		}},
		{"quotes and percent", `/abs/it's "%{cwd}".go`, 0, [][]string{
			esc("terminal_3"), chars("terminal_3", `:open '/abs/it''s "%{cwd}".go'`), enter("terminal_3"),
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &zellij.Fake{}
			if err := Open(f, "terminal_3", tt.path, tt.line); err != nil {
				t.Fatal(err)
			}
			check(t, f.Calls, tt.want)
		})
	}
}

func TestOpenControlChars(t *testing.T) {
	for _, path := range []string{"/abs/a\r:sh touch x\r.go", "/abs/\x1b.go", "/abs/a\n.go", "/abs/a\t.go", "/abs/a\x7f.go", "/abs/a\x00.go"} {
		f := &zellij.Fake{}
		err := Open(f, "terminal_3", path, 0)
		if err == nil || !strings.Contains(err.Error(), "control characters") || strings.ContainsFunc(err.Error(), unicode.IsControl) {
			t.Errorf("Open(%q) err = %q; want quoted control characters error", path, err)
		}
		if len(f.Calls) != 0 {
			t.Errorf("Open(%q) calls = %q; want none", path, f.Calls)
		}
	}
}

func TestOpenRelativeResolvesAgainstCwd(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	f := &zellij.Fake{}
	if err := Open(f, "terminal_0", "sub/a.go", 0); err != nil {
		t.Fatal(err)
	}
	check(t, f.Calls, [][]string{
		esc("terminal_0"), chars("terminal_0", ":open '"+dir+"/sub/a.go'"), enter("terminal_0"),
	})
}

func TestReload(t *testing.T) {
	f := &zellij.Fake{}
	if err := Reload(f, "terminal_3"); err != nil {
		t.Fatal(err)
	}
	check(t, f.Calls, [][]string{esc("terminal_3"), chars("terminal_3", ":reload-all"), enter("terminal_3")})
}
