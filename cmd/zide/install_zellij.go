package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/relevatist/zide/assets"
)

const (
	zellijBegin = "    // zide: begin (managed by `zide install`)\n"
	zellijEnd   = "    // zide: end\n"
)

var (
	systemZellijDir = "/etc/zellij"
	errByHand       = errors.New(`zide cannot find where to add the key; add the shared_except "locked" node from assets/zellij/keys.kdl to the keybinds block by hand`)
	keybindsLine    = regexp.MustCompile(`^keybinds([\s{]|$)`)
	altWBind        = regexp.MustCompile(`\bbind\b.*"Alt w"`)
)

// zellijNode is the shared_except node inside the keybinds block of
// assets/zellij/keys.kdl.
func zellijNode() string {
	return strings.TrimSuffix(strings.TrimPrefix(assets.ZellijKeys, "keybinds {\n"), "}\n")
}

// zellijConfigPath returns the config file that Zellij loads, in Zellij's
// lookup order, or the file to create when there is none.
func zellijConfigPath(home string) (string, error) {
	if f := os.Getenv("ZELLIJ_CONFIG_FILE"); f != "" {
		return f, nil
	}
	if d := os.Getenv("ZELLIJ_CONFIG_DIR"); d != "" {
		return filepath.Join(d, "config.kdl"), nil
	}
	user := filepath.Join(home, ".config/zellij")
	for _, d := range []string{user, platformZellijDir(home), systemZellijDir} {
		if _, err := os.Stat(d); err != nil {
			continue
		}
		if d == systemZellijDir {
			return "", fmt.Errorf("zellij uses %s; copy its config.kdl to %s first", d, user)
		}
		return filepath.Join(d, "config.kdl"), nil
	}
	return filepath.Join(user, "config.kdl"), nil
}

// platformZellijDir is the config dir of the Rust directories crate.
func platformZellijDir(home string) string {
	if runtime.GOOS == "darwin" {
		return filepath.Join(home, "Library/Application Support/org.Zellij-Contributors.Zellij")
	}
	if x := os.Getenv("XDG_CONFIG_HOME"); filepath.IsAbs(x) {
		return filepath.Join(x, "zellij")
	}
	return filepath.Join(home, ".config/zellij")
}

// editZellijConfig returns cfg with the marked zide node in its first
// top-level keybinds block.
func editZellijConfig(cfg string, force bool) (string, error) {
	block := zellijBegin + zellijNode() + zellijEnd
	code, depth, ok := kdlCode(cfg)
	if !ok {
		return "", errByHand
	}
	lines := strings.SplitAfter(cfg, "\n")
	begin, end := -1, -1
	for i, l := range lines {
		switch strings.TrimSpace(l) {
		case strings.TrimSpace(zellijBegin):
			if begin >= 0 {
				return "", errByHand
			}
			begin = i
		case strings.TrimSpace(zellijEnd):
			if end >= 0 {
				return "", errByHand
			}
			end = i
		}
	}
	if (begin < 0) != (end < 0) || end < begin {
		return "", errByHand
	}
	keybinds := -1
	for i, c := range code {
		if begin >= 0 && i >= begin && i <= end {
			continue
		}
		if altWBind.MatchString(c) {
			return "", fmt.Errorf(`"Alt w" is already bound (%s); remove that bind first`, strings.TrimSpace(lines[i]))
		}
		if keybinds < 0 && depth[i] == 0 && keybindsLine.MatchString(strings.TrimSpace(c)) {
			keybinds = i
		}
	}
	join := func(ls []string) string { return strings.Join(ls, "") }
	switch {
	case begin >= 0:
		if join(lines[begin:end+1]) == block {
			return cfg, nil
		}
		if !force {
			return "", errors.New("the zide key node has changes; use --force to overwrite")
		}
		return join(lines[:begin]) + block + join(lines[end+1:]), nil
	case keybinds < 0:
		if cfg != "" && !strings.HasSuffix(cfg, "\n") {
			cfg += "\n"
		}
		return cfg + "keybinds {\n" + block + "}\n", nil
	case !strings.HasSuffix(strings.TrimSpace(code[keybinds]), "{") || !strings.HasSuffix(lines[keybinds], "\n"):
		return "", errByHand
	}
	return join(lines[:keybinds+1]) + block + join(lines[keybinds+1:]), nil
}

// kdlCode returns the lines of cfg with comments blanked out, and the brace
// depth at the start of each line. ok is false for KDL that it does not
// handle: raw strings, `/-` comments, and an unclosed comment or string.
func kdlCode(cfg string) (lines []string, depth []int, ok bool) {
	var b strings.Builder
	depth = []int{0}
	d, comment, inString := 0, 0, false
	for i := 0; i < len(cfg); i++ {
		c, next := cfg[i], byte(0)
		if i+1 < len(cfg) {
			next = cfg[i+1]
		}
		switch {
		case c == '\n':
			depth = append(depth, d)
		case comment > 0:
			if c == '*' && next == '/' || c == '/' && next == '*' {
				if c == '*' {
					comment--
				} else {
					comment++
				}
				i++
				b.WriteString("  ")
				continue
			}
			c = ' '
		case inString:
			if c == '\\' && next != '\n' && next != 0 {
				b.WriteByte(c)
				i++
				c = next
			} else if c == '"' {
				inString = false
			}
		case c == '"':
			inString = true
		case c == '/' && next == '/':
			for i+1 < len(cfg) && cfg[i+1] != '\n' {
				i++
			}
			c = ' '
		case c == '/' && next == '*':
			comment = 1
			i++
			b.WriteString("  ")
			continue
		case c == '/' && next == '-', c == 'r' && (next == '"' || next == '#'), c == '#' && next == '"':
			return nil, nil, false
		case c == '{':
			d++
		case c == '}':
			d--
		}
		b.WriteByte(c)
	}
	if comment > 0 || inString {
		return nil, nil, false
	}
	return strings.Split(b.String(), "\n"), depth, true
}

// checkZellijConfig runs `zellij setup --check` on cfg in a temp file.
func checkZellijConfig(cfg string) error {
	f, err := os.CreateTemp("", "zide-zellij-*.kdl")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(f.Name()) }()
	if _, err := f.WriteString(cfg); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if out, err := exec.Command("zellij", "--config", f.Name(), "setup", "--check").CombinedOutput(); err != nil {
		return fmt.Errorf("zellij setup --check fails on the new config: %v\n%s", err, out)
	}
	return nil
}

type zellijTarget struct {
	home, path, old, data string
	exists                bool
}

// check finds the config and makes the new content. It returns a problem,
// or "".
func (z *zellijTarget) check(force bool) string {
	path, err := zellijConfigPath(z.home)
	if err != nil {
		return err.Error()
	}
	if fi, err := os.Lstat(path); err == nil && fi.Mode()&fs.ModeSymlink != 0 {
		if path, err = filepath.EvalSymlinks(path); err != nil {
			return err.Error()
		}
	}
	z.path = path
	old, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err.Error()
	}
	z.old, z.exists = string(old), err == nil
	if z.data, err = editZellijConfig(z.old, force); err != nil {
		return path + ": " + err.Error()
	}
	if z.exists && z.data == z.old {
		return ""
	}
	if _, err := exec.LookPath("zellij"); err != nil {
		return "" // check names the missing command
	}
	if err := checkZellijConfig(z.data); err != nil {
		return path + ": " + err.Error()
	}
	return ""
}

func (z *zellijTarget) apply(out io.Writer) error {
	if z.exists && z.data == z.old {
		_, err := fmt.Fprintln(out, z.path, "unchanged")
		return err
	}
	dir := filepath.Dir(z.path)
	mode := fs.FileMode(0o644)
	if z.exists {
		fi, err := os.Stat(z.path)
		if err != nil {
			return err
		}
		mode = fi.Mode().Perm()
		if err := os.WriteFile(z.path+".zide.bak", []byte(z.old), mode); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(out, z.path+".zide.bak"); err != nil {
			return err
		}
	} else if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".config.kdl.zide-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(f.Name()) }()
	_, err = f.WriteString(z.data)
	if err == nil {
		err = f.Chmod(mode)
	}
	if err := errors.Join(err, f.Close()); err != nil {
		return err
	}
	if err := os.Rename(f.Name(), z.path); err != nil {
		return err
	}
	_, err = fmt.Fprintln(out, z.path)
	return err
}
