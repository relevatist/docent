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

// calls returns the calls of f without the plugin pipes. The Fake plugin
// never answers, so each paste also falls back to action paste.
func calls(f *zellij.Fake) [][]string {
	return slices.DeleteFunc(slices.Clone(f.Calls), func(c []string) bool { return c[0] == "pipe" })
}

func write(text string) []string {
	return []string{"action", "paste", "--pane-id", "terminal_4", "--", text}
}

// pasted returns the text that call pastes.
func pasted(t *testing.T, call []string) string {
	t.Helper()
	if len(call) != 6 || !slices.Equal(call[:5], []string{"action", "paste", "--pane-id", "terminal_4", "--"}) {
		t.Fatalf("call %q is not a paste to terminal_4", call)
	}
	return call[5]
}

func TestPaste(t *testing.T) {
	for _, focus := range []bool{true, false} {
		f := fake(t)
		if err := Paste(f, "@a.go:1 ", focus); err != nil {
			t.Fatal(err)
		}
		want := [][]string{
			{"action", "list-panes", "--all", "--json"},
			write("@a.go:1 "),
		}
		if focus {
			want = append(want, []string{"action", "focus-pane-id", "terminal_4"})
		}
		if !slices.EqualFunc(calls(f), want, slices.Equal) {
			t.Errorf("focus=%v calls = %q; want %q", focus, f.Calls, want)
		}
	}
}

func TestPasteMultiline(t *testing.T) {
	f := fake(t)
	text := "line 1\nline 2\n"
	if err := Paste(f, text, false); err != nil {
		t.Fatal(err)
	}
	if got := pasted(t, calls(f)[1]); got != text {
		t.Errorf("pasted %q; want %q", got, text)
	}
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
	if got := pasted(t, calls(f)[1]); got != want {
		t.Errorf("Paste pasted %q; want %q", got, want)
	}
	f = fake(t)
	if err := PasteRef(f, "a\x1b[201~.go:1", "aider", false); err != nil {
		t.Fatal(err)
	}
	if got := pasted(t, calls(f)[1]); got != "a[201~.go:1 " {
		t.Errorf("PasteRef pasted %q", got)
	}
}

func TestPasteEndMarkerAndCR(t *testing.T) {
	f := fake(t)
	if err := Paste(f, "a\x1b[201~\rb", false); err != nil {
		t.Fatal(err)
	}
	if got := calls(f)[1]; !slices.Equal(got, write("a[201~\rb")) {
		t.Errorf("call %q; want %q", got, write("a[201~\rb"))
	}
}

func TestPasteRefLayoutMarker(t *testing.T) {
	t.Setenv("ZELLIJ_PANE_ID", "3")
	f := &zellij.Fake{Out: map[string]string{"action list-panes --all --json": `[
		{"id":3,"title":"shell","tab_id":0},
		{"id":4,"title":"harness","tab_id":0,"terminal_command":"/usr/bin/env DOCENT_HARNESS_KIND=claude DOCENT_LAYOUT=/x.kdl claude"}]`}}
	if err := PasteRef(f, "a.go:1-2", "aider", false); err != nil {
		t.Fatal(err)
	}
	if got := pasted(t, calls(f)[1]); got != "@a.go:1-2 " {
		t.Errorf("PasteRef pasted %q; want %q", got, "@a.go:1-2 ")
	}
}
