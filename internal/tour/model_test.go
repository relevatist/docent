package tour

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

type recorder struct{ opened, sent []string }

func model(t *testing.T, tr Tour) (*Model, *recorder, string) {
	t.Helper()
	dir := repo(t, files)
	rec := &recorder{}
	m := &Model{
		Items: Build(tr, dir),
		Open: func(path string, line int) error {
			rec.opened = append(rec.opened, fmt.Sprintf("%s:%d", path, line))
			return nil
		},
		Send: func(ref string) error {
			rec.sent = append(rec.sent, ref)
			return nil
		},
	}
	m.Start()
	return m, rec, dir
}

func TestModelNavigation(t *testing.T) {
	m, rec, dir := model(t, Tour{Steps: []Step{{File: "a.go", Lines: []int{1, 3}}, {File: "b.go"}}})
	if m.Label() != "1/2" {
		t.Fatalf("label %q", m.Label())
	}
	m.Key("p")
	if m.Pos != 0 {
		t.Fatal("p moved before first step")
	}
	m.Key("n")
	if m.Label() != "2/2" {
		t.Fatalf("label %q", m.Label())
	}
	m.Key("n")
	if m.Pos != 1 {
		t.Fatalf("n moved past last step: %d", m.Pos)
	}
	m.Key("o")
	want := []string{filepath.Join(dir, "a.go") + ":1", filepath.Join(dir, "b.go") + ":0", filepath.Join(dir, "b.go") + ":0"}
	if !slices.Equal(rec.opened, want) {
		t.Fatalf("opened %q, want %q", rec.opened, want)
	}
	if !m.Key("q") {
		t.Fatal("q does not quit")
	}
}

func TestModelSend(t *testing.T) {
	m, rec, _ := model(t, Tour{Steps: []Step{{File: "a.go", Lines: []int{1, 3}}, {File: "b.go"}}})
	m.Key("s")
	m.Key("n")
	m.Key("s")
	if want := []string{"a.go:1-3", "b.go"}; !slices.Equal(rec.sent, want) {
		t.Fatalf("sent %q, want %q", rec.sent, want)
	}
}

func TestNoteOnlyStep(t *testing.T) {
	m, rec, dir := model(t, Tour{Steps: []Step{{Note: "## Overview"}, {File: "a.go", Lines: []int{2, 3}}, {Note: "wrap-up"}}})
	m.Key("o")
	m.Key("s")
	m.Key("n")
	m.Key("n")
	m.Key("o")
	m.Key("s")
	if want := []string{filepath.Join(dir, "a.go") + ":2"}; !slices.Equal(rec.opened, want) {
		t.Fatalf("opened %q, want %q", rec.opened, want)
	}
	if len(rec.sent) != 0 {
		t.Fatalf("sent %q", rec.sent)
	}
}

func TestInvalidSteps(t *testing.T) {
	m, rec, _ := model(t, Tour{Steps: []Step{
		{File: "a.go"},
		{File: "missing.go", Lines: []int{1, 2}},
		{File: "b.go", Lines: []int{5, 9}},
	}})
	m.Key("n")
	m.Key("n")
	if got := m.Items[1].Invalid; got != "invalid: missing.go not found" {
		t.Errorf("missing file: %q", got)
	}
	if got := m.Items[2].Invalid; !strings.Contains(got, "outside") {
		t.Errorf("range outside file: %q", got)
	}
	if len(rec.opened) != 1 {
		t.Fatalf("opened %q, want only a.go", rec.opened)
	}
}

func TestControlCharStep(t *testing.T) {
	m, rec, _ := model(t, Tour{Steps: []Step{{File: "a.go"}, {File: "x\r:sh touch y\r.go"}, {File: "e\x1b.go"}}})
	for _, it := range m.Items[1:3] {
		if !strings.Contains(it.Invalid, "control characters") || strings.ContainsAny(it.Invalid, "\r\x1b") || it.Editor {
			t.Errorf("step %q: invalid %q, editor %v", it.File, it.Invalid, it.Editor)
		}
	}
	m.Key("n")
	m.Key("n")
	if len(rec.opened) != 1 {
		t.Fatalf("opened %q, want only a.go", rec.opened)
	}
}
