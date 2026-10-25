package main

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/relevatist/docent/internal/zellij"
)

func workstream(t *testing.T) *zellij.Fake {
	t.Helper()
	t.Setenv("ZELLIJ_PANE_ID", "5")
	return &zellij.Fake{Out: map[string]string{"action list-panes --all --json": `[
		{"id":0,"title":"editor","tab_id":0},
		{"id":3,"title":"editor","tab_id":1},
		{"id":5,"title":"shell","tab_id":1}]`}}
}

func hasCall(f *zellij.Fake, args ...string) bool {
	return slices.ContainsFunc(f.Calls, func(c []string) bool { return slices.Equal(c, args) })
}

func tempFile(t *testing.T, name string) string {
	t.Helper()
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.WriteFile(name, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	return filepath.Join(dir, name)
}

func TestOpenLine(t *testing.T) {
	abs := tempFile(t, "a.go")
	f := workstream(t)
	var out bytes.Buffer
	if code := openCmd(f, []string{"a.go:42"}, &out); code != 0 {
		t.Fatalf("exit %d: %s", code, out.String())
	}
	if !hasCall(f, "action", "write-chars", "--pane-id", "terminal_3", "--", ":open '"+abs+"'") ||
		!hasCall(f, "action", "write-chars", "--pane-id", "terminal_3", "--", ":goto 42") {
		t.Fatalf("calls %q", f.Calls)
	}
	if slices.ContainsFunc(f.Calls, func(c []string) bool { return c[1] == "focus-pane-id" }) {
		t.Fatalf("focus changed without --focus: %q", f.Calls)
	}
}

func TestOpenFocus(t *testing.T) {
	tempFile(t, "a.go")
	for _, args := range [][]string{{"--focus", "a.go"}, {"a.go", "--focus"}} {
		f := workstream(t)
		var out bytes.Buffer
		if code := openCmd(f, args, &out); code != 0 {
			t.Fatalf("%q: exit %d: %s", args, code, out.String())
		}
		if last := f.Calls[len(f.Calls)-1]; !slices.Equal(last, []string{"action", "focus-pane-id", "terminal_3"}) {
			t.Fatalf("%q: last call %q", args, last)
		}
	}
}

func TestOpenMissingFile(t *testing.T) {
	t.Chdir(t.TempDir())
	f := workstream(t)
	var out bytes.Buffer
	if code := openCmd(f, []string{"missing.go:3"}, &out); code == 0 {
		t.Fatal("want non-zero exit")
	}
	if !strings.Contains(out.String(), "missing.go") {
		t.Fatalf("error does not name file: %q", out.String())
	}
	if len(f.Calls) != 0 {
		t.Fatalf("zellij called: %q", f.Calls)
	}
}

func TestOpenControlChars(t *testing.T) {
	tempFile(t, "a\r.go")
	for _, arg := range []string{"a\r.go", "missing\x1b.go:3"} {
		f := workstream(t)
		var out bytes.Buffer
		if code := openCmd(f, []string{arg}, &out); code == 0 {
			t.Errorf("%q: want non-zero exit", arg)
		}
		if msg := out.String(); !strings.Contains(msg, "control characters") || strings.ContainsAny(msg, "\r\x1b") {
			t.Errorf("%q: output %q", arg, msg)
		}
		if slices.ContainsFunc(f.Calls, func(c []string) bool { return c[1] == "write-chars" }) {
			t.Errorf("%q: calls %q", arg, f.Calls)
		}
	}
}

func TestOpenBadArgs(t *testing.T) {
	tempFile(t, "a.go")
	for _, args := range [][]string{{}, {"a.go", "b.go"}, {"a.go:0"}} {
		f := workstream(t)
		var out bytes.Buffer
		if code := openCmd(f, args, &out); code == 0 {
			t.Errorf("%q: want non-zero exit", args)
		}
		if len(f.Calls) != 0 {
			t.Errorf("%q: zellij called: %q", args, f.Calls)
		}
	}
}

func TestReload(t *testing.T) {
	f := workstream(t)
	var out bytes.Buffer
	if code := reloadCmd(f, nil, &out); code != 0 {
		t.Fatalf("exit %d: %s", code, out.String())
	}
	if !hasCall(f, "action", "write-chars", "--pane-id", "terminal_3", "--", ":reload-all") {
		t.Fatalf("calls %q", f.Calls)
	}
}

func TestReloadOutsideWorkstream(t *testing.T) {
	noEditor := &zellij.Fake{Out: map[string]string{"action list-panes --all --json": `[{"id":0,"title":"x","tab_id":0}]`}}
	for name, setup := range map[string]func() *zellij.Fake{
		"no pane id": func() *zellij.Fake { t.Setenv("ZELLIJ_PANE_ID", ""); return &zellij.Fake{} },
		"no editor":  func() *zellij.Fake { t.Setenv("ZELLIJ_PANE_ID", "0"); return noEditor },
	} {
		var out bytes.Buffer
		if code := reloadCmd(setup(), []string{"--quiet"}, &out); code != 0 || out.Len() != 0 {
			t.Errorf("%s --quiet: exit %d, output %q", name, code, out.String())
		}
		out.Reset()
		if code := reloadCmd(setup(), nil, &out); code == 0 || out.Len() == 0 {
			t.Errorf("%s: exit %d, output %q; want error", name, code, out.String())
		}
	}
}

func TestReloadHelpWarns(t *testing.T) {
	var out bytes.Buffer
	if code := reloadCmd(&zellij.Fake{}, []string{"--help"}, &out); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out.String(), "unsaved changes in all buffers are discarded") {
		t.Fatalf("help lacks warning:\n%s", out.String())
	}
}
