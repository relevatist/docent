# Commands

Run these inside a workstream tab. Add `--help` to any command for details.

```sh
docent open src/main.go:42       # show the file at line 42 in the editor
docent open --focus README.md    # same, and move focus to the editor
docent reload                    # reload all editor buffers from disk
docent send ref src/a.go:10-20   # paste a reference into the harness input
echo "some text" | docent send text
docent tour /tmp/tour.json       # start a guided tour, see tours.md
```

## `docent open`

Shows a file in the editor. `:line` is optional.

## `docent reload`

Reloads all editor buffers from disk.

WARNING: `docent reload` discards unsaved changes in all editor buffers.

With `--quiet`, it does nothing and exits zero outside a workstream. The reload hooks use this.

## `docent send`

Pastes into the harness input. It never submits the input. You submit it.

| Command | Pastes |
|---------|--------|
| `docent send ref src/a.go:10-20` | `@src/a.go:10-20 ` for claude and omp, `src/a.go:10-20 ` for others |
| `docent send text` | standard input |

The harness must enable bracketed paste (Claude Code, omp, and codex do). Otherwise, a newline in the sent text can submit it.

After a send, docent moves focus to the harness, unless the config sets `focus_on_send = false`.
