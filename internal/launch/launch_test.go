package launch

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/relevatist/zide/internal/zellij"
)

func TestLayout(t *testing.T) {
	kdl, err := Layout(`omp --model "x"`, `my "app"`)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`tab name="my \"app\"" {`,
		`pane name="editor" command="hx" size="60%" focus=true`,
		`pane name="harness" command="omp"`,
		`args "--model" "\"x\""`,
		`pane name="shell"`,
	} {
		if !strings.Contains(kdl, want) {
			t.Errorf("layout missing %q:\n%s", want, kdl)
		}
	}
	editor, harness, shell := strings.Index(kdl, `"editor"`), strings.Index(kdl, `"harness"`), strings.Index(kdl, `"shell"`)
	if editor >= harness || harness >= shell {
		t.Errorf("want editor, harness, shell order:\n%s", kdl)
	}
}

func TestLayoutNoArgs(t *testing.T) {
	kdl, err := Layout("claude", "app")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(kdl, "args") {
		t.Errorf("unexpected args:\n%s", kdl)
	}
}

func TestLayoutEmptyHarness(t *testing.T) {
	if _, err := Layout("  ", "app"); err == nil {
		t.Error("want error for empty harness")
	}
}

type attachCall struct {
	dir  string
	args []string
}

func run(t *testing.T, name string, inZellij bool, sessions string) (*zellij.Fake, []attachCall, string, error) {
	t.Helper()
	f := &zellij.Fake{Out: map[string]string{
		"--version":                     "zellij 0.44.0",
		"list-sessions --no-formatting": sessions,
	}}
	var calls []attachCall
	attach := func(dir string, args ...string) error {
		if len(args) == 4 {
			b, err := os.ReadFile(args[3])
			if err != nil {
				t.Error(err)
			}
			args[3] = string(b)
		}
		calls = append(calls, attachCall{dir, args})
		return nil
	}
	var warn bytes.Buffer
	err := Run(f, attach, &warn, t.TempDir(), name, "cat", inZellij)
	return f, calls, warn.String(), err
}

type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }

func TestRunWarningWriteFails(t *testing.T) {
	f := &zellij.Fake{Out: map[string]string{"--version": "zellij 0.44.0"}}
	if err := Run(f, nil, failWriter{}, t.TempDir(), "", "cat", true); err == nil {
		t.Error("want write error")
	}
}

func TestRunInsideZellij(t *testing.T) {
	f, calls, warn, err := run(t, "", true, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(warn, "0.45") {
		t.Errorf("want version warning, got %q", warn)
	}
	last := f.Calls[len(f.Calls)-1]
	dir := last[len(last)-1]
	kdl, _ := Layout("cat", filepath.Base(dir))
	want := []string{"action", "new-tab", "--layout-string", kdl, "--name", filepath.Base(dir), "--cwd", dir}
	if !slices.Equal(last, want) {
		t.Errorf("got %q, want %q", last, want)
	}
	if len(calls) != 0 {
		t.Errorf("unexpected attach: %v", calls)
	}
}

func TestRunOutsideZellij(t *testing.T) {
	_, calls, _, err := run(t, "", false, "other [Created 3s ago] \n")
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 {
		t.Fatalf("want one attach, got %v", calls)
	}
	session := filepath.Base(calls[0].dir)
	kdl, _ := Layout("cat", session)
	want := []string{"--session", session, "--new-session-with-layout", kdl}
	if !slices.Equal(calls[0].args, want) {
		t.Errorf("got %q, want %q", calls[0].args, want)
	}
	if !strings.Contains(calls[0].args[3], `tab name="`+session+`" {`) {
		t.Errorf("layout tab not named %q:\n%s", session, calls[0].args[3])
	}
}

func TestRunOutsideZellijSessionExists(t *testing.T) {
	f := &zellij.Fake{}
	var calls []attachCall
	attach := func(dir string, args ...string) error {
		calls = append(calls, attachCall{dir, args})
		return nil
	}
	dir := filepath.Join(t.TempDir(), "app")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	f.Out = map[string]string{"--version": "zellij 0.45.1", "list-sessions --no-formatting": "x [Created 1s ago] \napp [Created 2s ago] \n"}
	var warn bytes.Buffer
	if err := Run(f, attach, &warn, dir, "", "cat", false); err != nil {
		t.Fatal(err)
	}
	if warn.Len() != 0 {
		t.Errorf("unexpected warning %q", warn.String())
	}
	last := f.Calls[len(f.Calls)-1]
	kdl, _ := Layout("cat", "app")
	want := []string{"--session", "app", "action", "new-tab", "--layout-string", kdl, "--name", "app", "--cwd", dir}
	if !slices.Equal(last, want) {
		t.Errorf("got %q, want %q", last, want)
	}
	if len(calls) != 1 || !slices.Equal(calls[0].args, []string{"attach", "app"}) {
		t.Errorf("want attach app, got %v", calls)
	}
}

func TestRunName(t *testing.T) {
	f, _, _, err := run(t, "app-retry", true, "")
	if err != nil {
		t.Fatal(err)
	}
	last := f.Calls[len(f.Calls)-1]
	if i := slices.Index(last, "--name"); i < 0 || last[i+1] != "app-retry" {
		t.Errorf("got %q, want --name app-retry", last)
	}

	_, calls, _, err := run(t, "app-retry", false, "app-retry [Created 3s ago] \n")
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || !slices.Equal(calls[0].args, []string{"attach", "app-retry"}) {
		t.Errorf("want attach app-retry, got %v", calls)
	}

	_, calls, _, err = run(t, "app-retry", false, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || calls[0].args[1] != "app-retry" {
		t.Errorf("want new session app-retry, got %v", calls)
	}
}

func TestSessionState(t *testing.T) {
	out := "zide-x [Created 1m ago] \n" +
		"zide [Created 3m 9s ago] (EXITED - attach to resurrect)\n" +
		"live [Created 3s ago] (current)\n"
	for name, want := range map[string][2]bool{
		"live":   {true, false},
		"zide":   {true, true},
		"zide-x": {true, false},
		"zid":    {false, false},
		"absent": {false, false},
	} {
		found, exited := sessionState(out, name)
		if [2]bool{found, exited} != want {
			t.Errorf("sessionState(%q) = %v, %v, want %v", name, found, exited, want)
		}
	}
}

func TestRunOutsideZellijSessions(t *testing.T) {
	kdl, _ := Layout("cat", "app")
	for _, tc := range []struct {
		sessions string
		calls    [][]string
		attach   []string
		stderr   string
	}{
		{"", nil, []string{"--session", "app", "--new-session-with-layout", kdl}, ""},
		{"app [Created 3s ago] \n", [][]string{{"--session", "app", "action", "new-tab", "--layout-string", kdl, "--name", "app", "--cwd", "DIR"}}, []string{"attach", "app"}, ""},
		{"app [Created 3m ago] (EXITED - attach to resurrect)\n", [][]string{{"delete-session", "app"}}, []string{"--session", "app", "--new-session-with-layout", kdl}, "zide: removed exited session record \"app\"\n"},
	} {
		f := &zellij.Fake{Out: map[string]string{"--version": "zellij 0.45.1", "list-sessions --no-formatting": tc.sessions}}
		var got []string
		attach := func(_ string, args ...string) error {
			if len(args) == 4 {
				b, err := os.ReadFile(args[3])
				if err != nil {
					t.Error(err)
				}
				args[3] = string(b)
			}
			got = args
			return nil
		}
		dir := filepath.Join(t.TempDir(), "app")
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		var stderr bytes.Buffer
		if err := Run(f, attach, &stderr, dir, "", "cat", false); err != nil {
			t.Fatal(err)
		}
		want := append([][]string{{"--version"}, {"list-sessions", "--no-formatting"}}, tc.calls...)
		for _, c := range want {
			if i := slices.Index(c, "DIR"); i >= 0 {
				c[i] = dir
			}
		}
		if !slices.EqualFunc(f.Calls, want, slices.Equal) {
			t.Errorf("sessions %q: calls\n got %q\nwant %q", tc.sessions, f.Calls, want)
		}
		if !slices.Equal(got, tc.attach) {
			t.Errorf("sessions %q: attach got %q, want %q", tc.sessions, got, tc.attach)
		}
		if stderr.String() != tc.stderr {
			t.Errorf("sessions %q: stderr %q, want %q", tc.sessions, stderr.String(), tc.stderr)
		}
	}
}

func TestRunMissingDir(t *testing.T) {
	f := &zellij.Fake{}
	attach := func(string, ...string) error { t.Error("attach called"); return nil }
	err := Run(f, attach, io.Discard, "/no/such/dir", "", "cat", true)
	if err == nil || !strings.Contains(err.Error(), "/no/such/dir") {
		t.Errorf("want error naming dir, got %v", err)
	}
	if len(f.Calls) != 0 {
		t.Errorf("unexpected zellij calls: %q", f.Calls)
	}
}

func TestOldVersion(t *testing.T) {
	for out, want := range map[string]bool{
		"zellij 0.45.1\n": false,
		"zellij 0.46.0":   false,
		"zellij 1.0.0":    false,
		"zellij 0.44.3":   true,
		"zellij 0.9.0":    true,
		"garbage":         true,
	} {
		if got := OldVersion(out); got != want {
			t.Errorf("OldVersion(%q) = %v, want %v", out, got, want)
		}
	}
}
