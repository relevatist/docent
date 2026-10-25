// Package tour parses change tours and holds the step model shown by the tour
// pane.
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

type Tour struct {
	Title string `json:"title,omitempty"`
	Steps []Step `json:"steps"`
}

// Step is one stop of a tour. Without Lines it covers the whole file. Without
// File it is a note only.
type Step struct {
	File  string `json:"file,omitempty"`
	Lines []int  `json:"lines,omitempty"`
	Note  string `json:"note,omitempty"`
}

// Parse reads and validates a tour. Unknown fields, such as the old "base",
// are ignored.
func Parse(data []byte) (Tour, error) {
	var t Tour
	if err := json.Unmarshal(data, &t); err != nil {
		return t, fmt.Errorf("invalid tour JSON: %w", err)
	}
	if len(t.Steps) == 0 {
		return t, errors.New("invalid tour: no steps")
	}
	for i, s := range t.Steps {
		switch {
		case s.Lines != nil && s.File == "":
			return t, fmt.Errorf("invalid tour: step %d: lines without file", i+1)
		case s.File == "" && s.Note == "":
			return t, fmt.Errorf("invalid tour: step %d: no file or note", i+1)
		case s.Lines != nil && (len(s.Lines) != 2 || s.Lines[0] < 1 || s.Lines[1] < s.Lines[0]):
			return t, fmt.Errorf("invalid tour: step %d: lines must be [start, end] with 1 <= start <= end", i+1)
		}
	}
	return t, nil
}

// Ref is the step as a reference: file, file:N, file:N-M, or "" without a file.
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

// Item is a step resolved against the working tree.
type Item struct {
	Step
	Path    string // absolute path of Step.File
	Invalid string // reason the step cannot be shown
	Editor  bool   // the editor follows this step
}

// Line is the editor line for the step, or 0 for the whole file.
func (it Item) Line() int {
	if it.Lines == nil {
		return 0
	}
	return it.Lines[0]
}

// Build resolves each step against the files in cwd.
func Build(t Tour, cwd string) []Item {
	var items []Item
	for _, s := range t.Steps {
		if s.File == "" {
			items = append(items, Item{Step: s})
			continue
		}
		items = append(items, item(s, absPath(cwd, s.File)))
	}
	return items
}

func item(s Step, path string) Item {
	it := Item{Step: s, Path: path}
	if strings.ContainsFunc(s.File, unicode.IsControl) {
		it.Invalid = fmt.Sprintf("invalid: file name %q contains control characters", s.File)
		return it
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		it.Invalid = "invalid: " + s.File + " not found"
		return it
	}
	if err != nil {
		it.Invalid = "invalid: " + err.Error()
		return it
	}
	n := 0
	for range strings.Lines(string(data)) {
		n++
	}
	if s.Lines != nil && s.Lines[0] > n {
		it.Invalid = fmt.Sprintf("invalid: %s outside %s (%d lines)", s.Ref(), s.File, n)
		return it
	}
	it.Editor = true
	return it
}

func absPath(cwd, file string) string {
	if filepath.IsAbs(file) {
		return filepath.Clean(file)
	}
	return filepath.Join(cwd, file)
}
