# Troubleshooting

| Problem | Cause | Fix |
|---------|-------|-----|
| `must run inside a docent workstream` | The command ran outside a Zellij pane. | Run it in a pane of a docent tab. |
| `no "editor" pane in this tab` (or `harness`, `shell`) | docent did not open the tab, or a pane has a new name. | Open the tab with `docent`. Do not rename its panes. |
| `docent needs zellij 0.45 or later` | Zellij is too old. | Upgrade Zellij. |
| `docent new: must run inside Zellij` | `docent new` ran outside Zellij. | Use `Alt w`, or run `docent new` in a Zellij pane. |
| `fzf is not on PATH` | `docent new` needs fzf. | Install fzf. |
| `docent install: <name> is not on PATH` | docent installs only for commands that it finds. | Install the harness first, or leave it out. |
| The editor does not show the agent's edits. | No reload hook: the plugin or extension is not installed, Claude Code did not restart, or the harness is codex. | Run `docent install <harness>` and restart the harness. For codex, the skill tells it to run `docent reload`. |
| Unsaved editor changes are gone. | `docent reload` (also from the reload hook) discards unsaved changes. | Save in Helix before the agent edits files. |
| `docent new` opens another tab for a directory that is open. | You ran `:cd` in Helix, so docent cannot find the tab. | Go to the tab yourself. |
| The focused pane jumps to the bottom when an agent in another tab edits files. | The docent Zellij plugin is not installed, or the session started before the install. docent then uses `zellij action write` and `zellij action paste`. | Run `docent install zellij` again and start a new Zellij session. |
| `docent open`, `reload`, or `send` takes about 2 s. | Zellij did not grant the plugin its permissions: the entry in `permissions.kdl` is missing, so Zellij waits for an answer that it does not show. | Run `docent install zellij` again and start a new Zellij session. |
| `make install` fails with `cargo: command not found`. | Rust is not installed. docent embeds its Zellij plugin, which cargo builds. | Install Rust with [rustup](https://rustup.rs/), then run `make install` again. |
| `go build` or `go test` fails with `pattern zellij/docent.wasm: no matching files found`. | The plugin is not built. It is not in Git. | Run `make plugin` once. |
| `docent install zellij` does not add the key. | See the reasons in [Keys](keys.md#how-docent-install-zellij-changes-the-config). | Do what the message tells you. |
