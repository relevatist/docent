// Package launch builds the workstream layout and the zellij argv that opens it.
package launch

import (
	"cmp"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/relevatist/docent/internal/config"
	"github.com/relevatist/docent/internal/panes"
	"github.com/relevatist/docent/internal/zellij"
)

type fields struct {
	Name    string
	harness string
	used    bool
}

func (f *fields) Harness() string {
	f.used = true
	return f.harness
}

// Layout renders the KDL layout template tmpl for a tab called name. The
// template gets {{.Name}}, the quoted name, and {{.Harness}}, the harness pane
// command and args. layoutPath names the template in errors; "" means the
// built-in layout. The harness command is split on spaces into a command and
// its args; shell quoting is not supported. When the harness kind differs from
// the base name of the command, the pane runs through env with
// DOCENT_HARNESS_KIND set, so the pane command shows the kind. With a
// layoutPath, the pane always runs through env, with DOCENT_HARNESS_KIND and
// DOCENT_LAYOUT, the path URL-escaped.
func Layout(tmpl string, harness config.Harness, layoutPath, name string) (string, error) {
	f := strings.Fields(harness.Command)
	if len(f) == 0 {
		return "", errors.New("harness command is empty")
	}
	kind := cmp.Or(harness.Kind, filepath.Base(f[0]))
	switch {
	case layoutPath != "":
		f = append([]string{"/usr/bin/env", panes.KindVar + "=" + kind, panes.LayoutVar + "=" + (&url.URL{Path: layoutPath}).EscapedPath()}, f...)
	case kind != filepath.Base(f[0]):
		f = append([]string{"/usr/bin/env", panes.KindVar + "=" + kind}, f...)
	}
	data := &fields{Name: quote(name), harness: "command=" + quote(f[0])}
	if len(f) > 1 {
		q := make([]string, len(f)-1)
		for i, a := range f[1:] {
			q[i] = quote(a)
		}
		data.harness += " {\n                args " + strings.Join(q, " ") + "\n            }"
	}
	file := cmp.Or(layoutPath, "built-in layout")
	t, err := template.New(file).Option("missingkey=error").Parse(tmpl)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	if err := t.Execute(&b, data); err != nil {
		return "", fmt.Errorf("%w (fields: {{.Name}}, {{.Harness}})", err)
	}
	if !data.used {
		return "", fmt.Errorf("%s: layout does not use {{.Harness}}", file)
	}
	kdl := b.String()
	for _, p := range []string{"editor", "harness", "shell"} {
		if !strings.Contains(kdl, `name="`+p+`"`) {
			return "", fmt.Errorf("%s: layout has no pane named %q", file, p)
		}
	}
	return kdl, nil
}

func quote(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}

// Attach runs zellij with args in dir, attached to the terminal.
type Attach func(dir string, args ...string) error

// Run opens the workstream for the absolute dir with the layout template tmpl
// (see Layout): a new tab when inZellij, else a session. Both are called name,
// or the dir basename if name is empty. If that session is live, the tab is
// added to it and docent attaches; if it is exited, its record is deleted and
// a new session starts. It warns on w when zellij is below 0.45.
func Run(r zellij.Runner, attach Attach, w io.Writer, dir, name string, harness config.Harness, tmpl, layoutPath string, inZellij bool) (err error) {
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		return fmt.Errorf("%s: not a directory", dir)
	}
	name = cmp.Or(name, filepath.Base(dir))
	kdl, err := Layout(tmpl, harness, layoutPath, name)
	if err != nil {
		return err
	}
	if out, _ := r.Run("--version"); OldVersion(string(out)) {
		if _, err := fmt.Fprintf(w, "docent: warning: docent needs zellij 0.45 or later, found %q\n", strings.TrimSpace(string(out))); err != nil {
			return err
		}
	}
	tab := []string{"action", "new-tab", "--layout-string", kdl, "--name", name, "--cwd", dir}
	if inZellij {
		_, err := r.Run(tab...)
		return err
	}
	out, _ := r.Run("list-sessions", "--no-formatting")
	found, exited := sessionState(string(out), name)
	if exited {
		// An exited session can not take a new tab, and resurrecting it would skip the docent layout.
		if _, err := r.Run("delete-session", name); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "docent: removed exited session record %q\n", name); err != nil {
			return err
		}
	}
	if !found || exited {
		// `zellij --session --layout-string` only adds a tab to an existing session.
		f, err := os.CreateTemp("", "docent-layout-*.kdl")
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
