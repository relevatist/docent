# Design

## Context

See proposal.md - Why. Facts checked on the target machine (Zellij 0.45.1):

- `zellij action list-panes --all --json` gives `terminal_command`, `pane_command`, `pane_cwd`, `title`, `tab_id`, and `tab_name` for terminal panes. In a live tab it shows `hx` and `claude` for the editor and harness panes. `pane_cwd` is the pane's working directory (at start, the tab or session start directory). zide uses `pane_cwd` of the `editor` pane to find open workstreams. `terminal_command` includes the full argv, joined with single spaces, with no quotes or escapes: a pane with `command="/usr/bin/env"` and args `"A=/tmp/a b" "hx"` shows `/usr/bin/env A=/tmp/a b hx`, and a pane with args `"ZIDE_HARNESS_KIND=omp" "sleep" "600"` shows `/usr/bin/env ZIDE_HARNESS_KIND=omp sleep 600`. Plugin panes show `null`, and so does a default shell pane.
- `zellij action go-to-tab-by-id <id>` exists. It takes the `tab_id` from `list-panes` or `list-tabs --json` and makes that tab active. An unknown id (`99`) exits 0 and does nothing.
- `zellij action new-tab --name t2` run from a `--floating --close-on-exit` pane makes the new tab active (`list-tabs --json` shows `"active": true` for it). When the command exits, the floating pane closes and is removed from the old tab. In `list-panes`, `is_focused` is per tab: the old tab's editor pane stays `is_focused: true`, so use `list-tabs --json` `active` to find the active tab.
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

### Open-tab detection by `pane_cwd`
A tab holds a directory when its `editor` pane has that directory as `pane_cwd` in `list-panes --all --json`. Zellij 0.45.1 shows this field for terminal panes. The editor pane stays `command="hx"`, with no wrapper.

### Harness kind marker as an `env` wrapper
The harness pane runs through `/usr/bin/env` only when the kind differs from the base name of the first word of the command, so plain tabs keep `claude` as the pane command:

```kdl
pane name="harness" command="/usr/bin/env" {
    args "ZIDE_HARNESS_KIND=omp" "sandbox-exec" "-f" "omp.sb" "omp"
}
```

`terminal_command` is the argv joined with single spaces, with no quotes: `/usr/bin/env ZIDE_HARNESS_KIND=omp sandbox-exec -f omp.sb omp`. The kind is one word (the config check enforces this), so splitting on spaces is safe for the kind. Parsing lives in `internal/panes`, next to `Lookup`:

```go
// Marker returns the value of ZIDE_HARNESS_KIND in an `env ZIDE_HARNESS_KIND=kind cmd...` pane command.
func Marker(cmd string) (string, bool)
// Unwrap returns cmd without a leading `env ZIDE_HARNESS_KIND=kind`.
func Unwrap(cmd string) string
```

Alternatives: a tab-to-dir state file (goes stale when tabs close, and needs a lock); the pane name (`harness` is the lookup key for all commands); the tab name (the user can rename it).

### Tab name in the layout
`launch.Layout` takes the resolved tab name (`cmp.Or(name, filepath.Base(dir))`) and writes `tab name="<name>" {`. Without it, `--new-session-with-layout` names the tab `Tab #1`. `new-tab --name` still sets the name inside Zellij.

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

An alias matches only when `s` is exactly its name. Otherwise `s` is a command, and its kind is the base name of its first word. `launch.Layout` takes a `Harness` and the tab name, not a command string. `config.Load` checks aliases (empty `command`, `kind` with spaces) and returns an error that names the alias.

`send` gets the kind as `Marker(cmd)`, or else from `Unwrap(cmd)`, as it does now.

### Recent state in `internal/recent`

```toml
# $XDG_STATE_HOME/zide/recent.toml
dirs = ["/Users/thane/wt/retry/app", "/Users/thane/src/app"]
```

`recent.Add(dir)` loads, moves `dir` to the front, cuts the list to 20 entries, and writes through a temp file and rename. `launchDir` calls it once per launch: outside Zellij just before it attaches (attach blocks until the user detaches), and inside Zellij after the new tab opens. So `zide [dir]` and `zide new` both record. A write error prints a warning and does not fail the launch. Two launches at the same time can lose one entry. This is acceptable for a history list.

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
- The model gets the recent list at start, and the root walk runs in a goroutine that sends hits on a channel. A `tea.Cmd` reads the next batch (up to 100 hits, or the hits in 50ms after the first). So the recent list shows at once, and the repos append while the walk runs. When the picker exits, it cancels the walk and waits for the goroutine to return.
- The picker returns a result (`dir`, harness string, open/switch/another). `cmd/zide/new.go` does all the Zellij calls, so the model tests do not need a fake runner.
- Open-tab prompt: after `enter`, `new.go` checks the `pane_cwd` of each `editor` pane. If one matches, the picker shows a second screen with `switch` / `open another`. To keep this in one program, the model gets an `isOpen(dir) (tab string, ok bool)` func at start.

Tab switching inside the popup was considered (`tab` / `shift+tab` cycles the harness). But the harness field is now a text field, so `tab` moves focus between fields and `up` / `down` cycles the harness.

### Root walk
`filepath.WalkDir` per root, with no depth limit (like the fzf walker). It returns an `iter.Seq` and stops when its context is done. A directory with a `.git` entry is a hit and returns `fs.SkipDir`. Hidden directories (name starts with `.`) are skipped, so `.cache` and similar trees are not walked. The root itself goes through `filepath.EvalSymlinks`, and hits are joined back onto the root as given. `WalkDir` does not follow symlinks below the root. Errors in subtrees (permissions) are skipped.

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

`close_on_exit` would hide errors and warnings, so on a terminal inside Zellij, after a non-zero exit or any printed output, `zide new` waits for `enter`. A read error of the recent list shows as a warning line in the picker.

## Risks / Trade-offs

- [`terminal_command` may not include the args] -> Checked (task 1.1): it includes them.
- [The `env` wrapper changes the pane command that users and other tools see] -> Only `send` reads it, and `Unwrap` handles it.
- [`env` is not on `PATH` in an unusual environment] -> `/usr/bin/env` exists on macOS and Linux. Use the absolute path `/usr/bin/env`.
- [`:cd` in Helix changes the editor `pane_cwd`, so that tab is no longer detected for its first directory] -> Accepted. The spec states it.
- [Large roots make the walk slow] -> No descent into a repo, hidden directories skipped, symlinks below the root not followed (no loops), and the walk is asynchronous and streams hits, so the picker does not block.

## Migration Plan

No migration. Existing configs keep working. Old tabs keep working, and the picker detects them by `pane_cwd`.
