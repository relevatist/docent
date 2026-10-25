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
	"slices"
	"strings"
	"syscall"

	"github.com/relevatist/zide/assets"
)

const (
	zellijBegin = "    // zide: begin (managed by `zide install`)\n"
	zellijEnd   = "    // zide: end\n"
)

var (
	systemZellijDir = "/etc/zellij"
	goos            = runtime.GOOS
	writeString     = (*os.File).WriteString
	errByHand       = errors.New(`zide cannot find where to add the key; add the shared_except "locked" node from assets/zellij/keys.kdl to the keybinds block by hand`)
	errMarkers      = errors.New("more than one zide marker, or markers out of order")
	errUnpaired     = errors.New("a `// zide: begin` line without a `// zide: end` line (or the reverse)")
	keybindsLine    = regexp.MustCompile(`^keybinds([\s{]|$)`)
	keybindsName    = regexp.MustCompile(`(^|[;{}])\s*"?keybinds"?([\s;{}]|$)`)
	altWBind        = regexp.MustCompile(`\bbind(\s+"[^"]*")*\s+"\s*(?i:alt)\s+w\s*"`)
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
	if goos == "darwin" {
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
	cfg, bom := strings.CutPrefix(cfg, "\ufeff")
	out, err := editZellijCode(cfg, force)
	if bom && err == nil {
		out = "\ufeff" + out
	}
	return out, err
}

func editZellijCode(cfg string, force bool) (string, error) {
	block := zellijBegin + zellijNode() + zellijEnd
	code, depth, open, ok := kdlCode(cfg)
	if !ok {
		return "", errByHand
	}
	lines := strings.SplitAfter(cfg, "\n")
	begin, end, hidden := -1, -1, false
	for i, l := range lines {
		l = strings.TrimSpace(l)
		isBegin := l == strings.TrimSpace(zellijBegin)
		if !isBegin && l != strings.TrimSpace(zellijEnd) {
			continue
		}
		if i > 0 && open[i-1] {
			hidden = true
			continue
		}
		switch {
		case isBegin && begin < 0:
			begin = i
		case !isBegin && end < 0:
			end = i
		default:
			return "", errMarkers
		}
	}
	onlyHidden := begin < 0 && end < 0 && hidden
	unpaired := (begin < 0) != (end < 0)
	switch {
	case onlyHidden:
		return "", errByHand
	case unpaired:
		return "", errUnpaired
	case end < begin:
		return "", errMarkers
	}
	keybinds := -1
	for i, c := range code {
		if begin >= 0 && i >= begin && i <= end {
			continue
		}
		if altWBind.MatchString(c) {
			if isAssetBind(lines[i:]) {
				return "", fmt.Errorf(`"Alt w" is already bound by the zide snippet, added by hand (%s); remove it so that zide can manage it`, strings.TrimSpace(lines[i]))
			}
			return "", fmt.Errorf(`"Alt w" is already bound (%s); remove that bind first`, strings.TrimSpace(lines[i]))
		}
		if keybinds < 0 && keybindsName.MatchString(c) {
			keybinds = i
		}
	}
	join := func(ls []string) string { return strings.Join(ls, "") }
	// eol returns s with the line ending of l.
	eol := func(s, l string) string {
		if strings.HasSuffix(l, "\r\n") {
			return strings.ReplaceAll(s, "\n", "\r\n")
		}
		return s
	}
	switch {
	case begin >= 0:
		if strings.ReplaceAll(join(lines[begin:end+1]), "\r", "") == block {
			return cfg, nil
		}
		if !force {
			return "", errors.New("the zide key node has changes; use --force to overwrite")
		}
		return join(lines[:begin]) + eol(block, lines[begin]) + join(lines[end+1:]), nil
	case keybinds < 0:
		nl := eol("\n", lines[0])
		if cfg != "" && !strings.HasSuffix(cfg, "\n") {
			cfg += nl
		}
		return cfg + eol("keybinds {\n"+block+"}\n", nl), nil
	}
	kb := strings.TrimSpace(code[keybinds])
	topLevel := depth[keybinds] == 0 && !open[keybinds]
	endsLine := strings.HasSuffix(lines[keybinds], "\n")
	opensBlock := keybindsLine.MatchString(kb) && strings.HasSuffix(kb, "{")
	if !topLevel || !endsLine || !opensBlock {
		return "", errByHand
	}
	return join(lines[:keybinds+1]) + eol(block, lines[keybinds]) + join(lines[keybinds+1:]), nil
}

// isAssetBind reports whether ls starts with the bind node of
// assets/zellij/keys.kdl.
func isAssetBind(ls []string) bool {
	want := strings.Split(strings.TrimSuffix(zellijNode(), "\n"), "\n")
	want = want[1 : len(want)-1]
	return len(ls) >= len(want) && slices.EqualFunc(ls[:len(want)], want, func(a, b string) bool {
		return strings.TrimSpace(a) == strings.TrimSpace(b)
	})
}

// kdlCode returns the lines of cfg with comments blanked out, the brace
// depth at the start of each line, and whether each line ends inside a
// comment or a string. ok is false for KDL that it does not
// handle: raw strings, `/-` comments, and an unclosed comment or string.
func kdlCode(cfg string) (lines []string, depth []int, open []bool, ok bool) {
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
			open = append(open, comment > 0 || inString)
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
			return nil, nil, nil, false
		case c == '{':
			d++
		case c == '}':
			d--
		}
		b.WriteByte(c)
	}
	if comment > 0 || inString {
		return nil, nil, nil, false
	}
	return strings.Split(b.String(), "\n"), depth, append(open, false), true
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
	if z.data, err = editZellijConfig(z.old, force); errors.Is(err, errMarkers) || errors.Is(err, errUnpaired) {
		return fmt.Sprintf("%v; fix %s by hand", err, path)
	} else if err != nil {
		return path + ": " + err.Error()
	}
	if z.exists && z.data == z.old {
		return ""
	}
	if fi, err := os.Stat(path); err == nil {
		if st, ok := fi.Sys().(*syscall.Stat_t); ok && st.Nlink > 1 {
			return path + " has other hard links; zide would split them. Add the key by hand, or replace the hard link with a symlink."
		}
	}
	if err := checkWritable(filepath.Dir(path)); err != nil {
		return path + ": " + err.Error()
	}
	if _, err := exec.LookPath("zellij"); err != nil {
		return "" // check names the missing command
	}
	if err := checkZellijConfig(z.data); err != nil {
		return path + ": " + err.Error()
	}
	return ""
}

// checkWritable makes and removes a temp file in dir, or in its nearest
// parent that exists.
func checkWritable(dir string) error {
	for {
		if _, err := os.Stat(dir); err == nil || filepath.Dir(dir) == dir {
			break
		}
		dir = filepath.Dir(dir)
	}
	f, err := os.CreateTemp(dir, ".zide-check-*")
	if err != nil {
		return err
	}
	return errors.Join(f.Close(), os.Remove(f.Name()))
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
		// The link makes a complete backup appear at once, and fails if one
		// exists.
		tmp, err := writeTemp(dir, z.old, mode)
		if err != nil {
			return err
		}
		bak := z.path + ".zide.bak"
		msg := bak
		err = os.Link(tmp, bak)
		_ = os.Remove(tmp)
		if errors.Is(err, fs.ErrExist) {
			msg, err = bak+" kept (it holds the config from before zide)", nil
		}
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintln(out, msg); err != nil {
			return err
		}
	} else if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := writeTemp(dir, z.data, mode)
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp) }()
	if err := os.Rename(tmp, z.path); err != nil {
		return err
	}
	_, err = fmt.Fprintln(out, z.path)
	return err
}

// writeTemp writes data with mode to a new temp file in dir, syncs it, and
// returns its name. It removes the file if it fails.
func writeTemp(dir, data string, mode fs.FileMode) (string, error) {
	f, err := os.CreateTemp(dir, ".config.kdl.zide-*")
	if err != nil {
		return "", err
	}
	_, err = writeString(f, data)
	if err == nil {
		err = f.Chmod(mode)
	}
	if err == nil {
		err = f.Sync()
	}
	if err := errors.Join(err, f.Close()); err != nil {
		_ = os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}
