---
name: zide
description: Show files in the user's editor, reload it after edits, and walk the user through a change with a guided tour. Use when running inside a zide workstream (a Zellij tab with editor, harness, and shell panes).
---

# zide

Use these commands only when the `$ZELLIJ` environment variable is set and `zide` is on `PATH`. Otherwise, do not run zide commands.

The editor, this harness, and a shell share one Zellij tab. The commands act on the editor in this tab. They do not move the user's focus.

## Show a file

```sh
zide open src/main.go:42   # file at line 42
zide open README.md        # file only
```

Use this when you talk about specific code, so the user sees it.

## Reload the editor

```sh
zide reload
```

WARNING: this discards unsaved changes in all editor buffers. Run it after you edit files, only when no hook does it for you already. Do not run it when the user may have unsaved work that you did not cause.

## Give a tour of a change

Write a tour file, then start it:

```sh
zide tour /tmp/tour.json
zide tour - < /tmp/tour.json   # or read from standard input
```

Tour format: a JSON object with optional `title`, optional `base` (a git revision; default `HEAD`), and a non-empty `steps` array. Each step has a `file`, optional `lines` as `[start, end]` (omit to cover the whole file), and optional `note`.

```json
{
  "title": "Add retry to the HTTP client",
  "base": "main",
  "steps": [
    {"file": "client/retry.go", "lines": [1, 40], "note": "New Retry type and its backoff policy."},
    {"file": "client/client.go", "lines": [88, 104], "note": "Do() now wraps each request in Retry."},
    {"file": "client/client.go", "lines": [12, 20], "note": "Unchanged: the timeout that Retry must stay inside."},
    {"file": "client/retry_test.go", "note": "Tests for backoff and give-up."}
  ]
}
```

- Order steps so the change is easy to understand (for example: types, then callers, then tests), not in file order.
- Steps can point at unchanged code when it helps to explain the change.
- You do not need to cover every hunk. zide adds a final step that lists hunks no step covers.
- Keep notes short: what changed and why.

The user steps through the tour; the editor follows each step.

## Walk through the raw diff

```sh
zide diff            # one step per hunk of `git diff`
zide diff main       # takes `git diff` arguments
```
