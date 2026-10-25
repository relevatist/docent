# Workstreams

A workstream is one Zellij tab with an editor, a harness, and a shell.

## Open a workstream

```sh
docent                       # workstream for the current directory
docent ~/src/app             # workstream for ~/src/app
docent --harness omp ~/src/app
docent --name app-retry ~/wt/retry/app
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
| `editor` | left, 60% of the width | `hx` |
| `harness` | top right | the harness command |
| `shell` | bottom right | your shell |
| `tour` | stacked with `shell` | the tour pane, while a tour runs |

docent commands find the panes in their tab by these names. If you rename a pane, the commands cannot find it.

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
- On macOS, the first `docent new` can show privacy prompts for folders such as Documents, Desktop and Downloads, because docent lists the directories under `~`.

## Recent directories

docent keeps the last 20 directories that you opened in `~/.local/state/docent/recent.toml` (or `$XDG_STATE_HOME/docent/recent.toml`).
