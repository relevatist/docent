package tour

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestParseValid(t *testing.T) {
	got, err := Parse([]byte(`{"title":"Retry","base":"main","steps":[{"file":"a.go","lines":[1,40],"note":"New type"},{"file":"b.go"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	want := Tour{Title: "Retry", Base: "main", Steps: []Step{
		{File: "a.go", Lines: []int{1, 40}, Note: "New type"},
		{File: "b.go"},
	}}
	if got.Title != want.Title || got.Base != want.Base || !slices.EqualFunc(got.Steps, want.Steps, stepEqual) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func stepEqual(a, b Step) bool {
	return a.File == b.File && a.Note == b.Note && slices.Equal(a.Lines, b.Lines)
}

func TestParseErrors(t *testing.T) {
	for in, want := range map[string]string{
		`{"title":"x"}`:                                "no steps",
		`{"steps":[]}`:                                 "no steps",
		`{"steps":[{"file":"a.go"}`:                    "invalid tour JSON",
		`not json`:                                     "invalid tour JSON",
		`{"steps":[{"lines":[1,2]}]}`:                  "step 1: no file",
		`{"steps":[{"file":"a","lines":[3]}]}`:         "step 1: lines",
		`{"steps":[{"file":"a","lines":[3,1]}]}`:       "step 1: lines",
		`{"steps":[{"file":"a","lines":[0,1]}]}`:       "step 1: lines",
		`{"base":"--output=x","steps":[{"file":"a"}]}`: "base",
		`{"base":"-p","steps":[{"file":"a"}]}`:         "base",
	} {
		_, err := Parse([]byte(in))
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: err %v, want %q", in, err, want)
		}
	}
}

func readDiff(t *testing.T, name string) []Hunk {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	h, err := ParseDiff(string(b))
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func TestParseDiffMulti(t *testing.T) {
	got := readDiff(t, "multi.diff")
	want := []Hunk{
		{File: "a.go", OldStart: 2, OldCount: 1, NewStart: 2, NewCount: 1, Lines: []string{"-l2", "+L2"}},
		{File: "a.go", OldStart: 4, OldCount: 1, NewStart: 3, NewCount: 0, Lines: []string{"-l4"}},
		{File: "a.go", OldStart: 6, OldCount: 0, NewStart: 6, NewCount: 1, Lines: []string{"+l7"}},
		{File: "b.go", OldStart: 1, OldCount: 0, NewStart: 2, NewCount: 1, Lines: []string{"+r"}},
		{File: "gone.go", Deleted: true, OldStart: 1, OldCount: 2, NewStart: 0, NewCount: 0, Lines: []string{"-x", "-y"}},
		{File: "new.go", OldStart: 0, OldCount: 0, NewStart: 1, NewCount: 2, Lines: []string{"+n1", "+n2"}},
	}
	if !slices.EqualFunc(got, want, hunkEqual) {
		t.Fatalf("got\n%+v\nwant\n%+v", got, want)
	}
}

func hunkEqual(a, b Hunk) bool {
	return a.File == b.File && a.Deleted == b.Deleted && a.OldStart == b.OldStart && a.OldCount == b.OldCount &&
		a.NewStart == b.NewStart && a.NewCount == b.NewCount && slices.Equal(a.Lines, b.Lines)
}

func TestParseDiffNewAndDeleted(t *testing.T) {
	if h := readDiff(t, "new.diff"); len(h) != 1 || h[0].File != "new.go" || h[0].Deleted || h[0].NewCount != 2 {
		t.Errorf("new.diff: %+v", h)
	}
	if h := readDiff(t, "deleted.diff"); len(h) != 1 || h[0].File != "gone.go" || !h[0].Deleted || len(h[0].Lines) != 2 {
		t.Errorf("deleted.diff: %+v", h)
	}
	if h, err := ParseDiff(""); err != nil || len(h) != 0 {
		t.Errorf("empty diff: %v %v", h, err)
	}
	if _, err := ParseDiff("--- a/x\n+++ b/x\n@@ bad @@\n"); err == nil {
		t.Error("bad hunk header: want error")
	}
}

// repo writes files into a temp dir and returns the dir and abs-path hunks of multi.diff.
func repo(t *testing.T, files map[string]string) (string, []Hunk) {
	t.Helper()
	dir := t.TempDir()
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	h := readDiff(t, "multi.diff")
	for i := range h {
		h[i].File = filepath.Join(dir, h[i].File)
	}
	return dir, h
}

var multiFiles = map[string]string{
	"a.go":   "l1\nL2\nl3\nl5\nl6\nl7\n",
	"b.go":   "q\nr\n",
	"new.go": "n1\nn2\n",
}

func titles(items []Item) []string {
	var s []string
	for _, it := range items {
		s = append(s, it.Heading())
	}
	return s
}

func TestCoverageUncovered(t *testing.T) {
	dir, h := repo(t, multiFiles)
	tr := Tour{Steps: []Step{
		{File: "a.go", Lines: []int{2, 3}}, // overlaps hunks 1 and 2 (deletion after line 3)
		{File: "new.go"},                   // file-only: covers all its hunks
	}}
	items := Build(tr, h, dir)
	if len(items) != 3 {
		t.Fatalf("items %q", titles(items))
	}
	last := items[2]
	if last.Title != NotCovered {
		t.Fatalf("last %q", last.Heading())
	}
	var refs []string
	for _, l := range last.Body {
		if l.Kind == Header {
			refs = append(refs, l.Text)
		}
	}
	if want := []string{"a.go:6", "b.go:2", "gone.go (deleted)"}; !slices.Equal(refs, want) {
		t.Fatalf("uncovered %q, want %q", refs, want)
	}
}

func TestCoverageFull(t *testing.T) {
	dir, h := repo(t, multiFiles)
	tr := Tour{Steps: []Step{{File: "a.go"}, {File: "b.go", Lines: []int{1, 2}}, {File: "gone.go"}, {File: "new.go", Lines: []int{2, 2}}}}
	if items := Build(tr, h, dir); len(items) != 4 || slices.Contains(titles(items), NotCovered) {
		t.Fatalf("items %q", titles(items))
	}
}

func TestDiffTour(t *testing.T) {
	dir, h := repo(t, multiFiles)
	tr := DiffTour(h, dir)
	want := []Step{
		{File: "a.go", Lines: []int{2, 2}},
		{File: "a.go", Lines: []int{3, 4}},
		{File: "a.go", Lines: []int{6, 6}},
		{File: "b.go", Lines: []int{2, 2}},
		{File: "gone.go", Lines: []int{1, 1}},
		{File: "new.go", Lines: []int{1, 2}},
	}
	if !slices.EqualFunc(tr.Steps, want, stepEqual) {
		t.Fatalf("steps %+v", tr.Steps)
	}
	if items := Build(tr, h, dir); len(items) != len(want) {
		t.Fatalf("items %q", titles(items))
	}
}

func TestGitChanges(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git")
	}
	dir := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.email=a@b", "-c", "user.name=a"}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %q: %v\n%s", args, err, out)
		}
	}
	write := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	git("init", "-q")
	write("a.go", "1\n2\n")
	write("b.go", "1\n")
	git("add", ".")
	git("commit", "-qm", "init")
	write("a.go", "1\nTWO\n") // unstaged
	write("b.go", "1\n2\n")
	git("add", "b.go")     // staged
	write("c.go", "new\n") // untracked

	h, err := Changes(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, x := range h {
		got = append(got, filepath.Base(x.File))
		if !filepath.IsAbs(x.File) {
			t.Errorf("not absolute: %s", x.File)
		}
	}
	if want := []string{"a.go", "b.go", "c.go"}; !slices.Equal(got, want) {
		t.Fatalf("files %q, want %q", got, want)
	}
	if c := h[2]; c.NewStart != 1 || c.NewCount != 1 || !slices.Equal(c.Lines, []string{"+new"}) {
		t.Errorf("untracked hunk %+v", c)
	}

	if h, err := GitDiff(dir); err != nil || len(h) != 1 {
		t.Fatalf("git diff (unstaged only): %+v %v", h, err)
	}
	if _, err := Changes(dir, "nosuchrev"); err == nil {
		t.Error("bad base: want error")
	}
	out := filepath.Join(t.TempDir(), "x")
	if _, err := Changes(dir, "--output="+out); err == nil {
		t.Error("option as base: want error")
	}
	if _, err := os.Stat(out); err == nil {
		t.Error("option as base: git wrote --output file")
	}
}

func TestGitChangesSymlinkedDir(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git")
	}
	real := filepath.Join(t.TempDir(), "real")
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Mkdir(real, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", link, "-c", "user.email=a@b", "-c", "user.name=a"}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %q: %v\n%s", args, err, out)
		}
	}
	write := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(link, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	git("init", "-q")
	write("a.go", "1\n")
	git("add", ".")
	git("commit", "-qm", "init")
	write("a.go", "1\n2\n")
	write("c.go", "new\n")

	h, err := Changes(link, "")
	if err != nil {
		t.Fatal(err)
	}
	items := Build(Tour{Steps: []Step{{File: "a.go"}, {File: "c.go"}}}, h, link)
	if len(items) != 2 {
		t.Fatalf("items %q", titles(items))
	}
	for _, it := range items {
		if !slices.ContainsFunc(it.Body, func(l Line) bool { return l.Kind == Added }) {
			t.Errorf("%s: no added lines: %+v", it.File, it.Body)
		}
	}
}
