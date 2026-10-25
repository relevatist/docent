package main

import (
	"cmp"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/relevatist/docent/internal/config"
	"github.com/relevatist/docent/internal/launch"
	"github.com/relevatist/docent/internal/recent"
	"github.com/relevatist/docent/internal/zellij"
)

func runLaunch(args []string) int {
	fs := flag.NewFlagSet("docent", flag.ContinueOnError)
	harness := fs.String("harness", "", "harness command or alias (overrides config)")
	name := fs.String("name", "", "tab or session name (default: directory basename)")
	var usageErr error
	fs.Usage = func() { usageErr = usage(os.Stderr) }
	if err := fs.Parse(args); err != nil {
		return exitParse(err, usageErr)
	}
	if fs.NArg() > 1 {
		fs.Usage()
		return 2
	}
	err := launchDir(zellij.Exec{}, attach, os.Stderr, fs.Arg(0), *name, *harness, "")
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "docent:", err)
		return 1
	}
	return 0
}

// launchDir opens the workstream for arg and records it as recent: before
// attach outside Zellij, as attach blocks until the user detaches, and after
// the new tab inside Zellij. A non-empty kind replaces the harness kind.
func launchDir(r zellij.Runner, attach launch.Attach, w io.Writer, arg, name, harness, kind string) error {
	dir, err := filepath.Abs(arg)
	if err != nil {
		return err
	}
	c, err := config.Load()
	if err != nil {
		return err
	}
	h, err := c.Resolve(c.WithHarness(harness).Harness)
	if err != nil {
		return err
	}
	h.Kind = cmp.Or(kind, h.Kind)
	record := func() error {
		if err := recent.Add(dir); err != nil {
			_, err = fmt.Fprintln(w, "docent: warning:", err)
			return err
		}
		return nil
	}
	inZellij := os.Getenv("ZELLIJ") != ""
	recordAndAttach := func(dir string, args ...string) error {
		if err := record(); err != nil {
			return err
		}
		return attach(dir, args...)
	}
	if err := launch.Run(r, recordAndAttach, w, dir, name, h, inZellij); err != nil || !inZellij {
		return err
	}
	return record()
}

func attach(dir string, args ...string) error {
	cmd := exec.Command("zellij", args...)
	cmd.Dir = dir
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}
