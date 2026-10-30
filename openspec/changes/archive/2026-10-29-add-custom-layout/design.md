# Design

## Context

See proposal.md - Why. Current state:

- `launch.Layout` (`internal/launch/launch.go`) fills a `fmt.Sprintf` KDL string with the quoted tab name, the harness command, and an optional `args` block. `launch.Run` sends it to `new-tab --layout-string`, or writes it to a temp file for `--new-session-with-layout`.
- The harness pane runs through `/usr/bin/env DOCENT_HARNESS_KIND=<kind>` only when the kind differs from the base name of the command.
- `list-panes` shows `terminal_command` as the argv joined with single spaces and no quotes. `panes.unwrap` reads exactly one `DOCENT_HARNESS_KIND=` word after `env`.
- `send.PasteRef` uses `panes.Marker`. Without a marker, it takes the kind from the base name of the raw pane command. `new.currentHarness` uses `panes.Unwrap` and `panes.Marker`.
- `assets/zellij/keys.kdl` is embedded in `assets` and linked from the docs. The built-in layout can follow the same pattern.

## Goals / Non-Goals

**Goals:**
- One render path for the built-in and custom layouts.
- No new dependencies.

**Non-Goals:**
- Parse KDL. Zellij reports KDL syntax errors.
- Layout aliases, a `builtin` keyword, a layout field in the picker, or a command that prints the built-in layout.
- Support an editor other than Helix. A custom layout can change the editor pane command, but `docent open` and `reload` still send Helix keys.

## Decisions

### Layout is a `text/template`

`assets/zellij/layout.kdl`, embedded as `assets.ZellijLayout`:

```kdl
layout {
    default_tab_template {
        pane size=1 borderless=true {
            plugin location="zellij:tab-bar"
        }
        children
        pane size=2 borderless=true {
            plugin location="zellij:status-bar"
        }
    }
    tab name={{.Name}} {
        pane split_direction="vertical" {
            pane name="harness" size="50%" focus=true {{.Harness}}
            pane split_direction="horizontal" {
                pane name="shell"
                pane name="editor" command="hx"
            }
        }
    }
}
```

`launch.Layout` takes the template text:

```go
func Layout(tmpl string, harness config.Harness, layoutPath, name string) (string, error)
```

The data is a struct whose `Harness` is a method, so the render can see if the template used it:

```go
type fields struct {
    Name    string // quoted
    harness string
    used    bool
}

func (f *fields) Harness() string { f.used = true; return f.harness }
```

`{{.Harness}}` gives `command="claude"`, or `command="..." {\n args ...\n}` when there are args. Pane properties must come before `{{.Harness}}`, because it can end with a child block.

Use `Option("missingkey=error")`. An unknown field (`{{.Nmae}}`) on a struct is already an exec error.

Alternatives: KDL parsing to inject the command (needs a library or more of the hand-written scanner in `install_zellij.go`); `fmt` placeholders (no names, and no way to report a field that is not used).

### Checks after the render

`Layout` returns an error that names the file when: the template does not parse or run; `used` is false; or the output has no `name="editor"`, `name="harness"`, or `name="shell"` (`strings.Contains`). The built-in template goes through the same checks, and a unit test covers it. `launch.Run` calls `Layout` before any `zellij` call, so a bad layout opens nothing.

The text match can find a name in a KDL comment. Accepted: the check is a guard against mistakes, not a parser.

### Layout path resolution

- `config.Layout` (`toml:"layout"`): `Load` expands `~` with `expandHome`. A relative path is joined to `filepath.Dir(Path())`.
- `--layout`: `~` is expanded, then `filepath.Abs` gives the path relative to the cwd.
- `launchDir` gets the already-resolved absolute path (or `""` for the built-in layout) and reads the file. A read error names the path.

`launchDir` takes the layout path in the same way it takes `harness`: the flag value, or the value that `docent new` found. `""` from the flag means "use the config".

`docent new` must tell "built-in" apart from "use the config". `currentLayout(r, c)` returns `(path string, builtin bool)`:

| Harness pane | Result |
|--------------|--------|
| `DOCENT_LAYOUT` marker | marker path |
| found, no marker | built-in |
| not found | `c.Layout`, else built-in |

### Markers in the `env` wrapper

The harness pane uses the wrapper when the kind differs from the base name of the command, or when the tab uses a custom layout. When the wrapper is present, it always has `DOCENT_HARNESS_KIND`, so `send.PasteRef` and `currentHarness` still work without changes to their fallback:

```kdl
pane name="harness" size="50%" focus=true command="/usr/bin/env" {
    args "DOCENT_HARNESS_KIND=claude" "DOCENT_LAYOUT=/Users/thane/my%20layouts/pair.kdl" "claude"
}
```

`terminal_command`: `/usr/bin/env DOCENT_HARNESS_KIND=claude DOCENT_LAYOUT=/Users/thane/my%20layouts/pair.kdl claude`.

The path goes through `(&url.URL{Path: p}).EscapedPath()`, so it is one word with no spaces. `url.PathEscape` is not used, because it escapes `/` as `%2F`. `url.PathUnescape` reads it back. The harness process sees the escaped value. This is accepted, because no consumer reads it from the environment.

`internal/panes` changes:

```go
const LayoutVar = "DOCENT_LAYOUT"

// Marker returns the kind in an `env DOCENT_HARNESS_KIND=v ... cmd` pane command.
func Marker(cmd string) (string, bool)
// LayoutMarker returns the unescaped DOCENT_LAYOUT path.
func LayoutMarker(cmd string) (string, bool)
// Unwrap returns cmd without `env` and the leading DOCENT_HARNESS_KIND and DOCENT_LAYOUT words.
func Unwrap(cmd string) string
```

`unwrap` reads the words after `env` while they set `DOCENT_HARNESS_KIND` or `DOCENT_LAYOUT`. Each value is one word.

Alternatives: a separate wrapper for the layout only (then `send` would get the kind `env`); a state file that maps tabs to layouts (gets stale when tabs close).

### Focus

The built-in layout gives `focus=true` to the harness pane. `docent open --focus` and `docent send` already move focus when needed.

## Risks / Trade-offs

- [The new default layout changes the screen for current users] -> Mention it in the docs. The old layout can be a custom layout, so the docs show it as an example.
- [Tabs opened before this change have no layout marker] -> `docent new` from them uses the built-in layout, as for any tab without a marker.
- [`{{.Harness}}` ends with a child block, so the harness pane cannot have other children] -> The docs say this. A command pane needs only `args`.
- [`demo.png` shows the old layout] -> Replace the screenshot later. It is out of scope here.
