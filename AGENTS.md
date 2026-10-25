# AGENTS.md

docent joins Helix, a coding harness, and a shell in one Zellij tab. See README.md.

## Go

- Use modern (Go v1.27+), idiomatic Go.

## Documentation

- `README.md` is the overview and quick start. `docs/` is the full user documentation.
- When a change alters user-visible behavior (commands, flags, config, keys, output, install), update the `docs/` page, and the README if needed, in the same commit.
- When the commands that the agent uses change, also update `assets/claude-plugin/skills/docent/SKILL.md`.
- Write like the existing docs: short sentences, simple technical English, tables and examples over prose.

## Git

- Make commits on the CURRENT branch, whatever branch that is (including `main`).** Do not create, switch, or check out branches unless the user explicitly asks.
- Commit messages: `type(scope): description`, where the scope is the package path (for example `cmd/docent`, `internal/tui`).

## Commands

```sh
make check       # fmt-check, vet, lint, test
make test-unit   # skips the integration test
make install     # go install ./cmd/docent
```

The integration test runs only when `zellij` and `hx` are on PATH.
