# Keys

## Helix keys

Add the tables from [`assets/helix/keys.toml`](../assets/helix/keys.toml) to `~/.config/helix/config.toml`:

| Key | Action |
|-----|--------|
| `space z r` | paste a reference to the selected lines into the harness input |
| `space z s` | paste the selected text into the harness input |

Limit: Helix gives `space z r` the buffer name through the shell, and cannot quote it for the shell. The binding puts the name in single quotes, so a name that contains `'` is not safe with this binding.

## Zellij key

| Key | Action |
|-----|--------|
| `Alt w` | open `docent new` in a floating pane (all modes except locked). The pane closes when `docent new` exits. |

To add it, run `docent install zellij`.

To add it by hand, add the `keybinds` block from [`assets/zellij/keys.kdl`](../assets/zellij/keys.kdl) to `~/.config/zellij/config.kdl`. If your config already has a `keybinds` block, put the `shared_except "locked"` block in it. Zellij uses only the first `keybinds` block in the file.

### How `docent install zellij` changes the config

docent changes the config file that Zellij loads. It uses the first of these:

1. `$ZELLIJ_CONFIG_FILE`
2. `$ZELLIJ_CONFIG_DIR/config.kdl`
3. `config.kdl` in the first of `~/.config/zellij` and the platform config dir that exists
4. A new `~/.config/zellij/config.kdl`

If only `/etc/zellij` exists, docent refuses and tells you to copy its `config.kdl` to `~/.config/zellij` first.

docent puts the key between `// docent: begin` and `// docent: end` lines, on the line after the first `keybinds ... {` line. If there is no `keybinds` block, docent appends one.

- docent changes nothing else in the file. Before it appends a block, it adds a newline at the end of the file if there is none.
- It keeps a byte order mark and CRLF line endings.
- If the config is a symlink, docent changes the file that it points to.

docent does not install the key, and tells you why, when:

- your config already binds `Alt w` (in any case, for example `"alt w"`), also when it is the snippet that you added by hand. Remove it, and docent adds it between the markers. docent does not find rare forms, for example `"Alt\tw"`, a quoted `"bind"`, a `\"` before the key, or a `\` line continuation.
- you changed the lines between the markers. Use `--force` to replace them.
- there is more than one `// docent: begin` or `// docent: end` line, they are out of order, or one of them is missing. Fix the file by hand.
- it cannot find where to put the key. For example, the `{` of `keybinds` is on the next line, another node is before `keybinds` on its line, or a comment or string opens on the `keybinds` line.
- it cannot write to the directory of the config.
- the config has other hard links. docent would split them. Add the key by hand, or replace the hard link with a symlink.
- `zellij setup --check` fails on the new config.

### Backup

The first time docent changes the config, it keeps the old file as `config.kdl.docent.bak`.

- docent writes the backup to a temp file and links it into place, so a failed write leaves no partial backup.
- If `config.kdl.docent.bak` exists, docent does not replace it, because it holds the config from before docent.
