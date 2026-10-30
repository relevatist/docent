package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/relevatist/docent/internal/helix"
	"github.com/relevatist/docent/internal/panes"
	"github.com/relevatist/docent/internal/tour"
	"github.com/relevatist/docent/internal/tui"
	"github.com/relevatist/docent/internal/zellij"
)

const tourUsage = `usage: docent tour <file|->

Open a tour pane stacked with the shell pane and walk through a change.
"-" reads the tour from standard input. Tour format:

  {"title": "...",
   "steps": [{"note": "## Overview ..."},
             {"file": "a.go", "lines": [1, 40], "note": "..."}]}

A step needs a file, a note, or both. "lines" needs a file. The note is
Markdown; mermaid code blocks show as text diagrams.

Keys in the tour pane: n next, p previous, o open in editor again,
s send step to harness, q close, j/k scroll, pgdown/space/ctrl+d and
pgup/b/ctrl+u page, g/G top/bottom.`

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
		_, _ = fmt.Fprintln(w, "docent tour:", err)
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
	if _, err := tour.Parse(data); err != nil {
		return err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp("", "docent-tour-*.json")
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

// openTourPane opens a tour pane running `docent tour-view args...`, stacked
// with the shell pane, and focuses it if the user still sees the shell's tab.
func openTourPane(r zellij.Runner, cwd string, args ...string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	shell, err := panes.Lookup(r, "shell")
	if err != nil {
		return err
	}
	cmd := append([]string{exe, "tour-view"}, args...)
	// --near-current-pane --no-focus places the pane next to $ZELLIJ_PANE_ID,
	// in its tab, instead of in the user's active tab, so point it at the
	// shell. focus-pane-id before new-pane does not move the client's focus,
	// and --tab-id conflicts with --near-current-pane (Zellij 0.45.1).
	self := os.Getenv("ZELLIJ_PANE_ID")
	if err := os.Setenv("ZELLIJ_PANE_ID", strings.TrimPrefix(shell.ID, "terminal_")); err != nil {
		return err
	}
	id, err := zellij.NewPane(r, append([]string{"--stacked", "--near-current-pane", "--no-focus", "--name", "tour", "--close-on-exit", "--cwd", cwd, "--"}, cmd...)...)
	if serr := os.Setenv("ZELLIJ_PANE_ID", self); serr != nil {
		return errors.Join(err, serr)
	}
	if err != nil || id == "" {
		return err
	}
	// Do not pull the user away from another tab. The pane is open, so an
	// unknown active tab only skips the focus.
	if active, err := panes.ActiveTab(r); err != nil || active != shell.Tab {
		return nil
	}
	return zellij.FocusPane(r, id)
}

func runTourView(args []string) int {
	items, title, err := loadTourView(args)
	if err == nil {
		err = tui.Run(newModel(zellij.Exec{}, items), title)
	}
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "docent tour-view:", err)
		return 1
	}
	return 0
}

// loadTourView reads `tour-view <file>` (and removes the file), and builds
// the steps.
func loadTourView(args []string) ([]tour.Item, string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, "", err
	}
	if len(args) != 1 {
		return nil, "", errors.New("usage: docent tour-view <file>")
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
	return tour.Build(t, cwd), t.Title, nil
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
