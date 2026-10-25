package tui

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"iter"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"unicode"

	"github.com/relevatist/zide/internal/recent"
)

// PickerInput is what the workstream picker starts with.
type PickerInput struct {
	Recent  []string // absolute directories, most recent first
	Roots   []string // searched for git repos while the picker shows
	Harness string   // the harness to use unless the user picks another
	Aliases []string // config harness alias names, in any order
	Warning string   // shown in the header
	// IsOpen returns the name of a tab that has dir open.
	IsOpen func(dir string) (tab string, ok bool)
}

// Action is what to do with the picked directory.
type Action int

const (
	Open    Action = iota // open a new tab
	Switch                // go to the tab that has the directory open
	Another               // open another tab for a directory that is open
)

// Picked is the user's choice.
type Picked struct {
	Dir     string // absolute
	Harness string // command or alias name
	Action  Action
}

// fzfFunc runs fzf with args and the lines of in as input. It returns the
// output lines and the exit code.
type fzfFunc func(args []string, in iter.Seq[string]) ([]string, int, error)

// PickWorkstream shows the picker in fzf. ok is false when the user cancels.
func PickWorkstream(in PickerInput) (Picked, bool, error) {
	if _, err := exec.LookPath("fzf"); err != nil {
		return Picked{}, false, errors.New("fzf is not on PATH")
	}
	return pickWorkstream(in, runFzf)
}

func pickWorkstream(in PickerInput, fzf fzfFunc) (Picked, bool, error) {
	home, _ := os.UserHomeDir()
	if home != "" {
		home = filepath.Clean(home)
	}
	ctx, cancel := context.WithCancel(context.Background())
	header := "enter: open with " + in.Harness + "  alt-enter: choose harness  esc: cancel"
	if in.Warning != "" {
		header += "\nwarning: " + in.Warning
	}
	out, ok, err := step(fzf, dirLines(ctx, in, home), "dir", header, "--scheme=path", "--print-query", "--expect=alt-enter")
	cancel()
	if !ok {
		return Picked{}, false, err
	}
	query, key, sel := at(out, 0), at(out, 1), at(out, 2)
	dir, isDir := typedDir(query, home)
	if !isDir {
		if query == "" && sel == "" {
			return Picked{}, false, nil
		}
		dir, isDir = typedDir(cmp.Or(sel, query), home)
	}
	if !isDir {
		return Picked{}, false, fmt.Errorf("no directory for %q", cmp.Or(sel, query))
	}
	p := Picked{Dir: dir, Harness: in.Harness}

	if key == "alt-enter" {
		out, ok, err := step(fzf, slices.Values(harnesses(in)), "harness", "enter: choose, or use the typed command when nothing matches  esc: cancel", "--print-query")
		if !ok {
			return Picked{}, false, err
		}
		// fzf matches each word on its own, so typed text with spaces is a command.
		query := strings.TrimSpace(at(out, 0))
		p.Harness = strings.TrimSpace(cmp.Or(at(out, 1), query))
		if strings.ContainsFunc(query, unicode.IsSpace) {
			p.Harness = query
		}
		if p.Harness == "" {
			return Picked{}, false, errors.New("harness is empty")
		}
	}

	if in.IsOpen == nil {
		return p, true, nil
	}
	tab, isOpen := in.IsOpen(dir)
	if !isOpen {
		return p, true, nil
	}
	switchTo, another := "switch to tab "+safe(cmp.Or(tab, "?")), "open another tab"
	out, ok, err = step(fzf, slices.Values([]string{switchTo, another}), "", tilde(dir, home)+" is open", "--no-input")
	if !ok {
		return Picked{}, false, err
	}
	p.Action = Switch
	if at(out, 0) == another {
		p.Action = Another
	}
	return p, true, nil
}

// step runs one fzf screen. ok is false when the user cancels or fzf fails.
func step(fzf fzfFunc, in iter.Seq[string], prompt, header string, args ...string) (out []string, ok bool, err error) {
	args = append(args, "--layout=reverse", "--prompt="+prompt+"> ", "--header="+safeHeader(header))
	out, code, err := fzf(args, in)
	switch {
	case err != nil:
		return nil, false, err
	case code == 130:
		return nil, false, nil
	case code != 0 && code != 1: // 1 is no match
		return nil, false, fmt.Errorf("fzf exited with code %d", code)
	}
	return out, true, nil
}

// safeHeader is safe(s) with its newlines kept.
func safeHeader(s string) string {
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = safe(lines[i])
	}
	return strings.Join(lines, "\n")
}

func at(lines []string, i int) string {
	if i < len(lines) {
		return lines[i]
	}
	return ""
}

// dirLines yields the recent directories, the repos under the roots, and
// each directory under home, each once, as absolute paths written as ~/...
// It drops paths that
// have control characters, as fzf output could not give them back.
func dirLines(ctx context.Context, in PickerInput, home string) iter.Seq[string] {
	return func(yield func(string) bool) {
		seqs := []iter.Seq[string]{slices.Values(recent.Candidates(in.Recent)), recent.Repos(ctx, in.Roots)}
		if home != "" {
			seqs = append(seqs, recent.Dirs(ctx, home))
		}
		seen := map[string]bool{}
		for _, seq := range seqs {
			for d := range seq {
				d, err := filepath.Abs(d)
				if err != nil || seen[d] || safe(d) != d || strings.Contains(d, "\t") {
					continue
				}
				seen[d] = true
				if !yield(tilde(d, home)) {
					return
				}
			}
		}
	}
}

// runFzf runs fzf with the lines of in on its stdin. It stops reading in when
// fzf exits.
func runFzf(args []string, in iter.Seq[string]) ([]string, int, error) {
	cmd := exec.Command("fzf", args...)
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, os.Stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, 0, err
	}
	if err := cmd.Start(); err != nil {
		return nil, 0, err
	}
	go func() {
		defer func() { _ = stdin.Close() }()
		for l := range in {
			if _, err := io.WriteString(stdin, l+"\n"); err != nil {
				return
			}
		}
	}()
	err = cmd.Wait()
	code := 0
	if ee, ok := errors.AsType[*exec.ExitError](err); ok {
		code, err = ee.ExitCode(), nil
	}
	return strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n"), code, err
}

// harnesses returns the harness candidates in order, each once.
func harnesses(in PickerInput) []string {
	var hs []string
	for _, h := range slices.Concat([]string{in.Harness}, slices.Sorted(slices.Values(in.Aliases)), []string{"claude", "omp", "codex"}) {
		if h != "" && !slices.Contains(hs, h) {
			hs = append(hs, h)
		}
	}
	return hs
}

// typedDir returns q as an absolute path when it is a path to a directory.
// Only text that starts with /, ~/, ./ or ../, or is ~, . or .., is a path.
// A leading ~ means home, and . means the current directory. With no home, ~
// is not a path.
func typedDir(q, home string) (string, bool) {
	switch {
	case q == "~" || strings.HasPrefix(q, "~/"):
		if home == "" {
			return "", false
		}
		q = filepath.Join(home, q[1:])
	case q == "." || q == ".." || strings.HasPrefix(q, "/") || strings.HasPrefix(q, "./") || strings.HasPrefix(q, "../"):
	default:
		return "", false
	}
	q, err := filepath.Abs(q)
	if err != nil {
		return "", false
	}
	fi, err := os.Stat(q)
	return q, err == nil && fi.IsDir()
}

// tilde writes a path under home as ~/...
func tilde(p, home string) string {
	if p == home {
		return "~"
	}
	if rest, ok := strings.CutPrefix(p, home+string(filepath.Separator)); ok && home != "" {
		return "~/" + rest
	}
	return p
}
