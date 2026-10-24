package tui

import (
	"slices"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func press(c *checklist, keys ...string) {
	for _, k := range keys {
		var msg tea.KeyPressMsg
		switch k {
		case "space":
			msg = tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
		case "enter":
			msg = tea.KeyPressMsg{Code: tea.KeyEnter}
		case "esc":
			msg = tea.KeyPressMsg{Code: tea.KeyEscape}
		default:
			msg = tea.KeyPressMsg{Code: rune(k[0]), Text: k}
		}
		c.Update(msg)
	}
}

func newChecklist() *checklist {
	return &checklist{opts: []Option{{Label: "a", Checked: true}, {Label: "b"}, {Label: "c", Checked: true}}}
}

func TestChecklistToggleAndConfirm(t *testing.T) {
	c := newChecklist()
	press(c, "space", "j", "space", "j", "j", "enter")
	picked, ok := c.result()
	if !ok || !slices.Equal(picked, []int{1, 2}) {
		t.Errorf("result = %v, %v; want [1 2], true", picked, ok)
	}
}

func TestChecklistConfirmDefaults(t *testing.T) {
	c := newChecklist()
	press(c, "enter")
	if picked, ok := c.result(); !ok || !slices.Equal(picked, []int{0, 2}) {
		t.Errorf("result = %v, %v; want [0 2], true", picked, ok)
	}
}

func TestChecklistCancel(t *testing.T) {
	for _, k := range []string{"esc", "q"} {
		c := newChecklist()
		press(c, "space", k)
		if picked, ok := c.result(); ok || picked != nil {
			t.Errorf("%s: result = %v, %v; want nil, false", k, picked, ok)
		}
	}
}
