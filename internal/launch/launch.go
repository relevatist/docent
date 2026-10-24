// Package launch builds the workstream layout and the zellij argv that opens it.
package launch

import (
	"cmp"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/relevatist/zide/internal/zellij"
)

const layout = `layout {
    default_tab_template {
        pane size=1 borderless=true {
            plugin location="zellij:tab-bar"
        }
        children
        pane size=2 borderless=true {
            plugin location="zellij:status-bar"
        }
    }
    tab {
        pane split_direction="vertical" {
            pane name="editor" command="hx" size="60%%" focus=true
            pane split_direction="horizontal" {
                pane name="harness" command=%s%s
                pane name="shell"
            }
        }
    }
}
`

// Layout returns the workstream KDL layout. harness is split on spaces into
// a command and its args; shell quoting is not supported.
func Layout(harness string) (string, error) {
	f := strings.Fields(harness)
	if len(f) == 0 {
		return "", errors.New("harness command is empty")
	}
	args := ""
	if len(f) > 1 {
		q := make([]string, len(f)-1)
		for i, a := range f[1:] {
			q[i] = quote(a)
		}
		args = " {\n                    args " + strings.Join(q, " ") + "\n                }"
	}
	return fmt.Sprintf(layout, quote(f[0]), args), nil
}

func quote(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}

// Attach runs zellij with args in dir, attached to the terminal.
type Attach func(dir string, args ...string) error

// Run opens the workstream for the absolute dir: a new tab when inZellij,
// else a session. Both are called name, or the dir basename if name is empty. If that session is live, the tab is added
// to it and zide attaches; if it is exited, its record is deleted and a new session starts. It warns on w when zellij is below 0.45.
func Run(r zellij.Runner, attach Attach, w io.Writer, dir, name, harness string, inZellij bool) (err error) {
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		return fmt.Errorf("%s: not a directory", dir)
	}
	kdl, err := Layout(harness)
	if err != nil {
		return err
	}
	if out, _ := r.Run("--version"); OldVersion(string(out)) {
		if _, err := fmt.Fprintf(w, "zide: warning: zide needs zellij 0.45 or later, found %q\n", strings.TrimSpace(string(out))); err != nil {
			return err
		}
	}
	name = cmp.Or(name, filepath.Base(dir))
	tab := []string{"action", "new-tab", "--layout-string", kdl, "--name", name, "--cwd", dir}
	if inZellij {
		_, err := r.Run(tab...)
		return err
	}
	out, _ := r.Run("list-sessions", "--no-formatting")
	found, exited := sessionState(string(out), name)
	if exited {
		// An exited session can not take a new tab, and resurrecting it would skip the zide layout.
		if _, err := r.Run("delete-session", name); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "zide: removed exited session record %q\n", name); err != nil {
			return err
		}
	}
	if !found || exited {
		// `zellij --session --layout-string` only adds a tab to an existing session.
		f, err := os.CreateTemp("", "zide-layout-*.kdl")
		if err != nil {
			return err
		}
		defer func() { err = errors.Join(err, os.Remove(f.Name())) }()
		if _, err := f.WriteString(kdl); err != nil {
			return err
		}
		if err := f.Close(); err != nil {
			return err
		}
		return attach(dir, "--session", name, "--new-session-with-layout", f.Name())
	}
	if _, err := r.Run(append([]string{"--session", name}, tab...)...); err != nil {
		return err
	}
	return attach(dir, "attach", name)
}

// sessionState reports whether `zellij list-sessions --no-formatting` output
// lists the session name, and whether that session is exited.
func sessionState(out, name string) (found, exited bool) {
	for line := range strings.Lines(out) {
		if f := strings.Fields(line); len(f) > 0 && f[0] == name {
			return true, strings.Contains(line, "(EXITED")
		}
	}
	return false, false
}

// OldVersion reports whether `zellij --version` output is below 0.45 or unparsable.
func OldVersion(out string) bool {
	_, v, _ := strings.Cut(strings.TrimSpace(out), " ")
	major, rest, _ := strings.Cut(v, ".")
	minor, _, _ := strings.Cut(rest, ".")
	ma, err1 := strconv.Atoi(major)
	mi, err2 := strconv.Atoi(minor)
	if err1 != nil || err2 != nil {
		return true
	}
	return ma == 0 && mi < 45
}
