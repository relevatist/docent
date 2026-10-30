// Command docent joins an editor, a coding harness, and a shell in one Zellij tab.
package main

import (
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
	"text/tabwriter"
)

type command struct {
	run  func(args []string) int
	help string
}

var commands = map[string]command{
	"open":      {runOpen, "open <file>[:line] [--focus]\tshow file in the editor"},
	"reload":    {runReload, "reload [--quiet]\treload all editor buffers (discards unsaved changes)"},
	"send":      {runSend, "send ref <file>[:a[-b]] | text\tpaste into the harness input"},
	"tour":      {runTour, "tour <file|->\tstart a guided tour of a change"},
	"tour-view": {runTourView, "tour-view <file>\trun the tour pane (internal)"},
	"new":       {runNew, "new\tpick a directory and open a workstream tab (inside Zellij)"},
	"install":   {runInstall, "install [claude|codex|omp|zellij]... [--force]\tinstall docent for coding harnesses"},
}

func usage(w io.Writer) error {
	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)
	if _, err := fmt.Fprintln(tw, "Usage:\n  docent [--name name] [--harness cmd|alias] [--layout file] [dir]\topen a workstream tab"); err != nil {
		return err
	}
	for _, name := range slices.Sorted(maps.Keys(commands)) {
		if _, err := fmt.Fprintln(tw, "  docent", commands[name].help); err != nil {
			return err
		}
	}
	return tw.Flush()
}

func main() {
	os.Exit(dispatch(os.Args[1:]))
}

func dispatch(args []string) int {
	if len(args) > 0 {
		switch args[0] {
		case "-h", "-help", "--help", "help":
			if usage(os.Stdout) != nil {
				return 1
			}
			return 0
		}
		if c, ok := commands[args[0]]; ok {
			return c.run(args[1:])
		}
	}
	return runLaunch(args)
}
