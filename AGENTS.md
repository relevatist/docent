# AGENTS.md

zide joins Helix, a coding harness, and a shell in one Zellij tab. See README.md.

## Go

- Use modern (Go v1.27+), idiomatic Go.

## Git

- Make commits on the CURRENT branch, whatever branch that is (including `main`).** Do not create, switch, or check out branches unless the user explicitly asks.
- Commit messages: `type(scope): description`, where the scope is the package path (for example `cmd/zide`, `internal/tui`).

## Commands

```sh
make check       # fmt-check, vet, lint, test
make test-unit   # skips the integration test
make install     # go install ./cmd/zide
```

The integration test runs only when `zellij` and `hx` are on PATH.
