# Design

## Context

See proposal.md - Why. Facts checked on the target machine (Zellij 0.45.1):

- `zellij action list-panes --all --json` gives `terminal_command`, `pane_command`, `pane_cwd`, `title`, `tab_id`, and `tab_name` for terminal panes. In a live tab it shows `hx` and `claude` for the editor and harness panes. `pane_cwd` is the pane's working directory (at start, the tab or session start directory). docent uses `pane_cwd` of the `editor` pane to find open workstreams. `terminal_command` includes the full argv, joined with single spaces, with no quotes or escapes: a pane with `command="/usr/bin/env"` and args `"A=/tmp/a b" "hx"` shows `/usr/bin/env A=/tmp/a b hx`, and a pane with args `"DOCENT_HARNESS_KIND=omp" "sleep" "600"` shows `/usr/bin/env DOCENT_HARNESS_KIND=omp sleep 600`. Plugin panes show `null`, and so does a default shell pane.
- `zellij action go-to-tab-by-id <id>` exists. It takes the `tab_id` from `list-panes` or `list-tabs --json` and makes that tab active. An unknown id (`99`) exits 0 and does nothing.
- `zellij action new-tab --name t2` run from a `--floating --close-on-exit` pane makes the new tab active (`list-tabs --json` shows `"active": true` for it). When the command exits, the floating pane closes and is removed from the old tab. In `list-panes`, `is_focused` is per tab: the old tab's editor pane stays `is_focused: true`, so use `list-tabs --json` `active` to find the active tab.
- Default keybinds use `Alt c f h i j k l n o p`. `Alt w` is free, and the user's `config.kdl` does not bind it.
- Two top-level `keybinds` blocks in one `config.kdl`: Zellij uses only the first and ignores the second, with or without `clear-defaults=true`. `zellij setup --check` still says "Well defined", so it does not detect this. A second `shared_except "locked"` node inside the first `keybinds` block works: both nodes apply, also with `keybinds clear-defaults=true {`.
- Config lookup (probed with `zellij setup --check` and a temp `HOME`; it prints `[CONFIG DIR]` and `[LOOKING FOR CONFIG FILE FROM]`): the file is `$ZELLIJ_CONFIG_FILE`, else `<config dir>/config.kdl`. The config dir is `$ZELLIJ_CONFIG_DIR` (used also when its `config.kdl` does not exist), else the first directory that exists of `~/.config/zellij`, the platform config dir, and `/etc/zellij`. Zellij checks that the directory exists, not the file. On macOS the platform dir is `~/Library/Application Support/org.Zellij-Contributors.Zellij`, and `XDG_CONFIG_HOME` is ignored. On Linux it is `$XDG_CONFIG_HOME/zellij` (default `~/.config/zellij`), from the `directories` crate (`find_default_config_dir` in `zellij-utils/src/setup.rs`).
- `zellij --config <file> setup --check` exits 1 when the file does not parse.
- `BurntSushi/toml` can only encode a whole file, so writing state into `config.toml` would drop user comments.
- `internal/send.FormatRef` takes the kind from the base name of the first word of the pane command. `panes.Lookup` returns that command.

## Goals / Non-Goals

**Goals:**
- `docent new` reuses `launch.Run` for the tab. The picker only produces `(dir, name, command, kind)`.
- All open-tab detection comes from `list-panes`, so no state can go stale.

**Non-Goals:**
- Creating directories or worktrees from the picker (the `workspace` spec forbids it).
- Path completion. fzf lists every directory under `~`, so a fuzzy query finds it. A typed path outside the list opens when nothing matches.
- Editing the Zellij config outside the marked node that `docent install zellij` manages.

## Decisions

### Open-tab detection by `pane_cwd`
A tab holds a directory when its `editor` pane has that directory as `pane_cwd` in `list-panes --all --json`. Zellij 0.45.1 shows this field for terminal panes. The editor pane stays `command="hx"`, with no wrapper.

### Harness kind marker as an `env` wrapper
The harness pane runs through `/usr/bin/env` only when the kind differs from the base name of the first word of the command, so plain tabs keep `claude` as the pane command:

```kdl
pane name="harness" command="/usr/bin/env" {
    args "DOCENT_HARNESS_KIND=omp" "sandbox-exec" "-f" "omp.sb" "omp"
}
```

`terminal_command` is the argv joined with single spaces, with no quotes: `/usr/bin/env DOCENT_HARNESS_KIND=omp sandbox-exec -f omp.sb omp`. The kind is one word (the config check enforces this), so splitting on spaces is safe for the kind. Parsing lives in `internal/panes`, next to `Lookup`:

```go
// Marker returns the value of DOCENT_HARNESS_KIND in an `env DOCENT_HARNESS_KIND=kind cmd...` pane command.
func Marker(cmd string) (string, bool)
// Unwrap returns cmd without a leading `env DOCENT_HARNESS_KIND=kind`.
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
# $XDG_STATE_HOME/docent/recent.toml
dirs = ["/Users/thane/wt/retry/app", "/Users/thane/src/app"]
```

`recent.Add(dir)` loads, moves `dir` to the front, cuts the list to 20 entries, and writes through a temp file and rename. `launchDir` calls it once per launch: outside Zellij just before it attaches (attach blocks until the user detaches), and inside Zellij after the new tab opens. So `docent [dir]` and `docent new` both record. A write error prints a warning and does not fail the launch. Two launches at the same time can lose one entry. This is acceptable for a history list.

### Picker in fzf

The first version was a bubbletea model with a subsequence filter. It had no ranking and no way to reach a directory outside `roots` except an exact typed path. fzf gives ranked fuzzy matching and is fast for tens of thousands of lines (a walk of `~` on the target machine gives 56k directories in 2.3s), so `docent new` runs fzf (`internal/tui/picker.go`) in up to three steps:

```
dir> rtap                          fzf --layout=reverse --scheme=path --print-query --expect=alt-enter
  enter: open with omp  alt-enter: choose harness  esc: cancel
> ~/wt/retry/app
harness> sand                      only after alt-enter; fzf --print-query
> omp-sandbox
> switch to tab app                only when the dir is open; fzf --no-input
  open another tab
```

- Input: docent writes the lines to fzf stdin from a goroutine: recent dirs, repos under `roots`, then `recent.Dirs(home)`, each once, as `~/...`. fzf does not use `--walker` when stdin is a pipe, so docent walks. Paths with control characters or tabs are dropped, as fzf output could not give them back. When fzf exits, `cmd.Wait` closes the pipe, the next write fails, and the goroutine stops; the walk context is cancelled after the first step.
- Output (checked with fzf 0.74.5 under a pty): with `--print-query --expect=alt-enter`, fzf prints the query, the key (empty for `enter`), and the selection, and exits 0. With no match it prints the query and an empty key, and exits 1. `esc` exits 130. Other codes are errors. `accept-or-print-query` is not used: with no match it prints only the query, with no key line.
- Typed path: a query that is a typed path to an existing directory wins over the selection, because with 56k lines a path such as `/tmp` nearly always fuzzy-matches another line. Else docent uses the selection, or with no match prints `no directory for "<query>"`. An empty query with no selection cancels. Each line is made absolute first, so a relative root such as `roots = ["src"]` gives paths that open.
- Harness step: the selection, or the query with no match. A query with a space (after trimming) is used as is, because fzf extended search matches each word on its own (`omp -s` selects `omp-sandbox`).
- `docent new` needs `fzf` on PATH, and prints an error without it. The picker returns the same `Picked` as before, so `cmd/docent/new.go` did not change.
- Tests inject the fzf runner and assert the args, input lines, and result. `runFzf` is tested with the real fzf in `--filter` mode, and is skipped when fzf is missing.

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
            Run "docent" "new" {
                floating true
                close_on_exit true
                name "docent new"
            }
        }
    }
}
```

`docent new` opens the tab from inside the floating pane. Zellij focuses the new tab. The floating pane exits and closes in the old tab.

`docent install zellij` adds the inner `shared_except "locked"` node of this asset (lines 2 to n-1, already indented 4 spaces) to the config, so the README snippet and the installed node come from one file:

```kdl
keybinds clear-defaults=true {
    // docent: begin (managed by `docent install`)
    shared_except "locked" {
        ...
    }
    // docent: end
    normal { ... }
}
```

- Path: the Zellij lookup above. If the found dir is `/etc/zellij`, docent refuses, because a new `~/.config/zellij/config.kdl` would hide the system config. A symlinked config is resolved, so docent changes the target file and keeps the link.
- Parse: a leading BOM is removed first and put back on write. A small line scanner blanks `//` and nested `/* */` comments outside strings, tracks the `{}` depth, and records if each line ends inside a comment or string. Raw strings (`r"`, `#"`) and `/-` comments make it refuse, with a message to add the snippet by hand. The first line with a `keybinds` node name (bare or quoted, at the line start or after `;`, `{` or `}`) must be at depth 0, start with bare `keybinds`, end with `{`, and not end inside a comment or string, else docent refuses. No `keybinds` node name at all: docent appends `keybinds {`, the node, and `}`. Inserted lines use the line ending of the `keybinds` line (CRLF or LF).
- Markers: one begin and one end line, which start outside a comment or string. More than one, out of order, or one without the other: refuse with "fix <path> by hand". Markers only in a comment: refuse. Same content (ignoring `\r`): no write. Other content: refuse without `--force`; with it, replace the lines between the markers. A `bind` with a key `"Alt w"` (modifier in any case, any spaces) outside the markers: refuse; when it is the asset's bind, the message says that it was added by hand. The scanner is not a tokenizer, so it does not find rare forms (`"Alt\tw"`, a quoted `"bind"`, a `\"` argument before the key, a `\` line continuation); `zellij setup --check` does not catch them either. Accepted: the README states the limit.
- Check: all of this runs in `install.check`, with `zellij --config <temp file> setup --check` and a temp file made and removed in the config directory, or in its nearest parent that exists, so a refusal writes nothing for any item. A config with other hard links is refused, because the rename would split them. `apply` writes the old content to a synced temp file and hard-links it to `config.kdl.docent.bak`, so the backup is complete or absent; if the link exists, it keeps it (the first backup holds the config from before docent). Then it writes a temp file in the same directory with the old mode (`0644` for a new file), syncs it, and renames it.

`close_on_exit` would hide errors and warnings, so on a terminal inside Zellij, after a non-zero exit or any printed output, `docent new` waits for `enter`. A read error of the recent list shows as a warning line in the fzf header.

## Risks / Trade-offs

- [`terminal_command` may not include the args] -> Checked (task 1.1): it includes them.
- [The `env` wrapper changes the pane command that users and other tools see] -> Only `send` reads it, and `Unwrap` handles it.
- [`env` is not on `PATH` in an unusual environment] -> `/usr/bin/env` exists on macOS and Linux. Use the absolute path `/usr/bin/env`.
- [`:cd` in Helix changes the editor `pane_cwd`, so that tab is no longer detected for its first directory] -> Accepted. The spec states it.
- [The home walk lists many directories] -> It streams into fzf, so the first lines show at once. Hidden directories, `node_modules`, and `~/Library` are skipped.
- [Large roots make the walk slow] -> No descent into a repo, hidden directories skipped, symlinks below the root not followed (no loops), and the walk is asynchronous and streams hits, so the picker does not block.

## Migration Plan

No migration. Existing configs keep working. Old tabs keep working, and the picker detects them by `pane_cwd`.
