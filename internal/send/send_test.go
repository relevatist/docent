package send

import (
	"os"
	"slices"
	"testing"

	"github.com/relevatist/docent/internal/zellij"
)

func TestFormatRef(t *testing.T) {
	tests := []struct{ harness, ref, want string }{
		{"claude", "a.go:10-20", "@a.go:10-20 "},
		{"omp", "a.go:10-20", "@a.go:10-20 "},
		{"claude", "a.go:10-10", "@a.go:10 "},
		{"claude", "a.go:10", "@a.go:10 "},
		{"claude", "a.go", "@a.go "},
		{"aider", "a.go:10-20", "a.go:10-20 "},
		{"aider", "a.go:10-10", "a.go:10 "},
		{"/usr/local/bin/claude --model x", "src/a.go:1-2", "@src/a.go:1-2 "},
		{"aider --yes", "src/a.go:1-2", "src/a.go:1-2 "},
		{"claude", "dir:x/a.go:3-4", "@dir:x/a.go:3-4 "},
		{"claude", "dir:x/a.go", "@dir:x/a.go "},
	}
	for _, tt := range tests {
		got, err := FormatRef(tt.harness, tt.ref)
		if err != nil || got != tt.want {
			t.Errorf("FormatRef(%q, %q) = %q, %v; want %q", tt.harness, tt.ref, got, err, tt.want)
		}
	}
}

func TestFormatRefErrors(t *testing.T) {
	for _, ref := range []string{"", ":1", "a.go:20-10", "a.go:0", "a.go:1-x"} {
		if got, err := FormatRef("claude", ref); err == nil {
			t.Errorf("FormatRef(%q) = %q; want error", ref, got)
		}
	}
}

func fake(t *testing.T) *zellij.Fake {
	t.Helper()
	b, err := os.ReadFile("../panes/testdata/two_tabs.json")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("ZELLIJ_PANE_ID", "3")
	return &zellij.Fake{Out: map[string]string{"action list-panes --all --json": string(b)}}
}

func noEnter(t *testing.T, f *zellij.Fake) {
	t.Helper()
	for _, c := range f.Calls {
		if slices.Contains(c, "send-keys") || slices.Contains(c, "Enter") {
			t.Errorf("call %q may submit the harness input", c)
		}
	}
}

func TestPaste(t *testing.T) {
	for _, focus := range []bool{true, false} {
		f := fake(t)
		if err := Paste(f, "@a.go:1 ", focus); err != nil {
			t.Fatal(err)
		}
		want := [][]string{
			{"action", "list-panes", "--all", "--json"},
			{"action", "paste", "--pane-id", "terminal_4", "--", "@a.go:1 "},
		}
		if focus {
			want = append(want, []string{"action", "focus-pane-id", "terminal_4"})
		}
		if !slices.EqualFunc(f.Calls, want, slices.Equal) {
			t.Errorf("focus=%v calls = %q; want %q", focus, f.Calls, want)
		}
		noEnter(t, f)
	}
}

func TestPasteMultiline(t *testing.T) {
	f := fake(t)
	text := "line 1\nline 2\n"
	if err := Paste(f, text, false); err != nil {
		t.Fatal(err)
	}
	if got := f.Calls[1][len(f.Calls[1])-1]; got != text {
		t.Errorf("pasted %q; want %q", got, text)
	}
	noEnter(t, f)
}

func TestPasteEmptySendsNothing(t *testing.T) {
	f := fake(t)
	if err := Paste(f, "", true); err != nil {
		t.Fatal(err)
	}
	if len(f.Calls) != 0 {
		t.Errorf("calls = %q; want none", f.Calls)
	}
}

func TestPasteStripsEsc(t *testing.T) {
	const evil = "a\x1b[201~\rrm -rf x\r\x1b[200~b"
	const want = "a[201~\rrm -rf x\r[200~b"
	f := fake(t)
	if err := Paste(f, evil, false); err != nil {
		t.Fatal(err)
	}
	if got := f.Calls[1][len(f.Calls[1])-1]; got != want {
		t.Errorf("Paste pasted %q; want %q", got, want)
	}
	f = fake(t)
	if err := PasteRef(f, "a\x1b[201~.go:1", "aider", false); err != nil {
		t.Fatal(err)
	}
	if got := f.Calls[1][len(f.Calls[1])-1]; got != "a[201~.go:1 " {
		t.Errorf("PasteRef pasted %q", got)
	}
}
