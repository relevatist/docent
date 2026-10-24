# Proposal

## Why

AI coding harnesses make code changes faster than people can understand them. Editor, harness, and diff tools run as separate programs, so the user moves context between them by hand. zide joins them in one Zellij tab, puts understanding of the code and its changes first, keeps the user in control, and lets the user focus on one thing at a time.

## What Changes

- Add a `zide` Go CLI that runs inside Zellij.
- `zide [dir]` opens one workstream: a Zellij tab with an editor pane (Helix), a harness pane, and a shell pane.
- `zide send` pastes file/line references or text from the editor into the harness input. It never submits.
- `zide open` and `zide reload` let the harness (and the tour) move the editor to a file and line, and reload changed files.
- `zide tour` opens a guided walk-through of a change, one step at a time. The editor follows each step. Hunks the tour does not cover are shown at the end.
- `zide diff` opens the same walk-through in plain `git diff` order, with no notes.
- A generic agent skill (`SKILL.md`) tells any harness how to use zide and how to write a tour. Claude Code and oh-my-pi get an adapter that reloads the editor after each file edit. `zide install` installs for the harnesses the user checks; for Claude Code it installs a plugin with the skill and the hook.
- v1 supports Helix only. Neovim and Kakoune are out of scope.

## Capabilities

### New Capabilities

- `workspace`: launching a workstream tab with editor, harness, and shell panes, and finding those panes from inside the tab.
- `editor-control`: moving the editor to a file and line, and reloading files, on request from other panes.
- `harness-input`: sending file/line references and text from the editor into the harness input without submitting.
- `change-tour`: guided, step-by-step walk-through of a code change, with editor sync and coverage of all hunks.
- `harness-integration`: the generic agent skill and per-harness adapters that let coding harnesses use zide.

### Modified Capabilities

None.

## Impact

- New Go module and `zide` binary. New dependencies: a TUI library for the tour pane, a TOML parser for config.
- Runtime dependencies: Zellij 0.45+ (for `--pane-id` and `list-panes --json`), Helix 25.07+, git.
- Adds files to user config only on explicit install: Helix keybindings, the agent skill, and the omp extension. Registers the Claude Code plugin through the `claude` CLI.
