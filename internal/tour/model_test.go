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
	dir, h := repo(t, multiFiles)
	rec := &recorder{}
	m := &Model{
		Items: Build(tr, h, dir),
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
	// a.go:6, gone.go and new.go are uncovered, so there are 3 steps.
	if m.Label() != "1/3" {
		t.Fatalf("label %q", m.Label())
	}
	m.Key("p")
	if m.Pos != 0 {
		t.Fatal("p moved before first step")
	}
	m.Key("n")
	if m.Label() != "2/3" {
		t.Fatalf("label %q", m.Label())
	}
	m.Key("n")
	m.Key("n")
	if m.Pos != 2 {
		t.Fatalf("n moved past last step: %d", m.Pos)
	}
	m.Key("o")
	want := []string{filepath.Join(dir, "a.go") + ":1", filepath.Join(dir, "b.go") + ":0"}
	if !slices.Equal(rec.opened, want) {
		t.Fatalf("opened %q, want %q (Not covered step must not open)", rec.opened, want)
	}
	if !m.Key("q") {
		t.Fatal("q does not quit")
	}
}

func TestModelSend(t *testing.T) {
	m, rec, _ := model(t, Tour{Steps: []Step{{File: "a.go", Lines: []int{1, 3}}, {File: "b.go"}, {File: "gone.go"}, {File: "new.go"}}})
	m.Key("s")
	m.Key("n")
	m.Key("s")
	if want := []string{"a.go:1-3", "b.go"}; !slices.Equal(rec.sent, want) {
		t.Fatalf("sent %q, want %q", rec.sent, want)
	}
}

func TestStepView(t *testing.T) {
	m, _, _ := model(t, Tour{Steps: []Step{{File: "a.go", Lines: []int{1, 6}, Note: "the change"}}})
	it := m.Items[0]
	if it.Heading() != "a.go:1-6" || it.Note != "the change" {
		t.Fatalf("heading %q note %q", it.Heading(), it.Note)
	}
	var got []string
	for _, l := range it.Body {
		got = append(got, fmt.Sprintf("%c%d %s", l.Kind, l.Num, l.Text))
	}
	want := []string{
		" 1 l1",
		"-0 l2",
		"+2 L2",
		" 3 l3",
		"-0 l4",
		" 4 l5",
		" 5 l6",
		"+6 l7",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("body\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestStepViewContext(t *testing.T) {
	m, _, _ := model(t, Tour{Steps: []Step{{File: "a.go", Lines: []int{5, 5}}}})
	for _, l := range m.Items[0].Body {
		if l.Kind != Context {
			t.Fatalf("unchanged range has %c line %q", l.Kind, l.Text)
		}
	}
	if len(m.Items[0].Body) != 1 {
		t.Fatalf("body %+v", m.Items[0].Body)
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

func TestDeletedFileStep(t *testing.T) {
	m, rec, _ := model(t, Tour{Steps: []Step{{File: "gone.go"}}})
	it := m.Items[0]
	if it.Invalid != "" || it.Editor || len(it.Body) != 2 || it.Body[0].Kind != Removed {
		t.Fatalf("deleted step %+v", it)
	}
	if len(rec.opened) != 0 {
		t.Fatalf("opened %q", rec.opened)
	}
}
