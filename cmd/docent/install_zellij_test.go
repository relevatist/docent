package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/relevatist/docent/assets"
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
	return "    // docent: begin (managed by `docent install`)\n" + node + "    // docent: end\n"
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
		{"BOM", "\ufeffkeybinds {\n}\n", "\ufeffkeybinds {\n" + block + "}\n"},
		{"CRLF", "keybinds {\r\n}\r\n", "keybinds {\r\n" + crlf(block) + "}\r\n"},
		{"CRLF same content", "keybinds {\r\n" + crlf(block) + "}\r\n", "keybinds {\r\n" + crlf(block) + "}\r\n"},
		{"CRLF no keybinds node", "theme \"x\"\r\n", "theme \"x\"\r\n" + crlf("keybinds {\n"+block+"}\n")},
		{"comment after the brace", "keybinds { // mine\n}\n", "keybinds { // mine\n" + block + "}\n"},
		{"Alt W is another key", "keybinds {\n    bind \"Alt W\" {}\n}\n", "keybinds {\n" + block + "    bind \"Alt W\" {}\n}\n"},
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
	changed := strings.Replace(managed(zellijNode()), "docent\" \"new\"", "docent\" \"new\" \"--x\"", 1)
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
		{"begin without end", "keybinds {\n" + zellijBegin + "}\n", "without a `// docent: end` line"},
		{"end without begin", "keybinds {\n" + zellijEnd + "}\n", "without a `// docent: end` line"},
		{"KDL v2 raw string", "keybinds {\n}\nx #\"a\"#\n", "by hand"},
		{"block comment opens on the keybinds line", "keybinds { /*\n*/\n}\n", "by hand"},
		{"string opens on the keybinds line", "keybinds { x \"{\n}\"\n}\n", "by hand"},
		{"keybinds after another node", "theme \"x\"; keybinds {\n}\n", "by hand"},
		{"quoted keybinds", "\"keybinds\" {\n}\n", "by hand"},
		{"BOM and keybinds after another node", "\ufefftheme \"x\"; keybinds {\n}\n", "by hand"},
		{"markers only in a block comment", "keybinds {\n/*\n" + managed(zellijNode()) + "*/\n}\n", "by hand"},
		{"two begin markers", "keybinds {\n" + managed(zellijNode()) + managed(zellijNode()) + "}\n", "markers out of order"},
		{"markers out of order", "keybinds {\n" + zellijEnd + zellijBegin + "}\n", "markers out of order"},
		{"alt w bound", "keybinds {\n    bind \"alt w\" { NewPane; }\n}\n", "already bound"},
		{"ALT  w bound", "keybinds {\n    bind \"ALT  w\" { NewPane; }\n}\n", "already bound"},
		{"Alt w second key", "keybinds {\n    bind \"Alt x\" \"Alt w\" { NewPane; }\n}\n", "already bound"},
		{"snippet added by hand", assets.ZellijKeys, "added by hand"},
	} {
		if _, err := editZellijConfig(tc.cfg, false); err == nil || !strings.Contains(err.Error(), tc.msg) {
			t.Errorf("%s: err = %v; want it to contain %q", tc.name, err, tc.msg)
		}
	}
}

func crlf(s string) string { return strings.ReplaceAll(s, "\n", "\r\n") }

func TestEditZellijConfigForce(t *testing.T) {
	block := managed(zellijNode())
	changed := strings.Replace(block, "docent\" \"new\"", "docent\" \"new\" \"--x\"", 1)
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

func TestPlatformZellijDir(t *testing.T) {
	t.Cleanup(func() { goos = runtime.GOOS })
	for _, tc := range []struct{ goos, xdg, want string }{
		{"darwin", "/xdg", "/h/Library/Application Support/org.Zellij-Contributors.Zellij"},
		{"linux", "/xdg", "/xdg/zellij"},
		{"linux", "xdg", "/h/.config/zellij"},
		{"linux", "", "/h/.config/zellij"},
	} {
		goos = tc.goos
		t.Setenv("XDG_CONFIG_HOME", tc.xdg)
		if got := platformZellijDir("/h"); got != tc.want {
			t.Errorf("%s, XDG_CONFIG_HOME=%q: got %q, want %q", tc.goos, tc.xdg, got, tc.want)
		}
	}
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
	if err := os.Chmod(dotfile, 0o640); err != nil {
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
	want := installedConfig(t, userConfig, home)
	if readFile(t, dotfile) != want {
		t.Errorf("config:\n%s", readFile(t, dotfile))
	}
	if fi, err := os.Lstat(cfg); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Error("symlink replaced")
	}
	if fi, err := os.Stat(dotfile); err != nil || fi.Mode().Perm() != 0o640 {
		t.Errorf("mode not kept: %v", fi.Mode())
	}
	if readFile(t, dotfile+".docent.bak") != userConfig {
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

	out.Reset()
	writeTestFile(t, dotfile, strings.Replace(want, "floating true", "floating false", 1))
	if code := installCmd([]string{"zellij", "--force"}, &out, &errw, false, nil); code != 0 {
		t.Fatalf("force: exit %d: %s", code, errw.String())
	}
	if readFile(t, dotfile) != want || readFile(t, dotfile+".docent.bak") != userConfig {
		t.Error("force rerun did not keep the first backup")
	}
	if !strings.Contains(out.String(), ".docent.bak kept") {
		t.Errorf("output does not say that the backup was kept:\n%s", out.String())
	}
}

func TestInstallZellijReadOnlyDir(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can write to a read-only dir")
	}
	home := tempHome(t)
	fakeHarnesses(t, "codex", "zellij")
	fakeZellij(t, "0")
	ro := filepath.Join(home, "ro")
	writeTestFile(t, filepath.Join(ro, "config.kdl"), userConfig)
	cfg := filepath.Join(home, ".config/zellij/config.kdl")
	if err := os.MkdirAll(filepath.Dir(cfg), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(ro, "config.kdl"), cfg); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(ro, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(ro, 0o755) })
	var out, errw bytes.Buffer
	if code := installCmd([]string{"codex", "zellij"}, &out, &errw, false, nil); code == 0 {
		t.Fatal("want non-zero exit")
	}
	if !strings.Contains(errw.String(), "permission denied") {
		t.Errorf("error: %s", errw.String())
	}
	if _, err := os.Stat(filepath.Join(home, ".agents")); !os.IsNotExist(err) {
		t.Error("wrote the codex skill although the Zellij dir is read-only")
	}
}

func TestInstallZellijBackupFails(t *testing.T) {
	home := tempHome(t)
	fakeHarnesses(t, "zellij")
	fakeZellij(t, "0")
	cfg := filepath.Join(home, ".config/zellij/config.kdl")
	writeTestFile(t, cfg, userConfig)
	writeString = func(f *os.File, s string) (int, error) {
		n, _ := f.WriteString(s[:len(s)/2])
		return n, errors.New("disk full")
	}
	t.Cleanup(func() { writeString = (*os.File).WriteString })
	var out, errw bytes.Buffer
	if code := installCmd([]string{"zellij"}, &out, &errw, false, nil); code == 0 {
		t.Fatal("want non-zero exit")
	}
	if !strings.Contains(errw.String(), "disk full") {
		t.Errorf("error: %s", errw.String())
	}
	if readFile(t, cfg) != userConfig {
		t.Error("config changed")
	}
	if names, _ := filepath.Glob(filepath.Join(filepath.Dir(cfg), "*")); len(names) != 1 {
		t.Errorf("files left: %v", names)
	}
	if _, err := os.Lstat(filepath.Join(filepath.Dir(cfg), "config.kdl.docent.bak")); !os.IsNotExist(err) {
		t.Error("partial backup left")
	}
}

func TestInstallZellijHardLink(t *testing.T) {
	home := tempHome(t)
	fakeHarnesses(t, "zellij")
	fakeZellij(t, "0")
	cfg := filepath.Join(home, ".config/zellij/config.kdl")
	writeTestFile(t, cfg, userConfig)
	if err := os.Link(cfg, filepath.Join(home, "config.kdl")); err != nil {
		t.Fatal(err)
	}
	var out, errw bytes.Buffer
	if code := installCmd([]string{"zellij"}, &out, &errw, false, nil); code == 0 {
		t.Fatal("want non-zero exit")
	}
	if want := cfg + " has other hard links; docent would split them"; !strings.Contains(errw.String(), want) {
		t.Errorf("error does not contain %q: %s", want, errw.String())
	}
	if readFile(t, cfg) != userConfig {
		t.Error("config changed")
	}
}

func TestInstallZellijNewFile(t *testing.T) {
	home := tempHome(t)
	fakeHarnesses(t, "zellij")
	fakeZellij(t, "0")
	dir := filepath.Join(t.TempDir(), "zj")
	t.Setenv("ZELLIJ_CONFIG_DIR", dir)
	var out, errw bytes.Buffer
	if code := installCmd([]string{"zellij"}, &out, &errw, false, nil); code != 0 {
		t.Fatalf("exit %d: %s", code, errw.String())
	}
	cfg := filepath.Join(dir, "config.kdl")
	if readFile(t, cfg) != installedConfig(t, "", home) {
		t.Errorf("config:\n%s", readFile(t, cfg))
	}
	if _, err := os.Stat(cfg + ".docent.bak"); !os.IsNotExist(err) {
		t.Error("backup written for a new file")
	}
	if fi, err := os.Stat(cfg); err != nil || fi.Mode().Perm() != 0o644 {
		t.Errorf("mode: %v", fi.Mode())
	}
}

func TestInstallZellijRefused(t *testing.T) {
	for _, tc := range []struct {
		name, cfg, code, msg string
	}{
		{"Alt w bound", "keybinds {\n    bind \"Alt w\" { NewPane; }\n}\n", "0", `bind "Alt w"`},
		{"check fails", userConfig, "1", "bad config"},
		{"markers out of order", "keybinds {\n" + zellijEnd + zellijBegin + "}\n", "0", "markers out of order; fix"},
		{"begin without end", "keybinds {\n" + zellijBegin + "}\n", "0", "(or the reverse); fix"},
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

// TestZellijNodeChecks runs the real zellij on configs with the marked
// nodes.
func TestZellijNodeChecks(t *testing.T) {
	if _, err := exec.LookPath("zellij"); err != nil || testing.Short() {
		t.Skip("needs zellij")
	}
	for _, cfg := range []string{userConfig, "", "load_plugins {\n    \"zellij:tab-bar\"\n}\n"} {
		if err := checkZellijConfig(installedConfig(t, cfg, "/no such home")); err != nil {
			t.Error(err)
		}
	}
	if err := checkZellijConfig(installedConfig(t, userConfig, t.TempDir())); err != nil {
		t.Error(err)
	}
}

func testWasm(home string) string {
	return filepath.Join(home, ".local/share/docent/zellij/docent.wasm")
}

// installedConfig is cfg with the key node and the plugin node for home.
func installedConfig(t *testing.T, cfg, home string) string {
	t.Helper()
	cfg, err := editZellijConfig(cfg, false)
	if err == nil {
		cfg, err = editMarked(cfg, pluginNode(testWasm(home)), false)
	}
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func managedPlugin(wasm string) string {
	return zellijPluginBegin + "    \"file:" + wasm + "\"\n" + zellijPluginEnd
}

func TestEditZellijPluginConfig(t *testing.T) {
	block := managedPlugin("/w/docent.wasm")
	for _, tc := range []struct {
		name, wasm, cfg, want string
	}{
		{"empty", "/w/docent.wasm", "", "load_plugins {\n    \"zellij:link\"\n" + block + "}\n"},
		{"insert after load_plugins line", "/w/docent.wasm",
			"theme \"x\"\nload_plugins {\n    \"zellij:link\"\n}\n",
			"theme \"x\"\nload_plugins {\n" + block + "    \"zellij:link\"\n}\n"},
		{"same content", "/w/docent.wasm",
			"load_plugins {\n" + block + "    \"zellij:link\"\n}\n",
			"load_plugins {\n" + block + "    \"zellij:link\"\n}\n"},
		{"no load_plugins node", "/w/docent.wasm", "theme \"x\"", "theme \"x\"\nload_plugins {\n    \"zellij:link\"\n" + block + "}\n"},
		{"only the first block", "/w/docent.wasm",
			"load_plugins {\n}\nload_plugins {\n}\n",
			"load_plugins {\n" + block + "}\nload_plugins {\n}\n"},
		{"key markers are not plugin markers", "/w/docent.wasm",
			"keybinds {\n" + managed(zellijNode()) + "}\nload_plugins {\n}\n",
			"keybinds {\n" + managed(zellijNode()) + "}\nload_plugins {\n" + block + "}\n"},
		{"CRLF", "/w/docent.wasm", "load_plugins {\r\n}\r\n", "load_plugins {\r\n" + crlf(block) + "}\r\n"},
		{"escaped path", "/a b/%\"/docent.wasm", "", "load_plugins {\n    \"zellij:link\"\n" +
			zellijPluginBegin + "    \"file:/a%20b/%25%22/docent.wasm\"\n" + zellijPluginEnd + "}\n"},
	} {
		got, err := editMarked(tc.cfg, pluginNode(tc.wasm), false)
		if err != nil {
			t.Errorf("%s: %v", tc.name, err)
		} else if got != tc.want {
			t.Errorf("%s: got\n%s\nwant\n%s", tc.name, got, tc.want)
		}
	}
}

func TestEditZellijPluginConfigRefused(t *testing.T) {
	block := managedPlugin("/w/docent.wasm")
	for _, tc := range []struct {
		name, cfg, msg string
	}{
		{"brace on next line", "load_plugins\n{\n}\n", `add "file:/w/docent.wasm" to the load_plugins block by hand`},
		{"node on one line", "load_plugins { \"zellij:link\"; }\n", "by hand"},
		{"not at line start", "theme \"x\"; load_plugins {\n}\n", "by hand"},
		{"nested", "x {\n    load_plugins {\n    }\n}\n", "by hand"},
		{"begin without end", "load_plugins {\n" + zellijPluginBegin + "}\n", "without a `// docent: plugin end` line"},
		{"end without begin", "load_plugins {\n" + zellijPluginEnd + "}\n", "without a `// docent: plugin end` line"},
		{"markers out of order", "load_plugins {\n" + zellijPluginEnd + zellijPluginBegin + "}\n", "markers out of order"},
		{"two begin markers", "load_plugins {\n" + block + block + "}\n", "markers out of order"},
		{"node changed", "load_plugins {\n" + managedPlugin("/old/docent.wasm") + "}\n", "plugin node has changes; use --force"},
	} {
		if _, err := editMarked(tc.cfg, pluginNode("/w/docent.wasm"), false); err == nil || !strings.Contains(err.Error(), tc.msg) {
			t.Errorf("%s: err = %v; want it to contain %q", tc.name, err, tc.msg)
		}
	}
}

func TestEditZellijPluginConfigForce(t *testing.T) {
	got, err := editMarked("load_plugins {\n"+managedPlugin("/old/docent.wasm")+"    x\n}\n", pluginNode("/w/docent.wasm"), true)
	if want := "load_plugins {\n" + managedPlugin("/w/docent.wasm") + "    x\n}\n"; err != nil || got != want {
		t.Errorf("got %v\n%s\nwant\n%s", err, got, want)
	}
}

func TestEditZellijPermissions(t *testing.T) {
	const (
		entry = "\"/w/docent.wasm\" {\n    WriteToStdin\n    ReadCliPipes\n    ReadApplicationState\n}\n"
		other = "\"/x/other.wasm\" {\n    RunCommands\n}\nzellij:link {\n}\n"
	)
	for _, tc := range []struct {
		name, cache, want string
	}{
		{"no file", "", entry},
		{"other entries kept", other, other + entry},
		{"same permissions", other + "\"/w/docent.wasm\" {\n    ReadApplicationState\n    WriteToStdin\n    ReadCliPipes\n}\n",
			other + "\"/w/docent.wasm\" {\n    ReadApplicationState\n    WriteToStdin\n    ReadCliPipes\n}\n"},
		{"other permissions", "\"/w/docent.wasm\" {\n    WriteToStdin\n}\n" + other, entry + other},
		{"denied", "a {\n}\n\"/w/docent.wasm\" {\n}\nb {\n}\n", "a {\n}\n" + entry + "b {\n}\n"},
		{"file: key is another plugin", "\"file:/w/docent.wasm\" {\n}\n", "\"file:/w/docent.wasm\" {\n}\n" + entry},
	} {
		got, ok := editPermissions(tc.cache, "/w/docent.wasm")
		if !ok || got != tc.want {
			t.Errorf("%s: got %v\n%s\nwant\n%s", tc.name, ok, got, tc.want)
		}
	}
	for _, bad := range []string{
		"x",
		"a {\n}",
		"a {\n    X\n",
		"a { X; }\n",
		"\n",
		"a {\n    X Y\n}\n",
		"}\n",
	} {
		if got, ok := editPermissions(bad, "/w/docent.wasm"); ok {
			t.Errorf("%q: got %q, want not ok", bad, got)
		}
	}
}

func TestZellijCacheDir(t *testing.T) {
	t.Cleanup(func() { goos = runtime.GOOS })
	for _, tc := range []struct{ goos, xdg, want string }{
		{"darwin", "/xdg", "/h/Library/Caches/org.Zellij-Contributors.Zellij"},
		{"linux", "/xdg", "/xdg/zellij"},
		{"linux", "xdg", "/h/.cache/zellij"},
		{"linux", "", "/h/.cache/zellij"},
	} {
		goos = tc.goos
		t.Setenv("XDG_CACHE_HOME", tc.xdg)
		if got := zellijCacheDir("/h"); got != tc.want {
			t.Errorf("%s, XDG_CACHE_HOME=%q: got %q, want %q", tc.goos, tc.xdg, got, tc.want)
		}
	}
}

func TestInstallZellijPlugin(t *testing.T) {
	home := tempHome(t)
	fakeHarnesses(t, "zellij")
	fakeZellij(t, "0")
	data := filepath.Join(t.TempDir(), "data")
	t.Setenv("XDG_DATA_HOME", data)
	wasm := filepath.Join(data, "docent/zellij/docent.wasm")
	cfg := filepath.Join(home, ".config/zellij/config.kdl")
	perms := filepath.Join(zellijCacheDir(home), "permissions.kdl")
	other := "\"/x/other.wasm\" {\n    RunCommands\n}\n"
	writeTestFile(t, perms, other)
	install := func(args ...string) (int, string, string) {
		var out, errw bytes.Buffer
		code := installCmd(append([]string{"zellij"}, args...), &out, &errw, false, nil)
		return code, out.String(), errw.String()
	}

	code, out, errs := install()
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errs)
	}
	if readFile(t, wasm) != string(assets.ZellijPlugin) {
		t.Error("wasm not written")
	}
	if !strings.Contains(readFile(t, cfg), "\"file:"+wasm+"\"") {
		t.Errorf("config does not load %s:\n%s", wasm, readFile(t, cfg))
	}
	if want, _ := editPermissions(other, wasm); readFile(t, perms) != want {
		t.Errorf("permissions:\n%s", readFile(t, perms))
	}
	for _, p := range []string{wasm, cfg, perms} {
		if !strings.Contains(out, p+"\n") {
			t.Errorf("output does not name %s:\n%s", p, out)
		}
	}

	if code, out, errs := install(); code != 0 || !strings.Contains(out, perms+" unchanged") || !strings.Contains(out, cfg+" unchanged") {
		t.Fatalf("rerun: exit %d: %s%s", code, out, errs)
	}

	writeTestFile(t, wasm, "old")
	if code, _, errs := install(); code != 0 {
		t.Fatalf("changed wasm: exit %d: %s", code, errs)
	}
	if readFile(t, wasm) != string(assets.ZellijPlugin) {
		t.Error("changed wasm not replaced")
	}
}

func TestInstallZellijBadPermissions(t *testing.T) {
	home := tempHome(t)
	fakeHarnesses(t, "codex", "zellij")
	fakeZellij(t, "0")
	cfg := filepath.Join(home, ".config/zellij/config.kdl")
	writeTestFile(t, cfg, userConfig)
	perms := filepath.Join(zellijCacheDir(home), "permissions.kdl")
	writeTestFile(t, perms, "a { X; }\n")
	var out, errw bytes.Buffer
	if code := installCmd([]string{"codex", "zellij"}, &out, &errw, false, nil); code == 0 {
		t.Fatal("want non-zero exit")
	}
	if !strings.Contains(errw.String(), perms+": docent cannot read") {
		t.Errorf("error does not name %s: %s", perms, errw.String())
	}
	if readFile(t, cfg) != userConfig || readFile(t, perms) != "a { X; }\n" {
		t.Error("config or permissions changed")
	}
	for _, p := range []string{testWasm(home), filepath.Join(home, ".agents")} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("wrote %s although the permission cache was refused", p)
		}
	}
}
