package config

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func write(t *testing.T, s string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	if s == "" {
		return
	}
	if err := os.MkdirAll(filepath.Join(dir, "docent"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "docent", "config.toml"), []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDefaults(t *testing.T) {
	write(t, "")
	c, err := Load()
	if err != nil || c.Harness != "claude" || !c.FocusOnSend || c.Roots != nil || c.Harnesses != nil {
		t.Fatalf("Load = %+v, %v", c, err)
	}
}

func TestFileOverrides(t *testing.T) {
	write(t, "harness = \"omp\"\nfocus_on_send = false\n")
	c, err := Load()
	if err != nil || c.Harness != "omp" || c.FocusOnSend {
		t.Fatalf("Load = %+v, %v", c, err)
	}
}

func TestPartialFileKeepsDefaults(t *testing.T) {
	write(t, "harness = \"omp\"\n")
	c, err := Load()
	if err != nil || !c.FocusOnSend {
		t.Fatalf("Load = %+v, %v", c, err)
	}
}

func TestHomeFallback(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", home)
	_ = os.MkdirAll(filepath.Join(home, ".config", "docent"), 0o755)
	_ = os.WriteFile(filepath.Join(home, ".config", "docent", "config.toml"), []byte(`harness = "aider"`), 0o644)
	c, err := Load()
	if err != nil || c.Harness != "aider" {
		t.Fatalf("Load = %+v, %v", c, err)
	}
}

func TestFlagOverridesFile(t *testing.T) {
	write(t, "harness = \"omp\"\n")
	c, _ := Load()
	if c.WithHarness("codex").Harness != "codex" {
		t.Fatal("flag did not override")
	}
	if c.WithHarness("").Harness != "omp" {
		t.Fatal("empty flag overrode file")
	}
}

func TestBadFile(t *testing.T) {
	write(t, "harness = \n")
	if _, err := Load(); err == nil {
		t.Fatal("want parse error")
	}
}

const aliases = `
harness = "omp-sandbox"
roots = ["~/src", "~", "/abs", "~x/y"]

[harnesses.omp-sandbox]
command = "sandbox-exec -f omp.sb omp"
kind = "omp"

[harnesses.fast]
command = "/opt/bin/claude --model haiku"
`

func TestResolve(t *testing.T) {
	write(t, aliases)
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	for s, want := range map[string]Harness{
		"omp-sandbox":          {"sandbox-exec -f omp.sb omp", "omp"},
		"fast":                 {"/opt/bin/claude --model haiku", "claude"},
		"codex --yolo":         {"codex --yolo", "codex"},
		"/usr/bin/aider":       {"/usr/bin/aider", "aider"},
		"omp-sandbox --x":      {"omp-sandbox --x", "omp-sandbox"},
		"  fast  ":             {"fast", "fast"},
		"Fast":                 {"Fast", "Fast"},
		"/opt/bin/omp-sandbox": {"/opt/bin/omp-sandbox", "omp-sandbox"},
	} {
		if got, err := c.Resolve(s); err != nil || got != want {
			t.Errorf("Resolve(%q) = %+v, %v; want %+v", s, got, err, want)
		}
	}
	if _, err := c.Resolve(" "); err == nil {
		t.Error("Resolve of empty harness: want error")
	}
}

func TestRoots(t *testing.T) {
	write(t, aliases)
	home := t.TempDir()
	t.Setenv("HOME", home)
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{filepath.Join(home, "src"), home, "/abs", "~x/y"}
	if !slices.Equal(c.Roots, want) {
		t.Errorf("Roots = %q; want %q", c.Roots, want)
	}
}

func TestBadAlias(t *testing.T) {
	for _, s := range []string{
		"[harnesses.broken]\ncommand = \"\"\n",
		"[harnesses.broken]\nkind = \"omp\"\n",
		"[harnesses.broken]\ncommand = \"  \"\n",
		"[harnesses.broken]\ncommand = \"omp\"\nkind = \"o mp\"\n",
	} {
		write(t, s)
		if _, err := Load(); err == nil || !strings.Contains(err.Error(), `"broken"`) {
			t.Errorf("Load(%q) = %v; want error naming the alias", s, err)
		}
	}
}

func TestReadmeExample(t *testing.T) {
	b, err := os.ReadFile("../../README.md")
	if err != nil {
		t.Fatal(err)
	}
	_, example, _ := strings.Cut(string(b), "```toml\n")
	example, _, _ = strings.Cut(example, "```")
	write(t, example)
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if h, err := c.Resolve(c.Harness); err != nil || h.Command != "claude" {
		t.Errorf("Resolve(%q) = %+v, %v", c.Harness, h, err)
	}
	if h, err := c.Resolve("omp-sandbox"); err != nil || h.Kind != "omp" || len(c.Roots) != 2 {
		t.Errorf("Load = %+v; Resolve = %+v, %v", c, h, err)
	}
}

func TestAliasHome(t *testing.T) {
	write(t, "[harnesses.box]\ncommand = \"~/bin/box -f ~/omp.sb ~x a~/b ~ omp\"\n")
	home := t.TempDir()
	t.Setenv("HOME", home)
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	want := home + "/bin/box -f " + home + "/omp.sb ~x a~/b " + home + " omp"
	if got := c.Harnesses["box"]; got.Command != want || got.Kind != "box" {
		t.Errorf("box = %+v; want command %q", got, want)
	}
}

func TestResolveHome(t *testing.T) {
	write(t, "")
	home := t.TempDir()
	t.Setenv("HOME", home)
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	want := Harness{home + "/bin/box -f " + home + "/omp.sb ~x omp", "box"}
	if got, err := c.Resolve("~/bin/box -f ~/omp.sb ~x omp"); err != nil || got != want {
		t.Errorf("Resolve = %+v, %v; want %+v", got, err, want)
	}
}
