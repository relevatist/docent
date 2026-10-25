# Design

## Context

Greenfield repo. See proposal.md - Why. Facts checked on the target machine:

- Zellij 0.45.1: `write-chars`, `paste`, `send-keys` take `--pane-id`; `list-panes --all --json` lists panes with tab info; `new-pane` has `--stacked`, `--name`, `--cwd`, `--close-on-exit`; `new-tab` and the top-level command accept `--layout-string`.
- Helix 25.07.1: no remote-control API. Command-line expansions include `%{buffer_name}`, `%{selection_line_start}`, `%{selection_line_end}`. `:reload-all` discards unsaved changes in all buffers.
- oh-my-pi (`omp`) reads skills from `~/.claude/skills`, `~/.agents/skills`, and Claude Code marketplace plugins (`~/.claude/plugins/cache/`), and loads TypeScript extensions. Same-named skills with identical content collapse with no warning (omp 18.12.6 binary). 7.4 result (omp 18.12.6 source, `packages/coding-agent/src/extensibility/extensions/types.ts`, and `docs/extensions.md`, `docs/extension-loading.md`): there is no dedicated after-edit event, but `pi.on("tool_result", ...)` fires after every tool with `toolName`, `input`, and `isError`. File-writing tools are `edit` (also covers `apply_patch` mode) and `write`; `ast_edit` only previews, and `resolve` applies it. `pi.exec(cmd, args)` runs a command and does not throw on non-zero exit. User extensions load from `~/.omp/agent/extensions/*.ts`.

## Goals / Non-Goals

**Goals:**
- One static binary, no daemon, no state files.
- Every cross-pane action is one or more `zellij action` calls, so behavior is easy to inspect and test.

**Non-Goals:**
- A Zellij WASM plugin.
- An editor abstraction. Helix code is plain functions until a second editor exists.
- Tracking editor state (open buffers, mode, unsaved changes).

## Decisions

### Go CLI with stdlib subcommands
`zide` uses `os.Args` dispatch and `flag.FlagSet` per subcommand. Alternatives: Rust (same shape, slower to iterate), WASM plugin (more moving parts, harder to test), shell (fragile JSON and TUI). Dependencies: `bubbletea` + `lipgloss` for the tour pane, `BurntSushi/toml` for config. Diff parsing is local code: only unified-diff hunk headers and lines are needed.

```
cmd/zide/main.go          dispatch
internal/zellij           Runner (exec wrapper) + typed calls; Fake for tests
internal/panes            Find(name) in caller's tab
internal/helix            Open / Reload key sequences
internal/send             ref formatting per harness
internal/tour             parse, git diff, hunk mapping, coverage, step model
internal/tui              bubbletea view over tour step model
internal/launch           layout KDL (embedded template), tab/session start
assets/                   claude-plugin/ (skill, hook, plugin.json), helix keys, omp extension (go:embed)
```

`zellij.Runner` is the only interface. It exists so tests can assert the exact `zellij action` argv without a running session.

### Pane routing by name in the caller's tab
The layout names panes `editor`, `harness`, `shell`; the tour pane is `tour`. A command reads `$ZELLIJ_PANE_ID`, runs `zellij action list-panes --all --json`, finds its own pane's tab, then picks the named pane in that tab. Alternative: a per-tab state file written at launch - rejected because it goes stale when panes close or restart. The exact JSON field names are captured as a test fixture from a real session in task 2.1.

### Helix control by key injection
```
open a.go:42  ->  send-keys -p E "Esc" "Esc"
                  write-chars -p E ":open '/abs/a.go'"   ; send-keys -p E "Enter"
                  write-chars -p E ':goto 42'            ; send-keys -p E "Enter"
reload        ->  send-keys -p E "Esc" "Esc"
                  write-chars -p E ':reload-all'         ; send-keys -p E "Enter"
```
Two `Esc` leave insert mode and close any prompt or picker. Paths are made absolute against the caller's cwd, because Helix's cwd may differ from the caller's. Paths go in single quotes with `'` doubled: Helix expands `%{...}` inside double quotes and does not treat backslash as an escape. `:open` keeps other buffers, so unsaved work there survives. `:reload-all` discards unsaved changes (user decision; documented in the skill, help text, and README).

### Harness input by bracketed paste
`zide send` uses `zellij action paste`, which wraps text in bracketed paste. Harness TUIs treat pasted newlines as text, so nothing is submitted. Ref format table: `claude`, `omp` -> `@path:range `; others -> `path:range `. The harness key is the basename of the configured harness command.

### Tour as a short-lived pane running the same binary
```
zide tour plan.json            (from harness)
  validate tour, copy to $TMPDIR/zide-tour-<pid>.json
  ZELLIJ_PANE_ID=<shell> new-pane --stacked --near-current-pane --name tour --cwd <dir> --close-on-exit -- zide tour-view <tmp>
  focus-pane-id <tour>
zide tour-view <tmp>           (inside tour pane)
  git diff <base> --no-color -U0 -> hunks
  map steps to hunks, append "Not covered" step
  bubbletea loop; on step enter -> helix.Open(file, start) in-process
```
Validation runs before any pane opens, so tour errors print in the harness's shell. `--near-current-pane` with `ZELLIJ_PANE_ID` set to the shell pane stacks the tour with the shell, wherever the user's focus is; `focus-pane-id` before `new-pane` did not move the client's focus reliably. zide then focuses the tour pane. `zide diff [args]` builds an in-memory tour (one step per hunk of `git diff [args]`) and starts the same pane.

5.4 result (Zellij 0.45.1): with the shell focused, `new-pane --stacked --name tour -- <cmd>` stacks the tour pane with the shell. `dump-layout` shows `pane stacked=true { pane name="shell"; pane name="tour" expanded=true }` in the right column under the harness; the harness keeps its geometry and stays visible. No `--direction down` fallback is needed. The new pane starts in the caller's cwd, not the shell's, so pass `--cwd` if that matters.

Step view: read the step's line range from the working tree, mark lines inside new-side hunk ranges as added, and insert the hunk's removed lines before them. Lines outside any hunk are plain context.

### Launch with an embedded layout string
```
inside Zellij:  zellij action new-tab --layout-string <kdl> --name <name> --cwd <dir>
outside, new:   zellij --session <name> --new-session-with-layout <tmp kdl file>   (--layout-string only adds to an existing session)
outside, exists: zellij --session <name> action new-tab ... ; zellij attach <name>
outside, exited: zellij delete-session <name> ; then the "outside, new" path   (list-sessions --no-formatting marks it "(EXITED ...)")
```
The KDL template has the editor at 60% on the left, harness above shell on the right, with `hx` and the harness command filled in. `<name>` is `--name` or the directory basename. The harness is a property of the tab: `zide send` reads the harness pane's `terminal_command` from `list-panes --all --json` (the command the pane started with; it stays set after the command exits) and uses its basename, falling back to the config file only when that field is null. Alternative: `ZIDE_HARNESS` in every pane's env - rejected because Zellij 0.45.1 layouts reject a per-pane `env` node, and wrapping the shell in `env` makes it a command pane. zide never creates directories or git worktrees.

### One generic skill, thin adapters, guided install
- `assets/claude-plugin/skills/zide/SKILL.md` is the only copy of the skill: when to use zide (`$ZELLIJ` set and `zide` on PATH), `open`/`reload`/`tour`, the tour format with an example, and the reload warning.
- Layout:
  ```
  .claude-plugin/marketplace.json   repo marketplace, source ./assets/claude-plugin
  assets/claude-plugin/
    .claude-plugin/plugin.json
    skills/zide/SKILL.md
    hooks/hooks.json
  assets/omp/zide-reload.ts
  ```
- Codex skill directory, 7.2 result: the codex docs (developers.openai.com/codex/skills, now learn.chatgpt.com/docs/build-skills) give `$HOME/.agents/skills` as the USER scope. codex-cli 0.164.0 also still scans `$CODEX_HOME/skills` (`~/.codex/skills`; strings `codex-home-skills` and `user-agent-skills` in the binary), but only `~/.agents/skills` is documented.
- Claude Code adapter: the plugin's `PostToolUse` hook (matcher `Edit|Write|MultiEdit|NotebookEdit`) running `zide reload --quiet`. The Claude Code tools reference lists `Edit`, `Write`, and `NotebookEdit` as the file-editing tools; `MultiEdit` is no longer listed and stays in the matcher for older versions. `--quiet` exits zero with no output when not in a zide workstream, so the hook never fails an edit.
- omp adapter: `assets/omp/zide-reload.ts`, a `tool_result` handler that runs `zide reload --quiet` after a successful `edit`, `write`, or `resolve`.
- `zide install`: a bubbletea checklist when no harness is named on a terminal, else the named harnesses. It plans every write and command first, and acts only when the whole plan is valid.
  - claude: write the plugin to `$XDG_DATA_HOME/zide/claude-plugin/`, then `claude plugin marketplace add|update` and `claude plugin install|update zide@zide --scope user`. zide writes `0.0.0-<hash of the plugin files>` as the version in `plugin.json`, because Claude Code caches a plugin by version (zide has no build version). The repo `plugin.json` has no version, so the repo marketplace uses the git commit. Alternative: the user edits `settings.json` by hand - rejected, because a plugin installs and updates the hook and skill as one unit, and zide never merges JSON into a file the user owns.
  - codex, omp: the skill goes to `~/.agents/skills/zide/` (omp reads it too); omp also gets the extension.
  - The repo is also a marketplace for users who install without zide. Both use the name `zide`; if the registered source is not zide's directory, zide stops instead of replacing it.

## Risks / Trade-offs

- [Key injection depends on Helix keymap defaults] -> Use only `Esc`, `:` command line, and `Enter`; integration test catches regressions.
- [`zide reload` discards unsaved edits] -> Documented in three places; hook reloads only after harness edits, which the user started.
- [`--stacked` placement differs from expectation] -> Task 5.4 checks it in a real session before the tour work builds on it; fallback is `new-pane --direction down` from the shell pane.
- [`claude plugin` CLI flags change] -> zide only uses `marketplace add|update|list` and `install|update`; task 7.9 runs them against a real Claude Code.
- [`list-panes` JSON shape changes between Zellij versions] -> Fixture from 0.45.1; zide checks the Zellij version at launch and warns below 0.45.

## Open Questions

- None. (Codex skill directory: resolved in 7.2 as `~/.agents/skills`.)
