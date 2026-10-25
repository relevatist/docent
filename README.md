# zide

zide joins an editor (Helix), a coding harness (Claude Code, oh-my-pi, codex, ...), and a shell in one Zellij tab. The harness can show you code in the editor and walk you through its changes one step at a time.

```
+------------------------+-----------+
|                        | harness   |
| editor (hx)            +-----------+
|                        | shell     |
|                        | [tour]    |
+------------------------+-----------+
```

Requirements: Zellij 0.45+, Helix 25.07+, git, Go (to build).

## Install

```sh
make install
```

This installs `zide` into `$(go env GOPATH)/bin` (or `$GOBIN`). Make sure that directory is on your `PATH`.

## Open a workstream

```sh
zide                       # workstream for the current directory
zide ~/src/app             # workstream for ~/src/app
zide --harness omp ~/src/app
zide --name app-retry ~/wt/retry/app
zide new                   # inside Zellij: pick a directory and a harness
```

- Inside Zellij, zide opens a new tab. Outside Zellij, it starts a Zellij session.
- Outside Zellij, if a session with that name has exited (for example after you quit Zellij), zide deletes its record and starts a new session.
- The tab (or session) name is `--name`, or the directory basename.
- zide opens only directories that exist. It never creates directories or git worktrees.
- For more workstreams, open one tab per project or worktree. Each zide command acts only on the panes in its own tab.
- `zide new` shows a picker. Type to filter the directories: the recent directories come first, then the git repos and worktrees under `roots` (they show while the search runs). If you type a path to a directory (it starts with `/`, `~/`, `./` or `../`), it comes first in the list. `tab` moves to the harness field, and `up`/`down` there cycles the harnesses and aliases. `enter` opens the tab. `esc` cancels.
- If the directory is already open in a tab, `zide new` asks if you want to go to that tab or open another tab (`app-2`, `app-3`, ...). zide finds that tab from the working directory of its editor pane. After `:cd` in Helix, zide does not find it.
- zide keeps the recent directories in `~/.local/state/zide/recent.toml` (or `$XDG_STATE_HOME/zide/recent.toml`).

Config file `~/.config/zide/config.toml` (all keys optional):

```toml
harness = "claude"         # harness command or alias; --harness overrides it for one tab
focus_on_send = true       # move focus to the harness after `zide send`
roots = ["~/src", "~/wt"]  # where `zide new` looks for git repos and worktrees

[harnesses.omp-sandbox]    # harness alias: use `--harness omp-sandbox`
command = "sandbox-exec -f ~/omp.sb omp"
kind = "omp"               # optional; default: base name of the first word of command
```

`kind` is the harness that the command runs. `zide send ref` uses it to format references, so a wrapped harness still gets `@` references. `kind` must be one word.

zide expands a leading `~` in `roots` and in each word of a harness command (an alias `command`, the `harness` key, `--harness`, or the `zide new` harness field). Zellij runs the command without a shell, so other shell syntax does not work.

A root can be a symlink. zide does not follow symlinks below a root.

## Helix keys

Add the tables from [`assets/helix/keys.toml`](assets/helix/keys.toml) to `~/.config/helix/config.toml`:

| Key | Action |
|-----|--------|
| `space z r` | paste a reference to the selected lines into the harness input |
| `space z s` | paste the selected text into the harness input |

Limit: Helix gives `space z r` the buffer name through the shell, and has no way to quote it for the shell. The binding puts the name in single quotes, so a name that contains `'` is not safe with this binding.

## Zellij key

Add the `keybinds` block from [`assets/zellij/keys.kdl`](assets/zellij/keys.kdl) to `~/.config/zellij/config.kdl`. If your config already has a `keybinds` block, put the `shared_except "locked"` block in it. zide does not change your Zellij config.

| Key | Action |
|-----|--------|
| `Alt w` | open `zide new` in a floating pane (all modes except locked); the pane closes when `zide new` exits |

## Commands

Run these inside a workstream tab. Add `--help` to any command for details.

```sh
zide send ref src/a.go:10-20   # paste "@src/a.go:10-20 " (claude, omp) or "src/a.go:10-20 " (others)
echo "some text" | zide send text
zide open src/main.go:42       # show the file at line 42 in the editor
zide open --focus README.md    # same, and move focus to the editor
zide reload                    # reload all editor buffers from disk
zide tour /tmp/tour.json       # start a guided tour of a change
zide tour - < /tmp/tour.json   # read the tour from standard input
zide diff                      # walk through `git diff`, one step per hunk
zide diff main                 # takes `git diff` arguments
```

`zide send` never submits the harness input. You submit it. This requires a harness that enables bracketed paste (Claude Code, omp, and codex do); otherwise a newline in the sent text can submit it.

WARNING: `zide reload` discards unsaved changes in all editor buffers.

### Tours

A tour is a JSON file. `base` is a git revision (default `HEAD`); staged and unstaged changes both count. A step without `lines` covers the whole file.

```json
{"title": "Retry", "base": "main",
 "steps": [{"file": "a.go", "lines": [1, 40], "note": "New type"}]}
```

zide adds a final step, `Not covered by this tour`, that lists the hunks no step covers.

Keys in the tour pane:

| Key | Action |
|-----|--------|
| `n` | next step |
| `p` | previous step |
| `o` | open the step in the editor again |
| `s` | send the step as a reference to the harness |
| `q` | close the tour |

## Harness setup

```sh
zide install                 # choose harnesses from a checklist
zide install claude omp      # or name them
zide install codex --force   # overwrite a skill or extension file that you changed
```

| Harness | What zide installs |
|---------|--------------------|
| `claude` | The `zide@zide` Claude Code plugin: the agent skill and a reload hook. zide writes it to `~/.local/share/zide/claude-plugin/` (or `$XDG_DATA_HOME/zide/claude-plugin/`) and installs it with the `claude plugin` CLI. Restart Claude Code after the install. |
| `codex` | The agent skill: `~/.agents/skills/zide/SKILL.md` |
| `omp` | The agent skill (same path as codex; oh-my-pi reads it) and a reload extension: `~/.omp/agent/extensions/zide-reload.ts` |

The checklist starts with a harness checked when its command is on `PATH`. zide checks everything before it changes anything: if a harness command is missing, or a file has changes and you did not give `--force`, it installs nothing and exits non-zero. Run `zide install` again after you upgrade zide.

The agent skill tells the harness how to use `zide open`, `zide reload`, `zide tour`, and `zide diff`. The reload hook and extension run `zide reload --quiet` each time the harness edits a file, so the editor shows the new content. Outside a workstream, `zide reload --quiet` does nothing and exits zero, so it never fails an edit. codex has no reload adapter; the skill tells it to run `zide reload`.

To install the Claude Code plugin without the zide installer:

```sh
claude plugin marketplace add relevatist/zide
claude plugin install zide@zide
```

To try the oh-my-pi extension for one session: `omp -e assets/omp/zide-reload.ts`. The extension reloads after the `edit`, `write`, and `resolve` tools.
