# Tasks

## 1. Setup

- [x] 1.1 Create Go module `zide` with `cmd/zide/main.go` dispatching subcommands (`open`, `reload`, `send`, `tour`, `tour-view`, `diff`, `install`) via `flag.FlagSet`; verify `go build ./...` succeeds and `zide --help` lists all subcommands
- [x] 1.2 Add `bubbletea`, `lipgloss`, `BurntSushi/toml`; verify `go mod tidy` and `go vet ./...` pass

## 2. Zellij runner and pane routing

- [x] 2.1 Start a background Zellij 0.45 session with three named panes, save `zellij action list-panes --all --json` output as `internal/panes/testdata/two_tabs.json` (edit to two tabs, both with `editor`/`harness`/`shell`); verify the fixture parses with `jq`
- [x] 2.2 Write failing tests for `panes.Find`: finds named pane in caller's tab only; errors name the missing pane; errors when `$ZELLIJ_PANE_ID` is unset ("must run inside a zide workstream")
- [x] 2.3 Implement `zellij.Runner` (exec wrapper) and `zellij.Fake` (records argv, returns canned output), then `panes.Find`; verify 2.2 tests pass

## 3. Editor control (Helix)

- [x] 3.1 Write failing tests for `helix.Open` and `helix.Reload` against `zellij.Fake`: exact argv for `Esc Esc`, `:open "<abs path>"`, `:goto N`, `:reload-all`, `Enter`; paths with spaces are quoted; relative paths resolve against cwd
- [x] 3.2 Implement `helix.Open`/`helix.Reload` and `zide open <file>[:line] [--focus]`, `zide reload [--quiet]`; missing file errors before any zellij call; no focus change without `--focus`; verify 3.1 tests pass and CLI tests with the fake cover missing file, `--focus`, and `--quiet` outside Zellij
- [x] 3.3 Write `zide reload --help` text stating unsaved changes in all buffers are discarded; verify a test asserts the help text contains that warning

## 4. Harness input

- [x] 4.1 Write failing tests for `send.FormatRef`: `@a.go:10-20 ` for claude/omp, `a.go:10 ` for single line, `a.go:10-20 ` for unknown harness, harness key taken from command basename
- [x] 4.2 Implement `zide send ref` and `zide send text` (stdin, empty input sends nothing), using `paste --pane-id` and `focus-pane-id` per `focus_on_send`; verify 4.1 tests pass and fake-runner tests assert no `Enter` is ever sent
- [x] 4.3 Add `assets/helix/keys.toml` with `space z r` (ref via `%{buffer_name}:%{selection_line_start}-%{selection_line_end}`) and `space z s` (`:pipe-to zide send text`); verify by running both bindings in Helix inside a workstream and checking the harness input

## 5. Workspace launch

- [x] 5.1 Write failing tests for config loading: default `harness = "claude"`, `focus_on_send = true`; file overrides; `--harness` overrides file
- [x] 5.2 Write failing tests for layout rendering: KDL contains panes `editor` (60%, `hx`, focused), `harness` (configured command), `shell`, in the agreed arrangement
- [x] 5.3 Implement `zide [dir]`: inside Zellij `new-tab --layout-string --name --cwd`; outside `zellij --session --layout-string`; missing dir errors before launch; warn when Zellij is below 0.45; verify 5.1/5.2 tests pass and fake-runner tests cover both launch paths
- [x] 5.5 Add `--name <name>` to launch (tab or session name; default dir basename); verify fake-runner tests assert `--name` on `new-tab` and the session name outside Zellij
- [x] 5.6 Make `zide send` take the harness from the harness pane's `terminal_command` in `list-panes --all --json`, over the config file; verify tests that `send` in a tab launched with `--harness omp` formats refs for `omp` while the config says `claude`
- [x] 5.4 In a real session, verify that focusing the shell then `new-pane --stacked --name tour` stacks the new pane with the shell; record the result in design.md and switch to `--direction down` if it does not

## 6. Change tour

- [x] 6.1 Write failing tests for tour parsing: valid tour, missing `steps`, empty `steps`, invalid JSON, step without `lines`
- [x] 6.2 Write failing tests for unified diff parsing (`-U0` hunks, new files, deleted files, multiple files) using `testdata/*.diff` fixtures
- [x] 6.3 Write failing tests for coverage: overlap counts as covered; file-only step covers all hunks in file; uncovered hunks become final `Not covered by this tour` step; no extra step when fully covered; `zide diff` tour has one step per hunk in git order
- [x] 6.4 Write failing tests for the step model: `n`/`p` bounds, step label `2/5`, step view marks added/removed/context lines, invalid steps (missing file, range outside file) show the reason and do not call `helix.Open`
- [x] 6.5 Implement `internal/tour` to pass 6.1-6.4; base defaults to `HEAD`, diff includes staged and unstaged changes; verify with a temp git repo test
- [x] 6.6 Implement `zide tour <file|->`, `zide diff [args]` (prints `no changes` and opens no pane when empty), and `zide tour-view` with bubbletea keys `n p o s q`; `s` uses `zide send ref`; verify fake-runner tests assert the `new-pane --stacked --name tour --close-on-exit` call and that entering a step calls `helix.Open` without a focus change

## 7. Harness integration

- [x] 7.1 Write `assets/skill/zide/SKILL.md`: when to use zide (`$ZELLIJ` set, `zide` on PATH), `open`/`reload`/`tour` usage, tour format with example, reload warning; verify a test checks the file has valid frontmatter (`name`, `description`) and no harness-specific steps
- [x] 7.2 Confirm the codex user skill directory (`~/.codex/skills` vs `~/.agents/skills`), then implement `zide install skill <claude|codex> [--force]`: prints paths, refuses to overwrite a changed file without `--force`; verify tests against a temp `$HOME`
- [x] 7.3 Move the skill to `assets/claude-plugin/skills/zide/SKILL.md` and the hook to `assets/claude-plugin/hooks/hooks.json`; add `plugin.json` and the root `.claude-plugin/marketplace.json`; embed the plugin directory; verify `claude plugin validate` passes for the plugin and the marketplace and the 7.1 test reads the new path
- [ ] 7.4 Check the omp extension API for an after-edit or tool-result event; if present, add `assets/omp/zide-reload.ts` running `zide reload --quiet` and verify an omp edit reloads the editor; if absent, record that in design.md and the README
- [x] 7.5 Write README: install, `zide [dir]`, Helix keys, skill install, adapter install steps for Claude Code and omp, reload warning; verify every command in the README runs as written
- [x] 7.6 Write failing tests for `zide install` against a temp `$HOME`/`$XDG_DATA_HOME` and a fake `claude` on `PATH` that records argv: names form; unknown harness and no-terminal exit 2; missing command and changed file write nothing for any harness; codex/omp paths; claude add vs update; marketplace from another source; legacy skill removed or kept
- [x] 7.7 Implement `zide install <harness>... [--force]` and remove `install skill`; verify 7.6 tests pass
- [x] 7.8 Implement the bubbletea checklist (checked when on `PATH`, space toggles, enter confirms, esc/q cancels); verify model tests for toggle, confirm, and cancel
- [ ] 7.9 In a real workstream, run `zide install` and select Claude Code: verify `claude plugin list` shows `zide@zide`, a Claude Code edit reloads the editor, the hook exits zero outside Zellij, and rerunning updates the plugin; then verify `claude plugin marketplace add` of the repo installs the same plugin
- [x] 7.10 Update the README install section for `zide install`; verify every install command in the README runs as written

## 8. Integration

- [x] 8.1 Add `integration_test.go` (skipped when `zellij` or `hx` is missing): start a background session with the zide layout, run `zide open` from the shell pane, check `dump-screen` of the editor pane shows the file at the line; verify `go test ./...` passes with and without the tools installed
- [ ] 8.2 Manual walk-through in two workstream tabs: send a ref from Helix, have Claude Code open a file and write a tour, step through it, run `zide diff` from the shell; verify each action affects only its own tab
