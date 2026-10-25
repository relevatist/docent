# Spec Delta

## MODIFIED Requirements

### Requirement: Configurable harness command
zide SHALL read the harness command from `~/.config/zide/config.toml` (`harness` key). A `--harness` flag SHALL override it. The default SHALL be `claude`. The `harness` key and `--harness` SHALL take either a command with args or the name of a harness alias.

#### Scenario: Config sets harness
- **WHEN** the config has `harness = "omp"` and the user runs `zide`
- **THEN** the harness pane runs `omp`

#### Scenario: Flag overrides config
- **WHEN** the config has `harness = "omp"` and the user runs `zide --harness codex`
- **THEN** the harness pane runs `codex`

#### Scenario: Flag names an alias
- **WHEN** the config has `[harnesses.omp-sandbox]` with `command = "sandbox-exec -f omp.sb omp"` and the user runs `zide --harness omp-sandbox`
- **THEN** the harness pane runs `sandbox-exec -f omp.sb omp`

## ADDED Requirements

### Requirement: Harness aliases
The config SHALL accept harness aliases as `[harnesses.<name>]` tables with a required `command` and an optional `kind`. `kind` names the harness that the command runs (for example `omp`), and its default SHALL be the base name of the first word of `command`. zide SHALL reject an alias with an empty `command`, and name the alias in the error.

#### Scenario: Alias with a kind
- **WHEN** the config has `[harnesses.omp-sandbox]` with `command = "sandbox-exec -f omp.sb omp"` and `kind = "omp"`
- **THEN** the alias `omp-sandbox` runs that command and has the kind `omp`

#### Scenario: Alias without a kind
- **WHEN** the config has `[harnesses.fast]` with `command = "/opt/bin/claude --model haiku"` and no `kind`
- **THEN** the alias `fast` has the kind `claude`

#### Scenario: Alias with empty command
- **WHEN** the config has `[harnesses.broken]` with `command = ""`
- **THEN** zide prints an error that names `broken` and exits non-zero

### Requirement: Harness kind in the tab
When the harness kind differs from the base name of the first word of the harness command, the harness pane SHALL show the kind in its pane command, so other zide commands in the tab can read it. The harness process SHALL get the environment variable `ZIDE_HARNESS_KIND` set to the kind.

#### Scenario: Wrapped harness
- **WHEN** a tab is launched with the alias `omp-sandbox` (kind `omp`)
- **THEN** the harness pane command shows the kind `omp`, and the harness process has `ZIDE_HARNESS_KIND=omp`

### Requirement: Directory marker on the editor pane
The editor pane SHALL show the absolute workstream directory in its pane command, so zide can find which tab holds which directory. The editor process SHALL get the environment variable `ZIDE_DIR` set to that directory.

#### Scenario: Marker present
- **WHEN** the user runs `zide ~/src/app` inside Zellij
- **THEN** the editor pane command in `zellij action list-panes` shows the absolute path of `~/src/app`

### Requirement: Recent directories
Each successful launch, by `zide [dir]` or `zide new`, SHALL put the absolute directory at the top of a recent list in `$XDG_STATE_HOME/zide/recent.toml` (default `~/.local/state/zide/recent.toml`), remove it from lower positions, and keep at most 20 entries. If zide cannot write the file, it SHALL print a warning and the launch SHALL still succeed. zide SHALL NOT write to `config.toml`.

#### Scenario: Launch moves a directory to the top
- **WHEN** the recent list is `~/src/a` then `~/src/b`, and the user runs `zide ~/src/b`
- **THEN** the recent list is `~/src/b` then `~/src/a`

#### Scenario: Limit
- **WHEN** the recent list holds 20 entries and the user launches a new directory
- **THEN** the list holds 20 entries with the new directory at the top and the oldest removed

#### Scenario: State file not writable
- **WHEN** the state directory is not writable and the user runs `zide ~/src/app`
- **THEN** the tab opens, and zide prints a warning that names the state file
