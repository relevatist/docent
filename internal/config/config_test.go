package config

import (
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, s string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	if s == "" {
		return
	}
	if err := os.MkdirAll(filepath.Join(dir, "zide"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "zide", "config.toml"), []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDefaults(t *testing.T) {
	write(t, "")
	c, err := Load()
	if err != nil || c != (Config{Harness: "claude", FocusOnSend: true}) {
		t.Fatalf("Load = %+v, %v", c, err)
	}
}

func TestFileOverrides(t *testing.T) {
	write(t, "harness = \"omp\"\nfocus_on_send = false\n")
	c, err := Load()
	if err != nil || c != (Config{Harness: "omp", FocusOnSend: false}) {
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
	_ = os.MkdirAll(filepath.Join(home, ".config", "zide"), 0o755)
	_ = os.WriteFile(filepath.Join(home, ".config", "zide", "config.toml"), []byte(`harness = "aider"`), 0o644)
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
