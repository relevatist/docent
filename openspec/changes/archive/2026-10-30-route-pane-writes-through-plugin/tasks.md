# Tasks

## 1. Byte writes in docent (fallback path only)

- [x] 1.1 Add `zellij.Write(r, pane, b)` that runs `zellij action write --pane-id <pane> <byte>...`; verify with a `zellij.Fake` unit test of the args.
- [x] 1.2 Change `helix.run` to one `Write` of `ESC[27u ESC[27u` + each `cmd + "\r"`; update `internal/helix` tests to assert the exact bytes, including the insert-mode case.
- [x] 1.3 Change `send.paste` to one `Write` of `ESC[200~` + text + `ESC[201~`; update `internal/send` tests, including the `a ESC [201~ CR b` scenario.
- [x] 1.4 Remove `zellij.Paste`, `WriteChars`, `SendKeys`; verify `make check` passes and `TestIntegrationOpen` still opens the file at the line.

## 2. Zellij plugin

- [x] 2.1 Create `plugin/` (`Cargo.toml` with `zellij-tile = "=0.45.1"` and the size profile, `Cargo.lock`, `rust-toolchain.toml` with `wasm32-wasip1`); verify `cargo build --locked --release --target wasm32-wasip1` succeeds.
- [x] 2.2 Implement `load` (request `WriteToStdin`, `ReadCliPipes`, `ReadApplicationState`) and `pipe` (`docent-write` from the CLI, lowest-client check, `write_to_pane_id`, `cli_pipe_output("ok")`); verify with a manual `zellij pipe` in a session that has the plugin loaded: the target pane gets the bytes and the CLI prints `ok`.
- [x] 2.3 `make plugin` runs `cargo build --locked --release --target wasm32-wasip1` in `plugin/` and copies the result to a gitignored `assets/zellij/docent.wasm`; every Go target in the Makefile depends on `plugin`; remove `rustc.sh`, the path remapping, and `plugin-check`; verify `make build` works from a clean tree and `git status` shows no `.wasm` or build output.
- [x] 2.4 Untrack `assets/zellij/docent.wasm` and keep embedding it as `assets.ZellijPlugin`; verify the `assets` test checks the WASM magic bytes and `git ls-files` lists no `.wasm`.
- [x] 2.5 Update `CONTRIBUTING.md`: Rust via rustup, `make plugin` before plain `go build`/`go test`; remove the reproducibility and `plugin-check` text; verify each documented command runs.

## 3. Write through the plugin

- [x] 3.1 Change `zellij.Write` to first run `zellij pipe --name docent-write --args pane=<pane> -- <bytes>` with a 2 s timeout, and fall back to `action write` unless stdout is `ok`; verify `Fake` tests for `ok`, empty output, error, and timeout.
- [x] 3.2 Extend the integration test to load the plugin (temp config with `load_plugins` and a temp permission cache through `ZELLIJ_CONFIG_FILE` and the cache dir), scroll up the focused pane in tab 1, run `docent reload` from tab 2, and check that the tab 1 pane is still scrolled (`dump-screen` or `list-panes` scroll data); verify it fails with the plugin removed.
- [x] 3.3 Add a two-client case to the integration test, or a manual check recorded in the change if a second `script` client is not practical: `docent open a.go:3` writes once.

## 4. Install

- [x] 4.1 Generalise the marked-node edit in `install_zellij.go` to take the node name (`keybinds`, `load_plugins`) and the marker pair; verify the existing `keybinds` tests still pass unchanged.
- [x] 4.2 Add the `load_plugins` target with the scenarios in `specs/harness-integration/spec.md` (existing block, no block with `"zellij:link"`, rerun, refused forms, unpaired markers); verify with table tests in `install_zellij_test.go`, and that `zellij setup --check` accepts the result.
- [x] 4.3 Always replace `$XDG_DATA_HOME/docent/zellij/docent.wasm` with the embedded file (temp file + rename, no `--force`); verify tests for a new file, a changed file replaced without `--force`, and the printed path.
- [x] 4.4 Add the `permissions.kdl` target: Zellij cache dir per OS, create or update only docent's entry, refuse an unparseable file; verify table tests for no file, other entries kept, existing docent entry, and a bad file. Check the written file against one written by Zellij after a manual grant.
- [x] 4.5 Run all new targets through the existing check-before-write flow; verify a test where the permission cache is refused writes nothing to `config.kdl`, and that install prints all three paths.
- [x] 4.6 Update `docs/install.md` (what `docent install zellij` writes, start a new session, uninstall steps), `docs/keys.md` (config changes), `docs/troubleshooting.md` (scroll jumps: reinstall and restart the session), and `README.md` if its install section names the Zellij changes; verify each documented path matches the code.
- [x] 4.7 Update `docs/install.md` and `README.md`: building docent needs Rust, and `go install ...@latest` is not supported; remove text about a shipped or committed `.wasm`; verify each documented command matches the Makefile.

## 5. Integration

- [x] 5.1 Run `make check` with `zellij`, `hx`, and `cargo` on PATH; verify all pass.
- [x] 5.2 Manual check in a real session after `docent install zellij`: scroll up in tab 1, let an agent in tab 2 edit files, and confirm tab 1 does not jump.

## Workflow follow-up

- Report the `ClearScroll(client_id)` bug to Zellij upstream, with the `route.rs` lines.
- Archive the change after review.
