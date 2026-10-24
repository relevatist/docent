package main

import "testing"

func TestLaunchMissingDir(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if code := runLaunch([]string{"--harness", "cat", "/no/such/dir"}); code == 0 {
		t.Error("want non-zero exit")
	}
}

func TestLaunchTooManyArgs(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if code := runLaunch([]string{"a", "b"}); code != 2 {
		t.Errorf("got %d, want 2", code)
	}
}
