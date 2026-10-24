package zellij

import (
	"errors"
	"slices"
	"testing"
)

func TestHelpersArgv(t *testing.T) {
	f := &Fake{}
	_, _ = ListPanes(f)
	_ = Paste(f, "terminal_1", "-x")
	_ = WriteChars(f, "terminal_2", ":goto 4")
	_ = SendKeys(f, "terminal_2", "Esc", "Esc")
	_ = FocusPane(f, "terminal_3")
	_, _ = NewPane(f, "--stacked", "--name", "tour", "--", "zide", "tour-view", "x")

	want := [][]string{
		{"action", "list-panes", "--all", "--json"},
		{"action", "paste", "--pane-id", "terminal_1", "--", "-x"},
		{"action", "write-chars", "--pane-id", "terminal_2", "--", ":goto 4"},
		{"action", "send-keys", "--pane-id", "terminal_2", "Esc", "Esc"},
		{"action", "focus-pane-id", "terminal_3"},
		{"action", "new-pane", "--stacked", "--name", "tour", "--", "zide", "tour-view", "x"},
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
