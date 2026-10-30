package zellij

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// fixNow sets now to a fixed time, so the pipe deadline is
// 1700000001500.
func fixNow(t *testing.T) {
	t.Cleanup(func() { now = time.Now })
	now = func() time.Time { return time.UnixMilli(1700000000000) }
}

func TestHelpersArgv(t *testing.T) {
	fixNow(t)
	f := &Fake{}
	_, _ = ListPanes(f)
	_ = Write(f, "terminal_2", []byte("\x1b[27u:q\r"))
	_ = Paste(f, "terminal_4", "a\nb")
	_ = FocusPane(f, "terminal_3")
	_, _ = NewPane(f, "--stacked", "--name", "tour", "--", "docent", "tour-view", "x")

	want := [][]string{
		{"action", "list-panes", "--all", "--json"},
		{"pipe", "--name", "docent-write", "--args", "pane=terminal_2,deadline=1700000001500", "--", "\x1b[27u:q\r"},
		{"action", "write", "--pane-id", "terminal_2", "27", "91", "50", "55", "117", "58", "113", "13"},
		{"pipe", "--name", "docent-paste", "--args", "pane=terminal_4,deadline=1700000001500", "--", "a\nb"},
		{"action", "paste", "--pane-id", "terminal_4", "--", "a\nb"},
		{"action", "focus-pane-id", "terminal_3"},
		{"action", "new-pane", "--stacked", "--name", "tour", "--", "docent", "tour-view", "x"},
	}
	if !slices.EqualFunc(f.Calls, want, slices.Equal) {
		t.Fatalf("calls:\n got %q\nwant %q", f.Calls, want)
	}
}

func TestFakeCannedOutput(t *testing.T) {
	boom := errors.New("boom")
	f := &Fake{
		Out:  map[string]string{"action list-panes --all --json": "[]"},
		Errs: map[string]error{"action focus-pane-id terminal_9": boom},
	}
	out, err := ListPanes(f)
	if err != nil || string(out) != "[]" {
		t.Fatalf("ListPanes = %q, %v", out, err)
	}
	if err := FocusPane(f, "terminal_9"); !errors.Is(err, boom) {
		t.Fatalf("FocusPane err = %v", err)
	}
}

func TestWrite(t *testing.T) {
	fixNow(t)
	b := []byte("\x1b[27u:q\r")
	pipe, write := PipeArgs("docent-write", "terminal_2", string(b), now().Add(1500*time.Millisecond)), WriteArgs("terminal_2", b)
	key := strings.Join(pipe, " ")
	tests := []struct {
		name string
		out  string
		err  error
		want [][]string
	}{
		{"ok", "ok\n", nil, [][]string{pipe}},
		{"two instances", "okok", nil, [][]string{pipe}},
		{"one late, one ok", "lateok", nil, [][]string{pipe}},
		{"no answer", "", nil, [][]string{pipe, write}},
		{"late", "late", nil, [][]string{pipe, write}},
		{"other answer", "nope", nil, [][]string{pipe, write}},
		{"error", "ok", errors.New("boom"), [][]string{pipe, write}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &Fake{Out: map[string]string{key: tt.out}, Errs: map[string]error{key: tt.err}}
			if err := Write(f, "terminal_2", b); err != nil {
				t.Fatal(err)
			}
			if !slices.EqualFunc(f.Calls, tt.want, slices.Equal) {
				t.Errorf("calls:\n got %q\nwant %q", f.Calls, tt.want)
			}
		})
	}
}

func TestPasteOK(t *testing.T) {
	fixNow(t)
	pipe := PipeArgs("docent-paste", "terminal_4", "x", now().Add(1500*time.Millisecond))
	f := &Fake{Out: map[string]string{strings.Join(pipe, " "): "ok"}}
	if err := Paste(f, "terminal_4", "x"); err != nil {
		t.Fatal(err)
	}
	if want := [][]string{pipe}; !slices.EqualFunc(f.Calls, want, slices.Equal) {
		t.Errorf("calls = %q; want %q", f.Calls, want)
	}
}

func TestWriteInvalidUTF8SkipsPlugin(t *testing.T) {
	f := &Fake{}
	if err := Write(f, "terminal_2", []byte{0xff}); err != nil {
		t.Fatal(err)
	}
	if want := [][]string{WriteArgs("terminal_2", []byte{0xff})}; !slices.EqualFunc(f.Calls, want, slices.Equal) {
		t.Errorf("calls = %q; want %q", f.Calls, want)
	}
}

// TestWriteTimeout runs a zellij stub whose pipe never answers.
func TestWriteTimeout(t *testing.T) {
	dir := t.TempDir()
	log := filepath.Join(dir, "log")
	stub := "#!/bin/sh\necho \"$@\" >> " + log + "\n[ \"$1\" = pipe ] && exec /bin/sleep 30\nexit 0\n"
	if err := os.WriteFile(filepath.Join(dir, "zellij"), []byte(stub), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	fixNow(t)
	start := time.Now()
	if err := Write(Exec{}, "terminal_2", []byte("x")); err != nil {
		t.Fatal(err)
	}
	if d := time.Since(start); d < PipeTimeout || d > PipeTimeout+5*time.Second {
		t.Errorf("Write took %v; want about %v", d, PipeTimeout)
	}
	got, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	if want := "pipe --name docent-write --args pane=terminal_2,deadline=1700000001500 -- x\naction write --pane-id terminal_2 120\n"; string(got) != want {
		t.Errorf("zellij calls:\n%s\nwant:\n%s", got, want)
	}
}
