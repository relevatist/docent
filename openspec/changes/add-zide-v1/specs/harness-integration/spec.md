# Spec Delta

## Purpose

Lets any coding harness use zide through one generic agent skill, with small per-harness adapters only where a harness can do more than the skill allows.

## ADDED Requirements

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

### Requirement: Install the skill
`zide install skill` SHALL copy the skill into the skill directory of each harness the user names, and SHALL print each path it writes. It SHALL NOT overwrite a changed skill without `--force`.

#### Scenario: Install for Claude Code
- **WHEN** the user runs `zide install skill claude`
- **THEN** the skill is written to `~/.claude/skills/zide/SKILL.md` and the path is printed

#### Scenario: Existing changed skill
- **WHEN** the target skill file exists and differs from the shipped one
- **THEN** zide prints the path, writes nothing, and exits non-zero unless `--force` is given

### Requirement: Reload after harness edits
For harnesses with an after-edit hook, zide SHALL ship an adapter that runs `zide reload` after the harness edits or writes a file, with documented install steps. v1 SHALL ship adapters for Claude Code and oh-my-pi. Harnesses without an adapter SHALL rely on the skill telling them to run `zide reload`.

#### Scenario: Claude Code edits a file
- **WHEN** the Claude Code adapter is installed and Claude Code edits `a.go`
- **THEN** the editor in the same tab reloads `a.go`

#### Scenario: oh-my-pi edits a file
- **WHEN** the oh-my-pi adapter is installed and oh-my-pi edits `a.go`
- **THEN** the editor in the same tab reloads `a.go`

#### Scenario: Hook outside zide
- **WHEN** an adapter fires while the harness runs outside a zide workstream
- **THEN** the adapter does nothing and does not fail the harness's edit
