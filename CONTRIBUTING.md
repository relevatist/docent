# Contributing

## Issues

Report bugs and request features in [GitHub issues](https://github.com/relevatist/docent/issues). For a bug, include:

- The docent version: `go version -m "$(command -v docent)" | head -3`
- The OS, `zellij --version`, `hx --version`, and the harness and its version
- The steps to reproduce the bug, what you expected, and what happened

## Pull requests

We do not accept pull requests from the public. Only trusted contributors open pull requests.

If you want a change, open an issue.

## Requirements

- Go 1.27+
- [golangci-lint](https://golangci-lint.run/) for `make lint`
- Zellij 0.45+ and Helix 25.07+ for the integration test

## Build and test

```sh
make build       # bin/docent
make check       # fmt-check, vet, lint, test
make test-unit   # skips the integration test
make vuln        # govulncheck
```

The integration test runs only when `zellij` and `hx` are on `PATH`.

## Specs

docent uses [OpenSpec](https://github.com/Fission-AI/OpenSpec). The specs in `openspec/specs/` describe how docent behaves. For a change to behavior, write or update a change in `openspec/changes/` first. Archive it when the code is merged.

## Code

- Use modern, idiomatic Go (1.27+). Run `make fmt`.
- Follow the conventions of the existing code.
- When user-visible behavior changes, update the page in `docs/`, `README.md` if needed, and the agent skill in `assets/claude-plugin/skills/docent/SKILL.md` in the same change.

## Commits

Use `type(scope): description`. The scope is the package path or file, for example `cmd/docent`, `internal/tui`, `README.md`, or `openspec`:

```
fix(cmd/docent): keep the first Zellij config backup
docs(README.md): describe the tour pane position
```

Sign off each commit under the [Developer Certificate of Origin](https://developercertificate.org/):

```sh
git commit -s
```

This adds a `Signed-off-by: Name <email>` line. With it, you certify that you have the right to submit the change under the project license.

## License

Contributions are licensed under the [Apache License 2.0](LICENSE), the same as docent.
