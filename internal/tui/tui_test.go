package tui

import "testing"

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
