// Package panes finds named panes in the caller's Zellij tab.
package panes

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/relevatist/zide/internal/zellij"
)

type pane struct {
	ID       int    `json:"id"`
	IsPlugin bool   `json:"is_plugin"`
	Title    string `json:"title"`
	TabID    int    `json:"tab_id"`
	Command  string `json:"terminal_command"`
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
	out, err := zellij.ListPanes(r)
	if err != nil {
		return Pane{}, err
	}
	var all []pane
	if err := json.Unmarshal(out, &all); err != nil {
		return Pane{}, fmt.Errorf("parse list-panes: %w", err)
	}
	var terms []pane
	for _, p := range all {
		if !p.IsPlugin {
			terms = append(terms, p)
		}
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
