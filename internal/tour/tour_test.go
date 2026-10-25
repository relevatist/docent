package tour

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestParseValid(t *testing.T) {
	got, err := Parse([]byte(`{"title":"Retry","steps":[{"file":"a.go","lines":[1,40],"note":"New type"},{"file":"b.go"},{"note":"## Overview"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	want := Tour{Title: "Retry", Steps: []Step{
		{File: "a.go", Lines: []int{1, 40}, Note: "New type"},
		{File: "b.go"},
		{Note: "## Overview"},
	}}
	if got.Title != want.Title || !slices.EqualFunc(got.Steps, want.Steps, stepEqual) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestParseIgnoresBase(t *testing.T) {
	got, err := Parse([]byte(`{"base":"--output=x","steps":[{"file":"a.go"}]}`))
	if err != nil || len(got.Steps) != 1 {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func stepEqual(a, b Step) bool {
	return a.File == b.File && a.Note == b.Note && slices.Equal(a.Lines, b.Lines)
}

func TestParseErrors(t *testing.T) {
	for in, want := range map[string]string{
		`{"title":"x"}`:                          "no steps",
		`{"steps":[]}`:                           "no steps",
		`{"steps":[{"file":"a.go"}`:              "invalid tour JSON",
		`not json`:                               "invalid tour JSON",
		`{"steps":[{}]}`:                         "step 1: no file or note",
		`{"steps":[{"file":"a"},{"title":"x"}]}`: "step 2: no file or note",
		`{"steps":[{"lines":[1,2]}]}`:            "step 1: lines without file",
		`{"steps":[{"lines":[1,2],"note":"x"}]}`: "step 1: lines without file",
		`{"steps":[{"file":"a","lines":[3]}]}`:   "step 1: lines",
		`{"steps":[{"file":"a","lines":[3,1]}]}`: "step 1: lines",
		`{"steps":[{"file":"a","lines":[0,1]}]}`: "step 1: lines",
	} {
		_, err := Parse([]byte(in))
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: err %v, want %q", in, err, want)
		}
	}
}

// repo writes files into a temp dir and returns it.
func repo(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

var files = map[string]string{
	"a.go": "l1\nl2\nl3\nl4\nl5\nl6\n",
	"b.go": "q\nr\n",
}

func TestBuild(t *testing.T) {
	dir := repo(t, files)
	items := Build(Tour{Steps: []Step{{File: "a.go", Lines: []int{2, 9}}, {Note: "overview"}}}, dir)
	if len(items) != 2 {
		t.Fatalf("items %+v", items)
	}
	if a := items[0]; !a.Editor || a.Invalid != "" || a.Path != filepath.Join(dir, "a.go") || a.Line() != 2 {
		t.Errorf("file step %+v", a)
	}
	if n := items[1]; n.Editor || n.Invalid != "" || n.Path != "" || n.Ref() != "" {
		t.Errorf("note step %+v", n)
	}
}
