package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/relevatist/zide/internal/tour"
	"github.com/relevatist/zide/internal/zellij"
)

// gitRepo makes a temp repo with one commit of a.go, chdirs into it, and
// returns its path.
func gitRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git")
	}
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	if err := os.WriteFile("a.go", []byte("1\n2\n3\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"init", "-q"}, {"add", "."}, {"commit", "-qm", "init"}} {
		cmd := exec.Command("git", append([]string{"-c", "user.email=a@b", "-c", "user.name=a"}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %q: %v\n%s", args, err, out)
		}
	}
	return dir
}

// paneFake records $ZELLIJ_PANE_ID at the new-pane call and returns a pane ID for it.
type paneFake struct {
	*zellij.Fake
	paneEnv string
}

func newPaneFake(t *testing.T) *paneFake { return &paneFake{Fake: workstream(t)} }

func (f *paneFake) Run(args ...string) ([]byte, error) {
	out, err := f.Fake.Run(args...)
	if len(args) > 1 && args[1] == "new-pane" {
		f.paneEnv = os.Getenv("ZELLIJ_PANE_ID")
		return []byte("terminal_9\n"), err
	}
	return out, err
}

// paneCmd checks that the tour pane opens stacked next to the shell and gets
// focus, and returns its command (after "--").
func paneCmd(t *testing.T, f *paneFake, cwd string) []string {
	t.Helper()
	i := slices.IndexFunc(f.Calls, func(c []string) bool { return len(c) > 1 && c[1] == "new-pane" })
	if i < 0 || f.paneEnv != "5" || os.Getenv("ZELLIJ_PANE_ID") != "5" {
		t.Fatalf("new-pane not issued from the shell pane (env %q), calls %q", f.paneEnv, f.Calls)
	}
	if last := f.Calls[len(f.Calls)-1]; !slices.Equal(last, []string{"action", "focus-pane-id", "terminal_9"}) {
		t.Fatalf("tour pane not focused, last call %q", last)
	}
	want := []string{"action", "new-pane", "--stacked", "--near-current-pane", "--name", "tour", "--close-on-exit", "--cwd", cwd, "--"}
	c := f.Calls[i]
	if len(c) < len(want) || !slices.Equal(c[:len(want)], want) {
		t.Fatalf("new-pane call %q", c)
	}
	return c[len(want):]
}

func TestTourOpensPane(t *testing.T) {
	dir := gitRepo(t)
	if err := os.WriteFile("plan.json", []byte(`{"steps":[{"file":"a.go","lines":[1,2]}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	f := newPaneFake(t)
	var out bytes.Buffer
	if code := tourCmd(f, []string{"plan.json"}, nil, &out); code != 0 {
		t.Fatalf("exit %d: %s", code, out.String())
	}
	cmd := paneCmd(t, f, dir)
	if len(cmd) != 3 || cmd[1] != "tour-view" {
		t.Fatalf("pane command %q", cmd)
	}
	tmp := cmd[2]
	defer func() {
		if err := os.Remove(tmp); err != nil {
			t.Error(err)
		}
	}()
	if b, err := os.ReadFile(tmp); err != nil || !strings.Contains(string(b), "a.go") {
		t.Fatalf("temp tour %q: %s %v", tmp, b, err)
	}
}

func TestTourStdin(t *testing.T) {
	dir := gitRepo(t)
	f := newPaneFake(t)
	var out bytes.Buffer
	in := strings.NewReader(`{"steps":[{"file":"a.go"}]}`)
	if code := tourCmd(f, []string{"-"}, in, &out); code != 0 {
		t.Fatalf("exit %d: %s", code, out.String())
	}
	cmd := paneCmd(t, f, dir)
	if err := os.Remove(cmd[len(cmd)-1]); err != nil {
		t.Error(err)
	}
}

func TestTourInvalid(t *testing.T) {
	gitRepo(t)
	for _, in := range []string{`nope`, `{"steps":[]}`, `{"base":"nosuchrev","steps":[{"file":"a.go"}]}`} {
		f := workstream(t)
		var out bytes.Buffer
		if code := tourCmd(f, []string{"-"}, strings.NewReader(in), &out); code == 0 {
			t.Errorf("%s: want non-zero exit", in)
		}
		if out.Len() == 0 {
			t.Errorf("%s: no error printed", in)
		}
		if len(f.Calls) != 0 {
			t.Errorf("%s: zellij called: %q", in, f.Calls)
		}
	}
}

func TestDiffNoChanges(t *testing.T) {
	gitRepo(t)
	f := workstream(t)
	var out bytes.Buffer
	if code := diffCmd(f, nil, &out, &out); code != 0 || strings.TrimSpace(out.String()) != "no changes" {
		t.Fatalf("exit %d: %q", code, out.String())
	}
	if len(f.Calls) != 0 {
		t.Fatalf("zellij called: %q", f.Calls)
	}
}

func TestDiffOpensPane(t *testing.T) {
	dir := gitRepo(t)
	if err := os.WriteFile("a.go", []byte("1\nTWO\n3\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	f := newPaneFake(t)
	var out bytes.Buffer
	if code := diffCmd(f, []string{"--", "a.go"}, &out, &out); code != 0 {
		t.Fatalf("exit %d: %s", code, out.String())
	}
	if cmd := paneCmd(t, f, dir); !slices.Equal(cmd[1:], []string{"tour-view", "--diff", "--", "a.go"}) {
		t.Fatalf("pane command %q", cmd)
	}
}

func TestStepOpensEditorWithoutFocus(t *testing.T) {
	dir := gitRepo(t)
	f := workstream(t)
	items := tour.Build(tour.Tour{Steps: []tour.Step{{File: "a.go", Lines: []int{2, 3}}}}, nil, dir)
	m := newModel(f, items)
	m.Start()
	if !hasCall(f, "action", "write-chars", "--pane-id", "terminal_3", "--", ":open '"+filepath.Join(dir, "a.go")+"'") ||
		!hasCall(f, "action", "write-chars", "--pane-id", "terminal_3", "--", ":goto 2") {
		t.Fatalf("calls %q", f.Calls)
	}
	if slices.ContainsFunc(f.Calls, func(c []string) bool { return c[1] == "focus-pane-id" }) {
		t.Fatalf("focus changed: %q", f.Calls)
	}
}

func TestTourViewRemovesTempFile(t *testing.T) {
	gitRepo(t)
	tmp := filepath.Join(t.TempDir(), "t.json")
	if err := os.WriteFile(tmp, []byte(`{"steps":[{"file":"a.go"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := loadTourView([]string{tmp}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(tmp); !os.IsNotExist(err) {
		t.Fatalf("temp tour not removed: %v", err)
	}
}
