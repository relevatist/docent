# Tours

A tour walks you through a change, one step at a time. Usually the agent writes the tour and starts it.

```sh
docent tour /tmp/tour.json       # start a tour
docent tour - < /tmp/tour.json   # read the tour from standard input
```

## Format

A tour is a JSON file:

```json
{"title": "Retry",
 "steps": [{"note": "## Overview\n\n`Do()` now retries."},
           {"file": "a.go", "lines": [1, 40], "note": "New `Retry` type"}]}
```

| Field | Required | Meaning |
|-------|----------|---------|
| `title` | no | shown on steps |
| `steps` | yes | one or more steps |
| `steps[].file` | no | file to show in the editor |
| `steps[].lines` | no | `[start, end]`. Needs `file`. Without it, the step covers the whole file. |
| `steps[].note` | no | Markdown |

- A step needs a `file`, a `note`, or both.
- A relative `file` is relative to the directory where you run `docent tour`.
- docent ignores unknown fields, such as `base` from older tours.

## What you see

The tour opens in a `tour` pane, stacked with the shell pane.

- The editor shows the step's file at its first line. A step without a file does not move the editor.
- The tour pane shows the step's position, its file and lines (or the tour title), and its note.
- `diff` code blocks show added and removed lines in color.
- `mermaid` code blocks (flowchart, sequence, and ER diagrams) show as text diagrams. If docent cannot draw a diagram, it shows the source and the reason.

## Keys

| Key | Action |
|-----|--------|
| `n` | next step |
| `p` | previous step |
| `o` | open the step in the editor again |
| `s` | send the step as a reference to the harness |
| `j` / `k` (`down` / `up`) | scroll one line |
| `pgdown` / `space` / `ctrl+d` | scroll one page down |
| `pgup` / `b` / `ctrl+u` | scroll one page up |
| `g` / `G` | go to the top / bottom of the note |
| `q` | close the tour |
