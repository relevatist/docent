# Spec Delta

## Purpose

Walks the user through a code change one step at a time, in an order that helps understanding, with the editor following each step and no change left out.

## ADDED Requirements

### Requirement: Tour file format
A tour SHALL be a JSON object with optional `title`, optional `base` (a git revision), and a non-empty `steps` array. Each step SHALL have a `file`, optional `lines` as `[start, end]`, and optional `note`. A step without `lines` SHALL cover the whole file.

#### Scenario: Valid tour
- **WHEN** zide reads `{"title":"Retry","base":"main","steps":[{"file":"a.go","lines":[1,40],"note":"New type"}]}`
- **THEN** the tour has one step for `a.go` lines 1 to 40 with note `New type`, compared against `main`

#### Scenario: Invalid JSON
- **WHEN** the tour file is not valid JSON or has no steps
- **THEN** zide prints an error with the reason and exits non-zero without opening a pane

### Requirement: Start a tour
`zide tour <file>` SHALL open a pane named `tour` in the caller's tab, stacked with the shell pane, and show the first step. `zide tour -` SHALL read the tour from standard input. The harness and shell panes SHALL stay visible.

#### Scenario: Harness starts a tour
- **WHEN** the harness runs `zide tour /tmp/tour.json`
- **THEN** a `tour` pane opens stacked with the shell, showing step 1, and the harness pane stays visible

#### Scenario: Tour from stdin
- **WHEN** the harness pipes a tour to `zide tour -`
- **THEN** the tour pane opens with that tour

### Requirement: Diff against base
zide SHALL compare the working tree against the tour's `base`, or `HEAD` when `base` is absent. Staged and unstaged changes SHALL both count.

#### Scenario: No base
- **WHEN** a tour has no `base` and the working tree has unstaged edits
- **THEN** the tour shows those edits as changes

### Requirement: Step view
Each step SHALL show its position (for example `2/5`), file, line range, and note. Changed lines in the step's range SHALL show as a diff. Unchanged lines in the range SHALL show as plain context.

#### Scenario: Step on changed code
- **WHEN** a step's range contains a hunk
- **THEN** the step view shows added and removed lines for that hunk

#### Scenario: Step on unchanged code
- **WHEN** a step points at a caller that the change did not touch
- **THEN** the step view shows those lines as plain context

### Requirement: Editor follows the tour
On entering a step, zide SHALL open the step's file in the editor at the step's first line, without moving focus away from the tour pane.

#### Scenario: Next step
- **WHEN** the user moves to a step for `b.go` lines 88 to 102
- **THEN** the editor shows `b.go` at line 88 and focus stays on the tour pane

### Requirement: User controls navigation
The tour SHALL move only on user input. Keys: `n` next step, `p` previous step, `o` open the step in the editor again, `s` send the step as a reference to the harness, `q` close the tour pane. At the first or last step, `p` or `n` SHALL do nothing.

#### Scenario: Last step
- **WHEN** the user is on the last step and presses `n`
- **THEN** the tour stays on the last step

#### Scenario: Send step to harness
- **WHEN** the user presses `s` on a step for `a.go` lines 1 to 40
- **THEN** the harness input receives the reference for `a.go:1-40` and nothing is submitted

#### Scenario: Close
- **WHEN** the user presses `q`
- **THEN** the tour pane closes and the shell pane takes its place

### Requirement: Show uncovered hunks
zide SHALL add a final step titled `Not covered by this tour` that lists every hunk no step covers. A hunk SHALL count as covered when a step for the same file overlaps it, or a step names the file without `lines`. When all hunks are covered, zide SHALL NOT add this step.

#### Scenario: Skipped hunk
- **WHEN** the change has hunks in `a.go` and `c.go` and the tour only has steps for `a.go`
- **THEN** the last step lists the `c.go` hunks

#### Scenario: Full coverage
- **WHEN** every hunk overlaps a step
- **THEN** the tour has no `Not covered by this tour` step

### Requirement: Invalid steps
A step whose file does not exist, or whose range is outside the file, SHALL show as invalid with the reason. The tour SHALL still run and the editor SHALL NOT move for that step.

#### Scenario: Missing file
- **WHEN** a step names `gone.go`, which does not exist
- **THEN** the step shows `invalid: gone.go not found` and the editor stays where it is

### Requirement: Plain diff walk-through
`zide diff [git diff args]` SHALL open a tour with one step per hunk of `git diff [args]`, in git's order, with no notes.

#### Scenario: Shell starts a diff
- **WHEN** the user runs `zide diff` in the shell pane with three changed hunks
- **THEN** a tour pane opens with three steps, one per hunk

#### Scenario: No changes
- **WHEN** the user runs `zide diff` and there are no changes
- **THEN** zide prints `no changes` and opens no pane
