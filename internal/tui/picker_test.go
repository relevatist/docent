package tui

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

func TestFilter(t *testing.T) {
	home := t.TempDir()
	retry, lib, x := filepath.Join(home, "wt/retry/app"), filepath.Join(home, "src/lib"), filepath.Join(home, "x")
	for _, d := range []string{lib, x} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	src := filepath.Dir(lib)
	if err := os.WriteFile(filepath.Join(home, "f"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	dirs := []string{retry, lib, "/opt/RTAP", x}
	tests := []struct {
		q    string
		want []string
	}{
		{"", dirs},
		{"rtap", []string{retry, "/opt/RTAP"}},
		{"RtaP", []string{retry, "/opt/RTAP"}},
		{"~/s", []string{lib}},
		{"~/src", []string{src, lib}},
		{"~", []string{home, retry, lib, x}},
		{"pat", nil},
		{"~/x", []string{x}},
		{"~/x/", []string{x}},
		{home, []string{home}},
		{lib, []string{lib}},
		{"~/missing", nil},
		{"/", []string{"/", retry, lib, "/opt/RTAP", x}},
		{"~/f", nil},
		{"~src", []string{lib}},
	}
	for _, tt := range tests {
		if got := filter(dirs, tt.q, home); !slices.Equal(got, tt.want) {
			t.Errorf("filter(%q) = %q; want %q", tt.q, got, tt.want)
		}
	}
}

func TestFilterRelative(t *testing.T) {
	cwd := t.TempDir()
	t.Chdir(cwd)
	app := mkdirs(t, cwd, "app")[0]
	other := "/opt/app"
	for _, tt := range []struct {
		q    string
		want []string
	}{
		{"app", []string{other, app}},
		{"./app", []string{app}},
		{"./app/", []string{app}},
		{"../" + filepath.Base(cwd) + "/app", []string{app}},
		{".", []string{cwd}},
		{"..", []string{filepath.Dir(cwd)}},
		{"~", nil},
		{"~" + cwd, nil},
	} {
		if got := filter([]string{other, app}, tt.q, ""); !slices.Equal(got, tt.want) {
			t.Errorf("filter(%q) = %q; want %q", tt.q, got, tt.want)
		}
	}
}

func mkdirs(t *testing.T, root string, rel ...string) []string {
	t.Helper()
	var dirs []string
	for _, r := range rel {
		d := filepath.Join(root, r)
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
		dirs = append(dirs, d)
	}
	return dirs
}

func typeText(m tea.Model, s string) {
	for _, r := range s {
		press(m, string(r))
	}
}

func newPicker(t *testing.T, in PickerInput) (*picker, []string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	dirs := mkdirs(t, home, "wt/retry/app", "src/app", "src/lib")
	if in.Recent == nil {
		in.Recent = dirs
	}
	return newPickerModel(in), dirs
}

func TestPickerFilterAndMove(t *testing.T) {
	p, dirs := newPicker(t, PickerInput{Harness: "claude"})
	typeText(p, "src")
	if got := p.matches(); !slices.Equal(got, dirs[1:]) {
		t.Fatalf("matches = %q; want %q", got, dirs[1:])
	}
	press(p, "down", "down", "up", "ctrl+n", "ctrl+p", "ctrl+n", "enter")
	if got, ok := p.result(); !ok || got != (Picked{Dir: dirs[2], Harness: "claude"}) {
		t.Errorf("result = %+v, %v; want %s", got, ok, dirs[2])
	}
	if !strings.Contains(p.View().Content, "~/src/lib") {
		t.Errorf("view does not show ~ paths:\n%s", p.View().Content)
	}
}

func TestPickerHarness(t *testing.T) {
	p, dirs := newPicker(t, PickerInput{Harness: "omp", Aliases: []string{"omp-sandbox", "fast"}})
	press(p, "tab")
	var got []string
	for range 6 {
		press(p, "down")
		got = append(got, p.harness)
	}
	if want := []string{"fast", "omp-sandbox", "claude", "codex", "omp", "fast"}; !slices.Equal(got, want) {
		t.Errorf("down cycles %q; want %q", got, want)
	}
	press(p, "up", "up")
	if p.harness != "codex" {
		t.Errorf("up gives %q; want codex", p.harness)
	}
	for range len("codex") {
		press(p, "backspace")
	}
	typeText(p, "claude --model opus")
	press(p, "tab")
	typeText(p, "lib")
	press(p, "enter")
	if got, ok := p.result(); !ok || got != (Picked{Dir: dirs[2], Harness: "claude --model opus"}) {
		t.Errorf("result = %+v, %v", got, ok)
	}
}

func TestPickerEmptyHarness(t *testing.T) {
	p, _ := newPicker(t, PickerInput{Harness: "omp"})
	press(p, "tab", "backspace", "backspace", "backspace", "enter")
	if _, ok := p.result(); ok {
		t.Fatal("picked with an empty harness")
	}
	if !strings.Contains(p.View().Content, "harness is empty") {
		t.Errorf("view does not say the harness is empty:\n%s", p.View().Content)
	}
}

func TestPickerCancel(t *testing.T) {
	for _, k := range []string{"esc", "ctrl+c"} {
		p, _ := newPicker(t, PickerInput{Harness: "claude"})
		press(p, k)
		if _, ok := p.result(); ok {
			t.Errorf("%s: want cancel", k)
		}
	}
}

func TestPickerWalk(t *testing.T) {
	root := t.TempDir()
	repo := mkdirs(t, root, "repo/.git")[0]
	p, dirs := newPicker(t, PickerInput{Harness: "claude", Roots: []string{root, root}})
	cmd := p.Init()
	if cmd == nil {
		t.Fatal("no walk command")
	}
	if got := p.matches(); !slices.Equal(got, dirs) {
		t.Fatalf("before walk: %q", got)
	}
	for cmd != nil {
		_, cmd = p.Update(cmd())
	}
	if got, want := p.matches(), append(dirs, filepath.Dir(repo)); !slices.Equal(got, want) || p.walking {
		t.Errorf("after walk: %q, walking %v; want %q", got, p.walking, want)
	}
}

func TestPickerWhileWalking(t *testing.T) {
	p, dirs := newPicker(t, PickerInput{Harness: "claude", Roots: []string{"unused"}})
	hits := make(chan string, 2)
	p.hits = hits
	hits <- dirs[0]
	hits <- "/opt/app"
	_, cmd := p.Update(p.next())
	if got, want := p.matches(), append(dirs, "/opt/app"); !slices.Equal(got, want) || !p.walking || cmd == nil {
		t.Fatalf("while walking: %q, walking %v; want %q", got, p.walking, want)
	}
	if !strings.Contains(p.View().Content, "searching roots") {
		t.Errorf("no walking indicator:\n%s", p.View().Content)
	}
	close(hits)
	if p.Update(cmd()); p.walking {
		t.Error("walking after the walk ended")
	}
}

func TestPickerStopWalk(t *testing.T) {
	root := t.TempDir()
	mkdirs(t, root, "a/.git", "b/.git", "c/.git")
	var p *picker
	_, ok, err := pickWorkstream(PickerInput{Harness: "claude", Roots: []string{root}}, func(m tea.Model) error {
		p = m.(*picker)
		p.Init()
		<-p.hits
		return nil
	})
	if ok || err != nil {
		t.Fatalf("pickWorkstream = %v, %v; want a cancel", ok, err)
	}
	select {
	case <-p.walked:
	case <-time.After(5 * time.Second):
		t.Fatal("walk did not stop")
	}
}

func TestPickerHint(t *testing.T) {
	p, _ := newPicker(t, PickerInput{Harness: "claude", Recent: []string{}})
	if p.Init() != nil {
		t.Error("walk without roots")
	}
	if !strings.Contains(p.View().Content, "Set roots") {
		t.Errorf("no roots hint:\n%s", p.View().Content)
	}
	p, _ = newPicker(t, PickerInput{Harness: "claude", Recent: []string{}, Roots: []string{t.TempDir()}})
	if strings.Contains(p.View().Content, "Set roots") {
		t.Errorf("roots hint with roots set:\n%s", p.View().Content)
	}
}

func TestPickerOpenTab(t *testing.T) {
	for _, tt := range []struct {
		keys []string
		want Action
	}{
		{[]string{"enter"}, Switch},
		{[]string{"down", "enter"}, Another},
		{[]string{"down", "down", "enter"}, Switch},
	} {
		var asked string
		in := PickerInput{Harness: "claude", IsOpen: func(dir string) (string, bool) {
			asked = dir
			return "app", strings.HasSuffix(dir, "src/app")
		}}
		p, dirs := newPicker(t, in)
		press(p, "down", "enter")
		if asked != dirs[1] {
			t.Fatalf("IsOpen(%q); want %q", asked, dirs[1])
		}
		if v := p.View().Content; !strings.Contains(v, "switch") || !strings.Contains(v, "open another") {
			t.Fatalf("no open-tab screen:\n%s", v)
		}
		press(p, tt.keys...)
		if got, ok := p.result(); !ok || got != (Picked{Dir: dirs[1], Harness: "claude", Action: tt.want}) {
			t.Errorf("%q: result = %+v, %v; want action %v", tt.keys, got, ok, tt.want)
		}
	}
}

func TestPickerOpenTabPaste(t *testing.T) {
	for _, keys := range [][]string{{"enter"}, {"tab", "enter"}} {
		p, _ := newPicker(t, PickerInput{Harness: "claude", IsOpen: func(string) (string, bool) { return "app", true }})
		press(p, keys...)
		p.Update(tea.PasteMsg{Content: "x"})
		if p.query != "" || p.harness != "claude" {
			t.Errorf("%q: query %q, harness %q after paste on the open-tab screen", keys, p.query, p.harness)
		}
	}
}

func TestPickerOpenTabBack(t *testing.T) {
	p, dirs := newPicker(t, PickerInput{Harness: "claude", IsOpen: func(string) (string, bool) { return "app", true }})
	press(p, "enter", "esc")
	if _, ok := p.result(); ok {
		t.Fatal("esc on the open-tab screen picked")
	}
	press(p, "down")
	if got := p.matches()[p.cur]; got != dirs[1] {
		t.Errorf("after esc, selection = %q; want the list back", got)
	}
}

func TestPickerScroll(t *testing.T) {
	p, dirs := newPicker(t, PickerInput{Harness: "claude"})
	p.Update(tea.WindowSizeMsg{Height: 6})
	press(p, "down", "down")
	v := p.View().Content
	if strings.Contains(v, "retry") || !strings.Contains(v, "~/src/lib") {
		t.Errorf("selection %s not in view:\n%s", dirs[2], v)
	}
}

func TestPickerHarnessKeepsSelection(t *testing.T) {
	p, dirs := newPicker(t, PickerInput{Harness: "claude"})
	press(p, "down", "down", "tab")
	typeText(p, " --xy")
	press(p, "backspace", "enter")
	if got, ok := p.result(); !ok || got != (Picked{Dir: dirs[2], Harness: "claude --x"}) {
		t.Errorf("result = %+v, %v; want %s", got, ok, dirs[2])
	}
}

func TestPickerPaste(t *testing.T) {
	p, dirs := newPicker(t, PickerInput{Harness: "claude"})
	press(p, "down")
	p.Update(tea.PasteMsg{Content: "li\r\nb"})
	if got := p.matches(); p.query != "lib" || p.cur != 0 || !slices.Equal(got, dirs[2:]) {
		t.Fatalf("query %q, cur %d, matches %q", p.query, p.cur, got)
	}
	press(p, "backspace", "backspace", "backspace", "down", "tab")
	p.Update(tea.PasteMsg{Content: " --x\n"})
	press(p, "enter")
	if got, ok := p.result(); !ok || got != (Picked{Dir: dirs[1], Harness: "claude --x"}) {
		t.Errorf("result = %+v, %v; want %s", got, ok, dirs[1])
	}
}

func TestPickerWarning(t *testing.T) {
	p, _ := newPicker(t, PickerInput{Harness: "claude", Warning: "bad recent.toml"})
	if v := p.View().Content; !strings.Contains(v, "warning: bad recent.toml") {
		t.Errorf("no warning:\n%s", v)
	}
}
