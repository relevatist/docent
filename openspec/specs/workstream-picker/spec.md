# workstream-picker Specification

## Purpose

Lets the user open another workstream from inside Zellij with a keybinding and a picker, without typing paths or harness commands in a shell.

## Requirements

### Requirement: Open the picker
`docent new` SHALL show a picker in its terminal, with fzf. When the user confirms a directory and a harness, docent SHALL open a workstream tab for them in the current Zellij session and exit zero. When the user cancels, docent SHALL open nothing and exit zero.

#### Scenario: Confirm a choice
- **WHEN** the user runs `docent new` in a tab whose harness is `claude`, selects `~/src/app`, and presses `enter`
- **THEN** a new tab named `app` opens for `~/src/app` with harness `claude`, and `docent new` exits zero

#### Scenario: Cancel
- **WHEN** the user runs `docent new` and presses `esc`
- **THEN** no tab opens and `docent new` exits zero

#### Scenario: fzf missing
- **WHEN** `fzf` is not on PATH
- **THEN** docent prints that fzf is not on PATH and exits non-zero

#### Scenario: Not inside Zellij
- **WHEN** the user runs `docent new` outside Zellij
- **THEN** docent prints that `docent new` must run inside Zellij and exits non-zero without showing the picker

### Requirement: Directory candidates
The picker SHALL list the recent workstream directories first, most recent at the top, then the git repos and worktrees found under the config key `roots`, then the home directory and each directory below it. Below the home directory, docent SHALL NOT look inside a hidden directory, a `node_modules` directory, `~/Library`, or a symlink. Each directory SHALL appear once, as `~/...` when it is under the home directory. A directory that no longer exists SHALL NOT appear.

#### Scenario: Recent first
- **WHEN** the recent list is `~/wt/retry/app` then `~/src/app`, and `roots = ["~/src"]` holds `~/src/app` and `~/src/lib`
- **THEN** the picker lists `~/wt/retry/app`, `~/src/app`, `~/src/lib`, then `~` and the other directories below it, in that order

#### Scenario: Recent directory deleted
- **WHEN** a recent directory no longer exists
- **THEN** the picker does not list it

#### Scenario: No roots and no recent directories
- **WHEN** the config has no `roots` and the recent list is empty
- **THEN** the picker lists `~` and the directories below it

#### Scenario: Skipped directories
- **WHEN** `~/.cache/x`, `~/Library/Caches` and `~/src/app/node_modules/x` exist
- **THEN** the picker does not list them

### Requirement: Find repos under roots
For each directory in `roots` (a leading `~` means the home directory), docent SHALL list each directory that has a `.git` entry (directory or file), at any depth below the root. docent SHALL NOT look inside a directory it listed, a hidden directory, or a symlink below the root. A root that does not exist SHALL be ignored. The picker SHALL show the repos while the walk runs, and SHALL stop the walk when it exits.

#### Scenario: Worktree two levels down
- **WHEN** `roots = ["~/wt"]` and `~/wt/retry/app/.git` is a file
- **THEN** the picker lists `~/wt/retry/app`

#### Scenario: Nested repo is not listed
- **WHEN** `~/src/app` is a repo and `~/src/app/vendor/x/.git` exists
- **THEN** the picker lists `~/src/app` and not `~/src/app/vendor/x`

#### Scenario: Deep repo
- **WHEN** `roots = ["~/src"]` and the only repo is `~/src/a/b/c/d/e`
- **THEN** the picker lists `~/src/a/b/c/d/e`

#### Scenario: Results while walking
- **WHEN** the walk under `roots` has found `~/src/app` and has not ended
- **THEN** the picker lists `~/src/app` and shows that the search continues

### Requirement: Filter directories
Typed text SHALL filter the directory list with the fzf fuzzy match (scheme `path`), and with an empty query the list SHALL keep its order. When the text is a path to an existing directory, `enter` SHALL choose that directory, also when other directories match. A path starts with `/`, `~/`, `./` or `../`, or is `~`, `.` or `..`.

#### Scenario: Fuzzy match
- **WHEN** the list holds `~/wt/retry/app` and `~/src/lib`, and the user types `rtap`
- **THEN** only `~/wt/retry/app` is listed

#### Scenario: Typed path outside the list
- **WHEN** the user types `/opt/app`, a directory that is not in the list, and presses `enter`
- **THEN** the picker chooses `/opt/app`, also when other directories match the text

#### Scenario: No match and not a directory
- **WHEN** the user types `nope`, nothing matches, and the user presses `enter`
- **THEN** docent prints that there is no directory for `nope` and opens nothing

### Requirement: Choose the harness
`enter` in the directory list SHALL open the tab with the default harness: the harness of the current tab, or the config `harness` when docent cannot find one. `alt-enter` SHALL first show a harness list: the default harness, then the config aliases in name order, then `claude`, `omp`, `codex`, with each value once. In the harness list, `enter` SHALL choose the typed text (a command with args, or an alias name) when it has a space or nothing matches, else the selected harness.

#### Scenario: Default from current tab
- **WHEN** `docent new` runs in a workstream tab whose harness is `omp`, and the user presses `enter` on a directory
- **THEN** the new tab runs `omp`

#### Scenario: Pick an alias
- **WHEN** the config has `[harnesses.omp-sandbox]`, and the user presses `alt-enter` on a directory, types `sand`, and presses `enter`
- **THEN** the new tab runs the `omp-sandbox` alias

#### Scenario: Typed command with args
- **WHEN** the user types `claude --model opus` in the harness list, nothing matches, and the user presses `enter`
- **THEN** the harness pane of the new tab runs `claude --model opus`

#### Scenario: Typed command that matches a harness
- **WHEN** the user types `omp -s` in the harness list, `omp-sandbox` is selected, and the user presses `enter`
- **THEN** the harness pane of the new tab runs `omp -s`

#### Scenario: Typed command in the home directory
- **WHEN** the user types `~/bin/box omp` in the harness list and confirms
- **THEN** the harness pane of the new tab runs `box` from the `bin` directory in the home directory, with the arg `omp`

#### Scenario: Empty harness
- **WHEN** nothing matches in the harness list, the typed text is empty or spaces, and the user presses `enter`
- **THEN** docent prints that the harness is empty and opens nothing

### Requirement: Directory already open
When the chosen directory is open in a workstream tab of the current session, the picker SHALL show a list with `switch to tab <name>` and `open another tab`. `esc` SHALL cancel and open nothing. Switch SHALL focus that tab and open nothing. Another SHALL open a new tab named after the directory with the first free suffix `-2`, `-3`, and so on. A tab SHALL count as open for a directory when its `editor` pane has that directory as its working directory (`pane_cwd` in `zellij action list-panes`).

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
docent SHALL ship a Zellij keybinding snippet that binds `Alt w`, in all modes except locked, to run `docent new` in a floating pane that closes when `docent new` exits. docent SHALL change the user's Zellij config only through `docent install zellij` (see `harness-integration`).

#### Scenario: Open the picker from the editor
- **WHEN** the user has added the snippet to the Zellij config (by hand or with `docent install zellij`) and presses `Alt w` in the editor pane
- **THEN** a floating pane shows the picker, and it closes after the user confirms or cancels

### Requirement: Layout of the new tab
`docent new` SHALL open the new tab with the layout of the current tab: the file in the `DOCENT_LAYOUT` marker of its harness pane, or the built-in layout when the harness pane has no marker. When the current tab has no harness pane, docent SHALL use the config `layout`, else the built-in layout. The picker SHALL NOT show a layout choice.

#### Scenario: Tab with a custom layout
- **WHEN** the current tab was opened with `docent --layout ~/layouts/pair.kdl`, and the user runs `docent new` and confirms a directory
- **THEN** the new tab opens with the layout in `~/layouts/pair.kdl`

#### Scenario: Tab with the built-in layout
- **WHEN** the current tab was opened with the built-in layout, the config now has `layout = "wide.kdl"`, and the user runs `docent new` and confirms a directory
- **THEN** the new tab opens with the built-in layout

#### Scenario: Not a docent tab
- **WHEN** the current tab has no harness pane, the config has `layout = "wide.kdl"`, and the user runs `docent new` and confirms a directory
- **THEN** the new tab opens with the layout in `wide.kdl` in the docent config directory

#### Scenario: Layout file deleted
- **WHEN** the current tab was opened with `docent --layout ~/layouts/pair.kdl`, that file has since been deleted, and the user runs `docent new` and confirms a directory
- **THEN** docent prints an error that names `~/layouts/pair.kdl` and opens nothing
