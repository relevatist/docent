package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/relevatist/docent/internal/helix"
	"github.com/relevatist/docent/internal/panes"
	"github.com/relevatist/docent/internal/zellij"
)

func runOpen(args []string) int {
	return openCmd(zellij.Exec{}, args, os.Stderr)
}

func openCmd(r zellij.Runner, args []string, w io.Writer) int {
	fs := flag.NewFlagSet("open", flag.ContinueOnError)
	fs.SetOutput(w)
	focus := fs.Bool("focus", false, "move focus to the editor")
	var usageErr error
	fs.Usage = func() {
		_, usageErr = fmt.Fprintln(w, "Usage: docent open [--focus] <file>[:line]\n\nShow file in the editor of this tab, with the cursor on line.")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return exitParse(err, usageErr)
	}
	// Allow flags after the file too.
	rest := fs.Args()
	if len(rest) > 0 {
		if err := fs.Parse(rest[1:]); err != nil {
			return exitParse(err, usageErr)
		}
	}
	if len(rest) == 0 || fs.NArg() > 0 {
		fs.Usage()
		return 2
	}
	path, line, err := parseTarget(rest[0])
	if err != nil {
		_, _ = fmt.Fprintln(w, "docent open:", err)
		return 2
	}
	if err := helix.CheckPath(path); err != nil {
		_, _ = fmt.Fprintln(w, "docent open:", err)
		return 2
	}
	if _, err := os.Stat(path); err != nil {
		_, _ = fmt.Fprintf(w, "docent open: %s: %v\n", path, errors.Unwrap(err))
		return 1
	}
	pane, err := panes.Find(r, "editor")
	if err == nil {
		err = helix.Open(r, pane, path, line)
	}
	if err == nil && *focus {
		err = zellij.FocusPane(r, pane)
	}
	if err != nil {
		_, _ = fmt.Fprintln(w, "docent open:", err)
		return 1
	}
	return 0
}

// parseTarget splits "file[:line]". A suffix that is not a number is part of the file name.
func parseTarget(s string) (string, int, error) {
	path, num, ok := strings.CutLast(s, ":")
	if !ok {
		return s, 0, nil
	}
	line, err := strconv.Atoi(num)
	if err != nil {
		return s, 0, nil
	}
	if line < 1 {
		return "", 0, fmt.Errorf("bad line %q", num)
	}
	return path, line, nil
}

// exitParse returns the exit code for a flag parse error. usageErr is the
// error from writing the usage, which fails an otherwise successful -h.
func exitParse(err, usageErr error) int {
	switch {
	case !errors.Is(err, flag.ErrHelp):
		return 2
	case usageErr != nil:
		return 1
	}
	return 0
}
