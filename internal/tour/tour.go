// Package tour parses change tours, maps them to git diff hunks, and holds
// the step model shown by the tour pane.
package tour

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// NotCovered is the title of the final step that lists hunks no step covers.
const NotCovered = "Not covered by this tour"

type Tour struct {
	Title string `json:"title,omitempty"`
	Base  string `json:"base,omitempty"`
	Steps []Step `json:"steps"`
}

// Step is one stop of a tour. Without Lines it covers the whole file.
type Step struct {
	File  string `json:"file"`
	Lines []int  `json:"lines,omitempty"`
	Note  string `json:"note,omitempty"`
}

// Parse reads and validates a tour.
func Parse(data []byte) (Tour, error) {
	var t Tour
	if err := json.Unmarshal(data, &t); err != nil {
		return t, fmt.Errorf("invalid tour JSON: %w", err)
	}
	if len(t.Steps) == 0 {
		return t, errors.New("invalid tour: no steps")
	}
	if strings.HasPrefix(t.Base, "-") {
		return t, fmt.Errorf("invalid tour: base %q must be a git revision, not an option", t.Base)
	}
	for i, s := range t.Steps {
		switch {
		case s.File == "":
			return t, fmt.Errorf("invalid tour: step %d: no file", i+1)
		case s.Lines != nil && (len(s.Lines) != 2 || s.Lines[0] < 1 || s.Lines[1] < s.Lines[0]):
			return t, fmt.Errorf("invalid tour: step %d: lines must be [start, end] with 1 <= start <= end", i+1)
		}
	}
	return t, nil
}

// Ref is the step as a reference: file, file:N, or file:N-M.
func (s Step) Ref() string {
	if s.Lines == nil {
		return s.File
	}
	return s.File + ":" + lineRange(s.Lines[0], s.Lines[1])
}

func lineRange(a, b int) string {
	if a == b {
		return fmt.Sprint(a)
	}
	return fmt.Sprintf("%d-%d", a, b)
}

func (s Step) covers(h Hunk) bool {
	a, b := h.Span()
	return s.Lines == nil || (s.Lines[0] <= b && s.Lines[1] >= a)
}

// Line kinds.
const (
	Context byte = ' '
	Added   byte = '+'
	Removed byte = '-'
	Header  byte = '@'
)

// Line is one line of a step view. Num is the working-tree line number, or 0.
type Line struct {
	Kind byte
	Num  int
	Text string
}

// Item is a step resolved against the working tree.
type Item struct {
	Step
	Path    string // absolute path of Step.File
	Title   string // set only for the NotCovered step
	Invalid string // reason the step cannot be shown
	Editor  bool   // the editor follows this step
	Body    []Line
}

func (it Item) Heading() string {
	if it.Title != "" {
		return it.Title
	}
	return it.Ref()
}

// Line is the editor line for the step, or 0 for the whole file.
func (it Item) Line() int {
	if it.Lines == nil {
		return 0
	}
	return it.Lines[0]
}

// Build resolves each step against the files in cwd and the hunks (absolute
// paths), and appends a NotCovered step when some hunk is not covered.
func Build(t Tour, hunks []Hunk, cwd string) []Item {
	covered := make([]bool, len(hunks))
	var items []Item
	for _, s := range t.Steps {
		path := absPath(cwd, s.File)
		var own []Hunk
		for i, h := range hunks {
			if h.File == path {
				own = append(own, h)
				covered[i] = covered[i] || s.covers(h)
			}
		}
		items = append(items, item(s, path, own))
	}
	var rest []Line
	for i, h := range hunks {
		if covered[i] {
			continue
		}
		rest = append(rest, Line{Kind: Header, Text: h.ref(cwd)})
		for _, l := range h.Lines {
			rest = append(rest, Line{Kind: l[0], Text: l[1:]})
		}
	}
	if rest != nil {
		items = append(items, Item{Title: NotCovered, Body: rest})
	}
	return items
}

func item(s Step, path string, hunks []Hunk) Item {
	it := Item{Step: s, Path: path}
	if strings.ContainsFunc(s.File, unicode.IsControl) {
		it.Invalid = fmt.Sprintf("invalid: file name %q contains control characters", s.File)
		return it
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) && len(hunks) > 0 && hunks[0].Deleted {
		for _, l := range hunks[0].Lines {
			it.Body = append(it.Body, Line{Kind: Removed, Text: l[1:]})
		}
		return it
	}
	if errors.Is(err, fs.ErrNotExist) {
		it.Invalid = "invalid: " + s.File + " not found"
		return it
	}
	if err != nil {
		it.Invalid = "invalid: " + err.Error()
		return it
	}
	var text []string
	for l := range strings.Lines(string(data)) {
		text = append(text, strings.TrimRight(l, "\r\n"))
	}
	n := len(text)
	start, end := 1, n
	if s.Lines != nil {
		start, end = s.Lines[0], min(s.Lines[1], n)
		if start > n {
			it.Invalid = fmt.Sprintf("invalid: %s outside %s (%d lines)", s.Ref(), s.File, n)
			return it
		}
	}
	it.Editor = true

	added := map[int]bool{}
	removed := map[int][]string{} // removed lines, keyed by the line they precede
	for _, h := range hunks {
		at := h.NewStart
		if h.NewCount == 0 {
			at++
		}
		for _, l := range h.Lines {
			switch l[0] {
			case '+':
				added[at] = true
				at++
			case '-':
				removed[at] = append(removed[at], l[1:])
			default:
				at++
			}
		}
	}
	emit := func(at int) {
		for _, r := range removed[at] {
			it.Body = append(it.Body, Line{Kind: Removed, Text: r})
		}
	}
	for i := start; i <= end; i++ {
		emit(i)
		kind := Context
		if added[i] {
			kind = Added
		}
		it.Body = append(it.Body, Line{Kind: kind, Num: i, Text: text[i-1]})
	}
	if end == n {
		emit(n + 1)
	}
	return it
}

// DiffTour has one step per hunk, in order, with no notes.
func DiffTour(hunks []Hunk, cwd string) Tour {
	var t Tour
	for _, h := range hunks {
		a, b := h.Span()
		t.Steps = append(t.Steps, Step{File: relPath(cwd, h.File), Lines: []int{a, b}})
	}
	return t
}

func absPath(cwd, file string) string {
	if filepath.IsAbs(file) {
		return filepath.Clean(file)
	}
	return filepath.Join(cwd, file)
}

func relPath(cwd, file string) string {
	if r, err := filepath.Rel(cwd, file); err == nil {
		return r
	}
	return file
}
