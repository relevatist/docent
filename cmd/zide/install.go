package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/relevatist/zide/assets"
)

// skillDirs maps a harness to its user skill directory, relative to $HOME.
var skillDirs = map[string]string{
	"claude": ".claude/skills",
	"codex":  ".agents/skills",
}

func runInstall(args []string) int {
	return installCmd(args, os.Stdout, os.Stderr)
}

func installCmd(args []string, out, w io.Writer) int {
	fs := flag.NewFlagSet("install", flag.ContinueOnError)
	fs.SetOutput(w)
	force := fs.Bool("force", false, "overwrite a changed skill file")
	var usageErr error
	fs.Usage = func() {
		_, usageErr = fmt.Fprintln(w, `Usage: zide install skill <claude|codex>... [--force]

Write the zide agent skill to each harness's user skill directory:
  claude  ~/.claude/skills/zide/SKILL.md (oh-my-pi reads it too)
  codex   ~/.agents/skills/zide/SKILL.md`)
		fs.PrintDefaults()
	}
	if len(args) == 0 || args[0] != "skill" {
		if err := fs.Parse(args); err != nil {
			return exitParse(err, usageErr)
		}
		fs.Usage()
		return 2
	}
	var names []string
	for rest := args[1:]; ; rest = rest[1:] {
		if err := fs.Parse(rest); err != nil {
			return exitParse(err, usageErr)
		}
		if rest = fs.Args(); len(rest) == 0 {
			break
		}
		names = append(names, rest[0])
	}
	if len(names) == 0 {
		fs.Usage()
		return 2
	}

	home, err := os.UserHomeDir()
	if err != nil {
		_, _ = fmt.Fprintln(w, "zide install:", err)
		return 1
	}
	var paths []string
	refused := false
	for _, name := range names {
		dir, ok := skillDirs[name]
		if !ok {
			_, _ = fmt.Fprintf(w, "zide install: unknown harness %q (want claude or codex)\n", name)
			return 2
		}
		path := filepath.Join(home, dir, "zide", "SKILL.md")
		if old, err := os.ReadFile(path); err == nil && !bytes.Equal(old, []byte(assets.Skill)) && !*force {
			_, _ = fmt.Fprintf(w, "zide install: %s has changes; use --force to overwrite\n", path)
			refused = true
		}
		paths = append(paths, path)
	}
	if refused {
		return 1
	}
	for _, path := range paths {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			_, _ = fmt.Fprintln(w, "zide install:", err)
			return 1
		}
		if err := os.WriteFile(path, []byte(assets.Skill), 0o644); err != nil {
			_, _ = fmt.Fprintln(w, "zide install:", err)
			return 1
		}
		if _, err := fmt.Fprintln(out, path); err != nil {
			_, _ = fmt.Fprintln(w, "zide install:", err)
			return 1
		}
	}
	return 0
}
