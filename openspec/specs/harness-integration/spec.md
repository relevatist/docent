# harness-integration Specification

## Purpose

Lets any coding harness use zide through one generic agent skill, with small per-harness adapters only where a harness can do more than the skill allows.

## Requirements

### Requirement: Generic agent skill
zide SHALL ship one agent skill in the standard `SKILL.md` format. The skill SHALL tell the harness when and how to run `zide open`, `zide reload`, and `zide tour`, and SHALL define the tour file format with an example. The skill SHALL NOT depend on any one harness.

#### Scenario: Harness reads the skill
- **WHEN** Claude Code, oh-my-pi, or codex loads the skill
- **THEN** the skill gives the commands and the tour format with no harness-specific steps

### Requirement: Skill only applies inside zide
The skill SHALL tell the harness to use zide commands only when the `ZELLIJ` environment variable is set and `zide` is on `PATH`.

#### Scenario: Harness outside zide
- **WHEN** the harness runs outside Zellij
- **THEN** the skill tells it not to run zide commands

### Requirement: Install for chosen harnesses
`zide install [claude|codex|omp]... [--force]` SHALL install zide for each named harness. With no names on a terminal, it SHALL show a checklist of all three harnesses, each checked when its command is on `PATH`, and SHALL install for the checked ones when the user confirms. With no names and no terminal, it SHALL print usage and exit 2. It SHALL print each path it writes and each command it runs.

#### Scenario: Names given
- **WHEN** the user runs `zide install codex omp`
- **THEN** zide installs for codex and omp and shows no checklist

#### Scenario: Checklist
- **WHEN** the user runs `zide install` on a terminal with `claude` and `omp` on `PATH`
- **THEN** a checklist shows Claude Code and oh-my-pi checked and codex unchecked

#### Scenario: Checklist cancelled
- **WHEN** the user cancels the checklist
- **THEN** zide writes nothing and exits non-zero

#### Scenario: No terminal
- **WHEN** the user runs `zide install` with no names and no terminal
- **THEN** zide prints usage and exits 2

#### Scenario: Unknown harness
- **WHEN** the user runs `zide install vim`
- **THEN** zide names the unknown harness and exits 2

### Requirement: Check before changing anything
zide SHALL check every chosen harness before it writes a file or runs a command. If a chosen harness's command is not on `PATH`, or a target file has changes and `--force` is not given, zide SHALL write nothing for any harness, name each problem, and exit non-zero.

#### Scenario: Missing command
- **WHEN** the user runs `zide install claude codex` and `claude` is not on `PATH`
- **THEN** zide names `claude`, writes nothing for codex either, and exits non-zero

#### Scenario: Existing changed file
- **WHEN** a target skill or extension file exists and differs from the shipped one
- **THEN** zide prints the path, writes nothing, and exits non-zero unless `--force` is given

### Requirement: Claude Code plugin
For `claude`, zide SHALL write its embedded Claude Code plugin (the skill and the reload hook) to `$XDG_DATA_HOME/zide/claude-plugin/` (default `~/.local/share/zide/claude-plugin/`), register that directory as the `zide` marketplace, and install `zide@zide` at user scope, all through the `claude` CLI. When the `zide` marketplace is already registered from that directory, zide SHALL update the marketplace and the plugin instead. zide SHALL NOT edit Claude Code settings files.

#### Scenario: First install
- **WHEN** the user installs for `claude`
- **THEN** zide writes the plugin, runs `claude plugin marketplace add <dir>` and `claude plugin install zide@zide --scope user`, and `claude plugin list` shows `zide@zide`

#### Scenario: Reinstall
- **WHEN** the user installs for `claude` again after upgrading zide
- **THEN** the installed plugin matches the new binary's skill and hook

#### Scenario: Marketplace from another source
- **WHEN** a `zide` marketplace is registered from another source
- **THEN** zide writes nothing, tells the user to run `claude plugin marketplace remove zide`, and exits non-zero

#### Scenario: Legacy skill
- **WHEN** `~/.claude/skills/zide/SKILL.md` exists
- **THEN** zide removes it if it matches the shipped skill, and otherwise warns and leaves it

### Requirement: Codex and oh-my-pi install
For `codex`, zide SHALL write the skill to `~/.agents/skills/zide/SKILL.md`. For `omp`, zide SHALL write the skill to the same path, because omp reads `~/.agents/skills`, and SHALL write the extension to `~/.omp/agent/extensions/zide-reload.ts`.

#### Scenario: Install for oh-my-pi
- **WHEN** the user runs `zide install omp`
- **THEN** zide writes `~/.agents/skills/zide/SKILL.md` and `~/.omp/agent/extensions/zide-reload.ts` and prints both paths

### Requirement: Repository marketplace
The repository root SHALL hold `.claude-plugin/marketplace.json` that lists the `zide` plugin from `./assets/claude-plugin`, so `claude plugin marketplace add relevatist/zide` installs the same plugin that `zide install` installs.

#### Scenario: Install from the repository
- **WHEN** the user runs `claude plugin marketplace add relevatist/zide` and `claude plugin install zide@zide`
- **THEN** Claude Code has the same skill and reload hook as after `zide install claude`

### Requirement: Reload after harness edits
For harnesses with an after-edit hook, zide SHALL ship an adapter that runs `zide reload` after the harness edits or writes a file, and `zide install` SHALL install it. v1 SHALL ship a `PostToolUse` hook in the Claude Code plugin and an oh-my-pi extension. Harnesses without an adapter SHALL rely on the skill telling them to run `zide reload`.

#### Scenario: Claude Code edits a file
- **WHEN** the Claude Code adapter is installed and Claude Code edits `a.go`
- **THEN** the editor in the same tab reloads `a.go`

#### Scenario: oh-my-pi edits a file
- **WHEN** the oh-my-pi adapter is installed and oh-my-pi edits `a.go`
- **THEN** the editor in the same tab reloads `a.go`

#### Scenario: Hook outside zide
- **WHEN** an adapter fires while the harness runs outside a zide workstream
- **THEN** the adapter does nothing and does not fail the harness's edit
