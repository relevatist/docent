package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/relevatist/zide/assets"
)

const userConfig = "// my keybinds\n" +
	"/* keybinds {\n" +
	"} */\n" +
	"keybinds clear-defaults=true {\n" +
	"    normal {\n" +
	"        bind \"Alt n\" { NewPane; }\n" +
	"    }\n" +
	"}\n" +
	"theme \"x\"\n"

func managed(node string) string {
	return "    // zide: begin (managed by `zide install`)\n" + node + "    // zide: end\n"
}

func TestZellijNode(t *testing.T) {
	node := zellijNode()
	if !strings.HasPrefix(node, "    shared_except \"locked\" {\n") || !strings.HasSuffix(node, "\n    }\n") {
		t.Errorf("node:\n%s", node)
	}
	if "keybinds {\n"+node+"}\n" != assets.ZellijKeys {
		t.Error("node is not the inner part of assets/zellij/keys.kdl")
	}
}

func TestEditZellijConfig(t *testing.T) {
	block := managed(zellijNode())
	lines := strings.SplitAfter(userConfig, "\n")
	inserted := strings.Join(lines[:4], "") + block + strings.Join(lines[4:], "")
	for _, tc := range []struct {
		name, cfg, want string
	}{
		{"empty", "", "keybinds {\n" + block + "}\n"},
		{"insert after first keybinds line", userConfig, inserted},
		{"same content", inserted, inserted},
		{"no keybinds node", "theme \"x\"", "theme \"x\"\nkeybinds {\n" + block + "}\n"},
		{"Alt w in a comment", "// bind \"Alt w\" {}\nkeybinds {\n}\n", "// bind \"Alt w\" {}\nkeybinds {\n" + block + "}\n"},
		{"braces in a string", "x \"{\"\nkeybinds {\n}\n", "x \"{\"\nkeybinds {\n" + block + "}\n"},
	} {
		got, err := editZellijConfig(tc.cfg, false)
		if err != nil {
			t.Errorf("%s: %v", tc.name, err)
		} else if got != tc.want {
			t.Errorf("%s: got\n%s\nwant\n%s", tc.name, got, tc.want)
		}
	}
}

func TestEditZellijConfigRefused(t *testing.T) {
	changed := strings.Replace(managed(zellijNode()), "zide\" \"new\"", "zide\" \"new\" \"--x\"", 1)
	for _, tc := range []struct {
		name, cfg, msg string
	}{
		{"brace on next line", "keybinds\n{\n}\n", "by hand"},
		{"node on one line", "keybinds { normal {} }\n", "by hand"},
		{"raw string", "keybinds {\n}\nx r\"a\"\n", "by hand"},
		{"slashdash", "/-keybinds {\n}\n", "by hand"},
		{"open block comment", "/* keybinds {\n", "by hand"},
		{"Alt w bound", "keybinds {\n    normal {\n        bind \"Alt w\" { NewPane; }\n    }\n}\n", `bind "Alt w"`},
		{"marked node changed", "keybinds {\n" + changed + "}\n", "--force"},
		{"begin without end", "keybinds {\n    // zide: begin (managed by `zide install`)\n}\n", "by hand"},
	} {
		if _, err := editZellijConfig(tc.cfg, false); err == nil || !strings.Contains(err.Error(), tc.msg) {
			t.Errorf("%s: err = %v; want it to contain %q", tc.name, err, tc.msg)
		}
	}
}

func TestEditZellijConfigForce(t *testing.T) {
	block := managed(zellijNode())
	changed := strings.Replace(block, "zide\" \"new\"", "zide\" \"new\" \"--x\"", 1)
	got, err := editZellijConfig("keybinds {\n"+changed+"    normal {}\n}\n", true)
	if want := "keybinds {\n" + block + "    normal {}\n}\n"; err != nil || got != want {
		t.Errorf("got %v\n%s\nwant\n%s", err, got, want)
	}
}

func TestZellijConfigPath(t *testing.T) {
	home := tempHome(t)
	sys := filepath.Join(t.TempDir(), "etc-zellij")
	systemZellijDir = sys
	t.Cleanup(func() { systemZellijDir = "/etc/zellij" })
	check := func(name, want string) {
		t.Helper()
		if got, err := zellijConfigPath(home); err != nil || got != want {
			t.Errorf("%s: got %q, %v; want %q", name, got, err, want)
		}
	}
	check("nothing exists", filepath.Join(home, ".config/zellij/config.kdl"))

	writeTestFile(t, filepath.Join(sys, "config.kdl"), "")
	if _, err := zellijConfigPath(home); err == nil || !strings.Contains(err.Error(), sys) {
		t.Errorf("system config only: err = %v", err)
	}

	if platformZellijDir(home) != filepath.Join(home, ".config/zellij") {
		platform := platformZellijDir(home)
		writeTestFile(t, filepath.Join(platform, "x"), "")
		check("platform dir", filepath.Join(platform, "config.kdl"))
	}
	if err := os.MkdirAll(filepath.Join(home, ".config/zellij"), 0o755); err != nil {
		t.Fatal(err)
	}
	check("~/.config/zellij exists without config.kdl", filepath.Join(home, ".config/zellij/config.kdl"))

	dir := t.TempDir()
	t.Setenv("ZELLIJ_CONFIG_DIR", dir)
	check("ZELLIJ_CONFIG_DIR", filepath.Join(dir, "config.kdl"))
	t.Setenv("ZELLIJ_CONFIG_FILE", "/x/my.kdl")
	check("ZELLIJ_CONFIG_FILE", "/x/my.kdl")
}

// fakeZellij replaces the fake zellij command with one that records its
// arguments and the config it checks, and exits with code.
func fakeZellij(t *testing.T, code string) (log string) {
	t.Helper()
	bin := filepath.SplitList(os.Getenv("PATH"))[0]
	log = filepath.Join(t.TempDir(), "zellij.log")
	writeTestFile(t, filepath.Join(bin, "zellij"), "#!/bin/sh\necho \"$*\" >> "+log+"\ncat \"$2\" >> "+log+"\necho bad config >&2\nexit "+code+"\n")
	return log
}

func TestInstallZellij(t *testing.T) {
	home := tempHome(t)
	fakeHarnesses(t, "zellij")
	log := fakeZellij(t, "0")
	dotfile := filepath.Join(home, "dotfiles/config.kdl")
	writeTestFile(t, dotfile, userConfig)
	if err := os.Chmod(dotfile, 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(home, ".config/zellij/config.kdl")
	if err := os.MkdirAll(filepath.Dir(cfg), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(dotfile, cfg); err != nil {
		t.Fatal(err)
	}
	var out, errw bytes.Buffer
	if code := installCmd([]string{"zellij"}, &out, &errw, false, nil); code != 0 {
		t.Fatalf("exit %d: %s", code, errw.String())
	}
	want, _ := editZellijConfig(userConfig, false)
	if readFile(t, dotfile) != want {
		t.Errorf("config:\n%s", readFile(t, dotfile))
	}
	if fi, err := os.Lstat(cfg); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Error("symlink replaced")
	}
	if fi, err := os.Stat(dotfile); err != nil || fi.Mode().Perm() != 0o600 {
		t.Errorf("mode not kept: %v", fi.Mode())
	}
	if readFile(t, dotfile+".zide.bak") != userConfig {
		t.Error("backup does not hold the old config")
	}
	if l := readFile(t, log); !strings.Contains(l, "setup --check") || !strings.Contains(l, want) {
		t.Errorf("zellij did not check the new config:\n%s", l)
	}
	if real, _ := filepath.EvalSymlinks(dotfile); !strings.Contains(out.String(), real+"\n") {
		t.Errorf("output does not name %s:\n%s", real, out.String())
	}

	out.Reset()
	if code := installCmd([]string{"zellij"}, &out, &errw, false, nil); code != 0 {
		t.Fatalf("rerun: exit %d: %s", code, errw.String())
	}
	if readFile(t, dotfile) != want || !strings.Contains(out.String(), "unchanged") {
		t.Errorf("rerun changed the config or did not say so:\n%s", out.String())
	}
}

func TestInstallZellijNewFile(t *testing.T) {
	tempHome(t)
	fakeHarnesses(t, "zellij")
	fakeZellij(t, "0")
	dir := filepath.Join(t.TempDir(), "zj")
	t.Setenv("ZELLIJ_CONFIG_DIR", dir)
	var out, errw bytes.Buffer
	if code := installCmd([]string{"zellij"}, &out, &errw, false, nil); code != 0 {
		t.Fatalf("exit %d: %s", code, errw.String())
	}
	cfg := filepath.Join(dir, "config.kdl")
	if readFile(t, cfg) != "keybinds {\n"+managed(zellijNode())+"}\n" {
		t.Errorf("config:\n%s", readFile(t, cfg))
	}
	if _, err := os.Stat(cfg + ".zide.bak"); !os.IsNotExist(err) {
		t.Error("backup written for a new file")
	}
}

func TestInstallZellijRefused(t *testing.T) {
	for _, tc := range []struct {
		name, cfg, code, msg string
	}{
		{"Alt w bound", "keybinds {\n    bind \"Alt w\" { NewPane; }\n}\n", "0", `bind "Alt w"`},
		{"check fails", userConfig, "1", "bad config"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := tempHome(t)
			fakeHarnesses(t, "codex", "zellij")
			fakeZellij(t, tc.code)
			cfg := filepath.Join(home, ".config/zellij/config.kdl")
			writeTestFile(t, cfg, tc.cfg)
			var out, errw bytes.Buffer
			if code := installCmd([]string{"codex", "zellij"}, &out, &errw, false, nil); code == 0 {
				t.Fatal("want non-zero exit")
			}
			if !strings.Contains(errw.String(), cfg) || !strings.Contains(errw.String(), tc.msg) {
				t.Errorf("error does not name %s and %q: %s", cfg, tc.msg, errw.String())
			}
			if readFile(t, cfg) != tc.cfg {
				t.Error("config changed")
			}
			if _, err := os.Stat(filepath.Join(home, ".agents")); !os.IsNotExist(err) {
				t.Error("wrote the codex skill although the Zellij config was refused")
			}
		})
	}
}

// TestZellijNodeChecks runs the real zellij on a config with clear-defaults
// and the marked node.
func TestZellijNodeChecks(t *testing.T) {
	if _, err := exec.LookPath("zellij"); err != nil || testing.Short() {
		t.Skip("needs zellij")
	}
	got, err := editZellijConfig(userConfig, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := checkZellijConfig(got); err != nil {
		t.Error(err)
	}
}
