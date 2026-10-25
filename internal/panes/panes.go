// Package panes finds named panes in the caller's Zellij tab, and the tabs of
// the session.
package panes

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/relevatist/zide/internal/zellij"
)

type pane struct {
	ID       int    `json:"id"`
	IsPlugin bool   `json:"is_plugin"`
	Title    string `json:"title"`
	TabID    int    `json:"tab_id"`
	TabName  string `json:"tab_name"`
	Command  string `json:"terminal_command"`
	Cwd      string `json:"pane_cwd"`
}

// Pane is a terminal pane in the caller's tab.
type Pane struct {
	ID      string // terminal_N
	Command string // command the pane was started with; empty for a shell pane
}

// Find returns the ID (terminal_N) of the terminal pane titled name in the
// tab of the pane named by $ZELLIJ_PANE_ID.
func Find(r zellij.Runner, name string) (string, error) {
	p, err := Lookup(r, name)
	return p.ID, err
}

// Lookup is Find, but also returns the pane's command.
func Lookup(r zellij.Runner, name string) (Pane, error) {
	self := os.Getenv("ZELLIJ_PANE_ID")
	if self == "" {
		return Pane{}, errors.New("must run inside a zide workstream")
	}
	selfID, err := strconv.Atoi(self)
	if err != nil {
		return Pane{}, fmt.Errorf("bad ZELLIJ_PANE_ID %q", self)
	}
	terms, err := terminals(r)
	if err != nil {
		return Pane{}, err
	}
	tab := -1
	for _, p := range terms {
		if p.ID == selfID {
			tab = p.TabID
		}
	}
	if tab < 0 {
		return Pane{}, fmt.Errorf("pane %d not found; must run inside a zide workstream", selfID)
	}
	for _, p := range terms {
		if p.TabID == tab && p.Title == name {
			return Pane{fmt.Sprintf("terminal_%d", p.ID), p.Command}, nil
		}
	}
	return Pane{}, fmt.Errorf("no %q pane in this tab", name)
}

// Tab is a tab of the session.
type Tab struct {
	ID   int
	Name string
	Dir  string // working directory of the tab's editor pane; empty without one
}

// Tabs returns the tabs of the session that have a terminal pane.
func Tabs(r zellij.Runner) ([]Tab, error) {
	terms, err := terminals(r)
	if err != nil {
		return nil, err
	}
	var tabs []Tab
	for _, p := range terms {
		i := slices.IndexFunc(tabs, func(t Tab) bool { return t.ID == p.TabID })
		if i < 0 {
			tabs = append(tabs, Tab{ID: p.TabID, Name: p.TabName})
			i = len(tabs) - 1
		}
		if p.Title == "editor" && tabs[i].Dir == "" {
			tabs[i].Dir = p.Cwd
		}
	}
	return tabs, nil
}

func terminals(r zellij.Runner) ([]pane, error) {
	out, err := zellij.ListPanes(r)
	if err != nil {
		return nil, err
	}
	var all []pane
	if err := json.Unmarshal(out, &all); err != nil {
		return nil, fmt.Errorf("parse list-panes: %w", err)
	}
	return slices.DeleteFunc(all, func(p pane) bool { return p.IsPlugin }), nil
}

// KindVar is the variable that a zide layout sets on a harness pane whose
// kind differs from the base name of its command.
const KindVar = "ZIDE_HARNESS_KIND"

// Marker returns the kind in an `env ZIDE_HARNESS_KIND=kind cmd...` pane command.
func Marker(cmd string) (string, bool) {
	kind, _, ok := unwrap(cmd)
	return kind, ok
}

// Unwrap returns cmd without a leading `env ZIDE_HARNESS_KIND=kind`.
func Unwrap(cmd string) string {
	if _, rest, ok := unwrap(cmd); ok {
		return rest
	}
	return cmd
}

// unwrap splits on single spaces: list-panes joins the argv with spaces and
// no quotes, and the kind is one word.
func unwrap(cmd string) (kind, rest string, ok bool) {
	env, rest, _ := strings.Cut(cmd, " ")
	kv, rest, _ := strings.Cut(rest, " ")
	kind, ok = strings.CutPrefix(kv, KindVar+"=")
	if filepath.Base(env) != "env" || !ok || kind == "" || rest == "" {
		return "", "", false
	}
	return kind, rest, true
}
