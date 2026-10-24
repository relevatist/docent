package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/relevatist/zide/assets"
)

func tempHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	return home
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestInstallSkill(t *testing.T) {
	home := tempHome(t)
	var out, errw bytes.Buffer
	if code := installCmd([]string{"skill", "claude", "codex"}, &out, &errw); code != 0 {
		t.Fatalf("exit %d: %s", code, errw.String())
	}
	for _, p := range []string{".claude/skills/zide/SKILL.md", ".agents/skills/zide/SKILL.md"} {
		path := filepath.Join(home, p)
		if readFile(t, path) != assets.Skill {
			t.Errorf("%s differs from the shipped skill", path)
		}
		if !strings.Contains(out.String(), path+"\n") {
			t.Errorf("output does not print %s:\n%s", path, out.String())
		}
	}
}

func TestInstallSkillSameFile(t *testing.T) {
	tempHome(t)
	var out, errw bytes.Buffer
	installCmd([]string{"skill", "claude"}, &out, &errw)
	if code := installCmd([]string{"skill", "claude"}, &out, &errw); code != 0 {
		t.Fatalf("reinstall of the same file: exit %d: %s", code, errw.String())
	}
}

func TestInstallSkillChangedFile(t *testing.T) {
	home := tempHome(t)
	changed := filepath.Join(home, ".claude/skills/zide/SKILL.md")
	if err := os.MkdirAll(filepath.Dir(changed), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(changed, []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errw bytes.Buffer
	if code := installCmd([]string{"skill", "codex", "claude"}, &out, &errw); code == 0 {
		t.Fatal("want non-zero exit for a changed file")
	}
	if !strings.Contains(errw.String(), changed) {
		t.Errorf("error does not name %s: %s", changed, errw.String())
	}
	if readFile(t, changed) != "mine" {
		t.Error("changed file was overwritten without --force")
	}
	if _, err := os.Stat(filepath.Join(home, ".agents")); !os.IsNotExist(err) {
		t.Error("wrote codex skill although another target was refused")
	}

	if code := installCmd([]string{"skill", "--force", "claude"}, &out, &errw); code != 0 {
		t.Fatalf("--force: exit %d: %s", code, errw.String())
	}
	if readFile(t, changed) != assets.Skill {
		t.Error("--force did not overwrite")
	}
}

func TestInstallHelp(t *testing.T) {
	var out, errw bytes.Buffer
	if code := installCmd([]string{"--help"}, &out, &errw); code != 0 {
		t.Errorf("exit %d, want 0", code)
	}
	if !strings.Contains(errw.String(), "Usage: zide install skill") {
		t.Errorf("no usage: %s", errw.String())
	}
}

func TestInstallSkillBadArgs(t *testing.T) {
	tempHome(t)
	for _, args := range [][]string{nil, {"skill"}, {"skill", "vim"}, {"hooks", "claude"}} {
		var out, errw bytes.Buffer
		if code := installCmd(args, &out, &errw); code != 2 {
			t.Errorf("%q: exit %d, want 2", args, code)
		}
		if out.Len() > 0 {
			t.Errorf("%q: wrote output %q", args, out.String())
		}
	}
}
