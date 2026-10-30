package main

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/relevatist/docent/assets"
)

func tempHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	for _, k := range []string{"XDG_DATA_HOME", "XDG_CONFIG_HOME", "XDG_CACHE_HOME", "ZELLIJ_CONFIG_DIR", "ZELLIJ_CONFIG_FILE"} {
		t.Setenv(k, "")
	}
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

func writeTestFile(t *testing.T, path, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o755); err != nil {
		t.Fatal(err)
	}
}

// fakeClaude records each claude call, one line of arguments per call, and
// answers `plugin marketplace list --json` from a file.
const fakeClaude = `#!/bin/sh
echo "$*" >> "$FAKE_CLAUDE_LOG"
if [ "$*" = "plugin marketplace list --json" ]; then
  cat "$FAKE_CLAUDE_MARKETPLACES" 2>/dev/null || echo '[]'
fi
`

type fakes struct {
	log, marketplaces string
}

// fakeHarnesses puts fake harness commands, and only those, on PATH. With no
// names it fakes all harnesses.
func fakeHarnesses(t *testing.T, names ...string) fakes {
	t.Helper()
	if len(names) == 0 {
		names = []string{"claude", "codex", "omp", "zellij"}
	}
	dir := t.TempDir()
	bin := filepath.Join(dir, "bin")
	for _, name := range names {
		script := "#!/bin/sh\n"
		if name == "claude" {
			script = fakeClaude
		}
		writeTestFile(t, filepath.Join(bin, name), script)
	}
	f := fakes{filepath.Join(dir, "claude.log"), filepath.Join(dir, "marketplaces.json")}
	t.Setenv("PATH", bin+":/usr/bin:/bin")
	t.Setenv("FAKE_CLAUDE_LOG", f.log)
	t.Setenv("FAKE_CLAUDE_MARKETPLACES", f.marketplaces)
	return f
}

func (f fakes) calls(t *testing.T) []string {
	t.Helper()
	b, err := os.ReadFile(f.log)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
}

func noPick(t *testing.T) pickFunc {
	return func([]bool) ([]string, bool, error) {
		t.Fatal("checklist shown")
		return nil, false, nil
	}
}

func TestInstallCodexAndOmp(t *testing.T) {
	home := tempHome(t)
	fakeHarnesses(t)
	var out, errw bytes.Buffer
	if code := installCmd([]string{"codex", "omp"}, &out, &errw, true, noPick(t)); code != 0 {
		t.Fatalf("exit %d: %s", code, errw.String())
	}
	for p, want := range map[string]string{
		".agents/skills/docent/SKILL.md":         assets.Skill,
		".omp/agent/extensions/docent-reload.ts": assets.OmpExtension,
	} {
		path := filepath.Join(home, p)
		if readFile(t, path) != want {
			t.Errorf("%s differs from the shipped file", path)
		}
		if strings.Count(out.String(), path+"\n") != 1 {
			t.Errorf("output does not print %s once:\n%s", path, out.String())
		}
	}
}

func TestInstallSameFile(t *testing.T) {
	tempHome(t)
	fakeHarnesses(t)
	var out, errw bytes.Buffer
	installCmd([]string{"codex"}, &out, &errw, false, nil)
	if code := installCmd([]string{"codex"}, &out, &errw, false, nil); code != 0 {
		t.Fatalf("reinstall of the same file: exit %d: %s", code, errw.String())
	}
}

func TestInstallChangedFile(t *testing.T) {
	home := tempHome(t)
	f := fakeHarnesses(t)
	changed := filepath.Join(home, ".omp/agent/extensions/docent-reload.ts")
	writeTestFile(t, changed, "mine")
	var out, errw bytes.Buffer
	if code := installCmd([]string{"claude", "omp"}, &out, &errw, false, nil); code == 0 {
		t.Fatal("want non-zero exit for a changed file")
	}
	if !strings.Contains(errw.String(), changed) {
		t.Errorf("error does not name %s: %s", changed, errw.String())
	}
	if readFile(t, changed) != "mine" {
		t.Error("changed file was overwritten without --force")
	}
	if _, err := os.Stat(filepath.Join(home, ".agents")); !os.IsNotExist(err) {
		t.Error("wrote the skill although another target was refused")
	}
	if calls := f.calls(t); calls != nil {
		t.Errorf("ran claude although a target was refused: %q", calls)
	}

	if code := installCmd([]string{"omp", "--force"}, &out, &errw, false, nil); code != 0 {
		t.Fatalf("--force: exit %d: %s", code, errw.String())
	}
	if readFile(t, changed) != assets.OmpExtension {
		t.Error("--force did not overwrite")
	}
}

func TestInstallMissingCommand(t *testing.T) {
	home := tempHome(t)
	fakeHarnesses(t, "codex")
	var out, errw bytes.Buffer
	if code := installCmd([]string{"claude", "codex"}, &out, &errw, false, nil); code == 0 {
		t.Fatal("want non-zero exit when claude is missing")
	}
	if !strings.Contains(errw.String(), "claude is not on PATH") {
		t.Errorf("error does not name claude: %s", errw.String())
	}
	if _, err := os.Stat(filepath.Join(home, ".agents")); !os.IsNotExist(err) {
		t.Error("wrote the codex skill although claude is missing")
	}
}

func TestInstallClaudeFirst(t *testing.T) {
	home := tempHome(t)
	f := fakeHarnesses(t)
	legacy := filepath.Join(home, ".claude/skills/docent/SKILL.md")
	writeTestFile(t, legacy, assets.Skill)
	var out, errw bytes.Buffer
	if code := installCmd([]string{"claude"}, &out, &errw, false, nil); code != 0 {
		t.Fatalf("exit %d: %s", code, errw.String())
	}
	dir := filepath.Join(home, ".local/share/docent/claude-plugin")
	want := []string{
		"plugin marketplace list --json",
		"plugin marketplace add " + dir,
		"plugin install docent@docent --scope user",
	}
	if calls := f.calls(t); !slices.Equal(calls, want) {
		t.Errorf("claude calls = %q\nwant %q", calls, want)
	}
	if readFile(t, filepath.Join(dir, "skills/docent/SKILL.md")) != assets.Skill {
		t.Error("plugin skill differs from the shipped skill")
	}
	manifest := readFile(t, filepath.Join(dir, ".claude-plugin/plugin.json"))
	if !strings.Contains(manifest, `"version": "0.0.0-`) {
		t.Errorf("plugin.json has no hash version:\n%s", manifest)
	}
	if !strings.Contains(readFile(t, filepath.Join(dir, ".claude-plugin/marketplace.json")), `"source": "./"`) {
		t.Error("marketplace.json does not list the plugin")
	}
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Error("legacy skill matching the shipped one was not removed")
	}
	for _, s := range []string{dir + "\n", "$ claude plugin marketplace add " + dir + "\n", "removed " + legacy + "\n"} {
		if !strings.Contains(out.String(), s) {
			t.Errorf("output does not contain %q:\n%s", s, out.String())
		}
	}
}

func TestInstallClaudeUpdate(t *testing.T) {
	home := tempHome(t)
	data := t.TempDir()
	t.Setenv("XDG_DATA_HOME", data)
	f := fakeHarnesses(t)
	dir := filepath.Join(data, "docent/claude-plugin")
	writeTestFile(t, f.marketplaces, `[{"name":"docent","source":"directory","path":"`+dir+`"}]`)
	legacy := filepath.Join(home, ".claude/skills/docent/SKILL.md")
	writeTestFile(t, legacy, "mine")
	var out, errw bytes.Buffer
	if code := installCmd([]string{"claude"}, &out, &errw, false, nil); code != 0 {
		t.Fatalf("exit %d: %s", code, errw.String())
	}
	want := []string{
		"plugin marketplace list --json",
		"plugin marketplace update docent",
		"plugin install docent@docent --scope user",
		"plugin update docent@docent --scope user",
	}
	if calls := f.calls(t); !slices.Equal(calls, want) {
		t.Errorf("claude calls = %q\nwant %q", calls, want)
	}
	if readFile(t, legacy) != "mine" {
		t.Error("changed legacy skill was removed")
	}
	if !strings.Contains(errw.String(), legacy) {
		t.Errorf("no warning about %s: %s", legacy, errw.String())
	}
}

func TestInstallClaudeOtherMarketplace(t *testing.T) {
	home := tempHome(t)
	f := fakeHarnesses(t)
	writeTestFile(t, f.marketplaces, `[{"name":"docent","source":"github","repo":"relevatist/docent"}]`)
	var out, errw bytes.Buffer
	if code := installCmd([]string{"claude", "codex"}, &out, &errw, false, nil); code == 0 {
		t.Fatal("want non-zero exit for a marketplace from another source")
	}
	if !strings.Contains(errw.String(), "claude plugin marketplace remove docent") {
		t.Errorf("error does not say how to fix: %s", errw.String())
	}
	if _, err := os.Stat(filepath.Join(home, ".agents")); !os.IsNotExist(err) {
		t.Error("wrote the codex skill although the marketplace was refused")
	}
	if calls := f.calls(t); len(calls) != 1 {
		t.Errorf("claude calls = %q; want only the marketplace list", calls)
	}
}

func TestInstallChecklist(t *testing.T) {
	home := tempHome(t)
	fakeHarnesses(t, "codex", "omp", "zellij")
	var found []bool
	pick := func(f []bool) ([]string, bool, error) {
		found = f
		return []string{"codex"}, true, nil
	}
	var out, errw bytes.Buffer
	if code := installCmd(nil, &out, &errw, true, pick); code != 0 {
		t.Fatalf("exit %d: %s", code, errw.String())
	}
	if !slices.Equal(found, []bool{false, true, true, true}) {
		t.Errorf("checked = %v; want claude unchecked, codex, omp, and zellij checked", found)
	}
	if readFile(t, filepath.Join(home, ".agents/skills/docent/SKILL.md")) != assets.Skill {
		t.Error("checked harness was not installed")
	}
}

func TestInstallChecklistCancelled(t *testing.T) {
	home := tempHome(t)
	fakeHarnesses(t)
	for _, pick := range []pickFunc{
		func([]bool) ([]string, bool, error) { return nil, false, nil },
		func([]bool) ([]string, bool, error) { return nil, true, nil },
	} {
		var out, errw bytes.Buffer
		if code := installCmd(nil, &out, &errw, true, pick); code != 1 {
			t.Errorf("exit %d, want 1", code)
		}
	}
	if entries, _ := os.ReadDir(home); len(entries) > 0 {
		t.Errorf("wrote %v", entries)
	}
}

func TestInstallHelp(t *testing.T) {
	var out, errw bytes.Buffer
	if code := installCmd([]string{"--help"}, &out, &errw, true, noPick(t)); code != 0 {
		t.Errorf("exit %d, want 0", code)
	}
	if !strings.Contains(errw.String(), "Usage: docent install") {
		t.Errorf("no usage: %s", errw.String())
	}
}

func TestInstallBadArgs(t *testing.T) {
	tempHome(t)
	fakeHarnesses(t)
	for _, args := range [][]string{nil, {"vim"}, {"skill", "claude"}, {"--bogus"}} {
		var out, errw bytes.Buffer
		if code := installCmd(args, &out, &errw, false, nil); code != 2 {
			t.Errorf("%q: exit %d, want 2", args, code)
		}
		if out.Len() > 0 {
			t.Errorf("%q: wrote output %q", args, out.String())
		}
	}
}
