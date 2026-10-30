# Proposal

## Why

The workstream layout is fixed in the docent binary. Users cannot change pane positions, sizes, or the Zellij bars. Also, the default layout puts the editor first, but the user spends most of the time typing in the harness, so the harness should be the main pane.

## What Changes

- **BREAKING** New default layout: `harness` on the left at 50% of the width, with focus. `shell` top right, `editor` bottom right, each 50% of the right column height.
- The built-in layout moves to `assets/zellij/layout.kdl`. It is a Go template with the fields `{{.Name}}` (the quoted tab name) and `{{.Harness}}` (the harness pane command and args).
- Add the config key `layout` and the flag `docent --layout <file>`. Each takes the path of a full Zellij layout file that uses the same template fields. The flag overrides the config. Without either, docent uses the built-in layout.
- At launch, docent rejects a custom layout that does not render, does not use `{{.Harness}}`, or has no pane named `editor`, `harness`, or `shell`.
- A tab opened with a custom layout records the absolute layout path in its harness pane command (`DOCENT_LAYOUT=...`). `docent new` opens the new tab with the layout of the current tab.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `workspace`: the default layout and focus change; add the custom layout (config, flag, template fields, checks) and the layout marker in the tab.
- `workstream-picker`: `docent new` uses the layout of the current tab.

## Impact

- Changed code: `internal/launch` (template rendering and checks), `internal/config` (`layout` key), `internal/panes` (unwrap more than one marker), `cmd/docent/launch.go` (`--layout`), `cmd/docent/new.go` (layout from the current tab), `assets/assets.go` (embed `layout.kdl`).
- The pane command of the harness pane changes to `/usr/bin/env DOCENT_LAYOUT=... <cmd>` in tabs that use a custom layout.
- Docs: `docs/workstreams.md`, `docs/config.md`, `docs/commands.md`, README. `demo.png` shows the old layout.
- No new Go dependencies. `text/template` is in the stdlib.
- No change to the commands that the agent uses, so `SKILL.md` does not change.
