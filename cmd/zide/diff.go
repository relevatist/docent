package main

import (
	"fmt"
	"io"
	"os"

	"github.com/relevatist/zide/internal/tour"
	"github.com/relevatist/zide/internal/zellij"
)

const diffUsage = `usage: zide diff [git diff args]

Open a tour pane with one step per hunk of "git diff [args]".`

func runDiff(args []string) int {
	return diffCmd(zellij.Exec{}, args, os.Stdout, os.Stderr)
}

func diffCmd(r zellij.Runner, args []string, stdout, stderr io.Writer) int {
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		if _, err := fmt.Fprintln(stdout, diffUsage); err != nil {
			_, _ = fmt.Fprintln(stderr, "zide diff:", err)
			return 1
		}
		return 0
	}
	cwd, err := os.Getwd()
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "zide diff:", err)
		return 1
	}
	hunks, err := tour.GitDiff(cwd, args...)
	if err == nil && len(hunks) == 0 {
		if _, err := fmt.Fprintln(stdout, "no changes"); err != nil {
			_, _ = fmt.Fprintln(stderr, "zide diff:", err)
			return 1
		}
		return 0
	}
	if err == nil {
		err = openTourPane(r, cwd, append([]string{"--diff"}, args...)...)
	}
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "zide diff:", err)
		return 1
	}
	return 0
}
