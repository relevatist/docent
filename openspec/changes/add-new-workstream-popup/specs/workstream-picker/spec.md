# Spec Delta

## Purpose

Lets the user open another workstream from inside Zellij with a keybinding and a picker, without typing paths or harness commands in a shell.

## ADDED Requirements

### Requirement: Open the picker
`zide new` SHALL show a picker in its terminal. When the user confirms a directory and a harness, zide SHALL open a workstream tab for them in the current Zellij session and exit zero. When the user cancels, zide SHALL open nothing and exit zero.

#### Scenario: Confirm a choice
- **WHEN** the user runs `zide new`, selects `~/src/app`, keeps the harness `claude`, and presses `enter`
- **THEN** a new tab named `app` opens for `~/src/app` with harness `claude`, and `zide new` exits zero

#### Scenario: Cancel
- **WHEN** the user runs `zide new` and presses `esc`
- **THEN** no tab opens and `zide new` exits zero

#### Scenario: Not inside Zellij
- **WHEN** the user runs `zide new` outside Zellij
- **THEN** zide prints that `zide new` must run inside Zellij and exits non-zero without showing the picker

### Requirement: Directory candidates
The picker SHALL list the recent workstream directories first, most recent at the top, then the git repos and worktrees found under the config key `roots`. Each directory SHALL appear once. A directory that no longer exists SHALL NOT appear.

#### Scenario: Recent first
- **WHEN** the recent list is `~/wt/retry/app` then `~/src/app`, and `roots = ["~/src"]` holds `~/src/app` and `~/src/lib`
- **THEN** the picker lists `~/wt/retry/app`, `~/src/app`, `~/src/lib`, in that order

#### Scenario: Recent directory deleted
- **WHEN** a recent directory no longer exists
- **THEN** the picker does not list it

#### Scenario: No roots and no recent directories
- **WHEN** the config has no `roots` and the recent list is empty
- **THEN** the picker shows an empty list and a hint to set `roots` in the config

### Requirement: Find repos under roots
For each directory in `roots` (a leading `~` means the home directory), zide SHALL list each directory that has a `.git` entry (directory or file), down to 3 levels below the root. zide SHALL NOT look inside a directory it listed. A root that does not exist SHALL be ignored.

#### Scenario: Worktree two levels down
- **WHEN** `roots = ["~/wt"]` and `~/wt/retry/app/.git` is a file
- **THEN** the picker lists `~/wt/retry/app`

#### Scenario: Nested repo is not listed
- **WHEN** `~/src/app` is a repo and `~/src/app/vendor/x/.git` exists
- **THEN** the picker lists `~/src/app` and not `~/src/app/vendor/x`

#### Scenario: Too deep
- **WHEN** `roots = ["~/src"]` and the only repo is `~/src/a/b/c/d`
- **THEN** the picker does not list it

### Requirement: Filter directories
Typed text SHALL filter the directory list with a fuzzy match on the path: each typed character SHALL appear in the path in the same order. Matching SHALL ignore case. The filter SHALL keep the list order. `up`/`down` (and `ctrl+p`/`ctrl+n`) SHALL move the selection.

#### Scenario: Fuzzy match
- **WHEN** the list holds `~/wt/retry/app` and `~/src/lib`, and the user types `rtap`
- **THEN** only `~/wt/retry/app` is listed

#### Scenario: Typed path
- **WHEN** no listed directory matches the text and the text is a path to an existing directory
- **THEN** the picker offers that directory as the only choice

### Requirement: Choose the harness
The picker SHALL have a harness field that takes a harness command with args, or an alias name from the config. `tab` SHALL move focus between the directory filter and the harness field. In the harness field, `up`/`down` SHALL cycle through the candidates: the harness of the current tab, then the config aliases in name order, then `claude`, `omp`, `codex`, with each value once. The field SHALL start with the harness of the current tab, or the config `harness` when zide cannot find one.

#### Scenario: Default from current tab
- **WHEN** `zide new` runs in a workstream tab whose harness is `omp`
- **THEN** the harness field starts as `omp`

#### Scenario: Cycle to an alias
- **WHEN** the config has `[harnesses.omp-sandbox]`, the field shows `omp`, and the user presses `down` in the harness field
- **THEN** the field shows `omp-sandbox`

#### Scenario: Typed command with args
- **WHEN** the user types `claude --model opus` in the harness field and confirms
- **THEN** the harness pane of the new tab runs `claude --model opus`

#### Scenario: Empty harness
- **WHEN** the harness field is empty and the user presses `enter`
- **THEN** the picker shows that the harness is empty and opens nothing

### Requirement: Directory already open
When the chosen directory is open in a workstream tab of the current session, the picker SHALL ask the user to switch to that tab or open another tab. Switch SHALL focus that tab and open nothing. Another SHALL open a new tab named after the directory with the first free suffix `-2`, `-3`, and so on. A tab SHALL count as open for a directory when its `editor` pane has that directory as its working directory (`pane_cwd` in `zellij action list-panes`).

#### Scenario: Switch
- **WHEN** `~/src/app` is open in tab `app` and the user chooses it and then chooses switch
- **THEN** Zellij focuses tab `app` and no tab opens

#### Scenario: Open another
- **WHEN** tabs `app` and `app-2` exist, `~/src/app` is open, and the user chooses open another
- **THEN** a new tab named `app-3` opens for `~/src/app`

#### Scenario: Detected by the editor pane directory
- **WHEN** a tab has an `editor` pane whose working directory in `zellij action list-panes` is `~/src/app`, and the user chooses `~/src/app`
- **THEN** the picker asks to switch to that tab or open another tab

#### Scenario: Editor directory changed with `:cd`
- **WHEN** a tab was opened for `~/src/app` and the user ran `:cd ~/src/lib` in Helix, and the user chooses `~/src/app`
- **THEN** the picker does not detect the tab and opens a new tab named after the directory (accepted limit)

### Requirement: Zellij keybinding
zide SHALL ship a Zellij keybinding snippet that binds `Alt w`, in all modes except locked, to run `zide new` in a floating pane that closes when `zide new` exits. zide SHALL NOT change the user's Zellij config.

#### Scenario: Open the picker from the editor
- **WHEN** the user has added the snippet to the Zellij config and presses `Alt w` in the editor pane
- **THEN** a floating pane shows the picker, and it closes after the user confirms or cancels
