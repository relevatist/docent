// Package config loads the docent user config.
package config

import (
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Harness     string             `toml:"harness"`
	FocusOnSend bool               `toml:"focus_on_send"`
	Roots       []string           `toml:"roots"`
	Harnesses   map[string]Harness `toml:"harnesses"`
	Layout      string             `toml:"layout"` // absolute path; empty for the built-in layout
}

// Harness is a harness command and its kind, the harness that the command
// runs (for example `omp` for a sandbox wrapper around omp).
type Harness struct {
	Command string `toml:"command"`
	Kind    string `toml:"kind"`
}

// Path returns $XDG_CONFIG_HOME/docent/config.toml, or ~/.config/docent/config.toml.
func Path() (string, error) {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "docent", "config.toml"), nil
}

// Load reads the config file. A missing file gives the defaults.
func Load() (Config, error) {
	c := Config{Harness: "claude", FocusOnSend: true}
	p, err := Path()
	if err != nil {
		return c, err
	}
	if _, err := toml.DecodeFile(p, &c); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return c, err
	}
	for name, h := range c.Harnesses {
		if strings.TrimSpace(h.Command) == "" {
			return c, fmt.Errorf("harness alias %q: command is empty", name)
		}
		if strings.ContainsFunc(h.Kind, unicode.IsSpace) {
			return c, fmt.Errorf("harness alias %q: kind %q must be one word", name, h.Kind)
		}
		h.Kind = cmp.Or(h.Kind, kindOf(h.Command))
		if h.Command, err = expandWords(h.Command); err != nil {
			return c, err
		}
		c.Harnesses[name] = h
	}
	if c.Layout != "" {
		if c.Layout, err = ExpandHome(c.Layout); err != nil {
			return c, err
		}
		if !filepath.IsAbs(c.Layout) {
			c.Layout = filepath.Join(filepath.Dir(p), c.Layout)
		}
	}
	for i := range c.Roots {
		if c.Roots[i], err = ExpandHome(c.Roots[i]); err != nil {
			return c, err
		}
	}
	return c, nil
}

// ExpandHome replaces a leading ~ or ~/ in p with the home directory.
func ExpandHome(p string) (string, error) {
	if p != "~" && !strings.HasPrefix(p, "~/") {
		return p, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return p, err
	}
	return filepath.Join(home, p[1:]), nil
}

// expandWords applies ExpandHome to each word of cmd.
func expandWords(cmd string) (string, error) {
	words := strings.Fields(cmd)
	for i := range words {
		var err error
		if words[i], err = ExpandHome(words[i]); err != nil {
			return cmd, err
		}
	}
	return strings.Join(words, " "), nil
}

// Resolve returns the alias named s, or s as a plain command whose kind is
// the base name of its first word. It expands a leading ~ in each word of a
// plain command.
func (c Config) Resolve(s string) (Harness, error) {
	if h, ok := c.Harnesses[s]; ok {
		return h, nil
	}
	if strings.TrimSpace(s) == "" {
		return Harness{}, errors.New("harness command is empty")
	}
	cmd, err := expandWords(s)
	return Harness{cmd, kindOf(s)}, err
}

func kindOf(cmd string) string {
	return filepath.Base(strings.Fields(cmd)[0])
}

// WithHarness returns c with Harness set to h, unless h is empty.
func (c Config) WithHarness(h string) Config {
	if h != "" {
		c.Harness = h
	}
	return c
}
