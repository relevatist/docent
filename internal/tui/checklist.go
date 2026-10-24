package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

const checklistHelp = "space toggle  enter confirm  esc cancel  j/k move"

// Option is one checklist line.
type Option struct {
	Label, Detail string
	Checked       bool
}

// Pick shows a checklist and returns the indexes of the checked options.
// ok is false when the user cancels.
func Pick(title string, opts []Option) (picked []int, ok bool, err error) {
	c := &checklist{title: title, opts: opts}
	if _, err := tea.NewProgram(c).Run(); err != nil {
		return nil, false, err
	}
	picked, ok = c.result()
	return picked, ok, nil
}

type checklist struct {
	title     string
	opts      []Option
	cur       int
	confirmed bool
}

func (c *checklist) Init() tea.Cmd { return nil }

func (c *checklist) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return c, nil
	}
	switch k.String() {
	case "j", "down":
		c.cur = min(c.cur+1, len(c.opts)-1)
	case "k", "up":
		c.cur = max(c.cur-1, 0)
	case "space", "x":
		c.opts[c.cur].Checked = !c.opts[c.cur].Checked
	case "enter":
		c.confirmed = true
		return c, tea.Quit
	case "esc", "q", "ctrl+c":
		return c, tea.Quit
	}
	return c, nil
}

func (c *checklist) View() tea.View {
	lines := []string{bold.Render(c.title)}
	for i, o := range c.opts {
		cursor, box := "  ", "[ ]"
		if i == c.cur {
			cursor = "> "
		}
		if o.Checked {
			box = "[x]"
		}
		lines = append(lines, cursor+box+" "+o.Label+faint.Render("  "+o.Detail))
	}
	if !c.confirmed {
		lines = append(lines, faint.Render(checklistHelp))
	}
	return tea.NewView(strings.Join(lines, "\n") + "\n")
}

func (c *checklist) result() ([]int, bool) {
	if !c.confirmed {
		return nil, false
	}
	var picked []int
	for i, o := range c.opts {
		if o.Checked {
			picked = append(picked, i)
		}
	}
	return picked, true
}
