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
`zide install [claude|codex|omp|zellij]... [--force]` SHALL install zide for each named item. With no names on a terminal, it SHALL show a checklist of the three harnesses and `zellij`, each checked when its command is on `PATH`, and SHALL install the checked ones when the user confirms. With no names and no terminal, it SHALL print usage and exit 2. It SHALL print each path it writes and each command it runs.

#### Scenario: Names given
- **WHEN** the user runs `zide install codex omp`
- **THEN** zide installs for codex and omp and shows no checklist

#### Scenario: Checklist
- **WHEN** the user runs `zide install` on a terminal with `claude`, `omp`, and `zellij` on `PATH`
- **THEN** a checklist shows Claude Code, oh-my-pi, and Zellij checked and codex unchecked

#### Scenario: Zellij item in the checklist
- **WHEN** the checklist shows the `zellij` item
- **THEN** its detail names the `Alt w` key and the Zellij config path that zide would change

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
zide SHALL check every chosen item before it writes a file or runs a command that changes anything. If a chosen item's command is not on `PATH`, a target file has changes and `--force` is not given, or the Zellij config cannot take the key (see "Install the Zellij key"), zide SHALL write nothing for any item, name each problem, and exit non-zero.

#### Scenario: Missing command
- **WHEN** the user runs `zide install claude codex` and `claude` is not on `PATH`
- **THEN** zide names `claude`, writes nothing for codex either, and exits non-zero

#### Scenario: Existing changed file
- **WHEN** a target skill or extension file exists and differs from the shipped one
- **THEN** zide prints the path, writes nothing, and exits non-zero unless `--force` is given

#### Scenario: Zellij config refused
- **WHEN** the user runs `zide install codex zellij` and the Zellij config cannot take the key
- **THEN** zide names the problem, writes nothing for codex either, and exits non-zero

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

### Requirement: Zellij config path
For `zellij`, zide SHALL change the Zellij config file that Zellij loads: `$ZELLIJ_CONFIG_FILE`, else `$ZELLIJ_CONFIG_DIR/config.kdl`, else `config.kdl` in the first user config directory that exists in Zellij's lookup order. When no user config directory exists, zide SHALL create `~/.config/zellij/config.kdl`.

#### Scenario: Config directory from the environment
- **WHEN** `ZELLIJ_CONFIG_DIR` is set and `ZELLIJ_CONFIG_FILE` is not
- **THEN** zide uses `$ZELLIJ_CONFIG_DIR/config.kdl`, also when that file does not exist

#### Scenario: Only the system config
- **WHEN** no user config directory exists and `/etc/zellij` exists
- **THEN** zide writes nothing, tells the user to copy the system config to `~/.config/zellij/config.kdl` first, and exits non-zero, because a new user config would hide the system config

### Requirement: Install the Zellij key
For `zellij`, zide SHALL add the `Alt w` bind from `assets/zellij/keys.kdl` to the Zellij config as a marked node: the `shared_except "locked"` node of the asset, with 4 spaces of indent, between the lines `    // zide: begin (managed by `zide install`)` and `    // zide: end`. zide SHALL change only the marked node.

#### Scenario: No config file
- **WHEN** no Zellij config file exists
- **THEN** zide creates `config.kdl` with `keybinds {`, the marked node, and `}`

#### Scenario: Config with a keybinds block
- **WHEN** the config has a top-level `keybinds` node (for example `keybinds clear-defaults=true {`) that opens on its own line
- **THEN** zide inserts the marked node on the line after the first such line, and the rest of the file is unchanged byte for byte

#### Scenario: Comments are not nodes
- **WHEN** the word `keybinds` is in a `//` or `/* */` comment before the `keybinds` node
- **THEN** zide inserts the marked node after the `keybinds` node line, not after the comment

#### Scenario: Config without a keybinds block
- **WHEN** the config has no `keybinds` node
- **THEN** zide appends one `keybinds` block that holds the marked node, because Zellij uses only the first `keybinds` block

#### Scenario: Keybinds in a form zide cannot change
- **WHEN** the first `keybinds` node does not start its line (for example `theme "x"; keybinds {`) or is quoted, does not end its line with `{` (for example the `{` is on the next line), or its line ends inside a comment or string, or the file has KDL that zide does not parse (raw strings, `/-` comments)
- **THEN** zide writes nothing, tells the user to add the snippet by hand, and exits non-zero; it does not append a second `keybinds` block

#### Scenario: Byte order mark and CRLF
- **WHEN** the config starts with a UTF-8 byte order mark, or the `keybinds` line ends with CRLF
- **THEN** zide keeps the byte order mark, writes the marked node with the line ending of the `keybinds` line, and ignores `\r` when it compares the marked node on a rerun

#### Scenario: Markers zide cannot use
- **WHEN** the config has more than one begin or end marker, or the end marker is before the begin marker
- **THEN** zide names the file, tells the user to fix it by hand, writes nothing, and exits non-zero; markers in a `/* */` comment do not count, and when only such markers exist zide tells the user to add the snippet by hand

#### Scenario: Begin marker without an end marker
- **WHEN** the config has a `// zide: begin` line without a `// zide: end` line, or the reverse
- **THEN** zide names the file, says which marker is missing, tells the user to fix the file by hand, writes nothing, and exits non-zero

#### Scenario: Rerun with the same content
- **WHEN** the marked node matches the shipped one
- **THEN** zide does not change the file

#### Scenario: Marked node changed by the user
- **WHEN** the marked node differs from the shipped one and `--force` is not given
- **THEN** zide names the file, writes nothing, and exits non-zero; with `--force` it replaces only the marked node

#### Scenario: Alt w bound outside the marked node
- **WHEN** the config binds `Alt w` outside the marked node, with the modifier in any case and with any spaces (for example `"alt w"` or `"ALT  w"`), as any key of a `bind`
- **THEN** zide names the file and the bind, writes nothing, and exits non-zero; when the bind is the snippet from `assets/zellij/keys.kdl`, zide says that it was added by hand and tells the user to remove it so that zide can manage it

### Requirement: Safe Zellij config write
Before it writes, zide SHALL run `zellij --config <new file> setup --check` on the new content and refuse if it fails, and SHALL refuse if it cannot make a file in the config directory. When it changes an existing file, zide SHALL keep the old file as `config.kdl.zide.bak` next to it unless that file exists, write through a temp file and rename, and keep the file mode. A new config file SHALL get mode `0644`. zide SHALL change the target of a symlinked config and refuse a hard-linked one.

#### Scenario: Config check fails
- **WHEN** `zellij setup --check` fails on the new content
- **THEN** zide prints its output, writes nothing, and exits non-zero

#### Scenario: Backup and mode
- **WHEN** zide changes an existing config with mode `0640`
- **THEN** `config.kdl.zide.bak` holds the old content and the new config keeps mode `0640`

#### Scenario: Backup kept
- **WHEN** zide changes the config and `config.kdl.zide.bak` exists
- **THEN** zide does not change `config.kdl.zide.bak`, because it holds the config from before zide, and says that it kept it

#### Scenario: Backup write fails
- **WHEN** zide cannot write all of the backup
- **THEN** zide leaves no `config.kdl.zide.bak`, does not change the config, and exits non-zero

#### Scenario: Hard-linked config
- **WHEN** the config file has other hard links
- **THEN** zide names the file, tells the user to add the key by hand or to replace the hard link with a symlink, writes nothing, and exits non-zero

#### Scenario: Config directory not writable
- **WHEN** the user runs `zide install codex zellij` and zide cannot make a file in the directory of the Zellij config (for example a symlink into a read-only directory)
- **THEN** zide names the problem, writes nothing for codex either, and exits non-zero
