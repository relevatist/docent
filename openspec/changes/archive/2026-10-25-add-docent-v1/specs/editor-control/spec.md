# Spec Delta

## Purpose

Lets the harness, the shell, and the change tour move the editor to a file and line and reload changed files, so the editor always shows the code under discussion.

## ADDED Requirements

### Requirement: Open a file at a line
`docent open <file>[:<line>]` SHALL make the editor in the caller's tab show `file`, with the cursor on `line` when given. Relative paths SHALL resolve against the caller's working directory. docent SHALL first return the editor to normal mode so the command is not typed into the buffer.

#### Scenario: Open with line
- **WHEN** the harness runs `docent open src/main.go:42`
- **THEN** the editor shows `src/main.go` with the cursor on line 42

#### Scenario: Open without line
- **WHEN** the harness runs `docent open README.md`
- **THEN** the editor shows `README.md`

#### Scenario: Editor in insert mode
- **WHEN** the editor is in insert mode and `docent open a.go:3` runs
- **THEN** no text is inserted into the current buffer and the editor shows `a.go` at line 3

#### Scenario: File does not exist
- **WHEN** `docent open missing.go` runs and `missing.go` does not exist
- **THEN** docent prints an error naming the file, exits non-zero, and sends nothing to the editor

### Requirement: Open does not lose unsaved work
Opening a file SHALL NOT discard unsaved changes in other buffers.

#### Scenario: Unsaved buffer
- **WHEN** buffer `a.go` has unsaved changes and `docent open b.go` runs
- **THEN** the editor shows `b.go` and `a.go` keeps its unsaved changes

### Requirement: Reload changed files
`docent reload` SHALL make the editor reload all buffers from disk. Reload SHALL discard unsaved changes in every buffer, because Helix cannot reload without doing so. The skill, `docent reload --help`, and the README SHALL state this.

#### Scenario: File changed on disk
- **WHEN** the harness edits `a.go` on disk and runs `docent reload`
- **THEN** the editor shows the new content of `a.go`

#### Scenario: Modified buffer
- **WHEN** buffer `b.go` has unsaved changes and `docent reload` runs
- **THEN** `b.go` shows its content on disk and the unsaved changes are gone

#### Scenario: Warning is documented
- **WHEN** the user runs `docent reload --help`
- **THEN** the help text states that unsaved changes in all buffers are discarded

### Requirement: Focus stays with the user
`docent open` and `docent reload` SHALL NOT move focus to the editor pane unless `--focus` is given.

#### Scenario: Harness opens a file
- **WHEN** the user types in the harness pane and the harness runs `docent open a.go`
- **THEN** focus stays on the harness pane

#### Scenario: Focus flag
- **WHEN** `docent open --focus a.go` runs from the shell pane
- **THEN** focus moves to the editor pane
