package panes

import (
	"encoding/json/v2"
	"os"
	"strings"
	"testing"

	"github.com/relevatist/zide/internal/zellij"
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
	if err == nil || !strings.Contains(err.Error(), "must run inside a zide workstream") {
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
	if err != nil || got != (Pane{ID: "terminal_1", Command: "/opt/bin/omp --model x"}) {
		t.Errorf("Lookup = %+v, %v", got, err)
	}
	t.Setenv("ZELLIJ_PANE_ID", "5")
	if got, err := Lookup(fake(t), "harness"); err != nil || got != (Pane{ID: "terminal_4"}) {
		t.Errorf("Lookup = %+v, %v; want no command", got, err)
	}
}
