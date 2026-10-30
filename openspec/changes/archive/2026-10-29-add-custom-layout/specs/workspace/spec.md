# Spec Delta

## MODIFIED Requirements

### Requirement: Launch a workstream tab
`docent [dir]` SHALL open a workstream for `dir` (default: the current directory). The workstream SHALL have three named panes: `editor` running Helix, `harness` running the configured harness command, and `shell`. With the built-in layout, the harness pane SHALL have focus. All panes SHALL start in `dir`.

#### Scenario: Launch inside Zellij
- **WHEN** the user runs `docent ~/src/app` inside a Zellij session
- **THEN** a new tab named `app` opens with panes `editor`, `harness`, and `shell`, all with working directory `~/src/app`, and the harness has focus

#### Scenario: Launch outside Zellij
- **WHEN** the user runs `docent ~/src/app` outside Zellij
- **THEN** a new Zellij session named `app` starts with the workstream tab

#### Scenario: Launch outside Zellij with an exited session
- **WHEN** the user runs `docent ~/src/app` outside Zellij and a session named `app` is listed as EXITED
- **THEN** docent deletes that session record, prints one line to stderr saying it removed the exited session record, and starts a new session named `app` with the workstream tab

#### Scenario: Directory does not exist
- **WHEN** the user runs `docent /no/such/dir`
- **THEN** docent prints an error naming the directory and exits non-zero without opening a tab

## REMOVED Requirements

### Requirement: Editor primacy in the layout
**Reason**: The built-in layout now makes the harness the main pane.
**Migration**: See `Default layout`. To keep the editor on the left, use a custom layout (see `Custom layout file`).

## ADDED Requirements

### Requirement: Default layout
Without a custom layout, docent SHALL use the built-in layout: the harness on the left at 50% of the width, and on the right the shell above the editor, each at 50% of the height. The tab SHALL have the Zellij tab bar at the top and the status bar at the bottom.

#### Scenario: Built-in layout
- **WHEN** the config has no `layout` key and the user runs `docent ~/src/app`
- **THEN** the harness fills the left half, the shell fills the top of the right half, and the editor fills the bottom of the right half

### Requirement: Custom layout file
docent SHALL take a custom layout from the `--layout <file>` flag, else from the config `layout` key. The flag SHALL override the config. The value SHALL be a file path. docent SHALL expand a leading `~` or `~/` to the home directory. A relative path SHALL be relative to the current directory for the flag, and to the config directory for the config key.

#### Scenario: Config sets a layout
- **WHEN** the config has `layout = "wide.kdl"` and the user runs `docent ~/src/app`
- **THEN** the tab opens with the layout in `wide.kdl` in the docent config directory

#### Scenario: Flag overrides config
- **WHEN** the config has `layout = "wide.kdl"` and the user runs `docent --layout ./pair.kdl ~/src/app` from `~/layouts`
- **THEN** the tab opens with the layout in `~/layouts/pair.kdl`

#### Scenario: Layout file missing
- **WHEN** the user runs `docent --layout ~/no-such.kdl ~/src/app`
- **THEN** docent prints an error that names the file, exits non-zero, and opens nothing

### Requirement: Layout template
A layout file SHALL be a full Zellij layout that docent renders as a Go text template before use. `{{.Name}}` SHALL give the quoted tab name. `{{.Harness}}` SHALL give the `command` property of the harness pane and, when the harness has args or markers, an `args` child block.

#### Scenario: Template fields
- **WHEN** the layout has `tab name={{.Name}} {` and `pane name="harness" {{.Harness}}`, and the user runs `docent --name app --harness "claude --model opus" --layout x.kdl`
- **THEN** the tab is named `app` and the harness pane runs `claude --model opus`

### Requirement: Custom layout checks
Before it opens a tab, docent SHALL reject a custom layout that does not parse or render as a template, that does not use `{{.Harness}}`, or whose rendered text has no `name="editor"`, `name="harness"`, or `name="shell"`. The error SHALL name the layout file and the problem, and docent SHALL exit non-zero and open nothing.

#### Scenario: Template error
- **WHEN** the layout file contains `{{.Nmae}}`
- **THEN** docent prints an error that names the layout file and exits non-zero

#### Scenario: Harness field not used
- **WHEN** the layout has `pane name="harness" command="claude"` and no `{{.Harness}}`
- **THEN** docent prints an error that names the layout file and `{{.Harness}}`, and exits non-zero

#### Scenario: Pane missing
- **WHEN** the layout has no pane named `shell`
- **THEN** docent prints an error that names the layout file and `shell`, and exits non-zero

### Requirement: Layout in the tab
When a tab uses a custom layout, the harness pane SHALL show the absolute path of the layout file in its pane command, so `docent new` can read it. The harness process SHALL get the environment variable `DOCENT_LAYOUT` set to that path, with URL path escaping. A tab with the built-in layout SHALL have no layout marker.

#### Scenario: Custom layout marker
- **WHEN** the user runs `docent --layout ~/layouts/pair.kdl ~/src/app`
- **THEN** the harness pane command shows `DOCENT_LAYOUT=` with the absolute path of `pair.kdl`, and `docent send` in that tab still finds the harness kind

#### Scenario: Path with a space
- **WHEN** the layout file is `~/my layouts/pair.kdl`
- **THEN** the marker value is the absolute path with the space escaped as `%20`, and `docent new` in that tab uses `~/my layouts/pair.kdl`

#### Scenario: Built-in layout has no marker
- **WHEN** the config has no `layout` key and the user runs `docent --harness claude ~/src/app`
- **THEN** the harness pane command is `claude`
