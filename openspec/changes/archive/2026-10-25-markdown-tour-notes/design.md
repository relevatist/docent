# Design

## Context

- `internal/tour` parses tours (`tour.go`), maps steps to git hunks (`diff.go`, `Build`), and holds the step model (`model.go`).
- `internal/tui` renders a step as header lines plus `Item.Body` (diff lines). `j/k` scroll only the body.
- `cmd/zide/tour.go` checks git (`tour.Changes`) before it opens the pane. `cmd/zide/diff.go` and `tour-view --diff` build tours from hunks.
- The stack is `charm.land` v2 (bubbletea v2, lipgloss v2).

## Goals / Non-Goals

**Goals:**
- One view per step: header, then the rendered note, then key help.
- Remove all git and hunk code from the tour path.

**Non-Goals:**
- Mermaid diagram kinds other than the ones mermaid-ascii supports.
- Links, images, or any output outside the pane.
- Caching of rendered notes between runs.

## Decisions

### Data model

```diff
 type Tour struct {
 	Title string
-	Base  string
 	Steps []Step
 }

 type Item struct {
 	Step
 	Path    string
-	Title   string
 	Invalid string
 	Editor  bool
-	Body    []Line
 }
```

```text
Parse
  json.Unmarshal (json/v2 ignores unknown fields such as "base")
  each step: needs file or note; lines needs file
Build(t, cwd) []Item          // hunks parameter removed
  file == ""  -> Item{Editor: false}
  file        -> read file; check range; Editor = true (as now)
```

`item()` keeps the file read only to check that the file and the range exist. `Line`, the line kinds, `Hunk`, `NotCovered`, and `DiffTour` go. `Heading()` returns `Ref()`, or "" for a step with no file. The view shows the tour title in that place.

### Rendering

```text
renderNote(note, width) []string
  note = safe(note)                       // strip control chars first
  for each ```mermaid fence
    out, err = render.RenderDiagram(src, diagram.DefaultConfig())
    ok  -> replace fence with ```text\n<out>\n```
    err -> keep fence; add "*mermaid: <err>*" after it
  glamour.Render(note, WordWrap(width), dark style)
  split on "\n"
```

- **glamour v2 over a small fence renderer.** It handles headings, lists, tables, and code highlighting (it colors `diff` fences through chroma). It is the library behind `glow` and uses the same lipgloss v2.
- **Mermaid before glamour.** The diagram becomes a plain code block, so glamour keeps its spacing and does not wrap it. Alternative: render after glamour by matching its output. That is fragile.
- **`safe` before rendering.** The note is untrusted input. glamour adds its own escape codes after the strip, so they stay intact.
- **Style.** Use glamour's `dark` standard style. Choose `light` when bubbletea reports a light background (`tea.BackgroundColorMsg`). Request it with `tea.RequestBackgroundColor` in `Init`.
- **Cache.** The view keeps the rendered lines for the current step and width. It renders again only when the step or width changes. glamour is too slow to run on each key press.

### Scroll

```text
room = height - len(header) - 1 (help line)
j/k        scroll +-1
pgdown, space, ctrl+d   scroll + room
pgup, b, ctrl+u         scroll - room
g / G      0 / len(lines) - room
clamp scroll to [0, max(len(lines) - room, 0)]
step change -> scroll = 0
```

The header (position, heading, status, invalid reason) stays fixed. Only the note scrolls. This follows the spec's "position, heading, and key help stay visible".

### Removals

- `internal/tour/diff.go`, its tests, `testdata/*.diff`.
- `cmd/zide/diff.go`, the `diff` entry in `main.go`, `tour-view --diff`, and the `tour.Changes` check in `startTour`.

### Dependencies

- `charm.land/glamour/v2` v2.0.1.
- `github.com/AlexanderGrooff/mermaid-ascii` at the latest pseudo-version. Import only `pkg/render` and `pkg/diagram`.

## Risks / Trade-offs

- [mermaid-ascii has no tags, and upstream changes can break the build] -> Pin the pseudo-version in `go.mod`. A golden test renders one flowchart.
- [mermaid-ascii logs through logrus and uses gookit/color (seen in `pkg/graph`), and this can damage the TUI] -> Set `logrus.SetOutput(io.Discard)` once at startup and pass `diagram.DefaultConfig()`. A test makes sure the rendered output has no escape codes.
- [mermaid-ascii `pkg/graph/mapping_edge.go` writes to stdout with `fmt.Printf`, which bypasses the logrus discard and can damage the TUI] -> Accepted. We have not seen it trigger. Consider an upstream fix.
- [mermaid-ascii panics on bad input] -> Recover around `RenderDiagram` and treat a panic as an error. This shows the source (spec: unsupported diagram).
- [A wide diagram is wider than the pane] -> The existing `MaxWidth` clip cuts it. The skill tells the harness to keep diagrams small.
- [A larger binary from glamour, chroma, and goldmark] -> Accepted.
- [Removing `base` and the coverage step means that zide no longer finds a hunk that the harness skipped] -> Accepted by the user. The skill tells the harness to cover all of the change.

## Migration Plan

Old tours still parse: `base` is ignored. A step that has only `file` shows an empty note area. Before archive, update the `change-tour` spec Purpose. It must not say "no change left out".
