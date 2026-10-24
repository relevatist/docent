// Package tui holds the terminal views: the tour pane and the install checklist.
package tui

import (
	"fmt"
	"strings"
	"unicode"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/relevatist/zide/internal/tour"
)

var (
	bold    = lipgloss.NewStyle().Bold(true)
	faint   = lipgloss.NewStyle().Faint(true)
	added   = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	removed = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	header  = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true)
)

const help = "n next  p prev  o open  s send  q quit  j/k scroll"

// Run enters the first step and runs the tour until the user quits.
func Run(m *tour.Model, title string) error {
	m.Start()
	_, err := tea.NewProgram(&view{m: m, title: title}).Run()
	return err
}

type view struct {
	m             *tour.Model
	title         string
	width, height int
	scroll        int
}

func (v *view) Init() tea.Cmd { return nil }

func (v *view) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width, v.height = msg.Width, msg.Height
	case tea.KeyPressMsg:
		switch k := msg.String(); k {
		case "j", "down":
			v.scroll = min(v.scroll+1, max(len(v.m.Cur().Body)-1, 0))
		case "k", "up":
			v.scroll = max(v.scroll-1, 0)
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
	return v, nil
}

func (v *view) View() tea.View {
	it := v.m.Cur()
	top := []string{bold.Render(v.m.Label() + "  " + safe(it.Heading()))}
	if v.title != "" {
		top[0] += faint.Render("  " + safe(v.title))
	}
	if it.Note != "" {
		top = append(top, v.wrap(lipgloss.NewStyle(), it.Note)...)
	}
	if it.Invalid != "" {
		top = append(top, v.wrap(removed, it.Invalid)...)
	}
	if v.m.Status != "" {
		top = append(top, v.wrap(faint, v.m.Status)...)
	}
	top = append(top, "")

	var body []string
	for _, l := range it.Body[min(v.scroll, len(it.Body)):] {
		body = append(body, line(l))
	}
	if room := v.height - len(top) - 1; v.height > 0 && len(body) > room {
		body = body[:max(room, 0)]
	}
	lines := append(append(top, body...), faint.Render(help))
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

func line(l tour.Line) string {
	text := strings.ReplaceAll(safe(l.Text), "\t", "    ")
	num := "     "
	if l.Num > 0 {
		num = fmt.Sprintf("%4d ", l.Num)
	}
	switch l.Kind {
	case tour.Header:
		return header.Render(text)
	case tour.Added:
		return added.Render(num + "+ " + text)
	case tour.Removed:
		return removed.Render(num + "- " + text)
	}
	return faint.Render(num) + "  " + text
}

// safe replaces control characters other than tab with U+FFFD, so tour and
// file text cannot send escape sequences to the terminal.
func safe(s string) string {
	return strings.Map(func(r rune) rune {
		if r != '\t' && unicode.IsControl(r) {
			return unicode.ReplacementChar
		}
		return r
	}, s)
}
