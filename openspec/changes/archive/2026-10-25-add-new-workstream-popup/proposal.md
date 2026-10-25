# Proposal

## Why

To open a second workstream, the user must go to a shell pane, type `docent <path>`, and remember the path and harness. This is slow when a workstream is already open. A keybinding that opens a picker lets the user open a new workstream from any pane in a few keystrokes.

## What Changes

- Add `docent new`: an fzf picker that runs in a floating Zellij pane. The user fuzzy-filters a list of directories, optionally picks another harness (`alt-enter`), and confirms. docent then opens a workstream tab for that directory and the pane closes.
- The directory list has the recent workstream directories first, then git repos and worktrees found under the config key `roots`, then each directory under `~`.
- docent saves each launched directory in a state file, `$XDG_STATE_HOME/docent/recent.toml`.
- The harness list holds the current tab's harness, the aliases, and the built-in harnesses. Typed text that matches nothing is used as a command with args, or an alias name.
- Add harness aliases to `config.toml`: `[harnesses.<name>]` with `command` and an optional `kind`. The `harness` key and `--harness` also take alias names.
- `docent send` takes the ref format from the alias `kind`, so wrapped commands (for example a sandbox) still get `@` refs.
- docent finds open workstreams from the working directory (`pane_cwd`) of each tab's `editor` pane. If the chosen directory is already open in a tab, `docent new` asks to switch to that tab or open another one.
- Fix: outside Zellij, the new session's tab is named `Tab #1`, because the layout `tab` node has no name. The layout now names the tab.
- Ship a Zellij keybinding snippet (`Alt w`) in `assets/zellij/keys.kdl` and document it in the README.
- `docent install` gets a `zellij` item (checked when `zellij` is on `PATH`) that adds the `Alt w` bind to the user's Zellij `config.kdl` as a marked node that docent manages.

## Capabilities

### New Capabilities

- `workstream-picker`: the `docent new` popup - directory list, filter, harness choice, open-tab handling, and the Zellij keybinding.

### Modified Capabilities

- `workspace`: harness aliases in the config, the harness kind marker, the tab name for a new session, and saving recent directories on launch.
- `harness-input`: the ref format comes from the harness kind, not only from the command name.
- `harness-integration`: `docent install` can install the Zellij `Alt w` key into the user's Zellij config.

## Impact

- New code: `cmd/docent/new.go`, an fzf picker in `internal/tui`, a directory walk and recent state (new package `internal/recent` or in `internal/config`).
- Changed code: `internal/config` (aliases, `roots`), `internal/launch` (harness kind marker, tab name in the layout), `internal/send` (kind), `cmd/docent/install.go` (the `zellij` item), README, new `assets/zellij/keys.kdl`.
- The pane command shown by `list-panes` changes to `/usr/bin/env DOCENT_HARNESS_KIND=...` for the harness pane when an alias sets a `kind`. The editor pane stays `hx`.
- After `:cd` in Helix, the editor pane cwd changes, so the picker no longer detects that tab for its first directory.
- `docent install zellij` edits the user's Zellij `config.kdl`. It keeps a backup (`config.kdl.docent.bak`) and changes only the marked node.
- No new Go dependencies. `docent new` needs `fzf` on PATH.
