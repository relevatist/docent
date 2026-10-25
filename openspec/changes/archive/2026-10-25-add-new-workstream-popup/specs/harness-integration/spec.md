# Spec Delta

## MODIFIED Requirements

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

## ADDED Requirements

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
