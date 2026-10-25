# Proposal

## Why

To open a second workstream, the user must go to a shell pane, type `zide <path>`, and remember the path and harness. This is slow when a workstream is already open. A keybinding that opens a picker lets the user open a new workstream from any pane in a few keystrokes.

## What Changes

- Add `zide new`: a TUI that runs in a floating Zellij pane. The user fuzzy-filters a list of directories, optionally changes the harness, and confirms. zide then opens a workstream tab for that directory and the pane closes.
- The directory list has the recent workstream directories first, then git repos and worktrees found under the config key `roots`.
- zide saves each launched directory in a state file, `$XDG_STATE_HOME/zide/recent.toml`.
- The harness field takes a command with args, or the name of an alias. `tab` moves focus between the directory filter and the harness field. In the harness field, `up`/`down` cycles through the current tab's harness, the aliases, and the built-in harnesses.
- Add harness aliases to `config.toml`: `[harnesses.<name>]` with `command` and an optional `kind`. The `harness` key and `--harness` also take alias names.
- `zide send` takes the ref format from the alias `kind`, so wrapped commands (for example a sandbox) still get `@` refs.
- zide finds open workstreams from the working directory (`pane_cwd`) of each tab's `editor` pane. If the chosen directory is already open in a tab, `zide new` asks to switch to that tab or open another one.
- Fix: outside Zellij, the new session's tab is named `Tab #1`, because the layout `tab` node has no name. The layout now names the tab.
- Ship a Zellij keybinding snippet (`Alt w`) in `assets/zellij/keys.kdl` and document it in the README. zide does not edit the Zellij config.

## Capabilities

### New Capabilities

- `workstream-picker`: the `zide new` popup - directory list, filter, harness choice, open-tab handling, and the Zellij keybinding.

### Modified Capabilities

- `workspace`: harness aliases in the config, the harness kind marker, the tab name for a new session, and saving recent directories on launch.
- `harness-input`: the ref format comes from the harness kind, not only from the command name.

## Impact

- New code: `cmd/zide/new.go`, a picker model in `internal/tui`, a directory walk and recent state (new package `internal/recent` or in `internal/config`).
- Changed code: `internal/config` (aliases, `roots`), `internal/launch` (harness kind marker, tab name in the layout), `internal/send` (kind), README, new `assets/zellij/keys.kdl`.
- The pane command shown by `list-panes` changes to `/usr/bin/env ZIDE_HARNESS_KIND=...` for the harness pane when an alias sets a `kind`. The editor pane stays `hx`.
- After `:cd` in Helix, the editor pane cwd changes, so the picker no longer detects that tab for its first directory.
- No new dependencies.
