# Tasks

## 1. Verify Zellij behavior

- [x] 1.1 In a background Zellij 0.45 session, open a pane with `command="/usr/bin/env"` and args `"A=/tmp/a b" "hx"`. Verify that `list-panes --all --json` shows the full argv in `terminal_command`, and record the exact format in design.md. If the args are missing, stop and ask the user (see design.md - Risks)
- [x] 1.2 In the same session, verify that `zellij action new-tab` run from a floating pane focuses the new tab, and that a `close_on_exit` floating pane closes in the old tab. Record the result in design.md

## 2. Harness aliases and kind

- [x] 2.0 Make `launch.Layout` take the tab name and emit `tab name=...`; verify `TestRunOutsideZellij` asserts the layout tab name equals the session name and `TestLayout` asserts the quoted name
- [x] 2.1 Write failing tests for `config.Load` and `Config.Resolve`: alias with `kind`; alias with default kind from the base name of the first word; plain command; exact name match only; empty `command` and `kind` with spaces give an error that names the alias; `roots` with a leading `~` expands to the home directory
- [x] 2.2 Implement `[harnesses.<name>]`, `roots`, and `Resolve`; make `runLaunch` resolve `--harness` and `harness`; verify the 2.1 tests pass and a launch test with `--harness omp-sandbox` asserts the alias command in the layout
- [ ] 2.3 Write failing tests for `panes.Marker` and `panes.Unwrap`: no wrapper, `ZIDE_HARNESS_KIND` wrapper, a command with args after the wrapper
- [ ] 2.4 Change `launch.Layout` to take a `config.Harness` and the tab name: wrap the harness pane in `/usr/bin/env ZIDE_HARNESS_KIND=<kind>` only when the kind differs from the command's base name; the editor pane stays `hx`; verify the updated `TestLayout*` tests assert both cases and that a plain `claude` harness is not wrapped
- [ ] 2.5 Make `send` take the kind from `ZIDE_HARNESS_KIND`, else from `Unwrap(cmd)`; verify a send test with a pane command `env ZIDE_HARNESS_KIND=omp sandbox-exec -f omp.sb omp` gives `@a.go:1 `
- [ ] 2.6 Update the README config section with `[harnesses.<name>]` and `roots`; verify the TOML example parses with a test that loads it

## 3. Recent directories

- [ ] 3.1 Write failing tests for `internal/recent` against a temp `$XDG_STATE_HOME`: missing file gives an empty list; `Add` moves an entry to the top; the limit is 20; the file is written through a rename; `Load` drops nothing (the picker filters deleted directories)
- [ ] 3.2 Implement `internal/recent` and call `recent.Add` from `launchDir` after `launch.Run` succeeds; a write error prints `zide: warning: ...` with the state file path; verify the 3.1 tests pass and a launch test with an unwritable state directory still exits zero with the warning

## 4. Directory candidates

- [ ] 4.1 Write failing tests for the root walk against a temp tree: `.git` directory and `.git` file are hits; no descent into a hit; depth 3 limit; hidden directories skipped; missing root ignored
- [ ] 4.2 Implement the walk and the candidate list (recent first, deleted directories removed, duplicates removed); verify the 4.1 tests pass and a test asserts the order from the `Recent first` scenario
- [ ] 4.3 Write failing tests for the fuzzy filter: subsequence match, case ignored, list order kept, a typed path to an existing directory is the only choice when nothing matches; implement it and verify the tests pass

## 5. Picker TUI

- [ ] 5.1 Write failing model tests for the picker in `internal/tui`: typing filters; `up`/`down` and `ctrl+p`/`ctrl+n` move; `tab` moves focus; `up`/`down` in the harness field cycles the deduped candidates in order (current tab, aliases by name, `claude`, `omp`, `codex`); empty harness on `enter` shows an error; `esc` cancels; walk results append to the list; the hint shows when the list is empty and there are no roots
- [ ] 5.2 Write failing model tests for the open-tab screen: `enter` on a directory for which `isOpen` (from the editor pane `pane_cwd`) returns a tab shows `switch` / `open another`; each choice returns the matching result
- [ ] 5.3 Implement the picker model; verify the 5.1 and 5.2 tests pass

## 6. `zide new` command

- [ ] 6.1 Write failing CLI tests with `zellij.Fake`: outside Zellij prints the error and exits non-zero before the picker runs; the default harness comes from the current tab's harness pane, else from the config; `open` calls `new-tab` with the resolved layout; `switch` calls `go-to-tab-by-id <tab_id>`; `open another` uses the first free `-N` suffix from the `tab_name` values
- [ ] 6.2 Implement `cmd/zide/new.go` and add `new` to the dispatch table and the usage text; verify the 6.1 tests pass and `main_test.go` lists `new`
- [ ] 6.3 Add `assets/zellij/keys.kdl` with the `Alt w` bind and a README section "Zellij key" next to "Helix keys"; verify `zellij setup --check` accepts a config that includes the snippet

## 7. Integration

- [ ] 7.1 Extend `integration_test.go`: launch a workstream for a temp directory, and verify `list-panes` shows that directory as the editor pane `pane_cwd` and that `zide open` still finds the editor pane
- [ ] 7.2 Manual check in a real session: press `Alt w` in the editor pane, filter to a worktree, choose an alias with a `kind`, confirm; verify the new tab opens, `space z r` sends an `@` ref, `Alt w` on the same directory offers switch and open another, and both work

## Workflow follow-up

- Run `make check` before each commit.
- Archive the change after review.
