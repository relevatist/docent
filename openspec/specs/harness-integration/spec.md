# harness-integration Specification

## Purpose

Lets any coding harness use docent through one generic agent skill, with small per-harness adapters only where a harness can do more than the skill allows.

## Requirements

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
`docent install [claude|codex|omp|zellij]... [--force]` SHALL install docent for each named item. With no names on a terminal, it SHALL show a checklist of the three harnesses and `zellij`, each checked when its command is on `PATH`, and SHALL install the checked ones when the user confirms. With no names and no terminal, it SHALL print usage and exit 2. It SHALL print each path it writes and each command it runs.

#### Scenario: Names given
- **WHEN** the user runs `docent install codex omp`
- **THEN** docent installs for codex and omp and shows no checklist

#### Scenario: Checklist
- **WHEN** the user runs `docent install` on a terminal with `claude`, `omp`, and `zellij` on `PATH`
- **THEN** a checklist shows Claude Code, oh-my-pi, and Zellij checked and codex unchecked

#### Scenario: Zellij item in the checklist
- **WHEN** the checklist shows the `zellij` item
- **THEN** its detail names the `Alt w` key and the Zellij config path that docent would change

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
docent SHALL check every chosen item before it writes a file or runs a command that changes anything. If a chosen item's command is not on `PATH`, a target file has changes and `--force` is not given, or the Zellij config cannot take the key (see "Install the Zellij key"), docent SHALL write nothing for any item, name each problem, and exit non-zero.

#### Scenario: Missing command
- **WHEN** the user runs `docent install claude codex` and `claude` is not on `PATH`
- **THEN** docent names `claude`, writes nothing for codex either, and exits non-zero

#### Scenario: Existing changed file
- **WHEN** a target skill or extension file exists and differs from the shipped one
- **THEN** docent prints the path, writes nothing, and exits non-zero unless `--force` is given

#### Scenario: Zellij config refused
- **WHEN** the user runs `docent install codex zellij` and the Zellij config cannot take the key
- **THEN** docent names the problem, writes nothing for codex either, and exits non-zero

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

### Requirement: Zellij config path
For `zellij`, docent SHALL change the Zellij config file that Zellij loads: `$ZELLIJ_CONFIG_FILE`, else `$ZELLIJ_CONFIG_DIR/config.kdl`, else `config.kdl` in the first user config directory that exists in Zellij's lookup order. When no user config directory exists, docent SHALL create `~/.config/zellij/config.kdl`.

#### Scenario: Config directory from the environment
- **WHEN** `ZELLIJ_CONFIG_DIR` is set and `ZELLIJ_CONFIG_FILE` is not
- **THEN** docent uses `$ZELLIJ_CONFIG_DIR/config.kdl`, also when that file does not exist

#### Scenario: Only the system config
- **WHEN** no user config directory exists and `/etc/zellij` exists
- **THEN** docent writes nothing, tells the user to copy the system config to `~/.config/zellij/config.kdl` first, and exits non-zero, because a new user config would hide the system config

### Requirement: Install the Zellij key
For `zellij`, docent SHALL add the `Alt w` bind from `assets/zellij/keys.kdl` to the Zellij config as a marked node: the `shared_except "locked"` node of the asset, with 4 spaces of indent, between the lines `    // docent: begin (managed by `docent install`)` and `    // docent: end`. docent SHALL change only the marked node.

#### Scenario: No config file
- **WHEN** no Zellij config file exists
- **THEN** docent creates `config.kdl` with `keybinds {`, the marked node, and `}`

#### Scenario: Config with a keybinds block
- **WHEN** the config has a top-level `keybinds` node (for example `keybinds clear-defaults=true {`) that opens on its own line
- **THEN** docent inserts the marked node on the line after the first such line, and the rest of the file is unchanged byte for byte

#### Scenario: Comments are not nodes
- **WHEN** the word `keybinds` is in a `//` or `/* */` comment before the `keybinds` node
- **THEN** docent inserts the marked node after the `keybinds` node line, not after the comment

#### Scenario: Config without a keybinds block
- **WHEN** the config has no `keybinds` node
- **THEN** docent appends one `keybinds` block that holds the marked node, because Zellij uses only the first `keybinds` block

#### Scenario: Keybinds in a form docent cannot change
- **WHEN** the first `keybinds` node does not start its line (for example `theme "x"; keybinds {`) or is quoted, does not end its line with `{` (for example the `{` is on the next line), or its line ends inside a comment or string, or the file has KDL that docent does not parse (raw strings, `/-` comments)
- **THEN** docent writes nothing, tells the user to add the snippet by hand, and exits non-zero; it does not append a second `keybinds` block

#### Scenario: Byte order mark and CRLF
- **WHEN** the config starts with a UTF-8 byte order mark, or the `keybinds` line ends with CRLF
- **THEN** docent keeps the byte order mark, writes the marked node with the line ending of the `keybinds` line, and ignores `\r` when it compares the marked node on a rerun

#### Scenario: Markers docent cannot use
- **WHEN** the config has more than one begin or end marker, or the end marker is before the begin marker
- **THEN** docent names the file, tells the user to fix it by hand, writes nothing, and exits non-zero; markers in a `/* */` comment do not count, and when only such markers exist docent tells the user to add the snippet by hand

#### Scenario: Begin marker without an end marker
- **WHEN** the config has a `// docent: begin` line without a `// docent: end` line, or the reverse
- **THEN** docent names the file, says which marker is missing, tells the user to fix the file by hand, writes nothing, and exits non-zero

#### Scenario: Rerun with the same content
- **WHEN** the marked node matches the shipped one
- **THEN** docent does not change the file

#### Scenario: Marked node changed by the user
- **WHEN** the marked node differs from the shipped one and `--force` is not given
- **THEN** docent names the file, writes nothing, and exits non-zero; with `--force` it replaces only the marked node

#### Scenario: Alt w bound outside the marked node
- **WHEN** the config binds `Alt w` outside the marked node, with the modifier in any case and with any spaces (for example `"alt w"` or `"ALT  w"`), as any key of a `bind`
- **THEN** docent names the file and the bind, writes nothing, and exits non-zero; when the bind is the snippet from `assets/zellij/keys.kdl`, docent says that it was added by hand and tells the user to remove it so that docent can manage it

### Requirement: Safe Zellij config write
Before it writes, docent SHALL run `zellij --config <new file> setup --check` on the new content and refuse if it fails, and SHALL refuse if it cannot make a file in the config directory. When it changes an existing file, docent SHALL keep the old file as `config.kdl.docent.bak` next to it unless that file exists, write through a temp file and rename, and keep the file mode. A new config file SHALL get mode `0644`. docent SHALL change the target of a symlinked config and refuse a hard-linked one.

#### Scenario: Config check fails
- **WHEN** `zellij setup --check` fails on the new content
- **THEN** docent prints its output, writes nothing, and exits non-zero

#### Scenario: Backup and mode
- **WHEN** docent changes an existing config with mode `0640`
- **THEN** `config.kdl.docent.bak` holds the old content and the new config keeps mode `0640`

#### Scenario: Backup kept
- **WHEN** docent changes the config and `config.kdl.docent.bak` exists
- **THEN** docent does not change `config.kdl.docent.bak`, because it holds the config from before docent, and says that it kept it

#### Scenario: Backup write fails
- **WHEN** docent cannot write all of the backup
- **THEN** docent leaves no `config.kdl.docent.bak`, does not change the config, and exits non-zero

#### Scenario: Hard-linked config
- **WHEN** the config file has other hard links
- **THEN** docent names the file, tells the user to add the key by hand or to replace the hard link with a symlink, writes nothing, and exits non-zero

#### Scenario: Config directory not writable
- **WHEN** the user runs `docent install codex zellij` and docent cannot make a file in the directory of the Zellij config (for example a symlink into a read-only directory)
- **THEN** docent names the problem, writes nothing for codex either, and exits non-zero

### Requirement: Skill explains how to write step notes
The skill SHALL tell the harness to write each step note in Markdown that explains the change at that step. The skill SHALL include the views the harness can use, each with an example: pseudocode, call trees, component trees, file trees, `diff` sketches of these, code blocks, and Mermaid flowchart, sequence, and ER diagrams. The skill SHALL be complete without other skills installed. The skill SHALL NOT suggest HTML or other output that the tour pane cannot show.

#### Scenario: Harness writes a tour
- **WHEN** the harness loads only the docent skill and writes a tour
- **THEN** the skill gives it the note format and the view examples

#### Scenario: No HTML
- **WHEN** the harness reads the skill
- **THEN** the skill does not suggest HTML files or opening a browser

### Requirement: Install the docent Zellij plugin
For `zellij`, docent SHALL also write its embedded plugin to `$XDG_DATA_HOME/docent/zellij/docent.wasm` (default `~/.local/share/docent/zellij/docent.wasm`). It SHALL add `"file:<path>"` to the first top-level `load_plugins` block of the Zellij config as a marked node, between `    // docent: plugin begin (managed by `docent install`)` and `    // docent: plugin end`, or append a `load_plugins` block that holds the marked node.

#### Scenario: Config with load_plugins
- **WHEN** the config has `load_plugins {` on its own line
- **THEN** docent inserts the marked node on the next line, and the rest of the file is unchanged byte for byte

#### Scenario: Config without load_plugins
- **WHEN** the config has no `load_plugins` node
- **THEN** docent appends a `load_plugins` block with `"zellij:link"` and the marked node, so the Zellij default is kept

#### Scenario: Rerun with the same content
- **WHEN** the marked node matches the shipped one
- **THEN** docent does not change the config

#### Scenario: Changed plugin file
- **WHEN** the `.wasm` file differs from the embedded one
- **THEN** docent replaces it without `--force`, like the Claude Code plugin


#### Scenario: Plugin node in a form docent cannot change
- **WHEN** the first `load_plugins` node does not start its line, or does not end its line with `{`, or its markers are unpaired or out of order
- **THEN** docent names the file and the problem, writes nothing for any item, and exits non-zero

### Requirement: Grant the plugin its permissions
For `zellij`, docent SHALL grant the plugin `WriteToStdin`, `ReadCliPipes`, and `ReadApplicationState` in the Zellij permission cache (`permissions.kdl` in the Zellij cache directory), so Zellij does not ask the user. docent SHALL change only the entry for its plugin path and SHALL keep the other entries.

#### Scenario: No permission cache
- **WHEN** the Zellij cache directory has no `permissions.kdl`
- **THEN** docent creates it with one entry for its plugin

#### Scenario: Other plugins granted
- **WHEN** `permissions.kdl` has entries for other plugins
- **THEN** docent adds its entry and the other entries are unchanged

#### Scenario: Install output
- **WHEN** `docent install zellij` writes the plugin, the config, and the permission cache
- **THEN** docent prints each of the three paths
