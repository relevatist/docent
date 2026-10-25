# Proposal

## Why

A tour step now shows a raw diff with a short plain-text note. The editor already shows the code, so the diff repeats it and does not explain the change. The harness can explain a change much better with show-me views (pseudocode, call trees, `diff` sketches, Mermaid diagrams), but the pane can show only plain text. Long steps also cannot be paged.

## What Changes

- The tour pane shows the step's `note` as rendered Markdown. Fenced `mermaid` blocks render as terminal diagrams.
- **BREAKING**: The pane no longer shows a diff body for a step. The harness puts `diff` snippets in the note when they help.
- **BREAKING**: Remove the `base` tour field, the `Not covered by this tour` step, and `zide diff`. zide no longer reads git.
- A step's `file` is optional. A step without a file shows only its note, and the editor does not move. A step must have a `file`, a `note`, or both.
- New scroll keys: page down (`pgdown`, `space`, `ctrl+d`), page up (`pgup`, `b`, `ctrl+u`), top (`g`), and bottom (`G`). All scroll keys move the full step view.
- The zide skill tells the harness how to write each step note in Markdown, with its own copy of the show-me views. The skill does not suggest HTML output.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `change-tour`: Markdown step notes replace the diff body. `file` is optional, and the page keys are new. The `base` field, the diff against base, the coverage step, and `zide diff` are removed.
- `harness-integration`: the generic skill must tell the harness how to write step notes in Markdown with show-me views, and must not suggest HTML.

## Impact

- Code: `internal/tour` (remove `diff.go`, the hunk mapping, and the coverage step), `internal/tui` (Markdown view, page keys), `cmd/zide` (remove `diff.go`, the `diff` command, and git checks in `tour`).
- New dependencies: `charm.land/glamour/v2` (Markdown rendering) and `github.com/AlexanderGrooff/mermaid-ascii` (only `pkg/render`, pinned to a pseudo-version because the project has no tags).
- Docs: `README.md`, `assets/claude-plugin/skills/zide/SKILL.md`, and the `zide tour` usage text.
- Tours that still have `base` keep working. zide ignores the unknown field.
