// Package config loads the zide user config.
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
}

// Harness is a harness command and its kind, the harness that the command
// runs (for example `omp` for a sandbox wrapper around omp).
type Harness struct {
	Command string `toml:"command"`
	Kind    string `toml:"kind"`
}

// Path returns $XDG_CONFIG_HOME/zide/config.toml, or ~/.config/zide/config.toml.
func Path() (string, error) {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "zide", "config.toml"), nil
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
		c.Harnesses[name] = h
	}
	for i, r := range c.Roots {
		if r == "~" || strings.HasPrefix(r, "~/") {
			home, err := os.UserHomeDir()
			if err != nil {
				return c, err
			}
			c.Roots[i] = filepath.Join(home, r[1:])
		}
	}
	return c, nil
}

// Resolve returns the alias named s, or s as a plain command whose kind is
// the base name of its first word.
func (c Config) Resolve(s string) (Harness, error) {
	if h, ok := c.Harnesses[s]; ok {
		return h, nil
	}
	if strings.TrimSpace(s) == "" {
		return Harness{}, errors.New("harness command is empty")
	}
	return Harness{s, kindOf(s)}, nil
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
