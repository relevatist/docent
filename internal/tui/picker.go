package tui

import (
	"cmp"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"

	"github.com/relevatist/zide/internal/recent"
)

const (
	pickerHelp  = "tab: field  up/down: move or cycle  enter: open  esc: cancel"
	openTabHelp = "up/down: move  enter: choose  esc: back"
)

// PickerInput is what the workstream picker starts with.
type PickerInput struct {
	Recent  []string // absolute directories, most recent first
	Roots   []string // searched for git repos after the picker shows
	Harness string   // first value of the harness field
	Aliases []string // config harness alias names, in any order
	Warning string   // shown under the fields
	// IsOpen returns the name of a tab that has dir open.
	IsOpen func(dir string) (tab string, ok bool)
}

// Action is what to do with the picked directory.
type Action int

const (
	Open    Action = iota // open a new tab
	Switch                // go to the tab that has the directory open
	Another               // open another tab for a directory that is open
)

// Picked is the user's choice.
type Picked struct {
	Dir     string // absolute
	Harness string // command or alias name
	Action  Action
}

// PickWorkstream shows the picker. ok is false when the user cancels.
func PickWorkstream(in PickerInput) (Picked, bool, error) {
	p := newPickerModel(in)
	if _, err := tea.NewProgram(p).Run(); err != nil {
		return Picked{}, false, err
	}
	picked, ok := p.result()
	return picked, ok, nil
}

type reposMsg []string

type picker struct {
	in             PickerInput
	home           string
	dirs           []string
	walking        bool
	query, harness string
	onHarness      bool
	cur            int
	err            string
	openTab        string // set on the open-tab screen
	another        bool   // the open-tab screen choice
	picked         Picked
	done           bool
	height         int
}

func newPickerModel(in PickerInput) *picker {
	home, _ := os.UserHomeDir()
	return &picker{
		in:      in,
		home:    home,
		dirs:    recent.Candidates(in.Recent, nil),
		walking: len(in.Roots) > 0,
		harness: in.Harness,
	}
}

func (p *picker) Init() tea.Cmd {
	if !p.walking {
		return nil
	}
	return func() tea.Msg { return reposMsg(recent.Repos(p.in.Roots)) }
}

func (p *picker) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		p.height = msg.Height
	case reposMsg:
		p.dirs, p.walking = recent.Candidates(p.dirs, msg), false
	case tea.PasteMsg:
		if p.openTab == "" {
			text := strings.NewReplacer("\r", "", "\n", "").Replace(msg.Content)
			p.edit(func(s string) string { return s + text })
		}
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return p, tea.Quit
		}
		if p.openTab != "" {
			return p, p.openTabKey(msg)
		}
		return p, p.key(msg)
	}
	return p, nil
}

func (p *picker) key(k tea.KeyPressMsg) tea.Cmd {
	p.err = ""
	switch k.String() {
	case "esc":
		return tea.Quit
	case "tab", "shift+tab":
		p.onHarness = !p.onHarness
	case "up", "ctrl+p":
		p.move(-1)
	case "down", "ctrl+n":
		p.move(1)
	case "backspace":
		p.edit(func(s string) string {
			_, n := utf8.DecodeLastRuneInString(s)
			return s[:len(s)-n]
		})
	case "enter":
		return p.confirm()
	default:
		if k.Text != "" {
			p.edit(func(s string) string { return s + k.Text })
		}
	}
	return nil
}

// edit sets the focused field to f(field). A query edit resets the selection.
func (p *picker) edit(f func(string) string) {
	if p.onHarness {
		p.harness = f(p.harness)
		return
	}
	p.query, p.cur = f(p.query), 0
}

func (p *picker) move(step int) {
	if !p.onHarness {
		p.cur = max(min(p.cur+step, len(p.matches())-1), 0)
		return
	}
	hs := p.harnesses()
	i := slices.Index(hs, p.harness)
	if i < 0 && step < 0 {
		i = 0
	}
	p.harness = hs[(i+step+len(hs))%len(hs)]
}

// harnesses returns the harness field values in cycle order, each once.
func (p *picker) harnesses() []string {
	var hs []string
	for _, h := range slices.Concat([]string{p.in.Harness}, slices.Sorted(slices.Values(p.in.Aliases)), []string{"claude", "omp", "codex"}) {
		if h != "" && !slices.Contains(hs, h) {
			hs = append(hs, h)
		}
	}
	return hs
}

func (p *picker) confirm() tea.Cmd {
	m := p.matches()
	switch {
	case len(m) == 0:
		p.err = "no directory to open"
		return nil
	case strings.TrimSpace(p.harness) == "":
		p.err = "harness is empty"
		return nil
	}
	p.picked = Picked{Dir: m[p.cur], Harness: strings.TrimSpace(p.harness)}
	if p.in.IsOpen != nil {
		if tab, ok := p.in.IsOpen(p.picked.Dir); ok {
			p.openTab, p.another = cmp.Or(tab, "?"), false
			return nil
		}
	}
	p.done = true
	return tea.Quit
}

func (p *picker) openTabKey(k tea.KeyPressMsg) tea.Cmd {
	switch k.String() {
	case "esc":
		p.openTab = ""
	case "up", "down", "ctrl+p", "ctrl+n":
		p.another = !p.another
	case "enter":
		p.picked.Action, p.done = Switch, true
		if p.another {
			p.picked.Action = Another
		}
		return tea.Quit
	}
	return nil
}

func (p *picker) matches() []string {
	return filter(p.dirs, p.query, p.home)
}

func (p *picker) View() tea.View {
	var lines []string
	if p.openTab != "" {
		lines = []string{
			safe(tilde(p.picked.Dir, p.home)) + " is open in tab " + bold.Render(safe(p.openTab)) + ".",
			choice(!p.another, "switch to that tab"),
			choice(p.another, "open another tab"),
			faint.Render(openTabHelp),
		}
		return tea.NewView(strings.Join(lines, "\n") + "\n")
	}
	lines = append(lines, field("dir: ", p.query, !p.onHarness))
	m := p.matches()
	room := len(m)
	if p.height > 0 {
		room = max(p.height-5, 1)
	}
	start := max(p.cur-room+1, 0)
	for i := start; i < min(start+room, len(m)); i++ {
		l := choice(i == p.cur, safe(tilde(m[i], p.home)))
		if slices.Contains(p.in.Recent, m[i]) {
			l += faint.Render("  recent")
		}
		lines = append(lines, l)
	}
	switch {
	case p.walking:
		lines = append(lines, faint.Render("  searching roots..."))
	case len(p.dirs) == 0 && len(p.in.Roots) == 0:
		lines = append(lines, faint.Render("  No directories. Set roots in the zide config to list git repos."))
	}
	lines = append(lines, field("harness: ", p.harness, p.onHarness))
	if p.in.Warning != "" {
		lines = append(lines, removed.Render("warning: "+safe(p.in.Warning)))
	}
	if p.err != "" {
		lines = append(lines, removed.Render(p.err))
	}
	lines = append(lines, faint.Render(pickerHelp))
	return tea.NewView(strings.Join(lines, "\n") + "\n")
}

func field(label, value string, focused bool) string {
	if focused {
		return bold.Render(label) + safe(value) + "_"
	}
	return faint.Render(label) + safe(value)
}

func choice(selected bool, s string) string {
	if selected {
		return "> " + bold.Render(s)
	}
	return "  " + s
}

func (p *picker) result() (Picked, bool) {
	if !p.done {
		return Picked{}, false
	}
	return p.picked, true
}

// filter returns the dirs whose display path (see tilde) holds the runes of q
// in order, ignoring case. When none match and q is a path to a directory, it
// returns that directory alone.
func filter(dirs []string, q, home string) []string {
	var out []string
	for _, d := range dirs {
		if subsequence(strings.ToLower(tilde(d, home)), strings.ToLower(q)) {
			out = append(out, d)
		}
	}
	if len(out) > 0 || q == "" {
		return out
	}
	p := q
	if p == "~" || strings.HasPrefix(p, "~/") {
		p = filepath.Join(home, p[1:])
	}
	if p, err := filepath.Abs(p); err == nil {
		if fi, err := os.Stat(p); err == nil && fi.IsDir() {
			return []string{p}
		}
	}
	return nil
}

func subsequence(s, q string) bool {
	for _, r := range q {
		i := strings.IndexRune(s, r)
		if i < 0 {
			return false
		}
		s = s[i+utf8.RuneLen(r):]
	}
	return true
}

// tilde writes a path under home as ~/...
func tilde(p, home string) string {
	if p == home {
		return "~"
	}
	if rest, ok := strings.CutPrefix(p, home+string(filepath.Separator)); ok && home != "" {
		return "~/" + rest
	}
	return p
}
