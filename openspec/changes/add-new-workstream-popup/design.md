# Design

## Context

See proposal.md - Why. Facts checked on the target machine (Zellij 0.45.1):

- `zellij action list-panes --all --json` has no cwd field. It has `terminal_command`, `title`, `tab_id`, `tab_name`. In a live tab it shows `hx` and `claude` for the editor and harness panes. Not yet checked: whether `terminal_command` includes the args.
- `zellij action go-to-tab-by-id <id>` exists.
- Default keybinds use `Alt c f h i j k l n o p`. `Alt w` is free, and the user's `config.kdl` does not bind it.
- `BurntSushi/toml` can only encode a whole file, so writing state into `config.toml` would drop user comments.
- `internal/send.FormatRef` takes the kind from the base name of the first word of the pane command. `panes.Lookup` returns that command.

## Goals / Non-Goals

**Goals:**
- `zide new` reuses `launch.Run` for the tab. The picker only produces `(dir, name, command, kind)`.
- All open-tab detection comes from `list-panes`, so no state can go stale.

**Non-Goals:**
- Creating directories or worktrees from the picker (the `workspace` spec forbids it).
- Ranked fuzzy matching. A subsequence filter in list order is enough for a few hundred entries.
- Editing the Zellij config.

## Decisions

### Markers as `env` wrappers in the layout
Each pane that needs a marker runs through `env`:

```kdl
pane name="editor" command="env" size="60%" focus=true {
    args "ZIDE_DIR=/abs/dir" "hx"
}
pane name="harness" command="env" {
    args "ZIDE_HARNESS_KIND=omp" "sandbox-exec" "-f" "omp.sb" "omp"
}
```

The harness pane is wrapped only when the kind differs from the base name of the first word of the command, so plain tabs keep `claude` as the pane command.

Parsing lives in `internal/panes`, next to `Lookup`:

```go
// Marker returns the value of KEY= in an `env KEY=value cmd...` pane command.
func Marker(cmd, key string) (string, bool)
// Unwrap returns cmd without a leading `env KEY=value...`.
func Unwrap(cmd string) string
```

A directory can contain spaces, and `terminal_command` may join the args with spaces. So the editor marker is parsed as the text between `env ZIDE_DIR=` and the final ` hx`. The `kind` value must be one word (the config check enforces this).

Alternatives: a tab-to-dir state file (goes stale when tabs close, and needs a lock); the pane name (`editor` is the lookup key for all commands); the tab name (the user can rename it).

### Harness resolution in `internal/config`

```toml
harness = "omp-sandbox"
roots = ["~/src", "~/wt"]

[harnesses.omp-sandbox]
command = "sandbox-exec -f ~/omp.sb omp"
kind = "omp"
```

```go
type Harness struct{ Command, Kind string }
// Resolve returns the alias named s, or s as a plain command.
func (c Config) Resolve(s string) (Harness, error)
```

An alias matches only when `s` is exactly its name. Otherwise `s` is a command, and its kind is the base name of its first word. `launch.Layout` takes a `Harness` and the directory, not a command string. `config.Load` checks aliases (empty `command`, `kind` with spaces) and returns an error that names the alias.

`send` gets the kind as `Marker(cmd, "ZIDE_HARNESS_KIND")`, or else from `Unwrap(cmd)`, as it does now.

### Recent state in `internal/recent`

```toml
# $XDG_STATE_HOME/zide/recent.toml
dirs = ["/Users/thane/wt/retry/app", "/Users/thane/src/app"]
```

`recent.Add(dir)` loads, moves `dir` to the front, cuts the list to 20 entries, and writes through a temp file and rename. `launchDir` calls it after `launch.Run` succeeds, so `zide [dir]` and `zide new` both record. A write error prints a warning and does not fail the launch. Two launches at the same time can lose one entry. This is acceptable for a history list.

### Picker as a bubbletea model in `internal/tui`

```
> rtap_
  ~/wt/retry/app        recent
  ~/src/app             recent
  ~/src/lib
harness: omp-sandbox
tab: field  up/down: move or cycle  enter: open  esc: cancel
```

- No new dependency. The text fields are a string and a cursor at the end: runes append and `backspace` deletes. `bubbles/textinput` is not needed for one-line input with no cursor movement.
- The model gets the recent list at start, and the root walk runs as a `tea.Cmd`, so the recent list shows at once and the repos append when the walk ends.
- The picker returns a result (`dir`, harness string, open/switch/another). `cmd/zide/new.go` does all the Zellij calls, so the model tests do not need a fake runner.
- Open-tab prompt: after `enter`, `new.go` checks the `ZIDE_DIR` markers. If one matches, the picker shows a second screen with `switch` / `open another`. To keep this in one program, the model gets an `isOpen(dir) (tab string, ok bool)` func at start.

Tab switching inside the popup was considered (`tab` / `shift+tab` cycles the harness). But the harness field is now a text field, so `tab` moves focus between fields and `up` / `down` cycles the harness.

### Root walk
`filepath.WalkDir` per root, with a depth check. A directory with a `.git` entry is a hit and returns `fs.SkipDir`. Hidden directories (name starts with `.`) are skipped, so `.cache` and similar trees are not walked. `WalkDir` does not follow symlinks. Errors in subtrees (permissions) are skipped.

### Tab name suffix
For "open another", `new.go` takes the set of `tab_name` values from `list-panes` and picks `<base>-2`, `<base>-3`, and so on. It passes the name to `launch.Run` as `--name`.

### Keybinding
`assets/zellij/keys.kdl`:

```kdl
keybinds {
    shared_except "locked" {
        bind "Alt w" {
            Run "zide" "new" {
                floating true
                close_on_exit true
                name "zide new"
            }
        }
    }
}
```

`zide new` opens the tab from inside the floating pane. Zellij focuses the new tab. The floating pane exits and closes in the old tab.

## Risks / Trade-offs

- [`terminal_command` may not include the args] -> Check this first (task 1.1). If the args are missing, use the pane title instead: name the pane `editor` and set the title through a command. If that also fails, stop and ask the user.
- [The `env` wrapper changes the pane command that users and other tools see] -> Only `send` reads it, and `Unwrap` handles it.
- [`env` is not on `PATH` in an unusual environment] -> `/usr/bin/env` exists on macOS and Linux. Use the absolute path `/usr/bin/env`.
- [Tabs from an older zide are not detected as open] -> Accepted. The spec states it.
- [Large roots make the walk slow] -> Depth 3, hidden directories skipped, and the walk is asynchronous, so the picker does not block.

## Migration Plan

No migration. Existing configs keep working. Old tabs keep working, but the picker does not detect them.
