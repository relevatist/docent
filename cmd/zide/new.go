package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/relevatist/zide/internal/config"
	"github.com/relevatist/zide/internal/panes"
	"github.com/relevatist/zide/internal/recent"
	"github.com/relevatist/zide/internal/tui"
	"github.com/relevatist/zide/internal/zellij"
)

type pickerFunc func(tui.PickerInput) (tui.Picked, bool, error)

func runNew(args []string) int {
	code := newCmd(zellij.Exec{}, args, os.Stderr, tui.PickWorkstream)
	if isTerminal(os.Stdin) {
		return pause(code, os.Stdin, os.Stderr)
	}
	return code
}

// pause waits for enter after a non-zero exit, so that the error stays on
// screen in a floating pane that closes on exit.
func pause(code int, r io.Reader, w io.Writer) int {
	if code != 0 {
		_, _ = fmt.Fprint(w, "press enter to close")
		_, _ = bufio.NewReader(r).ReadString('\n')
	}
	return code
}

func newCmd(r zellij.Runner, args []string, w io.Writer, pick pickerFunc) int {
	fs := flag.NewFlagSet("new", flag.ContinueOnError)
	fs.SetOutput(w)
	var usageErr error
	fs.Usage = func() {
		_, usageErr = fmt.Fprintln(w, "Usage: zide new\n\nPick a directory and a harness, and open a workstream tab for them.\nRun it inside Zellij, for example from a key bind.")
	}
	if err := fs.Parse(args); err != nil {
		return exitParse(err, usageErr)
	}
	if fs.NArg() > 0 {
		fs.Usage()
		return 2
	}
	if err := newWorkstream(r, w, pick); err != nil {
		_, _ = fmt.Fprintln(w, "zide new:", err)
		return 1
	}
	return 0
}

func newWorkstream(r zellij.Runner, w io.Writer, pick pickerFunc) error {
	if os.Getenv("ZELLIJ") == "" {
		return errors.New("must run inside Zellij")
	}
	c, err := config.Load()
	if err != nil {
		return err
	}
	dirs, err := recent.Load()
	warning := ""
	if err != nil {
		warning = "recent directories: " + err.Error()
	}
	tabs, err := panes.Tabs(r)
	if err != nil {
		return err
	}
	harness, kind := currentHarness(r, c)
	p, ok, err := pick(tui.PickerInput{
		Recent:  dirs,
		Roots:   c.Roots,
		Harness: harness,
		Warning: warning,
		Aliases: slices.Collect(maps.Keys(c.Harnesses)),
		IsOpen: func(dir string) (string, bool) {
			t, ok := openTab(tabs, dir)
			return t.Name, ok
		},
	})
	if err != nil || !ok {
		return err
	}
	name := ""
	switch t, _ := openTab(tabs, p.Dir); p.Action {
	case tui.Switch:
		_, err := r.Run("action", "go-to-tab-by-id", strconv.Itoa(t.ID))
		return err
	case tui.Another:
		name = freeName(tabs, filepath.Base(p.Dir))
	}
	if p.Harness != harness {
		kind = ""
	}
	return launchDir(r, attach, w, p.Dir, name, p.Harness, kind)
}

// currentHarness returns the harness of the caller's tab: an alias name when
// the pane runs an alias command, else the command and its kind marker, if
// any. Without one, it returns the config harness.
func currentHarness(r zellij.Runner, c config.Config) (harness, kind string) {
	p, err := panes.Lookup(r, "harness")
	cmd := panes.Unwrap(p.Command)
	if err != nil || cmd == "" {
		return c.Harness, ""
	}
	for _, name := range slices.Sorted(maps.Keys(c.Harnesses)) {
		if strings.Join(strings.Fields(c.Harnesses[name].Command), " ") == cmd {
			return name, ""
		}
	}
	kind, _ = panes.Marker(p.Command)
	return cmd, kind
}

// openTab returns the tab whose editor pane has dir as its working directory.
func openTab(tabs []panes.Tab, dir string) (panes.Tab, bool) {
	for _, t := range tabs {
		if t.Dir != "" && samePath(t.Dir, dir) {
			return t, true
		}
	}
	return panes.Tab{}, false
}

// freeName returns base-N for the first N from 2 that no tab uses.
func freeName(tabs []panes.Tab, base string) string {
	for n := 2; ; n++ {
		name := base + "-" + strconv.Itoa(n)
		if !slices.ContainsFunc(tabs, func(t panes.Tab) bool { return t.Name == name }) {
			return name
		}
	}
}
