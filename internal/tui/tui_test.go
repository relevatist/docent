package tui

import (
	"testing"

	"charm.land/lipgloss/v2"
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
