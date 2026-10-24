package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/relevatist/zide/internal/helix"
	"github.com/relevatist/zide/internal/panes"
	"github.com/relevatist/zide/internal/zellij"
)

func runReload(args []string) int {
	return reloadCmd(zellij.Exec{}, args, os.Stderr)
}

func reloadCmd(r zellij.Runner, args []string, w io.Writer) int {
	fs := flag.NewFlagSet("reload", flag.ContinueOnError)
	fs.SetOutput(w)
	quiet := fs.Bool("quiet", false, "exit 0 silently outside a zide workstream")
	var usageErr error
	fs.Usage = func() {
		_, usageErr = fmt.Fprintln(w, `Usage: zide reload [--quiet]

Reload all editor buffers in this tab from disk.

WARNING: unsaved changes in all buffers are discarded.`)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return exitParse(err, usageErr)
	}
	if fs.NArg() > 0 {
		fs.Usage()
		return 2
	}
	pane, err := panes.Find(r, "editor")
	if err != nil {
		if *quiet {
			return 0
		}
		_, _ = fmt.Fprintln(w, "zide reload:", err)
		return 1
	}
	if err := helix.Reload(r, pane); err != nil {
		_, _ = fmt.Fprintln(w, "zide reload:", err)
		return 1
	}
	return 0
}
