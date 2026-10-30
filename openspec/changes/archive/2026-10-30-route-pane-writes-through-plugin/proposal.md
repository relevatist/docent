# Proposal

## Why

When an agent in any tab edits a file, the user's focused pane in the visible tab scrolls back to the bottom. The `docent reload` hook drives Helix with `zellij action send-keys/write-chars --pane-id`. Zellij (0.45.1 and `main`) sends `ClearScroll` to the caller's client before each write. A CLI caller has no tab, so Zellij falls back to the first client's focused pane. One reload sends 4 such writes. `docent open`, `send`, and `tour` have the same problem.

The Zellij plugin API `write_to_pane_id` writes to a pane without `ClearScroll`.

## What Changes

- Add a small Zellij plugin (Rust, `zellij-tile`) that writes the bytes of a pipe message to a pane. `make` builds it and docent embeds the `.wasm`.
- docent sends all pane writes (Helix keys, Helix commands, pastes) as bytes through `zellij pipe` to that plugin, one pipe call for each write.
- When the plugin does not answer (not installed, or a session started before install), docent falls back to `zellij action write --pane-id`. The scroll jump stays in that case.
- `docent install zellij` also writes the `.wasm`, adds it to `load_plugins` in the Zellij config in a second marked node, and grants the plugin its permissions in Zellij's permission cache.
- Focus moves (`focus-pane-id`) are not changed: the user asks for them.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `workspace`: new requirement that docent pane writes do not change the scroll position of other panes.
- `harness-integration`: `docent install zellij` also installs, preloads, and grants permissions to the docent Zellij plugin.

## Impact

- New Rust crate `plugin/`. No `.wasm` is in Git. Building docent needs Rust; `go install ...@latest` no longer works.
- `internal/zellij`: new byte write through the plugin, with fallback. `Paste`, `WriteChars`, `SendKeys` go away.
- `internal/helix`, `internal/send`: build byte sequences.
- `cmd/docent/install_zellij.go`: `load_plugins` node, `.wasm` file, permission cache.
- Docs: `docs/install.md`, `docs/keys.md` (config changes), `docs/troubleshooting.md`, `README.md` if needed.
- Users must run `docent install zellij` again and start a new Zellij session to get the fix.
