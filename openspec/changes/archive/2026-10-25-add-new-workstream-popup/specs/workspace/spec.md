# Spec Delta

## MODIFIED Requirements

### Requirement: Configurable harness command
docent SHALL read the harness command from `~/.config/docent/config.toml` (`harness` key). A `--harness` flag SHALL override it. The default SHALL be `claude`. The `harness` key and `--harness` SHALL take either a command with args or the name of a harness alias.

#### Scenario: Config sets harness
- **WHEN** the config has `harness = "omp"` and the user runs `docent`
- **THEN** the harness pane runs `omp`

#### Scenario: Flag overrides config
- **WHEN** the config has `harness = "omp"` and the user runs `docent --harness codex`
- **THEN** the harness pane runs `codex`

#### Scenario: Flag names an alias
- **WHEN** the config has `[harnesses.omp-sandbox]` with `command = "sandbox-exec -f omp.sb omp"` and the user runs `docent --harness omp-sandbox`
- **THEN** the harness pane runs `sandbox-exec -f omp.sb omp`

### Requirement: Tab name
`docent --name <name> [dir]` SHALL name the tab (or new session) `<name>`. Without `--name`, the name SHALL be the directory basename.

#### Scenario: Two worktrees with the same basename
- **WHEN** the user runs `docent --name app-retry ~/wt/retry/app`
- **THEN** the new tab is named `app-retry`

#### Scenario: Launch outside Zellij names the tab
- **WHEN** the user runs `docent ~/src/app` outside Zellij and no session named `app` exists
- **THEN** a new session named `app` starts, and its workstream tab is named `app`

## ADDED Requirements

### Requirement: Harness aliases
The config SHALL accept harness aliases as `[harnesses.<name>]` tables with a required `command` and an optional `kind`. `kind` names the harness that the command runs (for example `omp`), and its default SHALL be the base name of the first word of `command`. docent SHALL reject an alias with an empty `command`, and name the alias in the error. docent SHALL expand a leading `~` or `~/` in each word of a harness command, alias or plain, to the home directory.

#### Scenario: Alias with a kind
- **WHEN** the config has `[harnesses.omp-sandbox]` with `command = "sandbox-exec -f omp.sb omp"` and `kind = "omp"`
- **THEN** the alias `omp-sandbox` runs that command and has the kind `omp`

#### Scenario: Alias without a kind
- **WHEN** the config has `[harnesses.fast]` with `command = "/opt/bin/claude --model haiku"` and no `kind`
- **THEN** the alias `fast` has the kind `claude`

#### Scenario: Home directory in a harness command
- **WHEN** the user runs `docent --harness '~/bin/box omp' ~/src/app`
- **THEN** the harness pane runs `box` from the `bin` directory in the home directory, with the arg `omp`, and has the kind `box`

#### Scenario: Alias with empty command
- **WHEN** the config has `[harnesses.broken]` with `command = ""`
- **THEN** docent prints an error that names `broken` and exits non-zero

### Requirement: Harness kind in the tab
When the harness kind differs from the base name of the first word of the harness command, the harness pane SHALL show the kind in its pane command, so other docent commands in the tab can read it. The harness process SHALL get the environment variable `DOCENT_HARNESS_KIND` set to the kind.

#### Scenario: Wrapped harness
- **WHEN** a tab is launched with the alias `omp-sandbox` (kind `omp`)
- **THEN** the harness pane command shows the kind `omp`, and the harness process has `DOCENT_HARNESS_KIND=omp`

### Requirement: Recent directories
Each successful launch, by `docent [dir]` or `docent new`, SHALL put the absolute directory at the top of a recent list in `$XDG_STATE_HOME/docent/recent.toml` (default `~/.local/state/docent/recent.toml`), remove it from lower positions, and keep at most 20 entries. If docent cannot write the file, it SHALL print a warning and the launch SHALL still succeed. docent SHALL NOT write to `config.toml`.

#### Scenario: Launch moves a directory to the top
- **WHEN** the recent list is `~/src/a` then `~/src/b`, and the user runs `docent ~/src/b`
- **THEN** the recent list is `~/src/b` then `~/src/a`

#### Scenario: Limit
- **WHEN** the recent list holds 20 entries and the user launches a new directory
- **THEN** the list holds 20 entries with the new directory at the top and the oldest removed

#### Scenario: State file not writable
- **WHEN** the state directory is not writable and the user runs `docent ~/src/app`
- **THEN** the tab opens, and docent prints a warning that names the state file
