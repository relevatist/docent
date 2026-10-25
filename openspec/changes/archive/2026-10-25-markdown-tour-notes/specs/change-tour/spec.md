# Spec Delta

## MODIFIED Requirements

### Requirement: Tour file format
A tour SHALL be a JSON object with optional `title` and a non-empty `steps` array. Each step SHALL have an optional `file`, optional `lines` as `[start, end]`, and optional `note` in Markdown. A step SHALL have a `file`, a `note`, or both. `lines` SHALL require `file`. A step with a `file` and without `lines` SHALL cover the whole file. zide SHALL ignore unknown fields.

#### Scenario: Valid tour
- **WHEN** zide reads `{"title":"Retry","steps":[{"file":"a.go","lines":[1,40],"note":"New type"}]}`
- **THEN** the tour has one step for `a.go` lines 1 to 40 with note `New type`

#### Scenario: Note-only step
- **WHEN** zide reads `{"steps":[{"note":"## Overview"}]}`
- **THEN** the tour has one step with no file and note `## Overview`

#### Scenario: Empty step
- **WHEN** a step has neither `file` nor `note`
- **THEN** zide prints an error that names the step and exits non-zero without opening a pane

#### Scenario: Lines without file
- **WHEN** a step has `lines` and no `file`
- **THEN** zide prints an error that names the step and exits non-zero without opening a pane

#### Scenario: Old tour with base
- **WHEN** zide reads a tour that has a `base` field
- **THEN** zide ignores `base` and runs the tour

#### Scenario: Invalid JSON
- **WHEN** the tour file is not valid JSON or has no steps
- **THEN** zide prints an error with the reason and exits non-zero without opening a pane

### Requirement: Editor follows the tour
On entering a step that has a file, zide SHALL open the step's file in the editor at the step's first line, without moving focus away from the tour pane. On entering a step without a file, the editor SHALL stay where it is.

#### Scenario: Next step
- **WHEN** the user moves to a step for `b.go` lines 88 to 102
- **THEN** the editor shows `b.go` at line 88 and focus stays on the tour pane

#### Scenario: Step without a file
- **WHEN** the user moves to a step with no file
- **THEN** the editor stays where it is

### Requirement: User controls navigation
The tour SHALL move only on user input. Keys: `n` next step, `p` previous step, `o` open the step in the editor again, `s` send the step as a reference to the harness, `q` close the tour pane. At the first or last step, `p` or `n` SHALL do nothing. On a step without a file or with an invalid file, `o` and `s` SHALL do nothing.

#### Scenario: Last step
- **WHEN** the user is on the last step and presses `n`
- **THEN** the tour stays on the last step

#### Scenario: Send step to harness
- **WHEN** the user presses `s` on a step for `a.go` lines 1 to 40
- **THEN** the harness input receives the reference for `a.go:1-40` and nothing is submitted

#### Scenario: Send on a note-only step
- **WHEN** the user presses `s` on a step with no file
- **THEN** nothing is sent to the harness

#### Scenario: Close
- **WHEN** the user presses `q`
- **THEN** the tour pane closes and the shell pane takes its place

## ADDED Requirements

### Requirement: Markdown step view
Each step SHALL show its position (for example `2/5`), its heading, and its note rendered as Markdown and wrapped to the pane width. The heading SHALL be the file and line range, or the tour title when the step has no file. The step view SHALL NOT show the file content or a diff of it.

#### Scenario: Step with a file
- **WHEN** the user is on step 2 of 5 for `a.go` lines 1 to 40
- **THEN** the pane shows `2/5  a.go:1-40` and the rendered note, and no lines of `a.go`

#### Scenario: Markdown note
- **WHEN** a note has a heading, a list, and a fenced `diff` block
- **THEN** the pane shows them formatted, with added and removed lines in the diff block in different colors

#### Scenario: Note-only step
- **WHEN** the user is on a step with no file in a tour titled `Retry`
- **THEN** the pane shows the position, `Retry`, and the rendered note

#### Scenario: Control characters in a note
- **WHEN** a note contains a terminal escape sequence
- **THEN** the pane shows a replacement character in its place and the terminal does not act on it

### Requirement: Scroll the step view
When the step view is taller than the pane, the user SHALL be able to scroll it. Keys: `j` or `down` one line down, `k` or `up` one line up, `pgdown`, `space`, or `ctrl+d` one page down, `pgup`, `b`, or `ctrl+u` one page up, `g` to the top, `G` to the bottom. A page SHALL be the number of visible view lines. The position, heading, and key help SHALL stay visible. Scroll SHALL stop at the top and at the last line. Moving to another step SHALL reset scroll to the top.

#### Scenario: Page down
- **WHEN** the view shows 20 lines of a 50-line note at the top and the user presses `pgdown`
- **THEN** the view shows lines 21 to 40

#### Scenario: Bottom
- **WHEN** the user presses `G`
- **THEN** the view shows the last line of the note

#### Scenario: Page up at the top
- **WHEN** the view is at the top and the user presses `pgup`
- **THEN** the view stays at the top

#### Scenario: New step
- **WHEN** the user has scrolled down and presses `n`
- **THEN** the next step shows from its top

### Requirement: Mermaid diagrams in notes
A fenced `mermaid` block in a note SHALL show as a text diagram. Flowchart, sequence, and ER diagrams SHALL be supported. When zide cannot draw a block, it SHALL show the block's source and the reason.

#### Scenario: Flowchart
- **WHEN** a note has a `mermaid` block with `graph LR` and `A --> B`
- **THEN** the pane shows boxes `A` and `B` joined by an arrow

#### Scenario: Unsupported diagram
- **WHEN** a note has a `mermaid` block that zide cannot draw
- **THEN** the pane shows the block's source and the reason it was not drawn

## REMOVED Requirements

### Requirement: Step view
**Reason**: The step view showed a diff of the step's lines. Markdown notes replace it.
**Migration**: See `Markdown step view`. Put `diff` blocks in notes where they help.

### Requirement: Diff against base
**Reason**: The pane no longer shows diffs, so zide does not need to read git.
**Migration**: Remove `base` from tours. zide ignores the field if it is present. To show what changed, put a `diff` block in the step's note.

### Requirement: Show uncovered hunks
**Reason**: Coverage requires the diff against base, which this change removes. The harness writes the tour and decides what to explain.
**Migration**: None. The skill tells the harness to explain all of the change.

### Requirement: Plain diff walk-through
**Reason**: Without a diff body, hunk-only steps have no content to show.
**Migration**: Use `git diff` in the shell pane, or ask the harness for a tour.
