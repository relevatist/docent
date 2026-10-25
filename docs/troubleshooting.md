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
| `docent install zellij` does not add the key. | See the reasons in [Keys](keys.md#how-docent-install-zellij-changes-the-config). | Do what the message tells you. |
