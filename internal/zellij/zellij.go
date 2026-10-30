// Package zellij runs zellij commands.
package zellij

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// Runner runs zellij with args and returns its stdout.
type Runner interface {
	Run(args ...string) ([]byte, error)
}

// Exec runs the real zellij binary.
type Exec struct{}

func (Exec) Run(args ...string) ([]byte, error) {
	var stderr bytes.Buffer
	cmd := exec.Command("zellij", args...)
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

// Paste pastes text into pane as a bracketed paste.
func Paste(r Runner, pane, text string) error {
	return action(r, "paste", "--pane-id", pane, "--", text)
}

func WriteChars(r Runner, pane, chars string) error {
	return action(r, "write-chars", "--pane-id", pane, "--", chars)
}

func SendKeys(r Runner, pane string, keys ...string) error {
	if len(keys) == 0 {
		return errors.New("send-keys: no keys")
	}
	return action(r, append([]string{"send-keys", "--pane-id", pane}, keys...)...)
}

func FocusPane(r Runner, pane string) error {
	return action(r, "focus-pane-id", pane)
}

// NewPane runs `zellij action new-pane args...` and returns the new pane ID.
func NewPane(r Runner, args ...string) (string, error) {
	out, err := r.Run(append([]string{"action", "new-pane"}, args...)...)
	return strings.TrimSpace(string(out)), err
}
