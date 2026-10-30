# Tasks

## 1. Built-in layout as a template

- [x] 1.1 Write failing tests in `internal/launch`: the built-in template renders the harness at `size="50%"` with `focus=true`, then `shell`, then `editor` with `command="hx"` and no `focus`; `{{.Name}}` gives the quoted name; `{{.Harness}}` gives `command=...` with and without an `args` block
- [x] 1.2 Add `assets/zellij/layout.kdl` (see design.md) and embed it as `assets.ZellijLayout`; change `launch.Layout` to render it with `text/template`; update the existing `TestLayout*` tests; verify `make test-unit` passes
- [x] 1.3 Update the Layout tables in `docs/workstreams.md` and `README.md` (positions, sizes, harness focus), link `assets/zellij/layout.kdl`, and note that `demo.png` shows the old layout; verify the tables match `layout.kdl`

## 2. Layout checks

- [x] 2.1 Write failing tests: a template parse error, `{{.Nmae}}`, no `{{.Harness}}`, and a missing `editor`, `harness`, or `shell` pane each give an error that names the layout file and the problem; the built-in template passes
- [x] 2.2 Implement the `fields` struct with the `Harness` method and the checks in `launch.Layout`; verify the 2.1 tests pass and a `launch.Run` test with a bad layout makes no `zellij` call

## 3. Markers

- [x] 3.1 Write failing tests for `panes.Marker`, `panes.LayoutMarker`, and `panes.Unwrap`: kind only; kind and layout; layout path with `%20` unescapes to a space; a word other than `DOCENT_HARNESS_KIND` or `DOCENT_LAYOUT` after `env` stops the read; no wrapper
- [x] 3.2 Implement them (see design.md - Markers); make `launch.Layout` add the wrapper with `DOCENT_HARNESS_KIND` and an escaped `DOCENT_LAYOUT` when a layout path is given; verify the 3.1 tests pass, a `TestLayout` case asserts the args for a custom layout, and the built-in layout with `claude` has no wrapper
- [x] 3.3 Add a `send` test with the pane command `/usr/bin/env DOCENT_HARNESS_KIND=claude DOCENT_LAYOUT=/x.kdl claude`; verify it gives the `claude` ref format

## 4. Config and flag

- [x] 4.1 Write failing tests for `config.Load`: `layout = "~/x.kdl"` expands `~`; `layout = "wide.kdl"` resolves to the config directory; no key gives `""`
- [x] 4.2 Add `Layout` to `config.Config` and implement the resolution; verify the 4.1 tests pass
- [x] 4.3 Add `--layout` to `runLaunch` (expand `~`, then `filepath.Abs`) and pass the path through `launchDir` to `launch.Run`; a missing file gives an error that names it; verify launch tests for flag over config, config only, and missing file
- [x] 4.4 Document `layout` in `docs/config.md` and `--layout` in `docs/commands.md`; add a custom layout section to `docs/workstreams.md` with the template fields table, the checks, the rule that pane properties come before `{{.Harness}}`, and the old layout as an example; verify a test renders the documented example layout without error

## 5. `docent new`

- [x] 5.1 Write failing tests for `currentLayout`: harness pane with a marker gives the marker path; harness pane without a marker gives built-in, also when the config sets `layout`; no harness pane gives the config `layout`, else built-in
- [x] 5.2 Implement `currentLayout` and pass its result to `launchDir` from `newWorkstream`; verify the 5.1 tests pass and a `newCmd` test with a deleted marker file prints an error that names the file
- [x] 5.3 Document in `docs/workstreams.md` that `docent new` uses the current tab's layout; verify the text matches the 5.1 table

## 6. Integration

- [x] 6.1 Update `cmd/docent/integration_test.go` for the new pane order and harness focus, and add a case with a custom layout that checks `docent new` keeps the layout marker; verify `make check` passes (the integration test runs when `zellij` and `hx` are on PATH)
