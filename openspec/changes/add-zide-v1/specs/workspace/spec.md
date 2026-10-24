# Spec Delta

## Purpose

Opens one workstream as one Zellij tab with editor, harness, and shell panes, and lets zide commands find those panes in the tab they run from.

## ADDED Requirements

### Requirement: Launch a workstream tab
`zide [dir]` SHALL open a workstream for `dir` (default: the current directory). The workstream SHALL have three named panes: `editor` running Helix, `harness` running the configured harness command, and `shell`. The editor pane SHALL be the largest pane and SHALL have focus. All panes SHALL start in `dir`.

#### Scenario: Launch inside Zellij
- **WHEN** the user runs `zide ~/src/app` inside a Zellij session
- **THEN** a new tab named `app` opens with panes `editor`, `harness`, and `shell`, all with working directory `~/src/app`, and the editor has focus

#### Scenario: Launch outside Zellij
- **WHEN** the user runs `zide ~/src/app` outside Zellij
- **THEN** a new Zellij session named `app` starts with the workstream tab

#### Scenario: Launch outside Zellij with an exited session
- **WHEN** the user runs `zide ~/src/app` outside Zellij and a session named `app` is listed as EXITED
- **THEN** zide deletes that session record, prints one line to stderr saying it removed the exited session record, and starts a new session named `app` with the workstream tab

#### Scenario: Directory does not exist
- **WHEN** the user runs `zide /no/such/dir`
- **THEN** zide prints an error naming the directory and exits non-zero without opening a tab

### Requirement: Editor primacy in the layout
The layout SHALL place the editor on the left at about 65% width, with the harness above the shell on the right. The harness and shell panes SHALL stay visible while the editor is in use.

#### Scenario: Default layout
- **WHEN** a workstream tab opens
- **THEN** the editor fills the left column and the right column holds the harness above the shell

### Requirement: Configurable harness command
zide SHALL read the harness command from `~/.config/zide/config.toml` (`harness` key). A `--harness` flag SHALL override it. The default SHALL be `claude`.

#### Scenario: Config sets harness
- **WHEN** the config has `harness = "omp"` and the user runs `zide`
- **THEN** the harness pane runs `omp`

#### Scenario: Flag overrides config
- **WHEN** the config has `harness = "omp"` and the user runs `zide --harness codex`
- **THEN** the harness pane runs `codex`

### Requirement: Tab name
`zide --name <name> [dir]` SHALL name the tab (or new session) `<name>`. Without `--name`, the name SHALL be the directory basename.

#### Scenario: Two worktrees with the same basename
- **WHEN** the user runs `zide --name app-retry ~/wt/retry/app`
- **THEN** the new tab is named `app-retry`

### Requirement: Harness is per tab
The harness chosen at launch SHALL apply to every zide command run in that tab, over the config file.

#### Scenario: Two tabs with different harnesses
- **WHEN** tab A was launched with the default `claude` and tab B with `--harness omp`, and the user runs `zide send ref a.go:1` in tab B
- **THEN** the harness input in tab B receives `@a.go:1 ` formatted for `omp`, from the harness set at launch

### Requirement: Existing directories only
zide SHALL NOT create directories or git worktrees. `zide [dir]` SHALL only open a directory that already exists.

#### Scenario: Worktree not created
- **WHEN** the user runs `zide ../app-feature` and that directory does not exist
- **THEN** zide prints an error naming the directory, exits non-zero, and creates nothing

### Requirement: Find panes in the caller's tab
zide commands SHALL find a named pane only in the tab of the pane they run from. They SHALL NOT act on panes in other tabs.

#### Scenario: Two workstreams open
- **WHEN** two workstream tabs are open and `zide open a.go` runs from the harness pane of the second tab
- **THEN** only the editor pane of the second tab changes

#### Scenario: Not inside Zellij
- **WHEN** a zide command other than launch runs outside Zellij
- **THEN** zide prints that it must run inside a zide workstream and exits non-zero

#### Scenario: Named pane missing
- **WHEN** `zide open a.go` runs in a tab with no `editor` pane
- **THEN** zide prints an error naming the missing pane and exits non-zero
