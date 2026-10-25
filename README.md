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

Requirements: Zellij 0.45+, Helix 25.07+, Go (to build), fzf (for `zide new`).

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
- `zide new` shows the directories in fzf: the recent directories, then the git repos and worktrees under `roots`, then each directory under `~` (hidden directories, `node_modules` and `~/Library` skipped). With an empty query, the recent directories come first. When you type, fzf ranks the results. The list fills while the search runs. If you typed a path to a directory that exists (it starts with `/`, `~/`, `./` or `../`, or is `~`, `.` or `..`), `enter` opens it, also when other directories match. `enter` opens the tab with the harness of the current tab (or the config `harness`). `alt-enter` first asks for the harness: pick one, or type a command with args. Typed text with a space is always used as the command. `esc` cancels.
- On macOS, the first `zide new` can show privacy prompts for folders such as Documents, Desktop and Downloads, because zide lists the directories under `~`.
- If the directory is already open in a tab, `zide new` asks if you want to go to that tab or open another tab (`app-2`, `app-3`, ...). zide finds that tab from the working directory of its editor pane. After `:cd` in Helix, zide does not find it.
- zide keeps the recent directories in `~/.local/state/zide/recent.toml` (or `$XDG_STATE_HOME/zide/recent.toml`).

Config file `~/.config/zide/config.toml` (all keys optional):

```toml
harness = "claude"         # harness command or alias; --harness overrides it for one tab
focus_on_send = true       # move focus to the harness after `zide send`
roots = ["~/src", "~/wt"]  # `zide new` lists the git repos and worktrees here before other directories

[harnesses.omp-sandbox]    # harness alias: use `--harness omp-sandbox`
command = "sandbox-exec -f ~/omp.sb omp"
kind = "omp"               # optional; default: base name of the first word of command
```

`kind` is the harness that the command runs. `zide send ref` uses it to format references, so a wrapped harness still gets `@` references. `kind` must be one word.

zide expands a leading `~` in `roots` and in each word of a harness command (an alias `command`, the `harness` key, `--harness`, or a command typed in the `zide new` harness list). Zellij runs the command without a shell, so other shell syntax does not work.

A root can be a symlink. zide does not follow symlinks below a root.

## Helix keys

Add the tables from [`assets/helix/keys.toml`](assets/helix/keys.toml) to `~/.config/helix/config.toml`:

| Key | Action |
|-----|--------|
| `space z r` | paste a reference to the selected lines into the harness input |
| `space z s` | paste the selected text into the harness input |

Limit: Helix gives `space z r` the buffer name through the shell, and has no way to quote it for the shell. The binding puts the name in single quotes, so a name that contains `'` is not safe with this binding.

## Zellij key

`zide install zellij` adds the key to your Zellij config (see [Harness setup](#harness-setup)).

To add it by hand: add the `keybinds` block from [`assets/zellij/keys.kdl`](assets/zellij/keys.kdl) to `~/.config/zellij/config.kdl`. If your config already has a `keybinds` block, put the `shared_except "locked"` block in it. Zellij uses only the first `keybinds` block in the file.

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
```

`zide send` never submits the harness input. You submit it. This requires a harness that enables bracketed paste (Claude Code, omp, and codex do); otherwise a newline in the sent text can submit it.

WARNING: `zide reload` discards unsaved changes in all editor buffers.

### Tours

A tour is a JSON file. Each step has an optional `file`, optional `lines` (needs `file`; without it the step covers the whole file), and an optional `note`. A step needs a `file`, a `note`, or both.

```json
{"title": "Retry",
 "steps": [{"note": "## Overview\n\n`Do()` now retries."},
           {"file": "a.go", "lines": [1, 40], "note": "New `Retry` type"}]}
```

The editor shows the step's file at its first line. A step without a file does not move the editor. The tour pane shows the step's position, its file and lines (or the tour title), and its note. The note is Markdown. `diff` code blocks show added and removed lines in color, and `mermaid` code blocks (flowchart, sequence, and ER diagrams) show as text diagrams. If zide cannot draw a diagram, it shows the source and the reason. zide ignores unknown fields, such as `base` from older tours.

Keys in the tour pane:

| Key | Action |
|-----|--------|
| `n` | next step |
| `p` | previous step |
| `o` | open the step in the editor again |
| `s` | send the step as a reference to the harness |
| `j` / `k` (`down` / `up`) | scroll one line |
| `pgdown` / `space` / `ctrl+d` | scroll one page down |
| `pgup` / `b` / `ctrl+u` | scroll one page up |
| `g` / `G` | go to the top / bottom of the note |
| `q` | close the tour |

## Harness setup

```sh
zide install                 # choose harnesses from a checklist
zide install claude omp      # or name them
zide install zellij          # add the Alt w key to the Zellij config
zide install codex --force   # overwrite a skill or extension file that you changed
```

| Harness | What zide installs |
|---------|--------------------|
| `claude` | The `zide@zide` Claude Code plugin: the agent skill and a reload hook. zide writes it to `~/.local/share/zide/claude-plugin/` (or `$XDG_DATA_HOME/zide/claude-plugin/`) and installs it with the `claude plugin` CLI. Restart Claude Code after the install. |
| `codex` | The agent skill: `~/.agents/skills/zide/SKILL.md` |
| `omp` | The agent skill (same path as codex; oh-my-pi reads it) and a reload extension: `~/.omp/agent/extensions/zide-reload.ts` |
| `zellij` | The `Alt w` key (see [Zellij key](#zellij-key)) in the Zellij config file that Zellij loads: `$ZELLIJ_CONFIG_FILE`, else `$ZELLIJ_CONFIG_DIR/config.kdl`, else `config.kdl` in the first of `~/.config/zellij` and the platform config dir that exists, else a new `~/.config/zellij/config.kdl`. If only `/etc/zellij` exists, zide refuses and tells you to copy its `config.kdl` to `~/.config/zellij` first. |

The checklist starts with an item checked when its command is on `PATH`. zide checks everything before it changes anything: if a command is missing, a file has changes and you did not give `--force`, or zide cannot add the Zellij key, it installs nothing and exits non-zero. Run `zide install` again after you upgrade zide.

For `zellij`, zide puts the key between `// zide: begin` and `// zide: end` lines on the line after the first `keybinds ... {` line, or appends a `keybinds` block if there is none. It changes nothing else in the file, but it adds a newline at the end of the file before it appends a block, if there is no newline there. It keeps a byte order mark and CRLF line endings. It does not install the key, and tells you why, when:

- your config already binds `Alt w` (in any case, for example `"alt w"`), also when it is the snippet that you added by hand (remove it, and zide adds it between the markers). zide does not find rare forms, for example `"Alt\tw"`, a quoted `"bind"`, a `\"` before the key, or a `\` line continuation,
- you changed the lines between the markers (use `--force` to replace them),
- there is more than one `// zide: begin` or `// zide: end` line, they are out of order, or one of them is missing (fix the file by hand),
- it cannot find where to put the key (for example the `{` of `keybinds` is on the next line, another node is before `keybinds` on its line, or a comment or string opens on the `keybinds` line),
- it cannot write to the directory of the config,
- the config has other hard links (zide would split them; add the key by hand, or replace the hard link with a symlink), or
- `zellij setup --check` fails on the new config.

The first time zide changes the config, it keeps the old file as `config.kdl.zide.bak`. zide writes the backup to a temp file and links it into place, so a failed write leaves no partial backup. If `config.kdl.zide.bak` exists, zide does not replace it, because it holds the config from before zide. If the config is a symlink, zide changes the file that it points to.

The agent skill tells the harness how to use `zide open`, `zide reload`, and `zide tour`, and how to write tour notes. The reload hook and extension run `zide reload --quiet` each time the harness edits a file, so the editor shows the new content. Outside a workstream, `zide reload --quiet` does nothing and exits zero, so it never fails an edit. codex has no reload adapter; the skill tells it to run `zide reload`.

To install the Claude Code plugin without the zide installer:

```sh
claude plugin marketplace add relevatist/zide
claude plugin install zide@zide
```

To try the oh-my-pi extension for one session: `omp -e assets/omp/zide-reload.ts`. The extension reloads after the `edit`, `write`, and `resolve` tools.

## License

```
Copyright 2026 Relevatist Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
```
