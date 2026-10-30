package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"syscall"

	"github.com/relevatist/docent/assets"
)

const (
	zellijBegin       = "    // docent: begin (managed by `docent install`)\n"
	zellijEnd         = "    // docent: end\n"
	zellijPluginBegin = "    // docent: plugin begin (managed by `docent install`)\n"
	zellijPluginEnd   = "    // docent: plugin end\n"
)

var (
	systemZellijDir = "/etc/zellij"
	goos            = runtime.GOOS
	writeString     = (*os.File).WriteString
	errMarkers      = errFix("more than one docent marker, or markers out of order")
	altWBind        = regexp.MustCompile(`\bbind(\s+"[^"]*")*\s+"\s*(?i:alt)\s+w\s*"`)
)

// errFix is a problem with the docent markers, which the user must fix by
// hand.
type errFix string

func (e errFix) Error() string { return string(e) }

// markedNode is the docent node in a top-level block of the Zellij config.
type markedNode struct {
	block      string // keybinds, load_plugins
	begin, end string // marker lines
	node       string // lines between the markers
	head       string // lines before the markers in a new block
	what       string // name of the node in messages
	byHand     string // what to add by hand
	// conflict, if set, returns an error when a line outside the markers
	// conflicts with the node. code is the line without comments, and lines
	// starts with it.
	conflict func(code string, lines []string) error
}

func keysNode() markedNode {
	return markedNode{
		block: "keybinds", begin: zellijBegin, end: zellijEnd, node: zellijNode(),
		what:   "key",
		byHand: `the shared_except "locked" node from assets/zellij/keys.kdl to the keybinds block`,
		conflict: func(code string, lines []string) error {
			if !altWBind.MatchString(code) {
				return nil
			}
			if isAssetBind(lines) {
				return fmt.Errorf(`"Alt w" is already bound by the docent snippet, added by hand (%s); remove it so that docent can manage it`, strings.TrimSpace(lines[0]))
			}
			return fmt.Errorf(`"Alt w" is already bound (%s); remove that bind first`, strings.TrimSpace(lines[0]))
		},
	}
}

// pluginNode loads the docent plugin at wasm.
func pluginNode(wasm string) markedNode {
	entry := kdlString("file:" + (&url.URL{Path: wasm}).EscapedPath())
	return markedNode{
		block: "load_plugins", begin: zellijPluginBegin, end: zellijPluginEnd,
		node:   "    " + entry + "\n",
		head:   "    \"zellij:link\"\n",
		what:   "plugin",
		byHand: entry + " to the load_plugins block",
	}
}

// kdlString quotes s as a KDL string.
func kdlString(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}

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

// editZellijConfig returns cfg with the marked docent key node in its first
// top-level keybinds block.
func editZellijConfig(cfg string, force bool) (string, error) {
	return editMarked(cfg, keysNode(), force)
}

// editMarked returns cfg with n in its first top-level n.block block.
func editMarked(cfg string, n markedNode, force bool) (string, error) {
	cfg, bom := strings.CutPrefix(cfg, "\ufeff")
	out, err := editMarkedCode(cfg, n, force)
	if bom && err == nil {
		out = "\ufeff" + out
	}
	return out, err
}

func editMarkedCode(cfg string, n markedNode, force bool) (string, error) {
	errByHand := fmt.Errorf("docent cannot find where to add the %s; add %s by hand", n.what, n.byHand)
	blockLine := regexp.MustCompile(`^` + n.block + `([\s{]|$)`)
	blockName := regexp.MustCompile(`(^|[;{}])\s*"?` + n.block + `"?([\s;{}]|$)`)
	block := n.begin + n.node + n.end
	code, depth, open, ok := kdlCode(cfg)
	if !ok {
		return "", errByHand
	}
	lines := strings.SplitAfter(cfg, "\n")
	begin, end, hidden := -1, -1, false
	for i, l := range lines {
		l = strings.TrimSpace(l)
		isBegin := l == strings.TrimSpace(n.begin)
		if !isBegin && l != strings.TrimSpace(n.end) {
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
		return "", errFix(fmt.Sprintf("a `%s` line without a `%s` line (or the reverse)", strings.TrimSpace(n.begin), strings.TrimSpace(n.end)))
	case end < begin:
		return "", errMarkers
	}
	at := -1
	for i, c := range code {
		if begin >= 0 && i >= begin && i <= end {
			continue
		}
		if n.conflict != nil {
			if err := n.conflict(c, lines[i:]); err != nil {
				return "", err
			}
		}
		if at < 0 && blockName.MatchString(c) {
			at = i
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
			return "", fmt.Errorf("the docent %s node has changes; use --force to overwrite", n.what)
		}
		return join(lines[:begin]) + eol(block, lines[begin]) + join(lines[end+1:]), nil
	case at < 0:
		nl := eol("\n", lines[0])
		if cfg != "" && !strings.HasSuffix(cfg, "\n") {
			cfg += nl
		}
		return cfg + eol(n.block+" {\n"+n.head+block+"}\n", nl), nil
	}
	head := strings.TrimSpace(code[at])
	topLevel := depth[at] == 0 && !open[at]
	endsLine := strings.HasSuffix(lines[at], "\n")
	opensBlock := blockLine.MatchString(head) && strings.HasSuffix(head, "{")
	if !topLevel || !endsLine || !opensBlock {
		return "", errByHand
	}
	return join(lines[:at+1]) + eol(block, lines[at]) + join(lines[at+1:]), nil
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
	f, err := os.CreateTemp("", "docent-zellij-*.kdl")
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
	home, wasm      string
	path, old, data string // config.kdl
	exists          bool
	perm            kdlFile
}

type kdlFile struct {
	path, old, data string
	exists          bool
}

// readLinked reads the file at path, or at the target of a symlink at path.
// It returns the path of the file that it read.
func readLinked(path string) (kdlFile, error) {
	if fi, err := os.Lstat(path); err == nil && fi.Mode()&fs.ModeSymlink != 0 {
		if path, err = filepath.EvalSymlinks(path); err != nil {
			return kdlFile{}, err
		}
	}
	old, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return kdlFile{}, err
	}
	return kdlFile{path: path, old: string(old), exists: err == nil}, nil
}

// check finds the config and the permission cache and makes their new
// content, and checks that the plugin file can be written. It returns the
// problems.
func (z *zellijTarget) check(force bool) []string {
	var problems []string
	if p := z.checkConfig(force); p != "" {
		problems = append(problems, p)
	}
	if p := z.checkPermissions(); p != "" {
		problems = append(problems, p)
	}
	if err := checkWritable(filepath.Dir(z.wasm)); err != nil {
		problems = append(problems, z.wasm+": "+err.Error())
	}
	return problems
}

func (z *zellijTarget) checkConfig(force bool) string {
	path, err := zellijConfigPath(z.home)
	if err != nil {
		return err.Error()
	}
	f, err := readLinked(path)
	if err != nil {
		return err.Error()
	}
	path, z.path, z.old, z.exists = f.path, f.path, f.old, f.exists
	z.data, err = editZellijConfig(z.old, force)
	if err == nil {
		z.data, err = editMarked(z.data, pluginNode(z.wasm), force)
	}
	if _, ok := errors.AsType[errFix](err); ok {
		return fmt.Sprintf("%v; fix %s by hand", err, path)
	} else if err != nil {
		return path + ": " + err.Error()
	}
	if z.exists && z.data == z.old {
		return ""
	}
	if fi, err := os.Stat(path); err == nil {
		if st, ok := fi.Sys().(*syscall.Stat_t); ok && st.Nlink > 1 {
			return path + " has other hard links; docent would split them. Add the key by hand, or replace the hard link with a symlink."
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

func (z *zellijTarget) checkPermissions() string {
	f, err := readLinked(filepath.Join(zellijCacheDir(z.home), "permissions.kdl"))
	if err != nil {
		return err.Error()
	}
	data, ok := editPermissions(f.old, z.wasm)
	if !ok {
		return f.path + ": docent cannot read this Zellij permission cache; fix or remove it"
	}
	f.data = data
	z.perm = f
	if f.exists && f.data == f.old {
		return ""
	}
	if err := checkWritable(filepath.Dir(f.path)); err != nil {
		return f.path + ": " + err.Error()
	}
	return ""
}

// zellijCacheDir is the cache dir of the Rust directories crate.
func zellijCacheDir(home string) string {
	if goos == "darwin" {
		return filepath.Join(home, "Library/Caches/org.Zellij-Contributors.Zellij")
	}
	if x := os.Getenv("XDG_CACHE_HOME"); filepath.IsAbs(x) {
		return filepath.Join(x, "zellij")
	}
	return filepath.Join(home, ".cache/zellij")
}

// zellijPermissions are the permissions that the plugin requests.
var zellijPermissions = []string{"WriteToStdin", "ReadCliPipes", "ReadApplicationState"}

var (
	permHeader = regexp.MustCompile(`^\S.* \{$`)
	permChild  = regexp.MustCompile(`^    \w+$`)
)

// editPermissions returns the Zellij permission cache with the entry for the
// plugin at key set to zellijPermissions. ok is false when cache is not in
// the form that Zellij writes.
func editPermissions(cache, key string) (out string, ok bool) {
	head := kdlString(key) + " {\n"
	entry := head + "    " + strings.Join(zellijPermissions, "\n    ") + "\n}\n"
	if cache != "" && !strings.HasSuffix(cache, "\n") {
		return "", false
	}
	lines := strings.SplitAfter(cache, "\n")
	lines = lines[:len(lines)-1]
	start, from, to := -1, -1, -1
	for i, l := range lines {
		switch l = strings.TrimSuffix(l, "\n"); {
		case start < 0 && permHeader.MatchString(l):
			start = i
		case start >= 0 && permChild.MatchString(l):
		case start >= 0 && l == "}":
			if from < 0 && lines[start] == head {
				from, to = start, i
			}
			start = -1
		default:
			return "", false
		}
	}
	if start >= 0 {
		return "", false
	}
	if from < 0 {
		return cache + entry, true
	}
	var got []string
	for _, l := range lines[from+1 : to] {
		got = append(got, strings.TrimSpace(l))
	}
	if slices.Equal(slices.Sorted(slices.Values(got)), slices.Sorted(slices.Values(zellijPermissions))) {
		return cache, true
	}
	return strings.Join(lines[:from], "") + entry + strings.Join(lines[to+1:], ""), true
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
	f, err := os.CreateTemp(dir, ".docent-check-*")
	if err != nil {
		return err
	}
	return errors.Join(f.Close(), os.Remove(f.Name()))
}

// apply always replaces the plugin file, like the Claude Code plugin.
func (z *zellijTarget) apply(out io.Writer) error {
	if err := replaceFile(z.wasm, string(assets.ZellijPlugin), 0o644); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(out, z.wasm); err != nil {
		return err
	}
	if err := z.applyPermissions(out); err != nil {
		return err
	}
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
		bak := z.path + ".docent.bak"
		msg := bak
		err = os.Link(tmp, bak)
		_ = os.Remove(tmp)
		if errors.Is(err, fs.ErrExist) {
			msg, err = bak+" kept (it holds the config from before docent)", nil
		}
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintln(out, msg); err != nil {
			return err
		}
	}
	if err := replaceFile(z.path, z.data, mode); err != nil {
		return err
	}
	_, err := fmt.Fprintln(out, z.path)
	return err
}

func (z *zellijTarget) applyPermissions(out io.Writer) error {
	f := z.perm
	if f.exists && f.data == f.old {
		_, err := fmt.Fprintln(out, f.path, "unchanged")
		return err
	}
	mode := fs.FileMode(0o644)
	if fi, err := os.Stat(f.path); err == nil {
		mode = fi.Mode().Perm()
	}
	if err := replaceFile(f.path, f.data, mode); err != nil {
		return err
	}
	_, err := fmt.Fprintln(out, f.path)
	return err
}

// replaceFile writes data with mode to a temp file and renames it to path.
func replaceFile(path, data string, mode fs.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := writeTemp(dir, data, mode)
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp) }()
	return os.Rename(tmp, path)
}

// writeTemp writes data with mode to a new temp file in dir, syncs it, and
// returns its name. It removes the file if it fails.
func writeTemp(dir, data string, mode fs.FileMode) (string, error) {
	f, err := os.CreateTemp(dir, ".config.kdl.docent-*")
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
