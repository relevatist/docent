package helix

import (
	"slices"
	"strings"
	"testing"
	"unicode"

	"github.com/relevatist/docent/internal/zellij"
)

// check checks that f got one plugin write of want, then the fallback,
// because the Fake plugin never answers.
func check(t *testing.T, f *zellij.Fake, pane, want string) {
	t.Helper()
	if len(f.Calls) != 2 || !slices.Equal(f.Calls[0][:3], []string{"pipe", "--name", "docent-write"}) || f.Calls[0][len(f.Calls[0])-1] != want {
		t.Fatalf("calls = %q; want a docent-write pipe of %q", f.Calls, want)
	}
	if w := zellij.WriteArgs(pane, []byte(want)); !slices.Equal(f.Calls[1], w) {
		t.Fatalf("fallback = %q; want %q", f.Calls[1], w)
	}
}

func TestOpen(t *testing.T) {
	tests := []struct {
		name, path string
		line       int
		want       string
	}{
		{"with line", "/abs/a.go", 42, "\x1b[27u\x1b[27u:open '/abs/a.go'\r:goto 42\r"},
		{"no line", "/abs/a.go", 0, "\x1b[27u\x1b[27u:open '/abs/a.go'\r"},
		{"spaces", "/abs/my dir/a b.go", 0, "\x1b[27u\x1b[27u:open '/abs/my dir/a b.go'\r"},
		{"quotes and percent", `/abs/it's "%{cwd}".go`, 0, "\x1b[27u\x1b[27u:open '/abs/it''s \"%{cwd}\".go'\r"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &zellij.Fake{}
			if err := Open(f, "terminal_3", tt.path, tt.line); err != nil {
				t.Fatal(err)
			}
			check(t, f, "terminal_3", tt.want)
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
	check(t, f, "terminal_0", "\x1b[27u\x1b[27u:open '"+dir+"/sub/a.go'\r")
}

func TestReload(t *testing.T) {
	f := &zellij.Fake{}
	if err := Reload(f, "terminal_3"); err != nil {
		t.Fatal(err)
	}
	check(t, f, "terminal_3", "\x1b[27u\x1b[27u:reload-all\r")
}
