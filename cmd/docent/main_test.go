package main

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestUsageListsSubcommands(t *testing.T) {
	var b bytes.Buffer
	if err := usage(&b); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(b.String(), "docent diff") {
		t.Errorf("usage lists the removed diff command:\n%s", b.String())
	}
	for _, name := range []string{"open", "reload", "send", "tour", "tour-view", "install", "new", "[dir]", "[--harness cmd|alias]", "[--layout file]"} {
		if !strings.Contains(b.String(), name) {
			t.Errorf("usage missing %q:\n%s", name, b.String())
		}
	}
}

func TestDispatch(t *testing.T) {
	for _, name := range []string{"open", "reload", "send", "tour", "tour-view", "install", "new"} {
		if commands[name].run == nil {
			t.Errorf("no handler for %q", name)
		}
	}
}

type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }

func TestOutputWriteFails(t *testing.T) {
	if usage(failWriter{}) == nil {
		t.Error("usage: want write error")
	}
	tempHome(t)
	fakeHarnesses(t)
	workdir(t)
	f := workstream(t)
	for name, run := range map[string]func(w io.Writer) int{
		"install codex":  func(w io.Writer) int { return installCmd([]string{"codex"}, w, io.Discard, false, nil) },
		"install --help": func(w io.Writer) int { return installCmd([]string{"--help"}, io.Discard, w, false, nil) },
		"new --help":     func(w io.Writer) int { return newCmd(f, []string{"--help"}, w, nil) },
		"open --help":    func(w io.Writer) int { return openCmd(f, []string{"--help"}, w) },
		"reload --help":  func(w io.Writer) int { return reloadCmd(f, []string{"--help"}, w) },
		"tour --help":    func(w io.Writer) int { return tourCmd(f, []string{"--help"}, nil, w) },
	} {
		if code := run(io.Discard); code != 0 {
			t.Errorf("%s: exit %d with a working writer, want 0", name, code)
		}
		if code := run(failWriter{}); code != 1 {
			t.Errorf("%s: exit %d with a failing writer, want 1", name, code)
		}
	}
}
