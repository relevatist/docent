package tui

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestFilter(t *testing.T) {
	home := t.TempDir()
	x := filepath.Join(home, "x")
	if err := os.Mkdir(x, 0o755); err != nil {
		t.Fatal(err)
	}
	retry, lib := filepath.Join(home, "wt/retry/app"), filepath.Join(home, "src/lib")
	dirs := []string{retry, lib, "/opt/RTAP"}
	tests := []struct {
		q    string
		want []string
	}{
		{"", dirs},
		{"rtap", []string{retry, "/opt/RTAP"}},
		{"RtaP", []string{retry, "/opt/RTAP"}},
		{"~/s", []string{lib}},
		{"~", []string{retry, lib}},
		{"pat", nil},
		{"~/x", []string{x}},
		{"~/x/", []string{x}},
		{home, []string{home}},
		{"~/missing", nil},
	}
	for _, tt := range tests {
		if got := filter(dirs, tt.q, home); !slices.Equal(got, tt.want) {
			t.Errorf("filter(%q) = %q; want %q", tt.q, got, tt.want)
		}
	}
}
