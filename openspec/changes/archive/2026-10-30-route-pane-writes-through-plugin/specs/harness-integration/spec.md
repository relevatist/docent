# Spec Delta

## ADDED Requirements

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
