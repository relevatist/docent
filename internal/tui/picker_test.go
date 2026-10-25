package tui

import (
	"iter"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func mkdirs(t *testing.T, root string, rel ...string) []string {
	t.Helper()
	var dirs []string
	for _, r := range rel {
		d := filepath.Join(root, r)
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
		dirs = append(dirs, d)
	}
	return dirs
}

// fzfCall is one fzf run that the fake saw.
type fzfCall struct {
	args, in []string
}

// fakeFzf returns each output in turn, and records the calls.
type fakeFzf struct {
	outs  [][]string
	codes []int
	calls []fzfCall
}

func (f *fakeFzf) run(args []string, in iter.Seq[string]) ([]string, int, error) {
	i := len(f.calls)
	f.calls = append(f.calls, fzfCall{args, slices.Collect(in)})
	return f.outs[i], f.codes[i], nil
}

func (f *fakeFzf) arg(call int, prefix string) string {
	for _, a := range f.calls[call].args {
		if strings.HasPrefix(a, prefix) {
			return a
		}
	}
	return ""
}

// home sets HOME to a temp dir with these dirs: wt/retry/app, src/app (a
// repo), src/lib.
func home(t *testing.T) (string, []string) {
	t.Helper()
	h := t.TempDir()
	t.Setenv("HOME", h)
	dirs := mkdirs(t, h, "wt/retry/app", "src/app/.git", "src/lib")
	dirs[1] = filepath.Dir(dirs[1])
	return h, dirs
}

func TestPickDirLines(t *testing.T) {
	h, dirs := home(t)
	f := &fakeFzf{outs: [][]string{{"", "", "~/src/lib"}}, codes: []int{0}}
	in := PickerInput{Recent: []string{dirs[2], filepath.Join(h, "gone")}, Roots: []string{filepath.Join(h, "src")}, Harness: "claude"}
	got, ok, err := pickWorkstream(in, f.run)
	if err != nil || !ok || got != (Picked{Dir: dirs[2], Harness: "claude"}) {
		t.Fatalf("pick = %+v, %v, %v", got, ok, err)
	}
	want := []string{"~/src/lib", "~/src/app", "~", "~/src", "~/wt", "~/wt/retry", "~/wt/retry/app"}
	if !slices.Equal(f.calls[0].in, want) {
		t.Errorf("fzf input = %q; want %q", f.calls[0].in, want)
	}
	if f.arg(0, "--expect=") != "--expect=alt-enter" || !strings.Contains(f.arg(0, "--header="), "claude") {
		t.Errorf("fzf args = %q", f.calls[0].args)
	}
}

func TestPickTypedPath(t *testing.T) {
	_, dirs := home(t)
	other := t.TempDir()
	for _, tt := range []struct {
		query string
		want  string
	}{
		{other, other},
		{"~/src/lib/", dirs[2]},
		{"nope", ""},
		{"~/missing", ""},
	} {
		f := &fakeFzf{outs: [][]string{{tt.query, ""}}, codes: []int{1}}
		got, ok, err := pickWorkstream(PickerInput{Harness: "claude"}, f.run)
		if tt.want == "" {
			if err == nil || !strings.Contains(err.Error(), tt.query) {
				t.Errorf("query %q: err = %v; want one that names it", tt.query, err)
			}
			continue
		}
		if err != nil || !ok || got.Dir != tt.want {
			t.Errorf("query %q: pick = %+v, %v, %v; want %s", tt.query, got, ok, err, tt.want)
		}
	}
}

func TestPickTypedPathOverSelection(t *testing.T) {
	home(t)
	other := t.TempDir()
	f := &fakeFzf{outs: [][]string{{other, "", "~/src/lib"}}, codes: []int{0}}
	got, ok, err := pickWorkstream(PickerInput{Harness: "claude"}, f.run)
	if err != nil || !ok || got.Dir != other {
		t.Errorf("pick = %+v, %v, %v; want %s", got, ok, err, other)
	}
}

func TestPickCancel(t *testing.T) {
	home(t)
	for _, outs := range [][][]string{
		{{""}},
		{{"", "alt-enter", "~"}, {""}},
	} {
		codes := make([]int, len(outs))
		codes[len(codes)-1] = 130
		f := &fakeFzf{outs: outs, codes: codes}
		if _, ok, err := pickWorkstream(PickerInput{Harness: "claude"}, f.run); ok || err != nil {
			t.Errorf("after %d fzf runs: ok %v, err %v; want a cancel", len(outs), ok, err)
		}
	}
}

func TestPickEmptyDirList(t *testing.T) {
	home(t)
	f := &fakeFzf{outs: [][]string{{"", ""}}, codes: []int{1}}
	if _, ok, err := pickWorkstream(PickerInput{Harness: "claude"}, f.run); ok || err != nil {
		t.Errorf("ok %v, err %v; want a cancel", ok, err)
	}
}

func TestPickFzfError(t *testing.T) {
	home(t)
	f := &fakeFzf{outs: [][]string{{""}}, codes: []int{2}}
	if _, _, err := pickWorkstream(PickerInput{Harness: "claude"}, f.run); err == nil {
		t.Error("fzf exit 2 gives no error")
	}
}

func TestPickHarness(t *testing.T) {
	h, _ := home(t)
	for _, tt := range []struct {
		out  []string
		code int
		want string
	}{
		{[]string{"sand", "omp-sandbox"}, 0, "omp-sandbox"},
		{[]string{" claude --model opus ", ""}, 1, "claude --model opus"},
		{[]string{"omp -s", "omp-sandbox"}, 0, "omp -s"},
	} {
		f := &fakeFzf{outs: [][]string{{"", "alt-enter", "~"}, tt.out}, codes: []int{0, tt.code}}
		in := PickerInput{Harness: "omp", Aliases: []string{"omp-sandbox", "claude", "box"}}
		got, ok, err := pickWorkstream(in, f.run)
		if err != nil || !ok || got != (Picked{Dir: h, Harness: tt.want}) {
			t.Errorf("pick = %+v, %v, %v; want harness %q", got, ok, err, tt.want)
		}
		if want := []string{"omp", "box", "claude", "omp-sandbox", "codex"}; !slices.Equal(f.calls[1].in, want) {
			t.Errorf("harness candidates = %q; want %q", f.calls[1].in, want)
		}
	}
}

func TestPickEmptyHarness(t *testing.T) {
	home(t)
	f := &fakeFzf{outs: [][]string{{"", "alt-enter", "~"}, {"  ", ""}}, codes: []int{0, 1}}
	if _, _, err := pickWorkstream(PickerInput{Harness: "claude"}, f.run); err == nil || !strings.Contains(err.Error(), "harness is empty") {
		t.Errorf("err = %v; want harness is empty", err)
	}
}

func TestTypedDir(t *testing.T) {
	h, dirs := home(t)
	if err := os.WriteFile(filepath.Join(h, "f"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(filepath.Join(h, "src"))
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		q, home, want string
	}{
		{".", h, wd},
		{"..", h, filepath.Dir(wd)},
		{"./app", h, filepath.Join(wd, "app")},
		{"../src/app", h, filepath.Join(wd, "app")},
		{"~", h, h},
		{"~/src/lib/", h, dirs[2]},
		{"~", "", ""},
		{"~/x", "", ""},
		{"~/f", h, ""},
		{"~src", h, ""},
		{"app", h, ""},
	} {
		got, ok := typedDir(tt.q, tt.home)
		if ok != (tt.want != "") || ok && got != tt.want {
			t.Errorf("typedDir(%q, %q) = %q, %v; want %q", tt.q, tt.home, got, ok, tt.want)
		}
	}
}

func TestDirLinesDropsTabAndNewline(t *testing.T) {
	h, _ := home(t)
	mkdirs(t, h, "a\tb", "c\nd")
	got := slices.Collect(dirLines(t.Context(), PickerInput{}, h))
	if want := []string{"~", "~/src", "~/src/app", "~/src/lib", "~/wt", "~/wt/retry", "~/wt/retry/app"}; !slices.Equal(got, want) {
		t.Errorf("lines = %q; want %q", got, want)
	}
}

func TestDirLinesRelativeRoot(t *testing.T) {
	t.Chdir(t.TempDir())
	mkdirs(t, ".", "src/app/.git")
	got := slices.Collect(dirLines(t.Context(), PickerInput{Roots: []string{"src"}}, ""))
	want, _ := filepath.Abs("src/app")
	if !slices.Equal(got, []string{want}) {
		t.Fatalf("lines = %q; want %q", got, want)
	}
	if d, ok := typedDir(got[0], ""); !ok || d != want {
		t.Errorf("typedDir(%q) = %q, %v", got[0], d, ok)
	}
}

func TestDirLinesHomeTrailingSlash(t *testing.T) {
	h, _ := home(t)
	t.Setenv("HOME", h+"/")
	f := &fakeFzf{outs: [][]string{{""}}, codes: []int{130}}
	_, _, _ = pickWorkstream(PickerInput{}, f.run)
	if !slices.Contains(f.calls[0].in, "~/src") {
		t.Errorf("fzf input = %q; want ~/src", f.calls[0].in)
	}
}

func TestPickOpenTab(t *testing.T) {
	h, _ := home(t)
	isOpen := func(dir string) (string, bool) { return "app", dir == h }
	for _, tt := range []struct {
		choice string
		want   Action
	}{
		{"switch to tab app", Switch},
		{"open another tab", Another},
	} {
		f := &fakeFzf{outs: [][]string{{"", "", "~"}, {tt.choice}}, codes: []int{0, 0}}
		got, ok, err := pickWorkstream(PickerInput{Harness: "claude", IsOpen: isOpen}, f.run)
		if err != nil || !ok || got != (Picked{Dir: h, Harness: "claude", Action: tt.want}) {
			t.Errorf("choice %q: pick = %+v, %v, %v", tt.choice, got, ok, err)
		}
		if want := []string{"switch to tab app", "open another tab"}; !slices.Equal(f.calls[1].in, want) {
			t.Errorf("open-tab lines = %q; want %q", f.calls[1].in, want)
		}
	}
}

func TestPickWarning(t *testing.T) {
	home(t)
	f := &fakeFzf{outs: [][]string{{""}}, codes: []int{130}}
	_, _, _ = pickWorkstream(PickerInput{Harness: "claude", Warning: "bad recent.toml"}, f.run)
	if !strings.Contains(f.arg(0, "--header="), "warning: bad recent.toml") {
		t.Errorf("no warning in fzf args %q", f.calls[0].args)
	}
}

func TestRunFzf(t *testing.T) {
	if _, err := exec.LookPath("fzf"); err != nil {
		t.Skip("fzf not on PATH")
	}
	for _, tt := range []struct {
		query string
		want  []string
		code  int
	}{
		{"b", []string{"b", "bb"}, 0},
		{"z", []string{""}, 1},
	} {
		out, code, err := runFzf([]string{"--filter=" + tt.query}, slices.Values([]string{"a", "b", "bb"}))
		if err != nil || code != tt.code || !slices.Equal(out, tt.want) {
			t.Errorf("filter %q: %q, %d, %v; want %q, %d", tt.query, out, code, err, tt.want, tt.code)
		}
	}
}

func TestRunFzfStopsInput(t *testing.T) {
	if _, err := exec.LookPath("fzf"); err != nil {
		t.Skip("fzf not on PATH")
	}
	stopped := make(chan struct{})
	endless := func(yield func(string) bool) {
		defer close(stopped)
		for yield("a") {
		}
	}
	if _, code, _ := runFzf([]string{"--no-such-flag"}, endless); code != 2 {
		t.Errorf("code = %d; want 2", code)
	}
	<-stopped
}
