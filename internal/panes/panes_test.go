package panes

import (
	"encoding/json/v2"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/relevatist/docent/internal/zellij"
)

func fake(t *testing.T) *zellij.Fake {
	t.Helper()
	b, err := os.ReadFile("testdata/two_tabs.json")
	if err != nil {
		t.Fatal(err)
	}
	return &zellij.Fake{Out: map[string]string{"action list-panes --all --json": string(b)}}
}

func TestFindInCallerTab(t *testing.T) {
	tests := []struct{ self, name, want string }{
		{"4", "editor", "terminal_3"},
		{"4", "shell", "terminal_5"},
		{"0", "harness", "terminal_1"},
		{"2", "editor", "terminal_0"},
	}
	for _, tt := range tests {
		t.Setenv("ZELLIJ_PANE_ID", tt.self)
		got, err := Find(fake(t), tt.name)
		if err != nil || got != tt.want {
			t.Errorf("Find(self=%s, %s) = %q, %v; want %q", tt.self, tt.name, got, err, tt.want)
		}
	}
}

func TestFindMissingPane(t *testing.T) {
	t.Setenv("ZELLIJ_PANE_ID", "4")
	_, err := Find(fake(t), "tour")
	if err == nil || !strings.Contains(err.Error(), `"tour"`) {
		t.Fatalf("err = %v; want it to name the pane", err)
	}
}

func TestFindOutsideZellij(t *testing.T) {
	f := &zellij.Fake{}
	t.Setenv("ZELLIJ_PANE_ID", "")
	_, err := Find(f, "editor")
	if err == nil || !strings.Contains(err.Error(), "must run inside a docent workstream") {
		t.Fatalf("err = %v", err)
	}
	if len(f.Calls) != 0 {
		t.Fatalf("zellij called: %q", f.Calls)
	}
}

func TestFindSelfNotListed(t *testing.T) {
	t.Setenv("ZELLIJ_PANE_ID", "99")
	if _, err := Find(fake(t), "editor"); err == nil {
		t.Fatal("want error")
	}
}

func TestLookupCommand(t *testing.T) {
	b, err := os.ReadFile("testdata/two_tabs.json")
	if err != nil {
		t.Fatal(err)
	}
	var all []map[string]any
	if err := json.Unmarshal(b, &all); err != nil {
		t.Fatal(err)
	}
	for _, p := range all {
		if p["title"] == "harness" && p["tab_id"] == 0.0 {
			p["terminal_command"] = "/opt/bin/omp --model x"
		}
	}
	if b, err = json.Marshal(all); err != nil {
		t.Fatal(err)
	}
	f := &zellij.Fake{Out: map[string]string{"action list-panes --all --json": string(b)}}
	t.Setenv("ZELLIJ_PANE_ID", "2")
	got, err := Lookup(f, "harness")
	if err != nil || got != (Pane{ID: "terminal_1", Command: "/opt/bin/omp --model x", Tab: 0}) {
		t.Errorf("Lookup = %+v, %v", got, err)
	}
	t.Setenv("ZELLIJ_PANE_ID", "5")
	if got, err := Lookup(fake(t), "harness"); err != nil || got != (Pane{ID: "terminal_4", Tab: 1}) {
		t.Errorf("Lookup = %+v, %v; want no command", got, err)
	}
}

func TestMarker(t *testing.T) {
	tests := []struct{ cmd, kind, rest string }{
		{"", "", ""},
		{"claude", "", "claude"},
		{"/opt/bin/omp --model x", "", "/opt/bin/omp --model x"},
		{"/usr/bin/env DOCENT_HARNESS_KIND=omp sandbox-exec -f omp.sb omp", "omp", "sandbox-exec -f omp.sb omp"},
		{"env DOCENT_HARNESS_KIND=omp omp", "omp", "omp"},
		{"/usr/bin/env FOO=1 omp", "", "/usr/bin/env FOO=1 omp"},
		{"/usr/bin/env DOCENT_DEBUG=1 claude", "", "/usr/bin/env DOCENT_DEBUG=1 claude"},
		{"/usr/bin/env DOCENT_HARNESS_KIND=omp", "", "/usr/bin/env DOCENT_HARNESS_KIND=omp"},
		{"/usr/bin/env DOCENT_HARNESS_KIND=claude DOCENT_LAYOUT=/x.kdl claude", "claude", "claude"},
		{"/usr/bin/env DOCENT_LAYOUT=/x.kdl DOCENT_HARNESS_KIND=claude claude --x", "claude", "claude --x"},
		{"/usr/bin/env DOCENT_HARNESS_KIND=omp FOO=1 DOCENT_LAYOUT=/x.kdl omp", "omp", "FOO=1 DOCENT_LAYOUT=/x.kdl omp"},
	}
	for _, tt := range tests {
		kind, ok := Marker(tt.cmd)
		if kind != tt.kind || ok != (tt.kind != "") {
			t.Errorf("Marker(%q) = %q, %v; want %q", tt.cmd, kind, ok, tt.kind)
		}
		if got := Unwrap(tt.cmd); got != tt.rest {
			t.Errorf("Unwrap(%q) = %q; want %q", tt.cmd, got, tt.rest)
		}
	}
}

func TestLayoutMarker(t *testing.T) {
	tests := []struct{ cmd, path string }{
		{"claude", ""},
		{"/usr/bin/env DOCENT_HARNESS_KIND=omp omp", ""},
		{"/usr/bin/env DOCENT_HARNESS_KIND=claude DOCENT_LAYOUT=/x.kdl claude", "/x.kdl"},
		{"/usr/bin/env DOCENT_HARNESS_KIND=claude DOCENT_LAYOUT=/my%20layouts/pair.kdl claude", "/my layouts/pair.kdl"},
		{"/usr/bin/env DOCENT_HARNESS_KIND=omp FOO=1 DOCENT_LAYOUT=/x.kdl omp", ""},
		{"/usr/bin/env DOCENT_HARNESS_KIND=claude DOCENT_LAYOUT=/x.kdl", ""},
		{"/usr/bin/env DOCENT_HARNESS_KIND=claude DOCENT_LAYOUT= claude", ""},
	}
	for _, tt := range tests {
		path, ok := LayoutMarker(tt.cmd)
		if path != tt.path || ok != (tt.path != "") {
			t.Errorf("LayoutMarker(%q) = %q, %v; want %q", tt.cmd, path, ok, tt.path)
		}
	}
}

func TestTabs(t *testing.T) {
	got, err := Tabs(fake(t))
	want := []Tab{{0, "one", "/Users/thane/work/relevatist/docent"}, {1, "two", "/Users/thane/work/relevatist/docent"}}
	if err != nil || !slices.Equal(got, want) {
		t.Errorf("Tabs = %+v, %v; want %+v", got, err, want)
	}
	f := &zellij.Fake{Out: map[string]string{"action list-panes --all --json": `[
		{"id":0,"is_plugin":true,"title":"editor","tab_id":3,"tab_name":"x","pane_cwd":"/p"},
		{"id":1,"title":"shell","tab_id":3,"tab_name":"x","pane_cwd":"/s"}]`}}
	if got, err := Tabs(f); err != nil || !slices.Equal(got, []Tab{{ID: 3, Name: "x"}}) {
		t.Errorf("Tabs = %+v, %v; want no editor dir", got, err)
	}
}
