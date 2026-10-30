// Package zellij runs zellij commands.
package zellij

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// Runner runs zellij with args and returns its stdout.
type Runner interface {
	Run(args ...string) ([]byte, error)
}

// PipeTimeout is how long Exec waits for `zellij pipe`. With no plugin to
// answer, the pipe can wait forever.
const PipeTimeout = 2 * time.Second

// Exec runs the real zellij binary.
type Exec struct{}

func (Exec) Run(args ...string) ([]byte, error) {
	ctx := context.Background()
	if len(args) > 0 && args[0] == "pipe" {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, PipeTimeout)
		defer cancel()
	}
	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "zellij", args...)
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return out, fmt.Errorf("zellij %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}

// Fake records each call and returns canned output keyed by the space-joined args.
type Fake struct {
	Calls [][]string
	Out   map[string]string
	Errs  map[string]error
}

func (f *Fake) Run(args ...string) ([]byte, error) {
	f.Calls = append(f.Calls, args)
	key := strings.Join(args, " ")
	return []byte(f.Out[key]), f.Errs[key]
}

func action(r Runner, args ...string) error {
	_, err := r.Run(append([]string{"action"}, args...)...)
	return err
}

func ListPanes(r Runner) ([]byte, error) {
	return r.Run("action", "list-panes", "--all", "--json")
}

func CurrentTabInfo(r Runner) ([]byte, error) {
	return r.Run("action", "current-tab-info", "--json")
}

// Write writes b to pane as raw input. It sends b to the docent plugin first,
// which writes without the scroll reset of `zellij action write`. If the
// plugin does not write, it falls back to `zellij action write`.
func Write(r Runner, pane string, b []byte) error {
	if pipe(r, "docent-write", pane, string(b)) {
		return nil
	}
	_, err := r.Run(WriteArgs(pane, b)...)
	return err
}

// Paste writes text to pane as one bracketed paste, through the docent
// plugin like Write. The fallback is `zellij action paste`.
func Paste(r Runner, pane, text string) error {
	if pipe(r, "docent-paste", pane, text) {
		return nil
	}
	return action(r, "paste", "--pane-id", pane, "--", text)
}

// now is time.Now, replaced in tests.
var now = time.Now

// pipe sends payload to the docent plugin as the pipe message name and
// reports whether the plugin wrote it. The plugin takes only UTF-8. It
// answers "ok" when it writes, and "late" when it gets the message after the
// deadline, because then docent may have fallen back already. In a detach
// race two plugin instances can both answer, so "ok" anywhere in the output
// means that the plugin wrote.
func pipe(r Runner, name, pane, payload string) bool {
	if !utf8.ValidString(payload) {
		return false
	}
	out, err := r.Run(PipeArgs(name, pane, payload, now().Add(PipeTimeout-500*time.Millisecond))...)
	return err == nil && strings.Contains(string(out), "ok")
}

// PipeArgs returns the zellij args that send payload to the plugin. The
// plugin does not write after deadline.
func PipeArgs(name, pane, payload string, deadline time.Time) []string {
	args := fmt.Sprintf("pane=%s,deadline=%d", pane, deadline.UnixMilli())
	return []string{"pipe", "--name", name, "--args", args, "--", payload}
}

// WriteArgs returns the zellij args of the Write fallback.
func WriteArgs(pane string, b []byte) []string {
	args := []string{"action", "write", "--pane-id", pane}
	for _, c := range b {
		args = append(args, strconv.Itoa(int(c)))
	}
	return args
}

func FocusPane(r Runner, pane string) error {
	return action(r, "focus-pane-id", pane)
}

// NewPane runs `zellij action new-pane args...` and returns the new pane ID.
func NewPane(r Runner, args ...string) (string, error) {
	out, err := r.Run(append([]string{"action", "new-pane"}, args...)...)
	return strings.TrimSpace(string(out)), err
}
