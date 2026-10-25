package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/relevatist/zide/assets"
	"github.com/relevatist/zide/internal/tui"
)

type harness struct {
	name, label, detail string
}

var harnesses = []harness{
	{"claude", "Claude Code", "plugin: skill + reload hook"},
	{"codex", "Codex", "skill"},
	{"omp", "oh-my-pi", "skill + reload extension"},
	{"zellij", "Zellij", "Alt w key for zide new"},
}

// pickFunc asks the user which harnesses to install for. ok is false when
// the user cancels.
type pickFunc func(found []bool) (names []string, ok bool, err error)

func runInstall(args []string) int {
	tty := isTerminal(os.Stdin) && isTerminal(os.Stdout)
	return installCmd(args, os.Stdout, os.Stderr, tty, pickHarnesses)
}

func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

func pickHarnesses(found []bool) ([]string, bool, error) {
	opts := make([]tui.Option, len(harnesses))
	for i, h := range harnesses {
		opts[i] = tui.Option{Label: h.label, Detail: h.detail, Checked: found[i]}
		if home, err := os.UserHomeDir(); h.name == "zellij" && err == nil {
			if path, err := zellijConfigPath(home); err == nil {
				opts[i].Detail += " (" + path + ")"
			}
		}
	}
	picked, ok, err := tui.Pick("Install zide for:", opts)
	var names []string
	for _, i := range picked {
		names = append(names, harnesses[i].name)
	}
	return names, ok, err
}

func installCmd(args []string, out, w io.Writer, tty bool, pick pickFunc) int {
	fs := flag.NewFlagSet("install", flag.ContinueOnError)
	fs.SetOutput(w)
	force := fs.Bool("force", false, "overwrite a changed skill or extension file, or a changed zide key in the Zellij config")
	var usageErr error
	fs.Usage = func() {
		_, usageErr = fmt.Fprintln(w, `Usage: zide install [claude|codex|omp|zellij]... [--force]

Install zide for each named harness. With no names on a terminal, choose
from a checklist.
  claude  Claude Code plugin (skill + reload hook), through the claude CLI
  codex   ~/.agents/skills/zide/SKILL.md
  omp     ~/.agents/skills/zide/SKILL.md, ~/.omp/agent/extensions/zide-reload.ts
  zellij  Alt w key for zide new, in the Zellij config.kdl (the first old file kept as config.kdl.zide.bak)`)
		fs.PrintDefaults()
	}
	var names []string
	for rest := args; ; rest = rest[1:] {
		if err := fs.Parse(rest); err != nil {
			return exitParse(err, usageErr)
		}
		if rest = fs.Args(); len(rest) == 0 {
			break
		}
		names = append(names, rest[0])
	}
	for _, name := range names {
		if !slices.ContainsFunc(harnesses, func(h harness) bool { return h.name == name }) {
			_, _ = fmt.Fprintf(w, "zide install: unknown harness %q (want claude, codex, omp, or zellij)\n", name)
			return 2
		}
	}
	if len(names) == 0 {
		if !tty {
			fs.Usage()
			return 2
		}
		found := make([]bool, len(harnesses))
		for i, h := range harnesses {
			_, err := exec.LookPath(h.name)
			found[i] = err == nil
		}
		var ok bool
		var err error
		if names, ok, err = pick(found); err != nil {
			_, _ = fmt.Fprintln(w, "zide install:", err)
			return 1
		} else if !ok || len(names) == 0 {
			_, _ = fmt.Fprintln(w, "zide install: nothing installed")
			return 1
		}
	}

	in, err := newInstall(names, *force)
	if err != nil {
		_, _ = fmt.Fprintln(w, "zide install:", err)
		return 1
	}
	if problems := in.check(); len(problems) > 0 {
		for _, p := range problems {
			_, _ = fmt.Fprintln(w, "zide install:", p)
		}
		return 1
	}
	if err := in.apply(out, w); err != nil {
		_, _ = fmt.Fprintln(w, "zide install:", err)
		return 1
	}
	return 0
}

type fileTarget struct {
	path, data string
}

type install struct {
	names     []string
	force     bool
	files     []fileTarget
	pluginDir string // empty unless installing for claude
	legacy    string // the skill path from before the Claude Code plugin
	added     bool   // the zide marketplace is already registered from pluginDir
	zellij    *zellijTarget
}

func newInstall(names []string, force bool) (*install, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	in := &install{names: names, force: force}
	skill := fileTarget{filepath.Join(home, ".agents/skills/zide/SKILL.md"), assets.Skill}
	for _, name := range names {
		switch name {
		case "claude":
			data := os.Getenv("XDG_DATA_HOME")
			if !filepath.IsAbs(data) {
				data = filepath.Join(home, ".local/share")
			}
			in.pluginDir = filepath.Join(data, "zide/claude-plugin")
			in.legacy = filepath.Join(home, ".claude/skills/zide/SKILL.md")
		case "codex":
			in.addFile(skill)
		case "omp":
			in.addFile(skill)
			in.addFile(fileTarget{filepath.Join(home, ".omp/agent/extensions/zide-reload.ts"), assets.OmpExtension})
		case "zellij":
			in.zellij = &zellijTarget{home: home}
		}
	}
	return in, nil
}

func (in *install) addFile(f fileTarget) {
	if !slices.Contains(in.files, f) {
		in.files = append(in.files, f)
	}
}

// check returns every reason not to install, before anything is changed.
func (in *install) check() []string {
	var problems []string
	for _, name := range in.names {
		if _, err := exec.LookPath(name); err != nil {
			problems = append(problems, fmt.Sprintf("%s is not on PATH", name))
		}
	}
	for _, f := range in.files {
		if old, err := os.ReadFile(f.path); err == nil && string(old) != f.data && !in.force {
			problems = append(problems, fmt.Sprintf("%s has changes; use --force to overwrite", f.path))
		}
	}
	if in.zellij != nil {
		if p := in.zellij.check(in.force); p != "" {
			problems = append(problems, p)
		}
	}
	if in.pluginDir != "" && len(problems) == 0 {
		if p := in.checkMarketplace(); p != "" {
			problems = append(problems, p)
		}
	}
	return problems
}

func (in *install) checkMarketplace() string {
	m, err := zideMarketplace()
	if err != nil {
		return "claude plugin marketplace list: " + err.Error()
	}
	if m != nil && (m.Source != "directory" || !samePath(m.Path, in.pluginDir)) {
		return "a zide marketplace is registered from another source; run `claude plugin marketplace remove zide` first"
	}
	in.added = m != nil
	return ""
}

type marketplace struct {
	Name   string `json:"name"`
	Source string `json:"source"`
	Path   string `json:"path"`
}

// zideMarketplace returns the registered zide marketplace, or nil.
func zideMarketplace() (*marketplace, error) {
	data, err := exec.Command("claude", "plugin", "marketplace", "list", "--json").Output()
	if err != nil {
		return nil, err
	}
	var ms []marketplace
	if err := json.Unmarshal(data, &ms); err != nil {
		return nil, err
	}
	if i := slices.IndexFunc(ms, func(m marketplace) bool { return m.Name == "zide" }); i >= 0 {
		return &ms[i], nil
	}
	return nil, nil
}

func samePath(a, b string) bool {
	if filepath.Clean(a) == filepath.Clean(b) {
		return true
	}
	ra, errA := filepath.EvalSymlinks(a)
	rb, errB := filepath.EvalSymlinks(b)
	return errA == nil && errB == nil && ra == rb
}

func (in *install) apply(out, w io.Writer) error {
	for _, f := range in.files {
		if err := os.MkdirAll(filepath.Dir(f.path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(f.path, []byte(f.data), 0o644); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(out, f.path); err != nil {
			return err
		}
	}
	if in.zellij != nil {
		if err := in.zellij.apply(out); err != nil {
			return err
		}
	}
	if in.pluginDir == "" {
		return nil
	}
	return in.applyClaude(out, w)
}

func (in *install) applyClaude(out, w io.Writer) error {
	if err := writePlugin(in.pluginDir); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(out, in.pluginDir); err != nil {
		return err
	}
	cmds := [][]string{
		{"plugin", "marketplace", "add", in.pluginDir},
		{"plugin", "install", "zide@zide", "--scope", "user"},
	}
	if in.added {
		cmds = [][]string{
			{"plugin", "marketplace", "update", "zide"},
			{"plugin", "install", "zide@zide", "--scope", "user"},
			{"plugin", "update", "zide@zide", "--scope", "user"},
		}
	}
	for _, args := range cmds {
		if _, err := fmt.Fprintln(out, "$ claude", strings.Join(args, " ")); err != nil {
			return err
		}
		cmd := exec.Command("claude", args...)
		cmd.Stdout, cmd.Stderr = out, w
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("claude %s: %w", strings.Join(args, " "), err)
		}
	}
	if old, err := os.ReadFile(in.legacy); err == nil {
		if string(old) != assets.Skill {
			_, err := fmt.Fprintf(w, "zide install: %s has changes, so it was kept; the plugin now provides the skill\n", in.legacy)
			return err
		}
		if err := os.Remove(in.legacy); err != nil {
			return err
		}
		_ = os.Remove(filepath.Dir(in.legacy))
		if _, err := fmt.Fprintln(out, "removed", in.legacy); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintln(out, "Restart Claude Code to load the zide plugin.")
	return err
}

// writePlugin replaces dir with the embedded plugin and a marketplace that
// lists it. The plugin version is a hash of its files, because Claude Code
// caches a plugin by version.
func writePlugin(dir string) error {
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	h := sha256.New()
	err := fs.WalkDir(assets.ClaudePlugin, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := fs.ReadFile(assets.ClaudePlugin, path)
		if err != nil {
			return err
		}
		h.Write([]byte(path))
		h.Write(data)
		if path != ".claude-plugin/plugin.json" {
			return writeFile(filepath.Join(dir, path), data)
		}
		return nil
	})
	if err != nil {
		return err
	}

	data, err := fs.ReadFile(assets.ClaudePlugin, ".claude-plugin/plugin.json")
	if err != nil {
		return err
	}
	var manifest map[string]any
	if err := json.Unmarshal(data, &manifest); err != nil {
		return err
	}
	manifest["version"] = "0.0.0-" + hex.EncodeToString(h.Sum(nil))[:12]
	marketplace := map[string]any{
		"name":    "zide",
		"owner":   map[string]any{"name": "zide"},
		"plugins": []any{map[string]any{"name": "zide", "source": "./"}},
	}
	for name, v := range map[string]any{"plugin.json": manifest, "marketplace.json": marketplace} {
		data, err := json.Marshal(v, json.Deterministic(true), jsontext.WithIndent("  "))
		if err != nil {
			return err
		}
		if err := writeFile(filepath.Join(dir, ".claude-plugin", name), append(data, '\n')); err != nil {
			return err
		}
	}
	return nil
}

func writeFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
