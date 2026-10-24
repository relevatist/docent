// Package config loads the zide user config.
package config

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Harness     string `toml:"harness"`
	FocusOnSend bool   `toml:"focus_on_send"`
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
	return c, nil
}

// WithHarness returns c with Harness set to h, unless h is empty.
func (c Config) WithHarness(h string) Config {
	if h != "" {
		c.Harness = h
	}
	return c
}
