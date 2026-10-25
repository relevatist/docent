# harness-input Specification

## Purpose

Lets the user send file/line references and selected text from the editor into the harness prompt, where the user reviews and submits it.

## Requirements

### Requirement: Send a file/line reference
`zide send ref <file>[:<start>[-<end>]]` SHALL paste a reference to that file and line range into the harness input in the caller's tab. The reference format SHALL match the harness kind: `@file:start-end` for `claude` and `omp`, and `file:start-end` for other harnesses. The kind SHALL be the kind shown in the harness pane command, or else the base name of the first word of that command. A single-line range SHALL be written as `file:start`. The reference SHALL end with one space.

#### Scenario: Range reference to Claude Code
- **WHEN** the harness is `claude` and the user runs `zide send ref src/a.go:10-20`
- **THEN** the harness input receives `@src/a.go:10-20 `

#### Scenario: Single line
- **WHEN** the harness is `claude` and the user runs `zide send ref src/a.go:10-10`
- **THEN** the harness input receives `@src/a.go:10 `

#### Scenario: Unknown harness
- **WHEN** the harness is `aider` and the user runs `zide send ref src/a.go:10-20`
- **THEN** the harness input receives `src/a.go:10-20 `

#### Scenario: Wrapped harness with a kind
- **WHEN** the tab was launched with an alias whose command is `sandbox-exec -f omp.sb omp` and whose kind is `omp`, and the user runs `zide send ref src/a.go:10-20`
- **THEN** the harness input receives `@src/a.go:10-20 `

### Requirement: Send text
`zide send text` SHALL paste its standard input into the harness input in the caller's tab, unchanged. As an exception, `zide send` SHALL remove ESC characters from all pasted text, so the text cannot end the bracketed paste early.

#### Scenario: Selected text
- **WHEN** the user pipes a Helix selection to `zide send text`
- **THEN** the harness input receives the selected text

#### Scenario: Text that ends the paste
- **WHEN** `zide send text` receives `a` ESC `[201~` CR `b`
- **THEN** the harness input receives `a[201~` CR `b` inside one paste and the harness does not submit it

#### Scenario: Empty input
- **WHEN** `zide send text` receives empty standard input
- **THEN** zide sends nothing and exits zero

### Requirement: Never submit
`zide send` SHALL NOT submit the harness input. The user SHALL submit it. This guarantee requires a harness that enables bracketed paste (Claude Code, omp, and codex do).

#### Scenario: After sending
- **WHEN** `zide send ref a.go:1` completes
- **THEN** the text is in the harness input and the harness has not received an Enter key

### Requirement: Focus after send
`zide send` SHALL move focus to the harness pane by default. The config key `focus_on_send = false` SHALL disable this.

#### Scenario: Default focus
- **WHEN** the user sends a reference from the editor
- **THEN** focus moves to the harness pane

#### Scenario: Focus disabled
- **WHEN** the config has `focus_on_send = false` and the user sends a reference
- **THEN** focus stays on the editor pane

### Requirement: Helix keybindings
zide SHALL ship Helix keybindings that send the primary selection's line range as a reference and send the primary selection as text.

#### Scenario: Send reference from Helix
- **WHEN** the user selects lines 10 to 20 of `a.go` in Helix and presses the reference binding
- **THEN** the harness input receives the reference for `a.go:10-20`
