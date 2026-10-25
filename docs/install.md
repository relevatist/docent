# Install

## Requirements

- macOS or Linux
- Zellij 0.45+
- Helix 25.07+
- Go 1.27+ (to build)
- fzf (for `docent new`)

## Install docent

```sh
go install github.com/relevatist/docent/cmd/docent@latest
```

Or, from a clone of this repository: `make install`.

Both install `docent` into `$(go env GOPATH)/bin` (or `$GOBIN`). Make sure that directory is on your `PATH`.

Then:

1. Run `docent install` to set up your harnesses and the Zellij key. See [Harnesses](harnesses.md).
2. Add the Helix keys. See [Keys](keys.md).

## Uninstall

```sh
rm "$(go env GOPATH)/bin/docent"                # or $GOBIN/docent
claude plugin uninstall docent@docent           # claude
claude plugin marketplace remove docent
rm -rf ~/.local/share/docent                    # or $XDG_DATA_HOME/docent
rm -rf ~/.agents/skills/docent                  # codex, omp
rm ~/.omp/agent/extensions/docent-reload.ts     # omp
rm -rf ~/.config/docent ~/.local/state/docent   # config and recent directories
```

- Zellij: in `config.kdl`, remove the lines from `// docent: begin` to `// docent: end`. Or, if you did not change the config after `docent install zellij`, replace it with `config.kdl.docent.bak`.
- Helix: remove the `[keys.normal.space.z]` and `[keys.select.space.z]` tables from `~/.config/helix/config.toml`.
