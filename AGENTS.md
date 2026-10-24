# AGENTS.md

zide joins Helix, a coding harness, and a shell in one Zellij tab. See README.md.

## Git

- Make commits on the CURRENT branch, whatever branch that is (including `main`).** Do not create, switch, or check out branches unless the user explicitly asks.
- Commit messages: `type(scope): description`, where the scope is the package path (for example `cmd/zide`, `internal/tui`).

## Layout

```
cmd/zide          CLI entry point and subcommands
internal/config   user config
internal/helix    drive Helix by injecting keys
internal/launch   workstream layout and zellij argv
internal/panes    find named panes in the current tab
internal/send     paste text into the harness pane
internal/tour     tour parsing, diff hunk mapping, step model
internal/tui      tour pane and install checklist (bubbletea)
internal/zellij   run zellij commands
assets            embedded files: Claude Code plugin, omp extension, Helix keys
openspec          specs and change proposals
```

## Commands

```sh
make check       # fmt-check, vet, lint, test
make test-unit   # skips the integration test
make install     # go install ./cmd/zide
```

The integration test runs only when `zellij` and `hx` are on PATH.
