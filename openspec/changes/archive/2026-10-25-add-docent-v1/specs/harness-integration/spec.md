# Spec Delta

## Purpose

Lets any coding harness use docent through one generic agent skill, with small per-harness adapters only where a harness can do more than the skill allows.

## ADDED Requirements

### Requirement: Generic agent skill
docent SHALL ship one agent skill in the standard `SKILL.md` format. The skill SHALL tell the harness when and how to run `docent open`, `docent reload`, and `docent tour`, and SHALL define the tour file format with an example. The skill SHALL NOT depend on any one harness.

#### Scenario: Harness reads the skill
- **WHEN** Claude Code, oh-my-pi, or codex loads the skill
- **THEN** the skill gives the commands and the tour format with no harness-specific steps

### Requirement: Skill only applies inside docent
The skill SHALL tell the harness to use docent commands only when the `ZELLIJ` environment variable is set and `docent` is on `PATH`.

#### Scenario: Harness outside docent
- **WHEN** the harness runs outside Zellij
- **THEN** the skill tells it not to run docent commands

### Requirement: Install for chosen harnesses
`docent install [claude|codex|omp]... [--force]` SHALL install docent for each named harness. With no names on a terminal, it SHALL show a checklist of all three harnesses, each checked when its command is on `PATH`, and SHALL install for the checked ones when the user confirms. With no names and no terminal, it SHALL print usage and exit 2. It SHALL print each path it writes and each command it runs.

#### Scenario: Names given
- **WHEN** the user runs `docent install codex omp`
- **THEN** docent installs for codex and omp and shows no checklist

#### Scenario: Checklist
- **WHEN** the user runs `docent install` on a terminal with `claude` and `omp` on `PATH`
- **THEN** a checklist shows Claude Code and oh-my-pi checked and codex unchecked

#### Scenario: Checklist cancelled
- **WHEN** the user cancels the checklist
- **THEN** docent writes nothing and exits non-zero

#### Scenario: No terminal
- **WHEN** the user runs `docent install` with no names and no terminal
- **THEN** docent prints usage and exits 2

#### Scenario: Unknown harness
- **WHEN** the user runs `docent install vim`
- **THEN** docent names the unknown harness and exits 2

### Requirement: Check before changing anything
docent SHALL check every chosen harness before it writes a file or runs a command. If a chosen harness's command is not on `PATH`, or a target file has changes and `--force` is not given, docent SHALL write nothing for any harness, name each problem, and exit non-zero.

#### Scenario: Missing command
- **WHEN** the user runs `docent install claude codex` and `claude` is not on `PATH`
- **THEN** docent names `claude`, writes nothing for codex either, and exits non-zero

#### Scenario: Existing changed file
- **WHEN** a target skill or extension file exists and differs from the shipped one
- **THEN** docent prints the path, writes nothing, and exits non-zero unless `--force` is given

### Requirement: Claude Code plugin
For `claude`, docent SHALL write its embedded Claude Code plugin (the skill and the reload hook) to `$XDG_DATA_HOME/docent/claude-plugin/` (default `~/.local/share/docent/claude-plugin/`), register that directory as the `docent` marketplace, and install `docent@docent` at user scope, all through the `claude` CLI. When the `docent` marketplace is already registered from that directory, docent SHALL update the marketplace and the plugin instead. docent SHALL NOT edit Claude Code settings files.

#### Scenario: First install
- **WHEN** the user installs for `claude`
- **THEN** docent writes the plugin, runs `claude plugin marketplace add <dir>` and `claude plugin install docent@docent --scope user`, and `claude plugin list` shows `docent@docent`

#### Scenario: Reinstall
- **WHEN** the user installs for `claude` again after upgrading docent
- **THEN** the installed plugin matches the new binary's skill and hook

#### Scenario: Marketplace from another source
- **WHEN** a `docent` marketplace is registered from another source
- **THEN** docent writes nothing, tells the user to run `claude plugin marketplace remove docent`, and exits non-zero

#### Scenario: Legacy skill
- **WHEN** `~/.claude/skills/docent/SKILL.md` exists
- **THEN** docent removes it if it matches the shipped skill, and otherwise warns and leaves it

### Requirement: Codex and oh-my-pi install
For `codex`, docent SHALL write the skill to `~/.agents/skills/docent/SKILL.md`. For `omp`, docent SHALL write the skill to the same path, because omp reads `~/.agents/skills`, and SHALL write the extension to `~/.omp/agent/extensions/docent-reload.ts`.

#### Scenario: Install for oh-my-pi
- **WHEN** the user runs `docent install omp`
- **THEN** docent writes `~/.agents/skills/docent/SKILL.md` and `~/.omp/agent/extensions/docent-reload.ts` and prints both paths

### Requirement: Repository marketplace
The repository root SHALL hold `.claude-plugin/marketplace.json` that lists the `docent` plugin from `./assets/claude-plugin`, so `claude plugin marketplace add relevatist/docent` installs the same plugin that `docent install` installs.

#### Scenario: Install from the repository
- **WHEN** the user runs `claude plugin marketplace add relevatist/docent` and `claude plugin install docent@docent`
- **THEN** Claude Code has the same skill and reload hook as after `docent install claude`

### Requirement: Reload after harness edits
For harnesses with an after-edit hook, docent SHALL ship an adapter that runs `docent reload` after the harness edits or writes a file, and `docent install` SHALL install it. v1 SHALL ship a `PostToolUse` hook in the Claude Code plugin and an oh-my-pi extension. Harnesses without an adapter SHALL rely on the skill telling them to run `docent reload`.

#### Scenario: Claude Code edits a file
- **WHEN** the Claude Code adapter is installed and Claude Code edits `a.go`
- **THEN** the editor in the same tab reloads `a.go`

#### Scenario: oh-my-pi edits a file
- **WHEN** the oh-my-pi adapter is installed and oh-my-pi edits `a.go`
- **THEN** the editor in the same tab reloads `a.go`

#### Scenario: Hook outside docent
- **WHEN** an adapter fires while the harness runs outside a docent workstream
- **THEN** the adapter does nothing and does not fail the harness's edit
