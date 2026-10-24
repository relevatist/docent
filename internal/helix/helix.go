// Package helix drives a Helix editor in a Zellij pane by injecting keys.
package helix

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/relevatist/zide/internal/zellij"
)

// Open shows path in the editor in pane, with the cursor on line when line > 0.
// A relative path resolves against the current working directory.
func Open(r zellij.Runner, pane, path string, line int) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if err := CheckPath(abs); err != nil {
		return err
	}
	// Single quotes: Helix expands %{...} inside double quotes, and doubles a quote to escape it.
	cmds := []string{":open '" + strings.ReplaceAll(abs, "'", "''") + "'"}
	if line > 0 {
		cmds = append(cmds, fmt.Sprintf(":goto %d", line))
	}
	return run(r, pane, cmds...)
}

// CheckPath rejects a path with control characters: typed into the Helix
// command line, they act as keys (Enter, Esc, ...).
func CheckPath(path string) error {
	if strings.ContainsFunc(path, unicode.IsControl) {
		return fmt.Errorf("path %q contains control characters", path)
	}
	return nil
}

// Reload reloads all buffers from disk. Unsaved changes in all buffers are discarded.
func Reload(r zellij.Runner, pane string) error {
	return run(r, pane, ":reload-all")
}

// run leaves insert mode and closes any prompt (Esc Esc), so cmds are never
// typed into the buffer, then runs each command.
func run(r zellij.Runner, pane string, cmds ...string) error {
	if err := zellij.SendKeys(r, pane, "Esc", "Esc"); err != nil {
		return err
	}
	for _, c := range cmds {
		if err := zellij.WriteChars(r, pane, c); err != nil {
			return err
		}
		if err := zellij.SendKeys(r, pane, "Enter"); err != nil {
			return err
		}
	}
	return nil
}
