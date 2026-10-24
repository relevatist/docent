package tour

import "fmt"

// Model is the tour pane state. It moves only on Key.
type Model struct {
	Items  []Item
	Pos    int
	Open   func(path string, line int) error // shows a file in the editor
	Send   func(ref string) error            // pastes a ref into the harness
	Status string                            // last error or notice
}

func (m *Model) Cur() Item { return m.Items[m.Pos] }

func (m *Model) Label() string { return fmt.Sprintf("%d/%d", m.Pos+1, len(m.Items)) }

// Start enters the first step.
func (m *Model) Start() { m.enter() }

func (m *Model) enter() {
	m.Status = ""
	if it := m.Cur(); it.Editor {
		m.report(m.Open(it.Path, it.Line()), "")
	}
}

func (m *Model) report(err error, ok string) {
	m.Status = ok
	if err != nil {
		m.Status = "error: " + err.Error()
	}
}

// Key handles n, p, o, s, q. It returns true when the tour should close.
func (m *Model) Key(k string) bool {
	switch k {
	case "n":
		if m.Pos < len(m.Items)-1 {
			m.Pos++
			m.enter()
		}
	case "p":
		if m.Pos > 0 {
			m.Pos--
			m.enter()
		}
	case "o":
		m.enter()
	case "s":
		if it := m.Cur(); it.Editor {
			m.report(m.Send(it.Ref()), "sent "+it.Ref())
		}
	case "q":
		return true
	}
	return false
}
