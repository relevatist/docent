package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/relevatist/zide/internal/helix"
	"github.com/relevatist/zide/internal/panes"
	"github.com/relevatist/zide/internal/tour"
	"github.com/relevatist/zide/internal/tui"
	"github.com/relevatist/zide/internal/zellij"
)

const tourUsage = `usage: zide tour <file|->

Open a tour pane stacked with the shell pane and walk through a change.
"-" reads the tour from standard input. Tour format:

  {"title": "...", "base": "<git rev, default HEAD>",
   "steps": [{"file": "a.go", "lines": [1, 40], "note": "..."}]}

Keys in the tour pane: n next, p previous, o open in editor again,
s send step to harness, q close.`

func runTour(args []string) int {
	return tourCmd(zellij.Exec{}, args, os.Stdin, os.Stderr)
}

func tourCmd(r zellij.Runner, args []string, stdin io.Reader, w io.Writer) int {
	if len(args) != 1 || args[0] == "-h" || args[0] == "--help" {
		_, err := fmt.Fprintln(w, tourUsage)
		switch {
		case len(args) != 1:
			return 2
		case err != nil:
			return 1
		}
		return 0
	}
	if err := startTour(r, args[0], stdin); err != nil {
		_, _ = fmt.Fprintln(w, "zide tour:", err)
		return 1
	}
	return 0
}

func startTour(r zellij.Runner, src string, stdin io.Reader) error {
	var data []byte
	var err error
	if src == "-" {
		data, err = io.ReadAll(stdin)
	} else {
		data, err = os.ReadFile(src)
	}
	if err != nil {
		return err
	}
	t, err := tour.Parse(data)
	if err != nil {
		return err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	// Check git and the base here, so errors print in the caller's terminal.
	if _, err := tour.Changes(cwd, t.Base); err != nil {
		return err
	}
	tmp, err := os.CreateTemp("", "zide-tour-*.json")
	if err != nil {
		return err
	}
	_, err = tmp.Write(data)
	if err = errors.Join(err, tmp.Close()); err == nil {
		err = openTourPane(r, cwd, tmp.Name())
	}
	if err != nil {
		err = errors.Join(err, os.Remove(tmp.Name()))
	}
	return err
}

// openTourPane opens a tour pane running `zide tour-view args...`, stacked
// with the shell pane, and focuses it.
func openTourPane(r zellij.Runner, cwd string, args ...string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	shell, err := panes.Find(r, "shell")
	if err != nil {
		return err
	}
	cmd := append([]string{exe, "tour-view"}, args...)
	// --near-current-pane places the pane next to $ZELLIJ_PANE_ID instead of
	// the user's focus, so point it at the shell. focus-pane-id before
	// new-pane does not move the client's focus (Zellij 0.45.1).
	self := os.Getenv("ZELLIJ_PANE_ID")
	if err := os.Setenv("ZELLIJ_PANE_ID", strings.TrimPrefix(shell, "terminal_")); err != nil {
		return err
	}
	id, err := zellij.NewPane(r, append([]string{"--stacked", "--near-current-pane", "--name", "tour", "--close-on-exit", "--cwd", cwd, "--"}, cmd...)...)
	if serr := os.Setenv("ZELLIJ_PANE_ID", self); serr != nil {
		return errors.Join(err, serr)
	}
	if err != nil || id == "" {
		return err
	}
	return zellij.FocusPane(r, id)
}

func runTourView(args []string) int {
	items, title, err := loadTourView(args)
	if err == nil {
		err = tui.Run(newModel(zellij.Exec{}, items), title)
	}
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "zide tour-view:", err)
		return 1
	}
	return 0
}

// loadTourView reads `tour-view <file>` (and removes the file) or
// `tour-view --diff [git diff args]`, and builds the steps.
func loadTourView(args []string) ([]tour.Item, string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, "", err
	}
	if len(args) > 0 && args[0] == "--diff" {
		hunks, err := tour.GitDiff(cwd, args[1:]...)
		if err != nil {
			return nil, "", err
		}
		if len(hunks) == 0 {
			return nil, "", errors.New("no changes")
		}
		return tour.Build(tour.DiffTour(hunks, cwd), hunks, cwd), "git diff", nil
	}
	if len(args) != 1 {
		return nil, "", errors.New("usage: zide tour-view <file> | --diff [git diff args]")
	}
	data, err := os.ReadFile(args[0])
	if err != nil {
		return nil, "", err
	}
	// The pane has nowhere to report a failure, and a leftover temp file does
	// not affect the tour.
	_ = os.Remove(args[0])
	t, err := tour.Parse(data)
	if err != nil {
		return nil, "", err
	}
	hunks, err := tour.Changes(cwd, t.Base)
	if err != nil {
		return nil, "", err
	}
	return tour.Build(t, hunks, cwd), t.Title, nil
}

// newModel wires the step model to the editor and harness panes of this tab.
func newModel(r zellij.Runner, items []tour.Item) *tour.Model {
	editor, findErr := panes.Find(r, "editor")
	return &tour.Model{
		Items: items,
		Open: func(path string, line int) error {
			if findErr != nil {
				return findErr
			}
			return helix.Open(r, editor, path, line)
		},
		Send: func(ref string) error { return doSend(r, nil, []string{"ref", ref}) },
	}
}
