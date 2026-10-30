# Workstreams

A workstream is one Zellij tab with an editor, a harness, and a shell.

## Open a workstream

```sh
docent                       # workstream for the current directory
docent ~/src/app             # workstream for ~/src/app
docent --harness omp ~/src/app
docent --name app-retry ~/wt/retry/app
docent --layout ~/layouts/pair.kdl ~/src/app
docent new                   # inside Zellij: pick a directory and a harness
```

- Inside Zellij, docent opens a new tab. Outside Zellij, it starts a Zellij session.
- Outside Zellij, if a session with that name is running, docent adds the tab to it and attaches.
- Outside Zellij, if a session with that name has exited (for example after you quit Zellij), docent deletes its record and starts a new session.
- The tab (or session) name is `--name`, or the directory basename.
- docent opens only directories that exist. It never creates directories or git worktrees.
- For more workstreams, open one tab per project or worktree. Each docent command acts only on the panes in its own tab.

## Layout

| Pane | Position | Runs |
|------|----------|------|
| `harness` | left, 50% of the width, has focus | the harness command |
| `shell` | top right, 50% of the height | your shell |
| `editor` | bottom right, 50% of the height | `hx` |
| `tour` | stacked with `shell` | the tour pane, while a tour runs |

The tab also has the Zellij tab bar at the top and the status bar at the bottom. The built-in layout is [`assets/zellij/layout.kdl`](../assets/zellij/layout.kdl).

The editor was on the left in earlier versions. The screenshot in the README (`demo.png`) still shows that layout.

docent commands find the panes in their tab by these names. If you rename a pane, the commands cannot find it.

## Custom layout

To use your own layout, set `layout` in the [config](config.md), or give `--layout <file>`. The flag overrides the config.

| Source | Relative path is relative to |
|--------|------------------------------|
| `--layout` | the current directory |
| config `layout` | the config directory |

docent expands a leading `~` in both.

The file is a full [Zellij layout](https://zellij.dev/documentation/layouts.html) and a Go [text template](https://pkg.go.dev/text/template). docent fills these fields:

| Field | Gives |
|-------|-------|
| `{{.Name}}` | the tab name, quoted |
| `{{.Harness}}` | the `command` property of the harness pane, and an `args` block when needed |

Put the pane properties before `{{.Harness}}`, because it can end with an `args` block. The harness pane cannot have other child nodes.

Before docent opens the tab, it checks the layout. It shows an error that names the file, and opens nothing, when:

- the template does not parse or run (for example `{{.Nmae}}`)
- the layout does not use `{{.Harness}}`
- the layout has no `name="editor"`, `name="harness"`, or `name="shell"`

docent does not check the KDL syntax. Zellij does that.

This is the old layout, with the editor on the left:

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
            pane name="editor" command="hx" size="60%" focus=true
            pane split_direction="horizontal" {
                pane name="harness" {{.Harness}}
                pane name="shell"
            }
        }
    }
}
```

The editor pane must run Helix. `docent open` and `docent reload` send Helix keys.

A tab with a custom layout keeps the absolute path of the file in the harness pane command, URL-escaped:

```text
/usr/bin/env DOCENT_HARNESS_KIND=claude DOCENT_LAYOUT=/Users/me/my%20layouts/pair.kdl claude
```

The harness process gets `DOCENT_LAYOUT` with the escaped value. A tab with the built-in layout has no `DOCENT_LAYOUT`.

## Pick a directory with `docent new`

`docent new` runs inside Zellij. The [Zellij key](keys.md#zellij-key) `Alt w` opens it in a floating pane.

It shows the directories in fzf, in this order:

1. The recent directories.
2. The git repos and worktrees under the config `roots` (see [Config](config.md)).
3. Each directory under `~`. It skips hidden directories, `node_modules`, and `~/Library`.

With an empty query, the recent directories come first. When you type, fzf ranks the results. The list fills while the search runs.

| Key | Action |
|-----|--------|
| `enter` | open the directory with the harness of the current tab (or the config `harness`) |
| `alt-enter` | first choose the harness: pick one, or type a command with args |
| `esc` | cancel |

- If you type a path to a directory that exists, `enter` opens it, also when other directories match. A path starts with `/`, `~/`, `./` or `../`, or is `~`, `.` or `..`.
- In the harness list, typed text with a space is always used as the command.
- If the directory is already open in a tab, `docent new` asks if you want to go to that tab or open another tab (`app-2`, `app-3`, ...). docent finds that tab from the working directory of its editor pane. After `:cd` in Helix, docent does not find it.
- `docent new` opens the new tab with the layout of the current tab:

  | Current tab | Layout of the new tab |
  |-------------|-----------------------|
  | harness pane with `DOCENT_LAYOUT` | that file |
  | harness pane without `DOCENT_LAYOUT` | the built-in layout, also when the config sets `layout` |
  | no harness pane | the config `layout`, else the built-in layout |

  If that file does not exist, `docent new` shows an error that names it and opens nothing.
- On macOS, the first `docent new` can show privacy prompts for folders such as Documents, Desktop and Downloads, because docent lists the directories under `~`.

## Recent directories

docent keeps the last 20 directories that you opened in `~/.local/state/docent/recent.toml` (or `$XDG_STATE_HOME/docent/recent.toml`).
