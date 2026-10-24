package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/relevatist/zide/internal/config"
	"github.com/relevatist/zide/internal/launch"
	"github.com/relevatist/zide/internal/zellij"
)

func runLaunch(args []string) int {
	fs := flag.NewFlagSet("zide", flag.ContinueOnError)
	harness := fs.String("harness", "", "harness command (overrides config)")
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
	err := launchDir(fs.Arg(0), *name, *harness)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "zide:", err)
		return 1
	}
	return 0
}

func launchDir(arg, name, harness string) error {
	dir, err := filepath.Abs(arg)
	if err != nil {
		return err
	}
	c, err := config.Load()
	if err != nil {
		return err
	}
	return launch.Run(zellij.Exec{}, attach, os.Stderr, dir, name, c.WithHarness(harness).Harness, os.Getenv("ZELLIJ") != "")
}

func attach(dir string, args ...string) error {
	cmd := exec.Command("zellij", args...)
	cmd.Dir = dir
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}
