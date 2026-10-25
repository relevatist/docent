// Package tui holds the terminal views: the tour pane, the install checklist,
// and the workstream picker.
package tui

import (
	"cmp"
	"fmt"
	"io"
	"strings"
	"unicode"

	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2"
	"charm.land/glamour/v2/styles"
	"charm.land/lipgloss/v2"
	"github.com/AlexanderGrooff/mermaid-ascii/pkg/diagram"
	"github.com/AlexanderGrooff/mermaid-ascii/pkg/render"
	"github.com/charmbracelet/x/ansi"
	"github.com/sirupsen/logrus"

	"github.com/relevatist/zide/internal/tour"
)

var (
	bold    = lipgloss.NewStyle().Bold(true)
	faint   = lipgloss.NewStyle().Faint(true)
	removed = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
)

const help = "n next  p prev  o open  s send  q quit  j/k scroll  space/b page  g/G top/bottom"

// Run enters the first step and runs the tour until the user quits.
func Run(m *tour.Model, title string) error {
	// mermaid-ascii logs through logrus, which would write over the pane.
	logrus.SetOutput(io.Discard)
	m.Start()
	_, err := tea.NewProgram(&view{m: m, title: title}).Run()
	return err
}

type view struct {
	m             *tour.Model
	title         string
	width, height int
	scroll        int
	light         bool

	// The rendered note of the current step, for key.
	note []string
	key  noteKey
}

type noteKey struct {
	pos, width int
	light      bool
}

func (v *view) Init() tea.Cmd { return tea.RequestBackgroundColor }

func (v *view) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width, v.height = msg.Width, msg.Height
	case tea.BackgroundColorMsg:
		v.light = !msg.IsDark()
	case tea.KeyPressMsg:
		room := max(v.room(), 1)
		switch k := msg.String(); k {
		case "j", "down":
			v.scroll++
		case "k", "up":
			v.scroll--
		case "pgdown", "space", "ctrl+d":
			v.scroll += room
		case "pgup", "b", "ctrl+u":
			v.scroll -= room
		case "g":
			v.scroll = 0
		case "G":
			v.scroll = len(v.lines())
		case "ctrl+c":
			return v, tea.Quit
		default:
			pos := v.m.Pos
			if v.m.Key(k) {
				return v, tea.Quit
			}
			if v.m.Pos != pos {
				v.scroll = 0
			}
		}
	}
	// A resize, a new background, or a new status can shorten the note or
	// grow the header.
	v.scroll = max(min(v.scroll, len(v.lines())-max(v.room(), 1)), 0)
	return v, nil
}

// header is the fixed part of the step view: position, heading, and status.
func (v *view) header() []string {
	it := v.m.Cur()
	top := v.m.Label() + "  "
	if it.File == "" {
		top = bold.Render(top + safe(v.title))
	} else {
		top = bold.Render(top + safe(it.Ref()))
		if v.title != "" {
			top += faint.Render("  " + safe(v.title))
		}
	}
	lines := []string{top}
	if it.Invalid != "" {
		lines = append(lines, v.wrap(removed, it.Invalid)...)
	}
	if v.m.Status != "" {
		lines = append(lines, v.wrap(faint, v.m.Status)...)
	}
	return lines
}

// room is the number of note lines that fit under the header and above the
// blank line and help.
func (v *view) room() int { return v.height - len(v.header()) - 2 }

// lines is the rendered note of the current step. It renders again only when
// the step, width, or background changes.
func (v *view) lines() []string {
	if k := (noteKey{v.m.Pos, v.width, v.light}); k != v.key || v.note == nil {
		v.key, v.note = k, renderNote(v.m.Cur().Note, cmp.Or(v.width, 80), v.light)
	}
	return v.note
}

func (v *view) View() tea.View {
	note := v.lines()
	note = note[min(v.scroll, len(note)):]
	if room := v.room(); v.height > 0 && len(note) > room {
		note = note[:max(room, 0)]
	}
	lines := append(append(v.header(), note...), "", faint.Render(help))
	if v.width > 0 {
		clip := lipgloss.NewStyle().MaxWidth(v.width)
		for i, l := range lines {
			lines[i] = clip.Render(l)
		}
	}
	tv := tea.NewView(strings.Join(lines, "\n"))
	tv.AltScreen = true
	return tv
}

// wrap renders text in style, wrapped to the pane width, as lines.
func (v *view) wrap(style lipgloss.Style, text string) []string {
	if v.width > 0 {
		style = style.Width(v.width)
	}
	return strings.Split(style.Render(safe(text)), "\n")
}

// renderNote renders a Markdown note as lines wrapped to width. Mermaid code
// blocks become text diagrams. The note is untrusted, so control characters
// go before rendering.
func renderNote(note string, width int, light bool) []string {
	if note == "" {
		return []string{}
	}
	// glamour counts a tab as width 0, so tabs go before it wraps.
	note = strings.NewReplacer("\r\n", "\n", "\t", "    ").Replace(note)
	lines := strings.Split(note, "\n")
	for i, l := range lines {
		lines[i] = safe(l)
	}
	note = mermaid(strings.Join(lines, "\n"))
	style := styles.DarkStyle
	if light {
		style = styles.LightStyle
	}
	r, err := glamour.NewTermRenderer(glamour.WithStandardStyle(style), glamour.WithWordWrap(width))
	if err == nil {
		note, err = r.Render(note)
	}
	if err != nil {
		note += "\n\nmarkdown: " + err.Error()
	}
	return strings.Split(strings.TrimRight(sgrOnly(note), "\n"), "\n")
}

// sgrOnly keeps text, newlines, and SGR (color and style) sequences, and drops
// all other escape sequences and control characters. goldmark decodes
// character references such as "&#27;" after safe, and glamour writes OSC 8
// links, so the rendered note needs this second pass.
func sgrOnly(s string) string {
	var b strings.Builder
	var state byte
	for len(s) > 0 {
		seq, _, n, st := ansi.DecodeSequence(s, state, nil)
		s, state = s[n:], st
		switch {
		case seq == "\n" || isSGR(seq):
			b.WriteString(seq)
		case seq[0] == ansi.ESC || (seq[0] >= 0x80 && seq[0] <= 0x9f) || strings.ContainsFunc(seq, unicode.IsControl):
		default:
			b.WriteString(seq)
		}
	}
	return b.String()
}

func isSGR(seq string) bool {
	params, ok := strings.CutPrefix(seq, "\x1b[")
	params, ok2 := strings.CutSuffix(params, "m")
	return ok && ok2 && strings.Trim(params, "0123456789;:") == ""
}

// mermaid replaces each mermaid code block in md with a text block of the
// drawn diagram. When a diagram cannot be drawn, the block stays and the
// reason follows it.
func mermaid(md string) string {
	var out, src []string
	in := false
	for l := range strings.Lines(md) {
		l = strings.TrimSuffix(l, "\n")
		switch t := strings.TrimSpace(l); {
		case !in && t == "```mermaid":
			in, src = true, nil
		case in && t == "```":
			in = false
			if d, err := drawMermaid(strings.Join(src, "\n")); err != nil {
				out = append(out, "```mermaid")
				out = append(out, src...)
				out = append(out, "```", "", "*mermaid: "+safe(err.Error())+"*")
			} else {
				out = append(out, "```text", d, "```")
			}
		case in:
			src = append(src, l)
		default:
			out = append(out, l)
		}
	}
	if in {
		out = append(append(out, "```mermaid"), src...)
	}
	return strings.Join(out, "\n")
}

func drawMermaid(src string) (d string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("cannot draw diagram: %v", r)
		}
	}()
	d, err = render.RenderDiagram(src, diagram.DefaultConfig())
	// mermaid-ascii colors classDef nodes with escape codes.
	return strings.TrimRight(ansi.Strip(d), "\n"), err
}

// safe replaces control characters other than tab, and bidi embedding,
// override, and isolate characters, with U+FFFD, so tour text cannot send
// escape sequences to the terminal or reorder the text it shows.
func safe(s string) string {
	return strings.Map(func(r rune) rune {
		if r != '\t' && unicode.IsControl(r) || r >= 0x202A && r <= 0x202E || r >= 0x2066 && r <= 0x2069 {
			return unicode.ReplacementChar
		}
		return r
	}, s)
}
