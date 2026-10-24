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
	for _, name := range []string{"open", "reload", "send", "tour", "tour-view", "diff", "install", "[dir]"} {
		if !strings.Contains(b.String(), name) {
			t.Errorf("usage missing %q:\n%s", name, b.String())
		}
	}
}

func TestDispatch(t *testing.T) {
	for _, name := range []string{"open", "reload", "send", "tour", "tour-view", "diff", "install"} {
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
	gitRepo(t)
	f := workstream(t)
	for name, run := range map[string]func(w io.Writer) int{
		"diff --help":    func(w io.Writer) int { return diffCmd(f, []string{"--help"}, w, io.Discard) },
		"diff":           func(w io.Writer) int { return diffCmd(f, nil, w, io.Discard) },
		"install skill":  func(w io.Writer) int { return installCmd([]string{"skill", "claude"}, w, io.Discard) },
		"install --help": func(w io.Writer) int { return installCmd([]string{"--help"}, io.Discard, w) },
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
