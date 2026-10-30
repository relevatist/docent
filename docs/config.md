# Config

The config file is `~/.config/docent/config.toml` (or `$XDG_CONFIG_HOME/docent/config.toml`). All keys are optional.

```toml
harness = "claude"         # harness command or alias; --harness overrides it for one tab
focus_on_send = true       # move focus to the harness after `docent send`
roots = ["~/src", "~/wt"]  # `docent new` lists the git repos and worktrees here before other directories
layout = "wide.kdl"        # Zellij layout file; --layout overrides it for one tab

[harnesses.omp-sandbox]    # harness alias: use `--harness omp-sandbox`
command = "sandbox-exec -f ~/omp.sb omp"
kind = "omp"               # optional; default: base name of the first word of command
```

| Key | Default | Meaning |
|-----|---------|---------|
| `harness` | `"claude"` | harness command or alias for new tabs |
| `focus_on_send` | `true` | move focus to the harness after `docent send` |
| `roots` | none | directories where `docent new` looks for git repos and worktrees |
| `layout` | built-in layout | Zellij layout file for new tabs, see [Custom layout](workstreams.md#custom-layout) |
| `harnesses.<alias>.command` | - | the command of the alias |
| `harnesses.<alias>.kind` | base name of the first word of `command` | the harness that the command runs |

## Harness kind

`docent send ref` uses `kind` to format references, so a wrapped harness still gets `@` references. `kind` must be one word.

## Commands and paths

- docent expands a leading `~` in `roots`, in `layout`, and in each word of a harness command (an alias `command`, the `harness` key, `--harness`, or a command typed in the `docent new` harness list).
- A relative `layout` path is relative to the config directory.
- Zellij runs the harness command without a shell, so other shell syntax does not work.
- A root can be a symlink. docent does not follow symlinks below a root.
