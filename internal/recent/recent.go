// Package recent keeps the list of recently launched workstream directories.
package recent

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"

	"github.com/BurntSushi/toml"
)

const limit = 20

type state struct {
	Dirs []string `toml:"dirs"`
}

// Path returns $XDG_STATE_HOME/zide/recent.toml, or ~/.local/state/zide/recent.toml.
func Path() (string, error) {
	dir := os.Getenv("XDG_STATE_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(dir, "zide", "recent.toml"), nil
}

// Load returns the recent directories, most recent first. A missing file
// gives an empty list.
func Load() ([]string, error) {
	p, err := Path()
	if err != nil {
		return nil, err
	}
	var s state
	if _, err := toml.DecodeFile(p, &s); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	return s.Dirs, nil
}

// Add moves dir to the top of the list and keeps at most 20 entries. Two
// Adds at the same time can lose one entry.
func Add(dir string) error {
	p, err := Path()
	if err != nil {
		return err
	}
	if err := add(p, dir); err != nil {
		return fmt.Errorf("save recent directories to %s: %w", p, err)
	}
	return nil
}

func add(p, dir string) (err error) {
	dirs, err := Load()
	if err != nil {
		return err
	}
	dirs = slices.DeleteFunc(dirs, func(d string) bool { return d == dir })
	dirs = append([]string{dir}, dirs[:min(len(dirs), limit-1)]...)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(p), ".recent-*.toml")
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, os.Remove(f.Name()))
		}
	}()
	if err := errors.Join(toml.NewEncoder(f).Encode(state{dirs}), f.Close()); err != nil {
		return err
	}
	return os.Rename(f.Name(), p)
}
