# Install

## Requirements

- macOS or Linux
- Zellij 0.45+
- Helix 25.07+
- Go 1.27+ and [rustup](https://rustup.rs/) (to build)
- fzf (for `docent new`)

## Install docent

```sh
git clone https://github.com/relevatist/docent
cd docent
make install
```

`make install` builds the docent Zellij plugin with cargo, then installs `docent` into `$(go env GOPATH)/bin` (or `$GOBIN`). On the first build, rustup installs the Rust version and the `wasm32-wasip1` target from `plugin/rust-toolchain.toml`, and cargo downloads the crates. This needs network access and takes a few minutes.

`go install github.com/relevatist/docent/cmd/docent@latest` does not work: docent embeds the plugin, which is not in Git.

Make sure that `$(go env GOPATH)/bin` (or `$GOBIN`) is on your `PATH`.

Then:

1. Run `docent install` to set up your harnesses, and the Zellij key and plugin. See [Harnesses](harnesses.md).
2. Add the Helix keys. See [Keys](keys.md).

## What `docent install zellij` writes

| File | Change |
|------|--------|
| Zellij `config.kdl` | The `Alt w` key, between `// docent: begin` and `// docent: end`. See [Keys](keys.md#how-docent-install-zellij-changes-the-config). |
| Zellij `config.kdl` | `"file:<plugin path>"` in the first `load_plugins` block, between `// docent: plugin begin` and `// docent: plugin end`. If there is no `load_plugins` block, docent appends one with `"zellij:link"` (the Zellij default) and the plugin. |
| `~/.local/share/docent/zellij/docent.wasm` (or `$XDG_DATA_HOME/docent/zellij/docent.wasm`) | The docent Zellij plugin. docent always replaces this file. |
| Zellij `permissions.kdl` | The plugin permissions `WriteToStdin`, `ReadCliPipes`, and `ReadApplicationState`, so Zellij does not ask you. docent changes only the entry for its plugin. |

The Zellij `permissions.kdl` is in the Zellij cache directory:

| OS | Path |
|----|------|
| macOS | `~/Library/Caches/org.Zellij-Contributors.Zellij/permissions.kdl` |
| Linux | `~/.cache/zellij/permissions.kdl` (or `$XDG_CACHE_HOME/zellij/permissions.kdl`) |

Start a new Zellij session after the install. Zellij loads the plugin only when a session starts.

The plugin writes docent's keys and text to the editor and harness panes. Without it, docent uses `zellij action write` and `zellij action paste`, which scroll the focused pane of the visible tab to the bottom.

- docent does not change the permission cache if it cannot read it. It names the file. Fix or remove the file, then run `docent install zellij` again.
- If you changed the lines between the `// docent: plugin` markers, use `--force` to replace them.

## Uninstall

```sh
rm "$(go env GOPATH)/bin/docent"                # or $GOBIN/docent
claude plugin uninstall docent@docent           # claude
claude plugin marketplace remove docent
rm -rf ~/.local/share/docent                    # or $XDG_DATA_HOME/docent
rm -rf ~/.agents/skills/docent                  # codex, omp
rm ~/.omp/agent/extensions/docent-reload.ts     # omp
rm -rf ~/.config/docent ~/.local/state/docent   # config and recent directories
```

- Zellij: in `config.kdl`, remove the lines from `// docent: begin` to `// docent: end`, and from `// docent: plugin begin` to `// docent: plugin end`. Or, if you did not change the config after `docent install zellij`, replace it with `config.kdl.docent.bak`.
- Zellij permission cache: in `permissions.kdl`, remove the block for `.../docent/zellij/docent.wasm`.
- Helix: remove the `[keys.normal.space.z]` and `[keys.select.space.z]` tables from `~/.config/helix/config.toml`.
