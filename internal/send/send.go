// Package send pastes references and text into the harness pane.
package send

import (
	"cmp"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/relevatist/zide/internal/panes"
	"github.com/relevatist/zide/internal/zellij"
)

// FormatRef formats ref (file[:start[-end]]) for the harness command, with one
// trailing space. The file is kept as given.
func FormatRef(harness, ref string) (string, error) {
	file, lines, err := parseRef(ref)
	if err != nil {
		return "", err
	}
	if lines != "" {
		file += ":" + lines
	}
	switch filepath.Base(strings.Fields(harness + " ")[0]) {
	case "claude", "omp":
		file = "@" + file
	}
	return file + " ", nil
}

func parseRef(ref string) (file, lines string, err error) {
	file, rng, ok := strings.CutLast(ref, ":")
	if !ok || rng == "" || rng[0] < '0' || rng[0] > '9' {
		file, rng = ref, ""
	}
	if file == "" {
		return "", "", fmt.Errorf("bad reference %q: no file", ref)
	}
	if rng == "" {
		return file, "", nil
	}
	a, b, isRange := strings.Cut(rng, "-")
	if !isRange {
		b = a
	}
	start, err1 := strconv.Atoi(a)
	end, err2 := strconv.Atoi(b)
	switch {
	case err1 != nil || err2 != nil:
		return "", "", fmt.Errorf("bad line range %q in %q", rng, ref)
	case start < 1 || end < start:
		return "", "", fmt.Errorf("bad line range %q: want 1 <= start <= end", rng)
	case start == end:
		return file, strconv.Itoa(start), nil
	}
	return file, fmt.Sprintf("%d-%d", start, end), nil
}

// Paste pastes text into the harness pane of the caller's tab, then focuses
// that pane if focus is set. Empty text sends nothing. It never sends Enter.
func Paste(r zellij.Runner, text string, focus bool) error {
	if text == "" {
		return nil
	}
	pane, err := panes.Find(r, "harness")
	if err != nil {
		return err
	}
	return paste(r, pane, text, focus)
}

// PasteRef is Paste for FormatRef(harness, ref). harness is the command the
// harness pane was started with, so it follows the tab; fallback is used
// when that pane has no command.
func PasteRef(r zellij.Runner, ref, fallback string, focus bool) error {
	if _, _, err := parseRef(ref); err != nil {
		return err
	}
	pane, err := panes.Lookup(r, "harness")
	if err != nil {
		return err
	}
	text, err := FormatRef(cmp.Or(pane.Command, fallback), ref)
	if err != nil {
		return err
	}
	return paste(r, pane.ID, text, focus)
}

// paste removes ESC from text, so the text cannot end the bracketed paste
// early and submit what follows.
func paste(r zellij.Runner, pane, text string, focus bool) error {
	text = strings.ReplaceAll(text, "\x1b", "")
	if err := zellij.Paste(r, pane, text); err != nil {
		return err
	}
	if focus {
		return zellij.FocusPane(r, pane)
	}
	return nil
}
