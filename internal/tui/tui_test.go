package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/relevatist/docent/internal/tour"
)

func TestSafe(t *testing.T) {
	tests := []struct{ in, want string }{
		{"plain text", "plain text"},
		{"a\tb", "a\tb"},
		{"\x1b]0;pwn\x07title", "�]0;pwn�title"},
		{"a\rb\nc", "a�b�c"},
		{"del\x7f", "del�"},
		{"csi\u009b2J", "csi�2J"},
		{"日本", "日本"},
		{"abc\u202edef", "abc\ufffddef"},
		{"\u202a\u202d\u2066\u2069", "\ufffd\ufffd\ufffd\ufffd"},
		{"👨\u200d👩", "👨\u200d👩"},
	}
	for _, tt := range tests {
		if got := safe(tt.in); got != tt.want {
			t.Errorf("safe(%q) = %q; want %q", tt.in, got, tt.want)
		}
	}
}

func TestWrap(t *testing.T) {
	v := &view{width: 10}
	got := v.wrap(lipgloss.NewStyle(), "one two three four five")
	if len(got) < 3 {
		t.Fatalf("wrap = %q; want at least 3 lines", got)
	}
	for _, l := range got {
		if w := lipgloss.Width(l); w > 10 {
			t.Errorf("line %q is %d wide; want <= 10", l, w)
		}
	}
}

func plain(lines []string) string { return ansi.Strip(strings.Join(lines, "\n")) }

func TestRenderNoteFlowchart(t *testing.T) {
	got := plain(renderNote("## Flow\n\n```mermaid\ngraph LR\nA --> B\n```\n", 60, false))
	for _, want := range []string{"Flow", "│ A ├────►│ B │"} {
		if !strings.Contains(got, want) {
			t.Errorf("note does not show %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "graph LR") {
		t.Errorf("note shows the mermaid source:\n%s", got)
	}
}

func TestRenderNoteBadMermaid(t *testing.T) {
	got := plain(renderNote("```mermaid\nnot a diagram ((\n```", 60, false))
	for _, want := range []string{"not a diagram ((", "mermaid: failed to parse"} {
		if !strings.Contains(got, want) {
			t.Errorf("note does not show %q:\n%s", want, got)
		}
	}
}

func TestRenderNoteEscape(t *testing.T) {
	raw := strings.Join(renderNote("esc \x1b[2J here\x07", 60, false), "\n")
	if strings.Contains(raw, "\x1b[2J") || strings.Contains(raw, "\x07") {
		t.Fatalf("escape sequence reached the output: %q", raw)
	}
	if got := ansi.Strip(raw); !strings.Contains(got, "esc \uFFFD[2J here\uFFFD") {
		t.Errorf("no replacement character: %q", got)
	}
}

func TestDrawMermaidNoEscapes(t *testing.T) {
	d, err := drawMermaid("graph LR\nclassDef hot color:#ff0000\nA:::hot --> B")
	if err != nil {
		t.Fatal(err)
	}
	if strings.ContainsRune(d, '\x1b') || !strings.Contains(d, "A") {
		t.Errorf("diagram %q", d)
	}
}

// tourView is a view of a tour in a pane w wide and h high.
func tourView(t *testing.T, tr tour.Tour, w, h int) *view {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.go"), []byte("SECRET LINE\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := &tour.Model{Items: tour.Build(tr, dir), Open: func(string, int) error { return nil }}
	m.Start()
	v := &view{m: m, title: tr.Title}
	v.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return v
}

func content(v *view) string { return ansi.Strip(v.View().Content) }

func TestStepViewShowsNoteNotFile(t *testing.T) {
	v := tourView(t, tour.Tour{Title: "Retry", Steps: []tour.Step{
		{File: "a.go", Lines: []int{1, 1}, Note: "New type"},
		{Note: "## Overview"},
	}}, 60, 20)
	got := content(v)
	if !strings.HasPrefix(got, "1/2  a.go:1  Retry") || !strings.Contains(got, "New type") || strings.Contains(got, "SECRET") {
		t.Errorf("file step:\n%s", got)
	}
	v.Update(key("n"))
	if got := content(v); !strings.HasPrefix(got, "2/2  Retry\n") || !strings.Contains(got, "Overview") {
		t.Errorf("note-only step:\n%s", got)
	}
}

func key(k string) tea.KeyPressMsg {
	switch k {
	case "pgdown":
		return tea.KeyPressMsg{Code: tea.KeyPgDown}
	case "pgup":
		return tea.KeyPressMsg{Code: tea.KeyPgUp}
	case "space":
		return tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
	case "ctrl+d", "ctrl+u":
		return tea.KeyPressMsg{Code: rune(k[5]), Mod: tea.ModCtrl}
	}
	return tea.KeyPressMsg{Code: rune(k[0]), Text: k}
}

func TestScroll(t *testing.T) {
	var note []string
	for i := range 60 {
		note = append(note, fmt.Sprintf("- item %d", i+1))
	}
	// 23 rows: the heading, 20 note lines, a blank line, and the help line.
	v := tourView(t, tour.Tour{Steps: []tour.Step{{Note: strings.Join(note, "\n")}, {Note: "next"}}}, 60, 23)
	lines := v.lines()
	if v.room() != 20 || len(lines) < 50 {
		t.Fatalf("room %d, %d lines", v.room(), len(lines))
	}
	shows := func(name string, from int) {
		t.Helper()
		got := strings.Split(content(v), "\n")
		want := plain(lines[from : from+20])
		if v.scroll != from || len(got) != 23 || strings.Join(got[1:21], "\n") != want || got[21] != "" {
			t.Errorf("%s: scroll %d, want %d:\n%s", name, v.scroll, from, strings.Join(got, "\n"))
		}
	}
	v.Update(key("pgup"))
	shows("pgup at top", 0)
	v.Update(key("pgdown"))
	shows("pgdown", 20)
	v.Update(key("j"))
	shows("j", 21)
	v.Update(key("b"))
	shows("b", 1)
	v.Update(key("ctrl+d"))
	shows("ctrl+d", 21)
	v.Update(key("ctrl+u"))
	shows("ctrl+u", 1)
	v.Update(key("G"))
	shows("G", len(lines)-20)
	v.Update(key("space"))
	shows("space at bottom", len(lines)-20)
	if got := content(v); !strings.Contains(got, "item 60") {
		t.Errorf("G does not show the last line:\n%s", got)
	}
	v.Update(key("g"))
	shows("g", 0)
	v.Update(key("G"))
	v.Update(key("n"))
	if v.scroll != 0 || !strings.Contains(content(v), "next") {
		t.Errorf("n: scroll %d", v.scroll)
	}
}

// onlySGR fails when out has a BEL, or an ESC that does not start an SGR
// sequence.
func onlySGR(t *testing.T, name, out string) {
	t.Helper()
	if strings.ContainsRune(out, '\a') {
		t.Errorf("%s: BEL in output: %q", name, out)
	}
	if rest := regexp.MustCompile(`\x1b\[[0-9;:]*m`).ReplaceAllString(out, ""); strings.ContainsRune(rest, '\x1b') {
		t.Errorf("%s: non-SGR escape in output: %q", name, out)
	}
}

func TestRenderNoteCharacterReferences(t *testing.T) {
	for _, in := range []string{
		"x &#27;]0;pwned&#7; y",
		"x &#x1b;[2J y",
		"x &#27;Pq#0;2;0;0;0&#27;&#92; y",
		"x <span>&#27;[2J</span> y",
		"x &#155;2J y",
	} {
		out := strings.Join(renderNote(in, 60, false), "\n")
		onlySGR(t, in, out)
		if !strings.Contains(ansi.Strip(out), "x") {
			t.Errorf("%s: text lost: %q", in, out)
		}
	}
}

func TestRenderNoteLinks(t *testing.T) {
	out := strings.Join(renderNote("[docs](https://example.com/a) <https://example.com/b> ![img](https://example.com/c.png)", 80, false), "\n")
	onlySGR(t, "links", out)
	if strings.Contains(out, "\x1b]8") {
		t.Errorf("OSC 8 in output: %q", out)
	}
	got := ansi.Strip(out)
	for _, want := range []string{"docs", "https://example.com/a", "https://example.com/b", "img"} {
		if !strings.Contains(got, want) {
			t.Errorf("output does not show %q: %q", want, got)
		}
	}
}

func TestRenderNoteDiffColors(t *testing.T) {
	out := strings.Join(renderNote("```diff\n context\n-old line\n+new line\n```", 60, false), "\n")
	sgr := func(text string) string {
		m := regexp.MustCompile(`\x1b\[([0-9;]*)m` + regexp.QuoteMeta(text)).FindStringSubmatch(out)
		if m == nil {
			t.Fatalf("no SGR before %q: %q", text, out)
		}
		return m[1]
	}
	if del, add := sgr("-old line"), sgr("+new line"); del == add {
		t.Errorf("removed and added lines have the same SGR %q", del)
	}
}

func TestRenderNoteSequenceAndER(t *testing.T) {
	for src, want := range map[string]string{
		"sequenceDiagram\n    participant Client\n    participant Server\n    Client->>Server: request": "│ Client │",
		"erDiagram\n    USER ||--o{ SESSION : owns":                                                     "│ USER │",
	} {
		got := plain(renderNote("```mermaid\n"+src+"\n```", 60, false))
		if !strings.Contains(got, want) || strings.Contains(got, "Diagram") || strings.Contains(got, "mermaid:") {
			t.Errorf("%s:\n%s", src, got)
		}
	}
}

func TestScrollClampsOnResize(t *testing.T) {
	var paras []string
	for i := range 8 {
		paras = append(paras, fmt.Sprintf("Paragraph %d has enough words in it to wrap a few times in a narrow pane.", i+1))
	}
	v := tourView(t, tour.Tour{Steps: []tour.Step{{Note: strings.Join(paras, "\n\n") + "\n\nEND"}}}, 30, 12)
	v.Update(key("G"))
	v.Update(tea.WindowSizeMsg{Width: 200, Height: 12})
	if got := content(v); !strings.Contains(got, "END") {
		t.Errorf("scroll %d of %d lines, view:\n%s", v.scroll, len(v.lines()), got)
	}
}

func TestRenderNoteTabs(t *testing.T) {
	note := "```go\nfunc f() {\n\tif ok {\n\t\treturn lastValue\n\t}\n}\n```\n\nA\tparagraph\twith\ttabs\tthat\twraps\tin\tthe\tpane\tLASTWORD"
	lines := renderNote(note, 40, false)
	for _, l := range lines {
		if w := ansi.StringWidth(l); w > 40 {
			t.Errorf("line is %d wide, want <= 40: %q", w, ansi.Strip(l))
		}
	}
	got := plain(lines)
	for _, want := range []string{"        return lastValue", "LASTWORD"} {
		if !strings.Contains(got, want) {
			t.Errorf("output does not show %q:\n%q", want, got)
		}
	}
}
