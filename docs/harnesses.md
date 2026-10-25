# Harnesses

A harness is the coding agent in the `harness` pane, for example Claude Code (`claude`), oh-my-pi (`omp`), or `codex`.

## `docent install`

```sh
docent install                 # choose from a checklist
docent install claude omp      # or name them
docent install zellij          # add the Alt w key to the Zellij config
docent install codex --force   # overwrite a skill or extension file that you changed
```

Run `docent install` again after you upgrade docent.

| Target | What docent installs |
|--------|----------------------|
| `claude` | The `docent@docent` Claude Code plugin: the agent skill and a reload hook. Restart Claude Code after the install. |
| `codex` | The agent skill: `~/.agents/skills/docent/SKILL.md` |
| `omp` | The agent skill (same path as codex) and a reload extension: `~/.omp/agent/extensions/docent-reload.ts` |
| `zellij` | The `Alt w` key. See [Keys](keys.md#zellij-key). |

- The checklist starts with an item checked when its command is on `PATH`.
- docent checks everything before it changes anything. If a command is missing, a file has changes and you did not give `--force`, or docent cannot add the Zellij key, it installs nothing and exits non-zero.
- For `claude`, docent writes the plugin to `~/.local/share/docent/claude-plugin/` (or `$XDG_DATA_HOME/docent/claude-plugin/`) and installs it with the `claude plugin` CLI.

## What the skill and hooks do

- The agent skill tells the harness how to use `docent open`, `docent reload`, and `docent tour`, and how to write tour notes.
- The reload hook (claude) and extension (omp) run `docent reload --quiet` each time the harness edits a file, so the editor shows the new content. The omp extension reloads after the `edit`, `write`, and `resolve` tools.
- codex has no reload hook. The skill tells it to run `docent reload`.

## Use with the agent

Ask the harness in plain language. For example:

| You ask | The agent runs |
|---------|----------------|
| "Show me where the config is loaded." | `docent open internal/config/config.go:45` |
| "Walk me through this change." | writes a tour, then `docent tour /tmp/tour.json` |
| "Give me a tour of the retry logic, tests last." | the same, with the steps in that order |

The agent uses docent only inside a workstream (when `$ZELLIJ` is set).

## Install without `docent install`

Claude Code plugin:

```sh
claude plugin marketplace add relevatist/docent
claude plugin install docent@docent
```

oh-my-pi extension, for one session: `omp -e assets/omp/docent-reload.ts`.
