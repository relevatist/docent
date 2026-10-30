# Design

## Context

See proposal.md - Why. Facts from the Zellij 0.45.1 source that shape the design:

| Fact | Source |
|---|---|
| `action write`, `write-chars`, `send-keys`, `paste` with `--pane-id` all send `ClearScroll(cli_client)` first | `zellij-server/src/route.rs` |
| The plugin API `write_to_pane_id` sends only `WriteToPaneId` | `zellij-server/src/plugins/zellij_exports.rs` |
| `action paste` is `ESC[200~` + text + `ESC[201~` written raw, as 3 writes | `Tab::paste_to_pane_id` |
| A pane without bracketed paste mode drops `ESC[200~` or `ESC[201~` only when a write is exactly that marker | `TerminalPane::adjust_input_to_terminal` |
| `send-keys` writes each key as its kitty encoding (`DISAMBIGUATE_ESCAPE_CODES`): Esc is `ESC[27u`, Enter is `\r` | `KeyWithModifier::serialize_kitty` |
| `zellij pipe --plugin <url>` loads the plugin in a visible pane when it does not run | `WasmBridge::get_or_load_plugins` |
| A `zellij pipe` without `--plugin` goes to every running plugin and never loads one | `pipe_to_all_plugins` |
| There is one plugin instance for each attached client; a pipe goes to all of them | `PluginMap::all_plugin_ids`, `WasmBridge::add_client` |
| A CLI pipe exits when every plugin that got it returns, unless a plugin blocks it | `plugins/pipes.rs` |
| `load_plugins` loads background plugins at session start; Zellij reads the first `load_plugins` node | `kdl/mod.rs` |
| Permissions are cached in `<zellij cache dir>/permissions.kdl`, keyed by the plugin location string | `input/permission.rs` |

docent runs `zellij` with stdin not a terminal, so the pipe CLI waits for the plugin, then reads EOF and exits.

## Goals / Non-Goals

**Goals:**
- One code path for all pane writes: bytes.
- Same bytes in the target pane as today.

**Non-Goals:**
- Fixing Zellij upstream (report it separately).
- `focus-pane-id`, `new-pane`, `list-panes`: they do not clear scroll, or the user asks for them.
- CI. There is none yet.
- A reproducible `.wasm`. No binary is shared, so no one compares builds.
- `go install github.com/relevatist/docent/cmd/docent@latest`. It cannot build the plugin.

## Decisions

### 1. Keys become one byte write; a paste is its own operation

```go
// internal/zellij
func Write(r Runner, pane string, b []byte) error   // pipe docent-write
func Paste(r Runner, pane, text string) error       // pipe docent-paste
```

| Caller | Operation | Plugin writes | Fallback |
|---|---|---|---|
| `helix.run` | `Write` | `ESC[27u ESC[27u` then each `cmd + "\r"`, all in one write | `zellij action write` |
| `send.paste` | `Paste` | `ESC[200~`, text, `ESC[201~` as 3 writes (ESC already removed from text) | `zellij action paste` |

`WriteChars`, `SendKeys` go away. One write for a Helix call replaces 3 to 6 zellij processes with one.

A paste is not one byte write. Zellij drops the paste markers for a pane without bracketed paste mode only when a write is exactly the marker. In one write, a harness without bracketed paste mode gets the markers as literal text. `action paste` makes 3 writes, so the plugin does the same. The `action paste` fallback also takes the text as one argument, so a large paste does not hit the argument limit of the per-byte `action write`.

Alternative: keep one plugin operation per action kind (`paste`, `keys`, `chars`). Rejected for keys: more plugin code, and the plugin would have to copy Zellij's kitty encoding.

### 2. Broadcast pipe, plugin answers, fallback on no answer

```sh
zellij pipe --name docent-write --args pane=terminal_3,deadline=<unix ms> -- "<bytes>"
```

- No `--plugin`, so Zellij never loads the plugin in a visible pane in a session that does not have it.
- The plugin that writes answers `ok` with `cli_pipe_output` (needs `ReadCliPipes`).
- docent reads stdout. `ok` means done. Anything else, or an error, means the plugin is not there: docent runs the fallback, which writes the same bytes (with the old scroll jump).
- docent sets a timeout (2 s) on the pipe call, then falls back. This covers a session with no running plugin, where the pipe could wait.
- `deadline` is the time of the call + 1.5 s. When the plugin gets the write after the deadline, it does not write and answers `late`, because docent may have fallen back already. Without a valid deadline, the plugin ignores the message.
- `ok` anywhere in the output means done, because two instances can both answer (decision 3).
- The plugin ignores a message with no payload. The CLI sends one after the plugin releases the pipe, because its stdin is not a terminal.

Alternative: `--plugin file:...`. Rejected: in an old session it opens a plugin pane and maybe a permission prompt.

### 3. Only one plugin instance writes

Each instance gets the message. An instance writes and answers only when its client ID is the lowest of the connected clients (`get_plugin_ids().client_id`, `list_clients()`, needs `ReadApplicationState`). The others return without action.

When the lowest client detaches between `list_clients()` and the answer, two instances can both write. This race is accepted.

An instance answers only after `PermissionRequestResult(Granted)`. Before that, or when denied, it does not hold the pipe, so docent falls back at once. When the permissions are not in the cache, Zellij holds the message until the user answers a prompt that a background plugin cannot show, so docent falls back after the timeout.

Alternative: docent sends a request ID and instances dedupe. Rejected: instances share no state.

### 4. Plugin crate

```
plugin/
  Cargo.toml          zellij-tile = "=0.45.1", [profile.release] opt-level="z", lto, strip
  Cargo.lock
  rust-toolchain.toml pinned version, target wasm32-wasip1
  src/main.rs
```

```rust
fn load(&mut self, _: BTreeMap<String, String>) {
    request_permission(&[WriteToStdin, ReadCliPipes, ReadApplicationState]);
    subscribe(&[ListClients, PermissionRequestResult]);
}
fn pipe(&mut self, msg: PipeMessage) -> bool {
    // source is Cli, payload is set, name is docent-write or docent-paste,
    // pane and deadline args are valid, permissions granted:
    // block the pipe, keep the write, list_clients()
    false
}
fn update(&mut self, event: Event) -> bool {
    // ListClients, this instance is the lowest client:
    // before the deadline: write_to_pane_id(each write, pane), cli_pipe_output(id, "ok")
    // after: cli_pipe_output(id, "late")
    // unblock the pipe
    false
}
```

Payload is a Rust `String`, so bytes must be UTF-8. Today's `write-chars` and `paste` have the same limit.

### 5. One docent plugin, one pipe name per operation

The plugin is the docent plugin, not a plugin for this bug. Each operation is a pipe name `docent-<op>`; this change adds `docent-write` and `docent-paste`. The plugin ignores unknown names and does not answer, so docent falls back, and a new docent binary works with an old installed plugin. A new operation adds its pipe name and, if needed, a permission that `docent install zellij` grants.

Alternative: build a general command protocol now. Rejected: no feature needs it yet, and each permission is one more thing that the user must trust.

### 6. Build the plugin with make, embed the result

No `.wasm` is in Git: each plugin change would add about 0.8 MB to the history.

```make
plugin:   # cd plugin && cargo build --locked --release --target wasm32-wasip1
          # then copy to assets/zellij/docent.wasm (gitignored)
build install test test-unit vet lint check: plugin
```

- `assets.ZellijPlugin` embeds `assets/zellij/docent.wasm`. `go:embed` fails without the file, so every Go target runs `make plugin` first.
- Building docent needs Rust (`rustup`, which reads `rust-toolchain.toml`). `docent install zellij` does not: it copies the embedded file.

Alternatives:
- Commit the `.wasm`. Rejected: history grows by about 0.8 MB for each plugin change.
- Git LFS. Rejected: `go install` gets the LFS pointer, and `go:embed` embeds the pointer without an error.
- Build at `docent install zellij` time. Rejected: every user then needs Rust and network at install.
- Download from a release. Rejected: needs a release process, which docent does not have.

### 7. Install

`docent install zellij` adds two targets to the existing zellij item. All are checked before any write (existing rule).

| File | Change |
|---|---|
| `$XDG_DATA_HOME/docent/zellij/docent.wasm` | always replace it with the embedded file (temp file + rename), like the Claude Code plugin; no `--force` |
| Zellij `config.kdl` | second marked node `"file:<abs path>"` in the first top-level `load_plugins`, or append `load_plugins { "zellij:link" <node> }` |
| `<zellij cache dir>/permissions.kdl` | set the entry `"<abs path>" { WriteToStdin; ReadCliPipes; ReadApplicationState }` (key is the bare path, as `RunPluginLocation::to_string` prints it, not `file:<path>`), keep other entries |

The `load_plugins` edit reuses `editZellijCode`'s parser (`kdlCode`), generalised to a node name and a marker pair. Zellij cache dir: macOS `~/Library/Caches/org.Zellij-Contributors.Zellij`, else `$XDG_CACHE_HOME/zellij` (default `~/.cache/zellij`).

## Risks / Trade-offs

- [The broadcast wakes every plugin in the session for each write] → The cost is small. Other plugins ignore an unknown pipe name.
- [Zellij changes the pipe, permission cache, or plugin API] → Pin `zellij-tile`. The fallback keeps docent working. The integration test covers the plugin path.
- [The permission cache format changes] → docent writes the format that Zellij writes. When the file does not parse, docent names it and does not change it.
- [Building docent now needs Rust, and plain `go build` fails until `make plugin` runs once] → CONTRIBUTING.md and the install docs say so.
- [A user's config has no `load_plugins`, so Zellij used its built-in list] → The appended block keeps `"zellij:link"`, which is the built-in default.
- [The plugin applies only to sessions started after install] → docs say so. The fallback covers old sessions.

## Migration Plan

1. Users run `docent install zellij` again, then start a new Zellij session.
2. Rollback: remove the `// docent: plugin` lines from `config.kdl`. docent then falls back to `zellij action write`.
